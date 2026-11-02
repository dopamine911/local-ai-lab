package machines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// host.sh's package list spans two lines and ends in a comment; marvin's later "apt-get install -y tuned" is not part of it.
func TestHostPackages(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "setup"), 0o755)
	sh := "apt-get update -q\napt-get install -y -q docker.io docker-compose-v2 \\\n  curl git gh   # gh: GitHub CLI\nstep x\n" +
		"  dpkg -s tuned || { apt-get update -q; apt-get install -y tuned; }\n"
	os.WriteFile(filepath.Join(dir, "setup", "host.sh"), []byte(sh), 0o644)
	got, err := hostPackages(&config.Config{Repo: dir})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "docker.io docker-compose-v2 curl git gh" {
		t.Errorf("packages = %q", got)
	}
}
