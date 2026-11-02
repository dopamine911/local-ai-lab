// Package bench: the benchmark commands (issue #6 step 4), ports of tuning/log, series, speed, quality, bench and kl.
// Every port keeps what reaches MLflow the same as the script it replaces, so old and new results stay comparable:
// the logging itself is still tuning/log_run.py, run in the same MLflow image with the same environment and mounts.
package bench

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
	"github.com/dopamine911/local-ai-lab/cli/internal/manifest"
)

//go:embed activity.sh
var activityScript string

var (
	viewLine  = regexp.MustCompile(`View (run|experiment) `)
	loggedRun = regexp.MustCompile(`Logged run [0-9a-f]+`)
	summaryRe = regexp.MustCompile(`^  (Terminal-Bench|tool-eval score|mean KLD|speed_|[a-z_]+ = )`)
)

// Log is `lab log <machine> <model> <note> [name=value ...] [files ...]` (port of tuning/log): logs one result to
// MLflow from the current folder (file arguments are relative to it), then sends the phone notification
// (NOTIFY=0 skips it). Returns log_run.py's exit code.
func Log(cfg *config.Config, machine, model, note string, rest []string) (int, error) {
	m := machine
	if strings.HasPrefix(model, "marvin/") { // peer models ("marvin/big") belong to that machine
		m = "marvin"
	}
	host, _ := os.Hostname()
	env := []string{
		"CONFIG_COMMIT=" + configCommit(cfg, m),
		"MANIFEST=" + manifestFor(cfg, m, host),
		"ACTIVITY=" + activityFor(cfg, m, host, model),
	}
	cwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	tuning := cfg.Tuning()
	args := []string{"run", "--rm", "--network", "host",
		"-e", "GIT_PYTHON_REFRESH=quiet", "-e", "CONFIG_COMMIT", "-e", "SKILL_NAME", "-e", "SKILL_SHA", "-e", "SERIES",
		"-e", "MANIFEST", "-e", "ACTIVITY", "-e", "MLFLOW_URL",
		"-v", filepath.Join(cfg.Repo, "setup") + ":/setup:ro",
		"--env-file", filepath.Join(cfg.Repo, "hub", ".env"),
		"-v", tuning + ":/tuning:ro", "-v", cwd + ":/work:ro", "-w", "/work",
		cfg.Images.MLflow, "python", "/tuning/log_run.py", machine, model, note} // as given, like the script's "$@"
	args = append(args, rest...)
	c := exec.Command("docker", args...)
	c.Env = append(os.Environ(), env...)
	out, rc := runFiltered(c)

	if os.Getenv("NOTIFY") != "0" {
		notify(cfg, m, model, note, out, rc)
	}
	return rc, nil
}

// configCommit: the last change to that machine's llama-swap folder ("<hash> <subject>"), or "unknown".
func configCommit(cfg *config.Config, m string) string {
	c := exec.Command("git", "log", "-1", "--format=%h %s", "--", ".")
	c.Dir = filepath.Join(cfg.Repo, m, "llama-swap")
	out, err := c.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimRight(string(out), "\n")
}

// manifestFor: machine and software facts from the machine that served the model (lab manifest there over ssh).
func manifestFor(cfg *config.Config, m, host string) string {
	if m == host {
		var b strings.Builder
		for i, f := range manifest.Collect(cfg.Container(host)) {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(f.Key + "=" + f.Value)
		}
		return b.String()
	}
	out, err := exec.Command("ssh", "-o", "ConnectTimeout=10", m, "~/.local/bin/lab", "manifest").Output()
	if err != nil { // the script printed nothing here; say why, on stderr (MLflow gets the same line as before)
		fmt.Fprintf(os.Stderr, "lab log: no manifest from %s (%v); is lab installed there? (cli/build.sh install)\n", m, err)
		return "host=" + m + " (unreachable)"
	}
	return strings.TrimRight(string(out), "\n")
}

