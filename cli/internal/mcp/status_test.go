package mcp

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func read(t *testing.T, n string) string {
	b, err := os.ReadFile("testdata/" + n)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Fixtures recorded from the live gateway 2026-11-02 (MCPJungle 0.4.7).
func TestParse(t *testing.T) {
	s := ParseTools(read(t, "list-tools.txt"))
	got := map[string]string{}
	for _, x := range s {
		got[x.Name] = strings.Join([]string{itoa(len(x.Enabled)), itoa(len(x.Enabled) + len(x.Disabled))}, "/")
	}
	want := map[string]string{"browser": "25/25", "memory": "2/2", "recall": "5/30", "search": "4/4"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("server %s: got %s, want %s", k, got[k], v)
		}
	}
	c := ParseClients(read(t, "list-clients.txt"))
	if len(c) != 3 || c[2].Name != "openwebui" || strings.Join(c[2].Allowed, ",") != "search,browser" {
		t.Errorf("clients: got %+v", c)
	}
}

func TestDrift(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "servers"), 0o755)
	os.MkdirAll(filepath.Join(dir, "clients"), 0o755)
	os.WriteFile(filepath.Join(dir, "servers", "recall.json"), []byte(`{}`), 0o644)
	os.WriteFile(filepath.Join(dir, "servers", "recall.tools"), []byte("# c\nb\na\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "clients", "enso.json"), []byte(`{"name":"enso","allowed_servers":["recall"],"access_token_ref":{"env":"X"}}`), 0o644)
	st := &Status{Servers: []Server{{Name: "recall", Enabled: []string{"a", "b"}}}, Clients: []Client{{Name: "enso", Allowed: []string{"recall"}}}}
	if d := Drift(st, dir); len(d) != 0 {
		t.Errorf("matching setup reported drift: %v", d)
	}
	st.Servers[0].Enabled = []string{"a"}
	st.Clients = append(st.Clients, Client{Name: "stranger", Allowed: []string{"recall"}})
	d := Drift(st, dir)
	if len(d) != 2 || !strings.Contains(strings.Join(d, "|"), "stranger") || !strings.Contains(strings.Join(d, "|"), "the .tools file says a,b") {
		t.Errorf("drift not found: %v", d)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
