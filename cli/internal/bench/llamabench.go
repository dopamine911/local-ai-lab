package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// Bench is `lab bench <card 1|2> <note> [llama-bench options]` (port of tuning/bench, zaphod): llama-bench on one card
// with llama-swap stopped, 3 repeats, saved to tuning/runs/<date>-card<N>.jsonl and logged. MODEL picks another
// file (default Qwen3.8-27B UD-Q4_K_XL). dryRun prints the commands and changes nothing.
func Bench(cfg *config.Config, dryRun bool, card, note string, extra []string) error {
	if err := enterTuning(cfg, "bench", append([]string{card, note}, extra...)...); err != nil {
		return err
	}
	var dev string
	switch card {
	case "1":
		dev = "renderD128"
	case "2":
		dev = "renderD129"
	default:
		fmt.Println("card must be 1 or 2")
		return exitErr(1)
	}
	model := os.Getenv("MODEL")
	if model == "" {
		model = "/models/Qwen3.8-27B-UD-Q4_K_XL.gguf"
	}
	out := "runs/" + time.Now().Format("2006-01-02-1504") + "-card" + card + ".jsonl"
	errPath := strings.TrimSuffix(out, ".jsonl") + ".err"
	return withSwapPaused(dryRun, func(g *guard) error {
		image, err := llamaSwapImage() // same image as llama-swap
		if err != nil {
			return err
		}
		fmt.Printf("== benchmarking card %s (%s, image %s)\n", card, filepath.Base(model), image)
		args := []string{"run", "--rm", "--device", "/dev/dri/" + dev, "-e", "GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1",
			"-v", "/srv/models:/models:ro", "--entrypoint", "llama-bench", image,
			"-m", model, "-ngl", "999", "-fa", "1", "-p", "512", "-n", "128", "-r", "3"}
		args = append(append(args, extra...), "-o", "jsonl")
		c := exec.Command("docker", args...)
		if dryRun {
			g.run(c)
			fmt.Printf("   output: %s (errors: %s)\n   would log: lab log zaphod card%s:%s %q %s\n",
				out, errPath, card, strings.TrimSuffix(filepath.Base(model), ".gguf"), note, out)
			return nil
		}
		of, err := os.Create(out)
		if err != nil {
			return err
		}
		ef, err := os.Create(errPath)
		if err != nil {
			of.Close()
			return err
		}
		c.Stdout, c.Stderr = of, ef
		runErr := g.run(c)
		of.Close()
		ef.Close()
		if runErr != nil {
			if g.wasInterrupted() { // stopped by a signal: the guard says so, not a failure report
				return runErr
			}
			fmt.Println("llama-bench failed:")
			b, _ := os.ReadFile(errPath)
			fmt.Println(lastLines(string(b), 5))
			return exitErr(1)
		}
		fmt.Printf("== results (tokens/s; saved in %s)\n", out)
		rc, err := Log(cfg, "zaphod", "card"+card+":"+strings.TrimSuffix(filepath.Base(model), ".gguf"), note, []string{out})
		if err != nil {
			return err
		}
		if rc != 0 {
			return exitErr(rc)
		}
		return nil
	})
}