// activityFor: real-use speed during the run, when the caller recorded RUN_START (quality, tbench).
func activityFor(cfg *config.Config, m, host, model string) string {
	start := os.Getenv("RUN_START")
	if start == "" {
		return ""
	}
	lm := model // model id on the serving machine ("marvin/big" -> "big")
	if _, after, ok := strings.Cut(model, "/"); ok {
		lm = after
	}
	end := strconv.FormatInt(time.Now().Unix(), 10)
	var c *exec.Cmd
	if m == host {
		c = exec.Command("bash", "-s", "--", lm, start, end, cfg.Images.Python)
		c.Stderr = os.Stderr
	} else {
		c = exec.Command("ssh", "-o", "ConnectTimeout=10", m, "bash", "-s", "--", lm, start, end, cfg.Images.Python)
	}
	c.Stdin = strings.NewReader(activityScript)
	out, _ := c.Output()
	return strings.TrimRight(string(out), "\n")
}

// runFiltered runs c with stdout and stderr together, drops MLflow's "View run/experiment" lines, prints the rest as
// it comes and returns it (with the exit code), like `2>&1 | grep -vE "View (run|experiment) " | tee`.
func runFiltered(c *exec.Cmd) (string, int) {
	pr, pw := io.Pipe()
	c.Stdout, c.Stderr = pw, pw
	var b strings.Builder
	done := make(chan struct{})
	go func() {
		r := bufio.NewReader(pr)
		for {
			line, err := r.ReadString('\n')
			if line != "" && !viewLine.MatchString(line) {
				fmt.Print(line)
				b.WriteString(line)
			}
			if err != nil {
				break
			}
		}
		close(done)
	}()
	rc := 0
	if err := c.Run(); err != nil {
		rc = exitCode(err)
	}
	pw.Close()
	<-done
	return b.String(), rc
}

func exitCode(err error) int {
	if e, ok := err.(*exec.ExitError); ok {
		if e.ExitCode() > 0 {
			return e.ExitCode()
		}
	}
	return 1
}

// notify: push notification to the phone (hub/notify, issue #11) with buttons (hub/actions/actiond.py): Rerun (zaphod
// runs started by a benchmark command that recorded LAB_RERUN; not marvin until reservations, #3), Skip, Open.
// A button that can't be made is left out; a failed notification never fails the run.
func notify(cfg *config.Config, m, model, note, out string, rc int) {
	runs := loggedRun.FindAllString(out, -1)
	ids := make([]string, len(runs))
	for i, r := range runs {
		ids[i] = strings.Fields(r)[2]
	}
	run := strings.Join(ids, "\n")
	var sum []string
	for _, l := range strings.Split(out, "\n") {
		if summaryRe.MatchString(l) && len(sum) < 6 {
			sum = append(sum, strings.TrimPrefix(l, "  "))
		}
	}
	summary := strings.Join(sum, "\n")
	actiond := filepath.Join(cfg.Repo, "hub", "actions", "actiond.py")
	notifyBin := filepath.Join(cfg.Repo, "hub", "notify")
	subject := model + " (" + m + "): " + note
	var btn []string
	button := func(args ...string) {
		if b, err := exec.Command(actiond, append([]string{"button"}, args...)...).Output(); err == nil {
			btn = append(btn, strings.TrimRight(string(b), "\n"))
		}
	}
	if m == "zaphod" && os.Getenv("LAB_RERUN") != "" {
		button("rerun", "Rerun", subject)
	}
	var c *exec.Cmd
	if rc == 0 && run != "" {
		button("skip", "Skip", run, subject)
		url := "https://mlflow.ai.dopamine911.com/#/runs/" + run
		btn = append(btn, "view, Open, "+url)
		if summary == "" {
			summary = "logged"
		}
		short := run
		if len(short) > 8 {
			short = short[:8]
		}
		c = exec.Command(notifyBin, "-g", "white_check_mark", "-c", url, "-a", strings.Join(btn, ";"),
			"Done: "+model+" ("+m+")", note+"\n"+summary+"\nMLflow run "+short)
	} else {
		a := []string{"-t", "lab-alert", "-p", "high", "-g", "warning"}
		if acts := strings.Join(btn, ";"); acts != "" {
			a = append(a, "-a", acts)
		}
		c = exec.Command(notifyBin, append(a, "Logging failed: "+model+" ("+m+")", note+"\n"+lastLines(out, 3))...)
	}
	_ = c.Run() // output to /dev/null, like the script
}

// lastLines: the last n lines of s, without the final newline (like "$(tail -n FILE)").
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
