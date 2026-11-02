package mcp

// `lab mcp setup`: port of hub/mcp/setup.sh. Sets up the MCP gateway (MCPJungle) after
// `docker compose up -d mcpjungle mcp-searxng`. Safe to run again:
//   1. installs the mcpjungle CLI (pinned version, checksum checked) to bin
//   2. creates the admin account once (token in ~/.mcpjungle.conf, mode 600; copy it to Proton Pass)
//   3. registers every tool server in hub/mcp/servers/*.json
//   4. a server with <name>.tools keeps only the tools listed there; the rest are off
//   5. recreates every agent login in hub/mcp/clients/*.json (tokens from hub/.env MCP_TOKEN_<NAME>; never printed)
// The JSON files name secrets as ${VAR}; mcpjungle fills them from the environment, so hub/.env's values are passed
// to the mcpjungle processes only (never printed, never written).

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Setup struct {
	Bin, Registry, Version, Repo string
	DryRun                       bool
	env                          []string
}

func (s *Setup) hubDir() string { return filepath.Join(s.Repo, "hub") }

// loadEnv: hub/.env as KEY=value pairs for the child processes (like the script's `set -a; . ./.env`).
func (s *Setup) loadEnv() error {
	f, err := os.Open(filepath.Join(s.hubDir(), ".env"))
	if err != nil {
		return err
	}
	defer f.Close()
	s.env = os.Environ()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") || !strings.Contains(l, "=") {
			continue
		}
		k, v, _ := strings.Cut(l, "=")
		if len(v) >= 2 && (v[0] == '\'' && v[len(v)-1] == '\'' || v[0] == '"' && v[len(v)-1] == '"') {
			v = v[1 : len(v)-1]
		}
		s.env = append(s.env, k+"="+v)
	}
	return sc.Err()
}

func (s *Setup) mj(args ...string) (string, error) {
	c := exec.Command(s.Bin, append([]string{"--registry", s.Registry}, args...)...)
	c.Dir = s.hubDir()
	if s.env != nil {
		c.Env = s.env
	}
	b, err := c.CombinedOutput() // mcpjungle prints its results on stderr
	return string(b), err
}

var versionRe = regexp.MustCompile(`v[0-9.]+`)

func (s *Setup) cliVersion() string {
	b, _ := exec.Command(s.Bin, "version").CombinedOutput()
	return versionRe.FindString(string(b))
}

// installCLI downloads the pinned release, checks it against the release's checksums.txt, installs it.
func (s *Setup) installCLI() error {
	base := "https://github.com/mcpjungle/MCPJungle/releases/download/" + s.Version
	get := func(u string) ([]byte, error) {
		r, err := (&http.Client{Timeout: 120 * time.Second}).Get(u)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return nil, fmt.Errorf("%s: HTTP %d", u, r.StatusCode)
		}
		return io.ReadAll(r.Body)
	}
	tgz, err := get(base + "/mcpjungle_Linux_x86_64.tar.gz")
	if err != nil {
		return err
	}
	sums, err := get(base + "/checksums.txt")
	if err != nil {
		return err
	}
	sum := sha256.Sum256(tgz)
	want := ""
	for _, l := range strings.Split(string(sums), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[1] == "mcpjungle_Linux_x86_64.tar.gz" {
			want = f[0]
		}
	}
	if want == "" || want != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("checksum of mcpjungle_Linux_x86_64.tar.gz does not match checksums.txt: not installed")
	}
	zr, err := gzip.NewReader(strings.NewReader(string(tgz)))
	if err != nil {
		return err
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err != nil {
			return fmt.Errorf("mcpjungle not found in the archive: %v", err)
		}
		if filepath.Base(h.Name) == "mcpjungle" && h.Typeflag == tar.TypeReg {
			tmp := s.Bin + ".new"
			os.MkdirAll(filepath.Dir(s.Bin), 0o755)
			f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
			return os.Rename(tmp, s.Bin)
		}
	}
}

