package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

var frontmatter = regexp.MustCompile(`(?s)\A---\n.*?\n---\n`)

// Quality is `lab quality <model> <note> [--full] [tool-eval-bench options]` (port of tuning/quality): the tool-use
// test (tool-eval-bench) through llama-swap, logged with live_* speed for the run's window. Settings from the
// environment like the script: SKILL (instruction file added to the system prompt), SERIES, BASE_URL, API_KEY,
// MACHINE. extra goes to tool-eval-bench as given.
func Quality(cfg *config.Config, model, note string, extra []string) error {
	_, err := qualityRun(cfg, model, note, extra)
	return err
}

// qualityRun is Quality; it also returns the result file (tuning/tool-eval/runs/<date>-<model>.json).
func qualityRun(cfg *config.Config, model, note string, extra []string) (string, error) {
	if err := enterTuning(cfg, "quality", append([]string{model, note}, extra...)...); err != nil {
		return "", err
	}
	suite := []string{"--short"}
	if len(extra) > 0 && extra[0] == "--full" {
		suite, extra = nil, extra[1:]
	}
	key := os.Getenv("API_KEY")
	if key == "" {
		k, err := zaphodKey(cfg)
		if err != nil {
			return "", err
		}
		key = k
	}
	for _, d := range []string{"tool-eval/data", "tool-eval/runs"} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", err
		}
	}
	out := time.Now().Format("2006-01-02-1504") + "-" + strings.ReplaceAll(model, "/", "_") + ".json"
	var promptArgs []string
	if skill := os.Getenv("SKILL"); skill != "" {
		pf, err := skillPrompt(skill)
		if err != nil {
			return "", err
		}
		promptArgs = []string{"--system-prompt-file", "/app/" + pf}
	}

	os.Setenv("RUN_START", strconv.FormatInt(time.Now().Unix(), 10)) // lab log reads live speed for this window
	suiteName := "(full)"
	if suite != nil {
		suiteName = suite[0]
	}
	fmt.Printf("== tool-eval-bench on %s %s\n", model, suiteName)
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8080/v1"
	}
	pwd, _ := os.Getwd()
	args := []string{"run", "--rm", "--network", "host", "-u", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"-v", pwd + "/tool-eval/data:/app/data", "-v", pwd + "/tool-eval/runs:/app/runs",
		"-v", pwd + "/tool-eval/prompts:/app/tool-eval/prompts:ro",
		cfg.Images.ToolEval, "run"}
	args = append(args, suite...)
	args = append(args, "--seed", "42", "--base-url", base, "--api-key", key, "--model", model,
		"--no-live", "--json-file", "/app/runs/"+out)
	args = append(append(args, promptArgs...), extra...)
	logPath := "tool-eval/runs/" + strings.TrimSuffix(out, ".json") + ".log"
	lf, err := os.Create(logPath)
	if err != nil {
		return "", err
	}
	c := exec.Command("docker", args...)
	c.Stdout, c.Stderr = lf, lf
	runErr := c.Run()
	lf.Close()
	if runErr != nil {
		fmt.Println("tool-eval-bench failed:")
		b, _ := os.ReadFile(logPath)
		fmt.Println(lastLines(string(b), 5))
		return "", exitErr(1)
	}

	var res struct {
		ReportPath string `json:"report_path"`
	}
	b, err := os.ReadFile("tool-eval/runs/" + out)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return "", err
	}
	report := strings.ReplaceAll(res.ReportPath, "/app/", "tool-eval/")
	fmt.Printf("== results (full report with every conversation: %s)\n", report)
	rc, err := Log(cfg, machineEnv(), model, note, []string{"tool-eval/runs/" + out, report})
	if err != nil {
		return "", err
	}
	if rc != 0 {
		return "", exitErr(rc)
	}
	return "tool-eval/runs/" + out, nil
}

// skillPrompt builds the system prompt file for SKILL like the script: the built-in prompt, a blank line, then the
// instruction file without its frontmatter. Sets SKILL_NAME and SKILL_SHA (logged to MLflow). Returns its path.
func skillPrompt(skill string) (string, error) {
	raw, err := os.ReadFile(skill)
	if err != nil {
		fmt.Println("SKILL file not found: " + skill)
		return "", exitErr(1)
	}
	sum := sha256.Sum256(raw)
	name := strings.TrimSuffix(filepath.Base(skill), ".md")
	sha := hex.EncodeToString(sum[:])[:12]
	os.Setenv("SKILL_NAME", name)
	os.Setenv("SKILL_SHA", sha)
	if err := os.MkdirAll("tool-eval/prompts", 0o755); err != nil {
		return "", err
	}
	def, err := os.ReadFile("tool-eval/default-system-prompt.txt")
	if err != nil {
		return "", err
	}
	body := strings.TrimSpace(frontmatter.ReplaceAllString(string(raw), "")) // count=1: \A anchors it to the start
	pf := "tool-eval/prompts/" + name + "-" + sha + ".txt"
	text := string(def) + "\n\n" + body + "\n"
	if err := os.WriteFile(pf, []byte(text), 0o644); err != nil {
		return "", err
	}
	fmt.Printf("== with instructions: %s (%s, %d bytes of system prompt)\n", name, sha, len(text))
	return pf, nil
}
