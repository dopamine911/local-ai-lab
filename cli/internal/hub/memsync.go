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

// MemorySyncCronLine is the crontab line sync.sh install writes (every hour at :41).
func MemorySyncCronLine(repo string) string {
	home, _ := os.UserHomeDir()
	return fmt.Sprintf("41 * * * * %s/memory/sync.sh >> %s 2>&1", HubDir(repo), filepath.Join(home, ".cache", "memory-sync.log"))
}

// MemorySyncInstall: port of sync.sh install: replace any memory/sync.sh line in the user's crontab with the
// current one. Run by Daniel (lab rule: no cron job without him); --dry-run prints the new crontab.
func MemorySyncInstall(repo string, dryRun bool) error {
	line := MemorySyncCronLine(repo)
	old, _ := exec.Command("crontab", "-l").Output() // no crontab yet: empty, like the script's "|| true"
	var keep []string
	for _, l := range strings.Split(strings.TrimRight(string(old), "\n"), "\n") {
		if l != "" && !strings.Contains(l, "memory/sync.sh") {
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
	if !strings.Contains(string(now), "memory/sync.sh") {
		return fmt.Errorf("install FAILED")
	}
	fmt.Println("installed: " + line)
	return nil
}
