package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// Terminal-Bench-Local (Core-19), pinned; the runner itself stays Python (tuning/terminal-bench, own licence).
const (
	tbRepo = "https://github.com/kyuz0/terminal-bench-mini.git"
	tbPin  = "60739af" // Terminal-Bench-Local commit, 2026-10-27
	tbDir  = "terminal-bench"
)

// TBenchHelp is the usage text (the script printed the same on a wrong call).
const TBenchHelp = `Terminal-Bench-Local (Core-19): 19 real terminal tasks, graded pass/fail.
Each task gets up to 2 attempts (pass@2). Results are logged to MLflow.

  lab tbench setup                    # once: download the benchmark (pinned version)
  lab tbench doctor                   # checks Docker, the endpoint and the tasks; no inference
  lab tbench smoke "first try"        # 1 task, about 10-30 minutes
  lab tbench full "coder MTP2"        # all 19 tasks, about 10 hours: run it inside tmux
  lab tbench log "note"               # log the newest finished result to MLflow (done
                                      #   automatically after smoke/full)

Model: MODEL=coder-tb by default (QUANT and PROFILE describe it in the results):
  MODEL=coder-2 lab tbench smoke "card 2"
  TARGET=marvin lab tbench smoke "flash-next"     (marvin's model, labelled as marvin)
  TARGET=marvin-strata lab tbench full "strata"   (Strata on marvin; start it first: marvin's strata-test.sh start)
  TARGET=marvin-fork lab tbench full "fork"       (our UnbiasedStrata build on marvin; start it first: fork.sh start)
  RERUN=1 TARGET=marvin lab tbench full "again"   (same model and profile again: without RERUN=1 the suite skips it)
`