func (s *Setup) Run() error {
	hub := s.hubDir()
	servers, _ := filepath.Glob(filepath.Join(hub, "mcp", "servers", "*.json"))
	toolsFiles, _ := filepath.Glob(filepath.Join(hub, "mcp", "servers", "*.tools"))
	clients, _ := filepath.Glob(filepath.Join(hub, "mcp", "clients", "*.json"))
	sort.Strings(servers)
	sort.Strings(toolsFiles)
	sort.Strings(clients)
	if s.DryRun {
		fmt.Printf("would: make sure the mcpjungle CLI is v%s (now: %s); check %s/health; create the admin once if ~/.mcpjungle.conf is empty\n", s.Version, orNone(s.cliVersion()), s.Registry)
		for _, f := range servers {
			fmt.Printf("would: register tool server %s (register --force)\n", base(f, ".json"))
		}
		for _, f := range toolsFiles {
			fmt.Printf("would: %s: keep only the tools in %s, switch off the rest\n", base(f, ".tools"), filepath.Base(f))
		}
		for _, f := range clients {
			fmt.Printf("would: recreate agent login %s (delete, then create from %s; its token from hub/.env)\n", base(f, ".json"), filepath.Base(f))
		}
		return nil
	}

	if !strings.Contains(s.cliVersion(), "v"+s.Version) {
		if err := s.installCLI(); err != nil {
			return err
		}
	}
	fmt.Printf("== CLI %s\n", s.cliVersion())

	if r, err := (&http.Client{Timeout: 10 * time.Second}).Get(s.Registry + "/health"); err != nil || r.StatusCode >= 400 {
		return fmt.Errorf("MCPJungle not answering on %s: docker compose up -d mcpjungle", s.Registry)
	}
	home, _ := os.UserHomeDir()
	if st, err := os.Stat(filepath.Join(home, ".mcpjungle.conf")); err != nil || st.Size() == 0 {
		old := setUmask077()
		_, err := s.mj("init-server")
		restoreUmask(old)
		if err != nil {
			return fmt.Errorf("init-server failed: %v", err)
		}
		fmt.Println("== admin created; token in ~/.mcpjungle.conf: copy it to Proton Pass (entry 'MCPJungle admin')")
	}

	if err := s.loadEnv(); err != nil {
		return err
	}
	for _, f := range servers {
		rel, _ := filepath.Rel(hub, f)
		if out, err := s.mj("register", "--force", "-c", rel); err != nil {
			first := strings.SplitN(strings.TrimSpace(out), "\n", 2)[0]
			fmt.Printf("== tool server: %s FAILED: %s\n", base(f, ".json"), first)
		} else {
			fmt.Printf("== tool server: %s\n", base(f, ".json"))
		}
	}
	for _, tf := range toolsFiles {
		srv := base(tf, ".tools")
		keep, _ := readTools(tf)
		keepSet := map[string]bool{}
		var keepOrder []string
		b, _ := os.ReadFile(tf)
		for _, l := range strings.Split(string(b), "\n") { // file order for the message, like the script
			if l != "" && !strings.HasPrefix(l, "#") {
				keepOrder = append(keepOrder, srv+"__"+l)
			}
		}
		for _, k := range keep {
			keepSet[srv+"__"+k] = true
		}
		list, _ := s.mj("list", "tools")
		re := regexp.MustCompile(regexp.QuoteMeta(srv) + `__[A-Za-z0-9_]+`)
		seen := map[string]bool{}
		var tools []string
		for _, t := range re.FindAllString(list, -1) {
			if !seen[t] {
				seen[t] = true
				tools = append(tools, t)
			}
		}
		sort.Strings(tools)
		for _, t := range tools {
			if keepSet[t] {
				s.mj("enable", "tool", t)
			} else {
				s.mj("disable", "tool", t)
			}
		}
		fmt.Printf("== tool server: %s: on: %s\n", srv, strings.Join(keepOrder, ","))
	}
	for _, f := range clients {
		n := base(f, ".json")
		s.mj("delete", "mcp-client", n)
		rel, _ := filepath.Rel(hub, f)
		out, err := s.mj("create", "mcp-client", "--conf", rel)
		if err != nil {
			fmt.Printf("== client %s: FAILED\n", n)
			for _, l := range strings.Split(out, "\n") {
				if !strings.Contains(strings.ToLower(l), "token") {
					fmt.Println(l)
				}
			}
			return fmt.Errorf("client %s failed", n)
		}
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(strings.ToLower(l), "servers accessible") {
				fmt.Printf("== client %s: %s\n", n, l)
			}
		}
	}
	return nil
}

func base(p, ext string) string { return strings.TrimSuffix(filepath.Base(p), ext) }

func orNone(s string) string {
	if s == "" {
		return "not installed"
	}
	return s
}
