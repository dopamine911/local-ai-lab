package machines

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// Backup of zaphod's data that is not in git -> marvin:~/backups/zaphod/<date>/ (port of setup/backup-to-marvin.sh).
const (
	backupHost = "marvin"
	backupDest = "backups/zaphod"
	backupKeep = 14
)

// copyPy: the script's copy step, word for word: hub/data and llama-swap/data copied inside a container (some files
// belong to root), every SQLite database through its online backup API, so services keep running.
const copyPy = `import os, shutil, sqlite3
SKIP_DIRS = {"caddy", "caddy-config", "searxng-cache", "cache"}   # rebuildable
SKIP_SUFFIX = ("-wal", "-shm", "-journal")
n_db = n_file = 0
def own_all():
    for root, dirs, files in os.walk("/out"):
        for x in dirs + files:
            os.chown(os.path.join(root, x), 1000, 1000)
    os.chown("/out", 1000, 1000)
try:
  for top in ("hub-data", "llama-swap-data"):
      for root, dirs, files in os.walk(f"/src/{top}"):
          dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
          out = root.replace("/src", "/out", 1)
          os.makedirs(out, exist_ok=True)
          for f in files:
              s, d = os.path.join(root, f), os.path.join(out, f)
              if f.endswith(SKIP_SUFFIX):
                  continue
              with open(s, "rb") as fh:
                  is_db = fh.read(16) == b"SQLite format 3\x00"
              if is_db:
                  src = sqlite3.connect(f"file:{s}?mode=ro", uri=True)
                  dst = sqlite3.connect(d)
                  src.backup(dst); dst.close(); src.close(); n_db += 1
              else:
                  shutil.copy2(s, d); n_file += 1
finally:
    own_all()
print(f"copied {n_db} databases (online backup) and {n_file} other files")
`

func home() string { h, _ := os.UserHomeDir(); return h }

func stage() string { return filepath.Join(home(), ".cache", "backup-stage") }

// BackupLog is where the nightly run writes (cron line).
func BackupLog() string { return filepath.Join(home(), ".cache", "backup-to-marvin.log") }

func run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

// clean empties the stage folder; the copy step runs as root in a container, so clean up the same way.
func clean(cfg *config.Config) error {
	if err := os.MkdirAll(stage(), 0o755); err != nil {
		return err
	}
	return run("docker", "run", "--rm", "-v", stage()+":/out", cfg.Images.Alpine, "sh", "-c", "rm -rf /out/* /out/.[!.]*")
}

// BackupRun is `lab backup` / `lab backup run`: one snapshot now. Unchanged files are hard links to the previous
// snapshot, so each one costs only what changed; the last 14 are kept.
func BackupRun(cfg *config.Config) error {
	fmt.Printf("== %s backup start\n", time.Now().Format("2006-01-02 15:04:05"))
	if err := clean(cfg); err != nil {
		return err
	}
	st := stage()
	c := exec.Command("docker", "run", "--rm", "-i", "--network", "none",
		"-v", filepath.Join(cfg.Repo, "hub", "data")+":/src/hub-data",
		"-v", filepath.Join(cfg.Repo, "zaphod", "llama-swap", "data")+":/src/llama-swap-data",
		"-v", st+":/out", cfg.Images.Python, "python3", "-")
	c.Stdin = strings.NewReader(copyPy)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	for _, must := range []string{"hub-data/open-webui/webui.db", "hub-data/beszel/data.db", "hub-data/mlflow/mlflow.db"} {
		if fi, err := os.Stat(filepath.Join(st, must)); err != nil || fi.Size() == 0 {
			fmt.Printf("BACKUP FAILED: %s missing from the copy\n", must)
			return fmt.Errorf("backup failed")
		}
	}
	// Secrets (also in Proton Pass): copied with mode 600, never printed.
	envDir := filepath.Join(st, "env")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return err
	}
	for _, f := range []string{"hub/.env", "zaphod/llama-swap/.env"} {
		b, err := os.ReadFile(filepath.Join(cfg.Repo, f))
		if err != nil {
			continue
		}
		dst := filepath.Join(envDir, strings.ReplaceAll(f, "/", "_"))
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o600); err != nil { // WriteFile keeps an existing file's mode
			return err
		}
	}
	if err := os.Chmod(envDir, 0o700); err != nil {
		return err
	}

	snap := time.Now().Format("2006-01-02_1504")
	if err := run("ssh", backupHost, "mkdir -p "+backupDest+" && chmod 700 backups"); err != nil {
		return err
	}
	prevB, _ := exec.Command("ssh", backupHost, "ls -1d "+backupDest+"/20* 2>/dev/null | tail -1").Output()
	prev := strings.TrimRight(string(prevB), "\n")
	args := []string{"-a", "--delete"}
	if prev != "" {
		args = append(args, "--link-dest="+filepath.Join(home(), prev))
	}
	args = append(args, st+"/", backupHost+":"+backupDest+"/"+snap+"/")
	if err := run("rsync", args...); err != nil {
		return err
	}
	if err := run("ssh", backupHost, fmt.Sprintf("cd %s && ls -1d 20* | head -n -%d | xargs -r rm -rf", backupDest, backupKeep)); err != nil {
		return err
	}
	du, _ := exec.Command("du", "-sh", st).Output()
	size, _, _ := strings.Cut(string(du), "\t")
	fmt.Printf("== %s done: %s:%s/%s (%s)\n", time.Now().Format("2006-01-02 15:04:05"), backupHost, backupDest, snap, size)
	return clean(cfg)
}

// BackupList is `lab backup list`: the snapshots on marvin and their sizes.
func BackupList() error {
	return run("ssh", backupHost, "du -sh "+backupDest+"/* 2>/dev/null; echo; du -sh "+backupDest)
}

// CronLine is the nightly line `lab backup install` writes (04:17 Chicago time).
func CronLine() string {
	return "17 4 * * * " + filepath.Join(home(), ".local", "bin", "lab") + " backup >> " + BackupLog() + " 2>&1"
}

// BackupInstall is `lab backup install`: puts CronLine in the crontab instead of any earlier backup line (the
// script's or lab's). dryRun shows the crontab it would write.
func BackupInstall(dryRun bool) error {
	cur, _ := exec.Command("crontab", "-l").Output()
	var keep []string
	for _, l := range strings.Split(strings.TrimRight(string(cur), "\n"), "\n") {
		if l != "" && !strings.Contains(l, "backup-to-marvin.sh") && !strings.Contains(l, "/lab backup") {
			keep = append(keep, l)
		}
	}
	keep = append(keep, CronLine())
	text := strings.Join(keep, "\n") + "\n"
	if dryRun {
		fmt.Print("would write this crontab:\n" + text)
		return nil
	}
	c := exec.Command("crontab", "-")
	c.Stdin = strings.NewReader(text)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	now, _ := exec.Command("crontab", "-l").Output()
	if !strings.Contains(string(now), CronLine()) {
		fmt.Println("install FAILED")
		return fmt.Errorf("install failed")
	}
	fmt.Println("installed: " + CronLine())
	return nil
}