// tbSettings: what the run describes itself as (the script's shell variables), from TARGET and the environment.
type tbSettings struct {
	Model, ModelName, Quant, Profile, Platform, PlatformName, Context, Endpoint, Engine, Backend, BackendVersion string
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func tbFromEnv() tbSettings {
	s := tbSettings{Model: os.Getenv("MODEL"), ModelName: os.Getenv("MODEL_NAME"), Quant: os.Getenv("QUANT"),
		Profile: os.Getenv("PROFILE"), Platform: os.Getenv("PLATFORM"), PlatformName: os.Getenv("PLATFORM_NAME"),
		Context: os.Getenv("CONTEXT_LENGTH"), Endpoint: os.Getenv("ENDPOINT"), Engine: os.Getenv("ENGINE"),
		Backend: os.Getenv("BACKEND"), BackendVersion: os.Getenv("BACKEND_VERSION")}
	or := func(p *string, def string) {
		if *p == "" {
			*p = def
		}
	}
	strix := "marvin, Radeon 8060S (Strix Halo)"
	switch os.Getenv("TARGET") {
	case "marvin": // marvin's big model through zaphod (marvin/big), labelled as marvin
		or(&s.Model, "marvin/big")
		or(&s.ModelName, "Qwen3.8-Flash-Next")
		or(&s.Quant, "UD-IQ4_XS")
		or(&s.Profile, "mtp3-128k-np2")
		s.Platform, s.PlatformName = "strixhalo", strix
		or(&s.Context, "131072") // peer models don't advertise it through zaphod
	case "marvin-strata": // Strata on marvin, started with marvin's ~/strata-test/strata-test.sh start
		or(&s.Model, "qwen3.8-flash-next-unsloth-ud-iq4_xs")
		or(&s.ModelName, "Qwen3.8-Flash-Next")
		or(&s.Quant, "UD-IQ4_XS")
		or(&s.Profile, "strata0.2.7-mtp-128k-np2")
		s.Platform, s.PlatformName = "strixhalo", strix
		or(&s.Context, "131072")
		s.Endpoint, s.Engine, s.Backend, s.BackendVersion = "http://marvin:8099/v1", "strata", "hip", "rocm-gfx1151"
		os.Setenv("MACHINE", "marvin")
	case "marvin-fork": // our UnbiasedStrata gfx1151 build on marvin, started with fork.sh start
		or(&s.Model, "big-fork")
		or(&s.ModelName, "Qwen3.8-Flash-Next")
		or(&s.Quant, "UD-IQ4_XS")
		or(&s.Profile, "unbiasedstrata-gfx1151-mtp-128k-np2-medium")
		s.Platform, s.PlatformName = "strixhalo", strix
		or(&s.Context, "131072")
		s.Endpoint, s.Engine, s.Backend, s.BackendVersion = "http://marvin:8099/v1", "unbiasedstrata", "hip", "rocm-10.1.0"
		os.Setenv("MACHINE", "marvin")
	}
	or(&s.Endpoint, "http://127.0.0.1:8080/v1")
	or(&s.Engine, "llama.cpp")
	or(&s.Backend, "vulkan")
	or(&s.BackendVersion, "mesa-26.0")
	or(&s.Platform, "r9700")
	or(&s.PlatformName, "zaphod, Radeon AI PRO R9700")
	or(&s.Model, "coder-tb") // card 1, 256K context, q8_0 context memory, MTP
	or(&s.ModelName, "Qwen3.8-27B")
	or(&s.Quant, "UD-Q4_K_XL")
	or(&s.Profile, "mtp2-256k-kvq8")
	return s
}

func (s tbSettings) ctx() []string {
	if s.Context == "" {
		return nil
	}
	return []string{"--context-length", s.Context}
}

// TBench is `lab tbench setup|doctor|smoke|full|log` (port of tuning/tbench). Same settings (TARGET, MODEL, PROFILE,
// QUANT, SKILL, RERUN, ...), same runner and the same MLflow fields.
func TBench(cfg *config.Config, args []string) error {
	if err := enterTuning(cfg, "tbench", args...); err != nil {
		return err
	}
	s := tbFromEnv()
	usage := func() error { fmt.Print(TBenchHelp); return exitErr(1) }
	if skill := os.Getenv("SKILL"); skill != "" {
		raw, err := os.ReadFile(skill)
		if err != nil {
			fmt.Println("SKILL file not found: " + skill)
			return exitErr(1)
		}
		sc, _ := os.ReadFile(filepath.Join(tbDir, "compat", "harbor_prompt", "sitecustomize.py"))
		if !strings.Contains(string(sc), "local-ai-lab: extra instructions") {
			fmt.Println("Skill support not installed. Run: lab tbench setup")
			return exitErr(1)
		}
		abs, _ := filepath.Abs(skill)
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		sum := sha256.Sum256(raw)
		name := strings.TrimSuffix(filepath.Base(skill), ".md")
		os.Setenv("TBENCH_EXTRA_INSTRUCTIONS", abs)
		os.Setenv("SKILL_NAME", name)
		os.Setenv("SKILL_SHA", hex.EncodeToString(sum[:])[:12])
		s.Profile += "+" + name
		fmt.Printf("== with instructions: %s (%s); profile %s\n", name, os.Getenv("SKILL_SHA"), s.Profile)
	} else {
		os.Unsetenv("TBENCH_EXTRA_INSTRUCTIONS")
	}
	what, note := "", ""
	if len(args) > 0 {
		what = args[0]
	}
	if len(args) > 1 {
		note = args[1]
	}
	switch what {
	case "setup":
		return tbSetup()
	case "doctor":
		if err := tbNeedRepo(); err != nil {
			return err
		}
		if err := tbKey(cfg, s); err != nil {
			return err
		}
		return tbRunner(append([]string{"doctor", "--suite", "core19", "--tier", "smoke",
			"--endpoint", s.Endpoint, "--model", s.Model}, s.ctx()...))
	case "smoke":
		if note == "" {
			return usage()
		}
		return tbTier(cfg, s, "smoke", note)
	case "full":
		if note == "" {
			return usage()
		}
		if os.Getenv("TMUX") == "" {
			fmt.Println("A full run takes about 10 hours. Start it inside tmux:")
			fmt.Println("  tmux new -s tbench")
			fmt.Printf("  cd ~/homelab/tuning && lab tbench full %q\n", note)
			fmt.Println("Detach with Ctrl-b then d; reattach with: tmux attach -t tbench")
			return exitErr(1)
		}
		return tbTier(cfg, s, "full", note)
	case "log":
		if note == "" {
			return usage()
		}
		return tbLogLatest(cfg, s, note)
	}
	return usage()
}

func tbNeedRepo() error {
	if st, err := os.Stat(tbDir); err != nil || !st.IsDir() {
		fmt.Println("Run lab tbench setup first.")
		return exitErr(1)
	}
	return nil
}

// tbKey: the API key for the endpoint, in TBENCH_API_KEY (never printed).
func tbKey(cfg *config.Config, s tbSettings) error {
	var key string
	switch s.Engine {
	case "strata": // made per Strata start
		b, err := exec.Command("ssh", "marvin", "cat", "strata-test/key").Output()
		if err != nil {
			return fmt.Errorf("Strata key from marvin: %v", err)
		}
		key = strings.TrimRight(string(b), "\n")
	case "unbiasedstrata": // fork.sh start
		b, err := exec.Command("ssh", "marvin", "cat", "strata-fork/key").Output()
		if err != nil {
			return fmt.Errorf("fork key from marvin: %v", err)
		}
		key = strings.TrimRight(string(b), "\n")
	default:
		k, err := zaphodKey(cfg)
		if err != nil {
			return err
		}
		key = k
	}
	os.Setenv("TBENCH_API_KEY", key)
	return nil
}

// tbRunner runs ./terminal_bench.py in terminal-bench/ with the terminal attached.
func tbRunner(args []string) error {
	c := exec.Command("./terminal_bench.py", args...)
	c.Dir = tbDir
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return exitErr(exitCode(err))
	}
	return nil
}

