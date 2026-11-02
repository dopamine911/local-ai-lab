// Package hub: operations on zaphod's hub (ports of hub/ scripts). Commands that change something take dryRun:
// they print what they would do and change nothing. Secrets (.env values) are read, never printed.
package hub

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnvValue reads NAME=value from an .env file (first match, quotes stripped).
func EnvValue(path, name string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), name+"="); ok {
			return strings.Trim(strings.TrimSpace(v), `'"`), nil
		}
	}
	return "", fmt.Errorf("%s missing in %s", name, path)
}

// Compose runs `docker compose args...` in dir, output to the terminal.
func Compose(dir string, args ...string) error {
	c := exec.Command("docker", append([]string{"compose"}, args...)...)
	c.Dir = dir
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

// Quiet runs `docker compose args...` in dir, output discarded (the script's >/dev/null 2>&1).
func Quiet(dir string, args ...string) error {
	c := exec.Command("docker", append([]string{"compose"}, args...)...)
	c.Dir = dir
	return c.Run()
}

func HubDir(repo string) string { return filepath.Join(repo, "hub") }
