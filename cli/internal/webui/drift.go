// Package webui: `lab webui drift`, port of the read-only part of hub/openwebui/drift.sh.
//
// Compares the running Open WebUI's settings with what compose.yaml would give a fresh install: starts a throwaway
// Open WebUI (same image, same environment, no network) on an empty volume, waits for it to write its settings, then
// compares the two settings tables. Go does the orchestration; the comparison is the script's Python, unchanged and
// run in a pinned Python image, because its output (Python's JSON formatting) is the output to match.
// Secrets are never printed: settings whose name holds key/secret/token/password show as <set>.
// --sync (step 3): makes the running Open WebUI match compose.yaml for the settings it manages (managed-keys.txt):
// stops Open WebUI for a few seconds and saves a copy of its database first (data/open-webui/webui.db.before-sync-<date>).
package webui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Python image for the comparison: the one compose already pins (drift.sh used the floating 3.13-slim-trixie tag).
const PyImage = "python:3.13.18-slim-trixie@sha256:76baad6037690d6e8c4397fbcaef17bca6f19eecef822b708fe8fa3417d30bb5"

// compare: drift.sh's Python, word for word (its SYNC branch runs only with SYNC=yes).
const compare = `import json, os, re, shutil, sqlite3
def load(p):
    c = sqlite3.connect(f"file:{p}?mode=ro", uri=True)
    rows = {}
    for k, v in c.execute("select key, value from config"):
        try: rows[k] = json.loads(v) if isinstance(v, (str, bytes)) else v
        except Exception: rows[k] = v
    c.close(); return rows
def norm(v):  # treat empty-ish lists and "300" vs 300 as equal
    if v in ([""], []): return []
    if isinstance(v, str) and v.isdigit(): return int(v)
    return v
live, fresh = load("/live/webui.db"), load("/fresh/webui.db")
managed = [l.strip() for l in open("/managed.txt") if l.strip() and not l.startswith("#")]
SEC = re.compile(r"(key|secret|token|password|passwd)", re.I)
def show(k, v):
    if SEC.search(k) and v not in (None, "", [], False, 0): return "<set>"
    return re.sub(r'"(key|api_key|token|secret|password|auth)": ?"[^"]+"', r'"\1": "<set>"', json.dumps(v))[:160]
diff = sorted(k for k in set(live) | set(fresh) if norm(live.get(k)) != norm(fresh.get(k)))
print(f"{len(diff)} setting(s) differ between the running Open WebUI and compose.yaml")
for k in diff:
    print(f"  {'in compose   ' if k in managed else 'web page only'}  {k}\n      running: {show(k, live.get(k))}\n      compose: {show(k, fresh.get(k))}")
if os.environ["SYNC"] == "yes":
    todo = [k for k in diff if k in managed]
    if todo:
        shutil.copy2("/live/webui.db", f"/live/webui.db.before-sync-{os.environ['STAMP']}")
        c = sqlite3.connect("/live/webui.db")
        for k in todo:
            c.execute("update config set value = ? where key = ?", (json.dumps(fresh[k]), k))
        c.commit(); c.close()
    print(f"== synced {len(todo)} managed setting(s); 'web page only' ones were left alone")
`

func docker(args ...string) (string, error) {
	b, err := exec.Command("docker", args...).CombinedOutput()
	return string(b), err
}

// Drift prints the differences; repo is the lab repo (hub/ inside it). sync: also make the running one match
// (managed settings only); dryRun: say what would happen.
func Drift(repo string, sync, dryRun bool) error {
	hub := filepath.Join(repo, "hub")
	cfgCmd := exec.Command("docker", "compose", "config", "--format", "json")
	cfgCmd.Dir = hub
	raw, err := cfgCmd.Output()
	if err != nil {
		return fmt.Errorf("docker compose config (in %s): %v", hub, err)
	}
	var cfg struct {
		Services map[string]struct {
			Image       string            `json:"image"`
			Environment map[string]string `json:"environment"`
		} `json:"services"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	svc, ok := cfg.Services["open-webui"]
	if !ok {
		return fmt.Errorf("no open-webui service in hub/compose.yaml")
	}

	// The exact environment compose gives open-webui, secrets included (mode 600 file, deleted at the end).
	tmp, err := os.MkdirTemp("", "lab-webui-drift-")
	if err != nil {
		return err
	}
	vol := "owui-drift-" + strconv.Itoa(os.Getpid())
	defer func() {
		docker("rm", "-f", vol)
		docker("volume", "rm", vol)
		os.RemoveAll(tmp)
	}()
	envf := filepath.Join(tmp, "env")
	var lines []string
	for k, v := range svc.Environment {
		if k != "PORT" && k != "HOST" {
			lines = append(lines, k+"="+strings.ReplaceAll(v, "\n", " "))
		}
	}
	if err := os.WriteFile(envf, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("would: start a throwaway %s (no network) from compose's environment, compare its settings with the running one", svc.Image)
		if sync {
			fmt.Print("; then stop open-webui, copy webui.db to webui.db.before-sync-<date>, write the differing managed settings, start open-webui")
		}
		fmt.Println()
		return nil
	}
	fmt.Printf("== building a fresh Open WebUI from compose.yaml (%s, no network, about a minute)\n", svc.Image)
	if out, err := docker("volume", "create", vol); err != nil {
		return fmt.Errorf("volume: %v %s", err, out)
	}
	if out, err := docker("run", "-d", "--name", vol, "--network", "none", "--env-file", envf, "-e", "PORT=3099",
		"-v", vol+":/app/backend/data", svc.Image); err != nil {
		return fmt.Errorf("start: %v %s", err, out)
	}
	count := `import sqlite3;print(sqlite3.connect('file:/app/backend/data/webui.db?mode=ro',uri=True).execute('select count(*) from config').fetchone()[0])`
	n := 0
	for i := 0; i < 60; i++ {
		out, err := docker("exec", vol, "python3", "-c", count)
		if err == nil {
			n, _ = strconv.Atoi(strings.TrimSpace(out))
		}
		if n > 100 {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if n <= 100 {
		return fmt.Errorf("fresh Open WebUI did not write its settings")
	}
	time.Sleep(5 * time.Second)
	docker("stop", vol)

	syncEnv, live := "no", ":/live:ro"
	if sync {
		syncEnv, live = "yes", ":/live" // the script's mount: writable for the update
		fmt.Println("== stopping open-webui")
		stop := exec.Command("docker", "compose", "stop", "open-webui")
		stop.Dir = hub
		if err := stop.Run(); err != nil {
			return fmt.Errorf("stop open-webui: %v", err)
		}
		defer func() {
			start := exec.Command("docker", "compose", "start", "open-webui")
			start.Dir = hub
			if err := start.Run(); err == nil {
				fmt.Println("== open-webui started")
			} else {
				fmt.Printf("== open-webui did NOT start: %v (cd hub && docker compose start open-webui)\n", err)
			}
		}()
	}
	c := exec.Command("docker", "run", "--rm", "-i", "--network", "none", "-e", "SYNC="+syncEnv,
		"-e", "STAMP="+time.Now().Format("2006-01-02-1504"),
		"-v", filepath.Join(hub, "data", "open-webui")+live, "-v", vol+":/fresh:ro",
		"-v", filepath.Join(hub, "openwebui", "managed-keys.txt")+":/managed.txt:ro",
		PyImage, "python3", "-")
	c.Stdin = strings.NewReader(compare)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}
