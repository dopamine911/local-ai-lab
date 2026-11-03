package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// CompareSkillHelp is the usage text (the script printed the same on a wrong call).
const CompareSkillHelp = `Does a skill (instruction file) change tool-use quality? A/B test with tool-eval-bench.

Usage: lab compare-skill <path/to/SKILL.md> [model ...]        (default models: coder flint)
Example:
  lab compare-skill instructions/the-craft@3c329d0.md coder flint

For each model it runs the full suite (69 scenarios) twice under identical conditions
(temperature 0, seed 42): once with the built-in system prompt, once with the skill
appended to it. Both runs go to MLflow; a side-by-side table is printed and saved in
tool-eval/compare/. Takes about 20-40 minutes per model per arm.
`

var skillNameLine = regexp.MustCompile(`(?m)^name: *(.*)$`)

// CompareSkill is `lab compare-skill <skill> [model ...]` (port of tuning/compare-skill). The skill path is relative to
// tuning/, like the script's. The table comes from tuning/tool-eval/compare.py, unchanged.
func CompareSkill(cfg *config.Config, skill string, models []string) error {
	if err := os.Chdir(cfg.Tuning()); err != nil {
		return err
	}
	raw, err := os.ReadFile(skill)
	if err != nil {
		fmt.Print(CompareSkillHelp)
		return exitErr(1)
	}
	if len(models) == 0 {
		models = []string{"coder", "flint"}
	}
	name := filepath.Base(strings.TrimSuffix(skill, ".md"))
	if m := skillNameLine.FindSubmatch(raw); m != nil && len(m[1]) > 0 {
		name = string(m[1])
	}
	if err := os.MkdirAll("tool-eval/compare", 0o755); err != nil {
		return err
	}
	report := "tool-eval/compare/" + time.Now().Format("2006-01-02-1504") + "-" + name + ".md"
	sum := sha256.Sum256(raw)
	head := "# Skill comparison: " + name + "\n\n" +
		"Skill file: `" + skill + "` (sha256 " + hex.EncodeToString(sum[:])[:12] + "). tool-eval-bench 3.0.0, full suite, temperature 0, seed 42.\n"
	if err := os.WriteFile(report, []byte(head), 0o644); err != nil {
		return err
	}

	// Each arm is its own run, as each ./quality was its own process: no setting from one arm may reach the next
	// (the skill, its name and hash, the Rerun value).
	outer := map[string]string{}
	for _, k := range []string{"LAB_RERUN", "LAB_RERUN_CWD", "SKILL", "SKILL_NAME", "SKILL_SHA"} {
		if v, ok := os.LookupEnv(k); ok {
			outer[k] = v
		}
	}
	fresh := func(withSkill bool) {
		for _, k := range []string{"LAB_RERUN", "LAB_RERUN_CWD", "SKILL", "SKILL_NAME", "SKILL_SHA"} {
			if v, ok := outer[k]; ok && (k == "LAB_RERUN" || k == "LAB_RERUN_CWD") {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
		if withSkill {
			os.Setenv("SKILL", skill)
		}
	}
	for _, m := range models {
		fmt.Printf("\n################ %s: without skill\n", m)
		fresh(false)
		base, err := qualityRun(cfg, m, "skill A/B: "+name+" OFF", []string{"--full"})
		if err != nil {
			return err
		}
		fmt.Printf("\n################ %s: with %s\n", m, name)
		fresh(true)
		with, err := qualityRun(cfg, m, "skill A/B: "+name+" ON", []string{"--full"})
		if err != nil {
			return err
		}
		if err := os.Chdir(cfg.Tuning()); err != nil { // qualityRun moved there too; the paths are relative to it
			return err
		}
		f, err := os.OpenFile(report, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		c := exec.Command("python3", "tool-eval/compare.py", m, base, with, name)
		c.Stdout = io.MultiWriter(os.Stdout, f) // like `| tee -a "$REPORT"`
		c.Stderr = os.Stderr
		runErr := c.Run()
		f.Close()
		if runErr != nil {
			return exitErr(exitCode(runErr))
		}
	}
	fresh(false)
	fmt.Printf("\nSaved: %s\n", report)
	return nil
}
