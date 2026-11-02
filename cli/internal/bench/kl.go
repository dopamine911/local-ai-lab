package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// KL-divergence setup, the same as tuning/kl: reference Q8_0 file, reference logits, fixed code sample, settings.
const (
	klRefFile = "Qwen3.8-27B-Q8_0.gguf"
	klRef     = "kld/qwen3.8-27b-q8_0.c512x32.kld" // under /srv/models
	klSample  = "/tuning/kld/code-sample.txt"      // fixed text; never edit (checksum in kld/code-sample.sha256)
)

var klCommon = []string{"-c", "512", "--chunks", "32", "-ngl", "999", "-fa", "on", "-b", "512", "-ub", "512"}

// KLBase is `lab kl base`: builds the reference from the Q8_0 file on both cards (about 10 minutes).
func KLBase(cfg *config.Config, dryRun bool) error {
	if err := klStart(cfg, "base"); err != nil {
		return err
	}
	if _, err := os.Stat("/srv/models/" + klRefFile); err != nil {
		fmt.Printf("Missing /srv/models/%s. Download it first (29 GB):\n", klRefFile)
		fmt.Printf("  wget -c -P /srv/models https://huggingface.co/unsloth/Qwen3.8-27B-GGUF/resolve/main/%s\n", klRefFile)
		return exitErr(1)
	}
	if !dryRun {
		if err := os.MkdirAll("/srv/models/kld", 0o755); err != nil {
			return err
		}
	}
	return withSwapPaused(dryRun, func(g *guard) error {
		fmt.Println("== building the reference from Q8_0 on both cards (about 10 minutes)")
		c := klRun(cfg, append(append([]string{"-m", "/models/" + klRefFile, "-f", klSample}, klCommon...),
			"-dev", "Vulkan0,Vulkan1", "--kl-divergence-base", "/models/"+klRef)...)
		if dryRun {
			return g.run(c)
		}
		var b strings.Builder // like `2>&1 | tail -4`: only the last 4 lines, and its exit status is not checked
		c.Stdout, c.Stderr = &b, &b
		runErr := g.run(c)
		if g.wasInterrupted() {
			return runErr
		}
		if s := lastLines(b.String(), 4); s != "" {
			fmt.Println(s)
		}
		ls := exec.Command("ls", "-lh", "/srv/models/"+klRef)
		ls.Stdout, ls.Stderr = os.Stdout, os.Stderr
		return ls.Run()
	})
}

// KLTest is `lab kl test <note> [llama-perplexity options]`: how far a model file or setting drifts from the Q8_0
// reference, on card 1 (Vulkan0), logged. MODEL picks the file (default Qwen3.8-27B UD-Q4_K_XL).
func KLTest(cfg *config.Config, dryRun bool, note string, extra []string) error {
	if err := klStart(cfg, append([]string{"test", note}, extra...)...); err != nil {
		return err
	}
	if _, err := os.Stat("/srv/models/" + klRef); err != nil {
		fmt.Println("No reference yet. Run: lab kl base")
		return exitErr(1)
	}
	model := os.Getenv("MODEL")
	if model == "" {
		model = "/models/Qwen3.8-27B-UD-Q4_K_XL.gguf"
	}
	base := strings.TrimSuffix(filepath.Base(model), ".gguf")
	if !dryRun {
		if err := os.MkdirAll("kld/runs", 0o755); err != nil {
			return err
		}
	}
	out := "kld/runs/" + time.Now().Format("2006-01-02-1504") + "-" + base + ".kld.txt"
	return withSwapPaused(dryRun, func(g *guard) error {
		joined := strings.Join(extra, " ") // "$*"
		fmt.Printf("== KL-divergence: %s %s (card 1)\n", filepath.Base(model), joined)
		args := append(append([]string{"-m", model, "-f", klSample}, klCommon...),
			"-dev", "Vulkan0", "--kl-divergence-base", "/models/"+klRef, "--kl-divergence")
		c := klRun(cfg, append(args, extra...)...)
		if dryRun {
			g.run(c)
			fmt.Printf("   output: %s\n   would log: lab log zaphod kld:%s %q %s\n", out, base, note, out)
			return nil
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		fmt.Fprintf(f, "# model=%s extra=%s\n", model, joined)
		c.Stdout, c.Stderr = f, f
		runErr := g.run(c)
		f.Close()
		if runErr != nil {
			if g.wasInterrupted() { // stopped by a signal: the guard says so, not a failure report
				return runErr
			}
			fmt.Println("llama-perplexity failed:")
			b, _ := os.ReadFile(out)
			fmt.Println(lastLines(string(b), 8))
			return exitErr(1)
		}
		rc, err := Log(cfg, "zaphod", "kld:"+base, note, []string{out})
		if err != nil {
			return err
		}
		if rc != 0 {
			return exitErr(rc)
		}
		return nil
	})
}

// klStart: into tuning/ (Rerun button), then the sample checksum check (results would not be comparable).
func klStart(cfg *config.Config, args ...string) error {
	if err := enterTuning(cfg, "kl", args...); err != nil {
		return err
	}
	b, err := os.ReadFile("kld/code-sample.txt")
	if err != nil {
		return err
	}
	want, _ := os.ReadFile("kld/code-sample.sha256")
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:])[:16] != strings.TrimRight(string(want), "\n") {
		fmt.Println("kld/code-sample.txt has changed; results would not be comparable. Restore it with git.")
		return exitErr(1)
	}
	return nil
}

func klRun(cfg *config.Config, args ...string) *exec.Cmd {
	pwd, _ := os.Getwd()
	a := []string{"run", "--rm", "--device", "/dev/dri", "-e", "GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1",
		"-v", "/srv/models:/models", "-v", pwd + ":/tuning:ro", "--entrypoint", "/app/llama-perplexity", cfg.Images.KL}
	return exec.Command("docker", append(a, args...)...)
}
