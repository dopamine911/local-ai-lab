// Package mcp: `lab mcp status`, one read-only screen of the MCP gateway (MCPJungle): registered tool servers, their
// tools (enabled/disabled), which agent may use which server, and drift against the repo (hub/mcp/clients/*.json,
// hub/mcp/servers/*.tools), which hub/mcp/setup.sh applies. No secrets: client files are read for name and
// allowed_servers only.
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Server struct {
	Name     string   `json:"name"`
	Enabled  []string `json:"enabled"`
	Disabled []string `json:"disabled"`
}

type Client struct {
	Name    string   `json:"name"`
	Allowed []string `json:"allowed_servers"`
}

type Status struct {
	Servers []Server `json:"servers"`
	Clients []Client `json:"clients"`
	Drift   []string `json:"drift"` // empty = the gateway matches the repo
}

var toolLine = regexp.MustCompile(`^\d+\. ([A-Za-z0-9_-]+)__([A-Za-z0-9_-]+)\s+\[(ENABLED|DISABLED)\]`)
var clientLine = regexp.MustCompile(`^\d+\. (\S+)`)

// ParseTools reads `mcpjungle list tools` output.
func ParseTools(out string) []Server {
	by := map[string]*Server{}
	for _, l := range strings.Split(out, "\n") {
		m := toolLine.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		s := by[m[1]]
		if s == nil {
			s = &Server{Name: m[1]}
			by[m[1]] = s
		}
		if m[3] == "ENABLED" {
			s.Enabled = append(s.Enabled, m[2])
		} else {
			s.Disabled = append(s.Disabled, m[2])
		}
	}
	var out2 []Server
	for _, s := range by {
		sort.Strings(s.Enabled)
		sort.Strings(s.Disabled)
		out2 = append(out2, *s)
	}
	sort.Slice(out2, func(i, j int) bool { return out2[i].Name < out2[j].Name })
	return out2
}

// ParseClients reads `mcpjungle list mcp-clients` output ("1. name" then "Allowed servers: a,b").
func ParseClients(out string) []Client {
	var cs []Client
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if m := clientLine.FindStringSubmatch(l); m != nil {
			cs = append(cs, Client{Name: m[1]})
		} else if a, ok := strings.CutPrefix(l, "Allowed servers:"); ok && len(cs) > 0 {
			for _, s := range strings.Split(a, ",") {
				if s = strings.TrimSpace(s); s != "" {
					cs[len(cs)-1].Allowed = append(cs[len(cs)-1].Allowed, s)
				}
			}
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

// Get asks MCPJungle (bin, registry) and compares with the repo's hub/mcp.
func Get(bin, registry, repo string) (*Status, error) {
	run := func(args ...string) (string, error) {
		// mcpjungle prints its results on stderr (hub/mcp/setup.sh note), so both streams are read
		b, err := exec.Command(bin, append([]string{"--registry", registry}, args...)...).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("mcpjungle %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
		}
		return string(b), nil
	}
	tools, err := run("list", "tools")
	if err != nil {
		return nil, err
	}
	clients, err := run("list", "mcp-clients")
	if err != nil {
		return nil, err
	}
	st := &Status{Servers: ParseTools(tools), Clients: ParseClients(clients)}
	st.Drift = Drift(st, filepath.Join(repo, "hub", "mcp"))
	return st, nil
}

// Drift: differences between the gateway and the repo files (what setup.sh would apply).
func Drift(st *Status, dir string) []string {
	var d []string
	live := map[string]Server{}
	for _, s := range st.Servers {
		live[s.Name] = s
	}
	files, _ := filepath.Glob(filepath.Join(dir, "servers", "*.json"))
	for _, f := range files {
		n := strings.TrimSuffix(filepath.Base(f), ".json")
		s, ok := live[n]
		if !ok {
			d = append(d, fmt.Sprintf("server %s: in the repo, not in the gateway (no tools listed)", n))
			continue
		}
		if len(s.Enabled) == 0 {
			d = append(d, fmt.Sprintf("server %s: no tool enabled", n))
		}
		if want, ok := readTools(filepath.Join(dir, "servers", n+".tools")); ok {
			if got := strings.Join(s.Enabled, ","); got != strings.Join(want, ",") {
				d = append(d, fmt.Sprintf("server %s: enabled %s, the .tools file says %s", n, got, strings.Join(want, ",")))
			}
		}
	}
	liveC := map[string]Client{}
	for _, c := range st.Clients {
		liveC[c.Name] = c
	}
	cf, _ := filepath.Glob(filepath.Join(dir, "clients", "*.json"))
	for _, f := range cf {
		var c struct {
			Name    string   `json:"name"`
			Allowed []string `json:"allowed_servers"`
		}
		b, err := os.ReadFile(f)
		if err != nil || json.Unmarshal(b, &c) != nil {
			d = append(d, "client file unreadable: "+filepath.Base(f))
			continue
		}
		l, ok := liveC[c.Name]
		if !ok {
			d = append(d, fmt.Sprintf("agent %s: in the repo, not in the gateway", c.Name))
			continue
		}
		if a, b := sorted(l.Allowed), sorted(c.Allowed); a != b {
			d = append(d, fmt.Sprintf("agent %s: gateway allows %s, the repo says %s", c.Name, a, b))
		}
		delete(liveC, c.Name)
	}
	for n := range liveC {
		d = append(d, fmt.Sprintf("agent %s: in the gateway, not in the repo (hub/mcp/clients/)", n))
	}
	sort.Strings(d)
	return d
}

func readTools(p string) ([]string, bool) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	var t []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			t = append(t, l)
		}
	}
	sort.Strings(t)
	return t, true
}

func sorted(s []string) string {
	c := append([]string(nil), s...)
	sort.Strings(c)
	return strings.Join(c, ",")
}

// Print: the one-screen text form.
func Print(st *Status, showTools bool) {
	fmt.Printf("Tool servers (%d)          enabled / total tools\n", len(st.Servers))
	for _, s := range st.Servers {
		fmt.Printf("  %-12s %3d / %d\n", s.Name, len(s.Enabled), len(s.Enabled)+len(s.Disabled))
		if showTools {
			for _, t := range s.Enabled {
				fmt.Printf("      on   %s\n", t)
			}
			for _, t := range s.Disabled {
				fmt.Printf("      off  %s\n", t)
			}
		}
	}
	fmt.Printf("\nAgents (%d) and the tool servers they may use\n", len(st.Clients))
	for _, c := range st.Clients {
		fmt.Printf("  %-12s %s\n", c.Name, strings.Join(c.Allowed, ", "))
	}
	if len(st.Drift) == 0 {
		fmt.Println("\nMatches the repo (hub/mcp/clients, servers/*.tools).")
		return
	}
	fmt.Printf("\nDiffers from the repo (%d): run hub/mcp/setup.sh to apply the repo\n", len(st.Drift))
	for _, x := range st.Drift {
		fmt.Println("  " + x)
	}
}
