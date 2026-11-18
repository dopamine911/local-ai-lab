package hub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MemorySync: port of hub/memory/sync.sh run: upload changed Markdown to LocalRecall.
// Good result ends with "sync: N uploaded, M removed, K unchanged".
func MemorySync(repo string, dryRun bool) error {
	if dryRun {
		fmt.Printf("would run (in %s): docker compose --profile sync run --rm -T memory-sync\n", HubDir(repo))
		return nil
	}
	fmt.Printf("== %s\n", time.Now().Format("2006-01-02 15:04:05"))
	return Compose(HubDir(repo), "--profile", "sync", "run", "--rm", "-T", "memory-sync")
}

// MemorySyncCronLine is the hourly crontab line (at :41). Since #52 it runs lab itself (was hub/memory/sync.sh,
// removed); full path: cron has no ~/.local/bin on its PATH. The watchdog reads the same log.
func MemorySyncCronLine(repo string) string {
	home, _ := os.UserHomeDir()
	return fmt.Sprintf("41 * * * * %s memory sync >> %s 2>&1", filepath.Join(home, ".local", "bin", "lab"),
		filepath.Join(home, ".cache", "memory-sync.log"))
}

// isMemorySyncLine: the current line or the old sync.sh one, so --install replaces either.
func isMemorySyncLine(l string) bool {
	return strings.Contains(l, "memory/sync.sh") || strings.Contains(l, "lab memory sync")
}

// MemorySyncInstall: replace any memory-sync line in the user's crontab (old sync.sh or lab) with the current one. Run by Daniel (lab rule: no cron job without him); --dry-run prints the new crontab.
func MemorySyncInstall(repo string, dryRun bool) error {
	line := MemorySyncCronLine(repo)
	old, _ := exec.Command("crontab", "-l").Output() // no crontab yet: empty, like the script's "|| true"
	var keep []string
	for _, l := range strings.Split(strings.TrimRight(string(old), "\n"), "\n") {
		if l != "" && !isMemorySyncLine(l) {
			keep = append(keep, l)
		}
	}
	keep = append(keep, line)
	newTab := strings.Join(keep, "\n") + "\n"
	if dryRun {
		fmt.Print("would install this crontab:\n" + newTab)
		return nil
	}
	c := exec.Command("crontab", "-")
	c.Stdin = strings.NewReader(newTab)
	if err := c.Run(); err != nil {
		return fmt.Errorf("install FAILED: %v", err)
	}
	now, _ := exec.Command("crontab", "-l").Output()
	if !strings.Contains(string(now), line) {
		return fmt.Errorf("install FAILED")
	}
	fmt.Println("installed: " + line)
	return nil
}
