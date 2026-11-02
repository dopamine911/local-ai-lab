package enso

// `lab enso build|setup|up|tui|shell|down`: ports of agents/enso/enso.sh (check is in check.go).
// Keys are copied from zaphod/llama-swap/.env (ZAPHOD_KEY) and hub/.env (MCP_TOKEN_ENSO) into
// ~/agents/enso-home/keys/ (mode 600); never printed. Enso's guards refuse reads of ENSO_HOME, so the agent can't read
// them through its tools.

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Ops struct {
	Repo, Image, Container string
	DryRun                 bool
}

func (o *Ops) here() string { return filepath.Join(o.Repo, "agents", "enso") }

func home() string { h, _ := os.UserHomeDir(); return h }

func (o *Ops) ehome() string { return filepath.Join(home(), "agents", "enso-home") }
func (o *Ops) work() string  { return filepath.Join(home(), "agents", "work") }

func run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

func (o *Ops) dc(args ...string) error {
	return run("docker", append([]string{"compose", "-f", filepath.Join(o.here(), "compose.yaml")}, args...)...)
}

// approveMCP: Enso denies every MCP server unless approved by name; approve ours ("lab"). permissions.json is seeded
// by Enso's first start, so this runs after `up` too. Same Python as the script, so the file keeps its key order.
func (o *Ops) approveMCP() error {
	p := filepath.Join(o.ehome(), "permissions.json")
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	return run("python3", "-I", "-c", `import json,sys; p=sys.argv[1]; j=json.load(open(p)); j["permission"].setdefault("mcp", {})["lab"]="allow"; open(p,"w").write(json.dumps(j, indent=2)+"\n")`, p)
}

func (o *Ops) Build() error {
	if o.DryRun {
		fmt.Printf("would: docker buildx build --load -t %s %s\n", o.Image, o.here())
		return nil
	}
	return run("docker", "buildx", "build", "--load", "-t", o.Image, o.here())
}

// envValue: the value after NAME= on its first line, newlines dropped (script: grep | cut -d= -f2- | tr -d '\n').
func envValue(path, name string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(l, name+"="); ok {
			return v
		}
	}
	return ""
}

func (o *Ops) Setup() error {
	eh, wk := o.ehome(), o.work()
	if o.DryRun {
		fmt.Printf("would: make %s/{keys,pi-agent} and %s; copy ZAPHOD_KEY and MCP_TOKEN_ENSO into %s/keys (mode 600); "+
			"install config/models.json, mcp.json, config.json; approve MCP server \"lab\"; clone enso-harness into %s if missing\n",
			eh, wk, eh, wk)
		return nil
	}
	for _, d := range []string{filepath.Join(eh, "keys"), filepath.Join(eh, "pi-agent"), wk} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	os.Chmod(eh, 0o700)
	os.Chmod(filepath.Join(eh, "keys"), 0o700)
	keys := []struct{ file, env, name string }{
		{"zaphod-key", filepath.Join(o.Repo, "zaphod", "llama-swap", ".env"), "ZAPHOD_KEY"},
		{"mcp-token", filepath.Join(o.Repo, "hub", ".env"), "MCP_TOKEN_ENSO"},
	}
	for _, k := range keys {
		v := envValue(k.env, k.name)
		if err := os.WriteFile(filepath.Join(eh, "keys", k.file), []byte(v), 0o600); err != nil {
			return err
		}
		os.Chmod(filepath.Join(eh, "keys", k.file), 0o600)
		if v == "" {
			return fmt.Errorf("a key is missing (see README)")
		}
	}
	// pi's models and MCP servers, Enso's config: from the repo, overwritten on every setup (edit them in the repo)
	for _, c := range []struct{ src, dst string }{
		{"models.json", filepath.Join(eh, "pi-agent", "models.json")},
		{"mcp.json", filepath.Join(eh, "pi-agent", "mcp.json")},
		{"config.json", filepath.Join(eh, "config.json")},
	} {
		b, err := os.ReadFile(filepath.Join(o.here(), "config", c.src))
		if err != nil {
			return err
		}
		if err := os.WriteFile(c.dst, b, 0o600); err != nil {
			return err
		}
		os.Chmod(c.dst, 0o600)
	}
	if err := o.approveMCP(); err != nil {
		return err
	}
	// The workspace: its own clone, so the agent's commits never touch Daniel's checkout. No push credentials inside.
	repo := filepath.Join(wk, "enso-harness")
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		if err := run("git", "clone", "-q", "https://github.com/dopamine911/enso-harness.git", repo); err != nil {
			return err
		}
	}
	run("git", "-C", repo, "config", "user.name", "enso-agent (local model)")
	run("git", "-C", repo, "config", "user.email", "enso-agent@zaphod.invalid")
	ents, _ := os.ReadDir(filepath.Join(eh, "keys"))
	var names []string
	for _, e := range ents {
		names = append(names, e.Name()+" ")
	}
	head, _ := exec.Command("git", "-C", repo, "log", "-1", "--format=%h").Output()
	fmt.Printf("setup done: %s (keys: %s), %s at %s\n", eh, strings.Join(names, ""), repo, strings.TrimSpace(string(head)))
	return nil
}

func pageUp() bool {
	r, err := (&http.Client{Timeout: 3 * time.Second}).Get("http://127.0.0.1:5178/")
	if err != nil {
		return false
	}
	r.Body.Close()
	return r.StatusCode < 400
}

func (o *Ops) Up() error {
	if o.DryRun {
		fmt.Println("would: docker compose up -d (agents/enso), wait up to 2 min for http://127.0.0.1:5178, approve MCP server \"lab\"")
		return nil
	}
	if err := o.dc("up", "-d"); err != nil {
		return err
	}
	for i := 0; i < 60 && !pageUp(); i++ {
		time.Sleep(2 * time.Second)
	}
	if err := o.approveMCP(); err != nil {
		return err
	}
	if pageUp() {
		fmt.Println("Enso: http://localhost:5178")
		return nil
	}
	fmt.Println("page not answering yet:")
	return run("docker", "logs", "--tail", "20", o.Container)
}

func (o *Ops) Down() error {
	if o.DryRun {
		fmt.Println("would: docker compose down (agents/enso: enso and enso-relay)")
		return nil
	}
	return o.dc("down")
}

// execDocker replaces lab with `docker args...`, so an interactive session gets the terminal directly.
func execDocker(args ...string) error {
	p, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	return syscall.Exec(p, append([]string{"docker"}, args...), os.Environ())
}

// TUI: Enso's terminal in the workspace. Enso starts pi with --no-extensions, which also drops pi's built-in MCP;
// -e loads just MCP and tool search back. Extra arguments go to pi, e.g. --model lab/coder.
func (o *Ops) TUI(extra []string) error {
	return execDocker(append([]string{"exec", "-it", "-w", "/work/enso-harness", o.Container, "bun", "/opt/enso/scripts/enso.ts",
		"-e", "builtin:mcp", "-e", "builtin:tool-search"}, extra...)...)
}

func (o *Ops) Shell() error { return execDocker("exec", "-it", o.Container, "bash") }
