package hub

import (
	"strings"
	"testing"
)

// --install must replace the old sync.sh line and an earlier lab line, and keep every other cron job (#52).
func TestMemorySyncLine(t *testing.T) {
	line := MemorySyncCronLine("/r")
	if !strings.HasSuffix(strings.Fields(line)[5], "/.local/bin/lab") || !strings.Contains(line, " memory sync >> ") {
		t.Errorf("cron line: %s", line)
	}
	for l, want := range map[string]bool{
		"41 * * * * /home/x/homelab/hub/memory/sync.sh >> /home/x/.cache/memory-sync.log 2>&1": true,
		line: true,
		"17 4 * * * /home/x/homelab/setup/backup-to-marvin.sh >> /home/x/.cache/backup-to-marvin.log 2>&1": false,
		"23 3 * * 0 docker builder prune -f --filter until=168h": false,
	} {
		if isMemorySyncLine(l) != want {
			t.Errorf("isMemorySyncLine(%q) = %v, want %v", l, !want, want)
		}
	}
}