func tbSetup() error {
	if _, err := os.Stat(tbDir); err != nil {
		if err := runAttached("git", "clone", "-q", tbRepo, tbDir); err != nil {
			return err
		}
	}
	for _, a := range [][]string{{"fetch", "-q"}, {"checkout", "-q", tbPin},
		{"log", "-1", "--format=Terminal-Bench-Local pinned at %h (%cd)"}} {
		c := exec.Command("git", a...)
		c.Dir = tbDir
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			return exitErr(exitCode(err))
		}
	}
	return runAttached("python3", "tbench-skill-patch.py", tbDir) // adds skill support to the runner's prompt hook
}

func runAttached(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return exitErr(exitCode(err))
	}
	return nil
}

// tbTier runs one tier, then logs it, but only if this call made a new job (a skipped run would otherwise log the
// old result under the new note).
func tbTier(cfg *config.Config, s tbSettings, tier, note string) error {
	if err := tbNeedRepo(); err != nil {
		return err
	}
	if err := tbKey(cfg, s); err != nil {
		return err
	}
	os.Setenv("RUN_START", fmt.Sprint(time.Now().Unix())) // lab log reads real-use speed for this window
	args := []string{"run", "--suite", "core19", "--tier", tier}
	if os.Getenv("RERUN") == "1" { // run every task again even if this model / profile already has results
		args = append(args, "--rerun")
	}
	args = append(args, "--endpoint", s.Endpoint, "--model", s.Model)
	args = append(args, s.ctx()...)
	args = append(args, "--platform", s.Platform, "--platform-name", s.PlatformName,
		"--model-name", s.ModelName, "--engine", s.Engine, "--backend", s.Backend, "--backend-version", s.BackendVersion,
		"--quant", s.Quant, "--inference-profile", s.Profile)
	mark := time.Now()
	if err := tbRunner(args); err != nil {
		return err
	}
	if !tbNewJob(mark) {
		fmt.Printf("No new job ran (results for %s / %s already exist?), so nothing logged. Again: RERUN=1\n", s.Model, s.Profile)
		return exitErr(1)
	}
	return tbLogLatest(cfg, s, note)
}

// tbNewJob: a folder in terminal-bench/jobs changed after mark (like find -newer a file made at mark).
func tbNewJob(mark time.Time) bool {
	es, _ := os.ReadDir(filepath.Join(tbDir, "jobs"))
	for _, e := range es {
		if fi, err := e.Info(); err == nil && e.IsDir() && fi.ModTime().After(mark) {
			return true
		}
	}
	return false
}

// tbLogLatest logs the result whose model id and profile are this run's (the suite rewrites every summary.json when a
// run ends, so file times can't tell which run it was).
func tbLogLatest(cfg *config.Config, s tbSettings, note string) error {
	var found []string
	filepath.WalkDir(filepath.Join(tbDir, "results", "suites"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "summary.json" || !strings.Contains(p, s.Platform) {
			return nil
		}
		var sum struct {
			Model struct {
				ID string `json:"id"`
			} `json:"model"`
			Profile string `json:"inference_profile"`
		}
		if b, err := os.ReadFile(p); err == nil && json.Unmarshal(b, &sum) == nil &&
			sum.Model.ID == s.Model && sum.Profile == s.Profile {
			found = append(found, p)
		}
		return nil
	})
	if len(found) == 0 {
		fmt.Printf("No finished %s result for %s / %s found yet.\n", s.Platform, s.Model, s.Profile)
		return exitErr(1)
	}
	if len(found) > 1 {
		fmt.Printf("More than one result for %s / %s:\n%s\n", s.Model, s.Profile, strings.Join(found, "\n"))
		return exitErr(1)
	}
	fmt.Println("== logging " + found[0])
	rc, err := Log(cfg, machineEnv(), s.Model, note, []string{found[0]})
	if err != nil {
		return err
	}
	if rc != 0 {
		return exitErr(rc)
	}
	return nil
}
