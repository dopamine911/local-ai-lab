package bench

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// guard: a run with zaphod's llama-swap stopped (bench, kl: the test needs the card to itself). llama-swap is started
// again whatever happens: normal end, error, Ctrl-C, kill (SIGTERM) or a closed terminal (SIGHUP). On a signal the
// running command (docker run, which passes it on to the container) gets the same signal, then lab exits with 130.
// Like the scripts' `trap ... EXIT`, llama-swap starts again after the result is logged.
type guard struct {
	dryRun      bool
	mu          sync.Mutex
	cur         *exec.Cmd
	interrupted bool
}

// swapDir: zaphod's llama-swap folder (~/llama-swap, a link into the repo), where the scripts ran docker compose.
func swapDir() string { return config.Expand("~/llama-swap") }

// withSwapPaused stops llama-swap, runs fn, and always starts llama-swap again.
func withSwapPaused(dryRun bool, fn func(g *guard) error) (err error) {
	g := &guard{dryRun: dryRun}
	if dryRun {
		fmt.Println("== would stop llama-swap: (cd ~/llama-swap && docker compose stop -t 30)")
		err = fn(g)
		fmt.Println("== would start llama-swap: (cd ~/llama-swap && docker compose start)")
		return err
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for s := range sig {
			g.mu.Lock()
			g.interrupted = true
			if g.cur != nil && g.cur.Process != nil {
				g.cur.Process.Signal(s)
			}
			g.mu.Unlock()
		}
	}()
	defer signal.Stop(sig)

	fmt.Println("== stopping llama-swap")
	if e := composeFn("stop", "-t", "30"); e != nil {
		return fmt.Errorf("stop llama-swap: %v", e)
	}
	defer func() {
		fmt.Println("== starting llama-swap")
		if e := composeFn("start"); e != nil {
			fmt.Printf("== llama-swap did NOT start: %v (cd ~/llama-swap && docker compose start)\n", e)
			if err == nil {
				err = exitErr(1)
			}
		}
	}()
	err = fn(g)
	g.mu.Lock()
	stopped := g.interrupted
	g.mu.Unlock()
	if stopped {
		fmt.Println("== stopped by a signal")
		return exitErr(130)
	}
	return err
}

// composeFn runs docker compose in ~/llama-swap (a variable so the signal test can fake it).
var composeFn = compose

func compose(args ...string) error {
	c := exec.Command("docker", append([]string{"compose"}, args...)...)
	c.Dir = swapDir()
	return c.Run() // output to /dev/null, like the scripts
}

// run runs c as the current command (gets any signal); in a dry run it only prints it.
func (g *guard) run(c *exec.Cmd) error {
	if g.dryRun {
		fmt.Println("   would run: " + shellLine(c.Args))
		return nil
	}
	g.mu.Lock()
	if g.interrupted {
		g.mu.Unlock()
		return exitErr(130)
	}
	if err := c.Start(); err != nil {
		g.mu.Unlock()
		return err
	}
	g.cur = c
	g.mu.Unlock()
	err := c.Wait()
	g.mu.Lock()
	g.cur = nil
	g.mu.Unlock()
	return err
}

// llamaSwapImage: the first image: line of ~/llama-swap/compose.yaml (the same image as llama-swap).
func llamaSwapImage() (string, error) {
	b, err := os.ReadFile(filepath.Join(swapDir(), "compose.yaml"))
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) >= 2 && f[0] == "image:" {
			return f[1], nil
		}
	}
	return "", fmt.Errorf("no image: line in ~/llama-swap/compose.yaml")
}

// shellLine prints a command so it can be pasted into a shell.
func shellLine(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		if a != "" && strings.IndexFunc(a, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=,@+%", r))
		}) < 0 {
			q[i] = a
		} else {
			q[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
	}
	return strings.Join(q, " ")
}
