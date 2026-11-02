package webui

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Runs models.sh's own Python (taken from the script in the repo) and the Go port on the same made-up configs and
// the same fake Open WebUI, and compares their output. Skipped when the script or python3+PyYAML are missing.
func TestPlanMatchesScript(t *testing.T) {
	script := filepath.Join("..", "..", "..", "hub", "openwebui", "models.sh")
	src, err := os.ReadFile(script)
	if err != nil {
		t.Skip("models.sh not found")
	}
	s := string(src)
	i, j := strings.Index(s, "<<'PY'\n"), strings.LastIndex(s, "\nPY\n")
	if i < 0 || j < 0 {
		t.Fatal("python block not found in models.sh")
	}
	py := s[i+len("<<'PY'\n") : j]
	if exec.Command("python3", "-c", "import yaml").Run() != nil {
		t.Skip("python3 with PyYAML not available")
	}
	cur := []map[string]any{
		{"id": "coder", "name": "Qwen3.8-27B, card 1", "params": map[string]any{"function_calling": "native"}, "meta": map[string]any{"capabilities": map[string]any{"vision": false}}, "is_active": true},
		{"id": "ornith", "name": "Old name", "params": map[string]any{}, "meta": map[string]any{"capabilities": map[string]any{"vision": false}}},
		{"id": "embed", "name": "embed", "params": map[string]any{"function_calling": "native"}, "meta": map[string]any{"capabilities": map[string]any{"vision": false}}, "is_active": true},
		{"id": "marvin/big", "params": map[string]any{"function_calling": "native"}, "meta": map[string]any{}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/base") {
			json.NewEncoder(w).Encode(cur)
			return
		}
		io.Copy(io.Discard, r.Body)
		w.Write([]byte("true"))
	}))
	defer srv.Close()
	repo := t.TempDir()
	write := func(p, c string) {
		os.MkdirAll(filepath.Dir(filepath.Join(repo, p)), 0o755)
		os.WriteFile(filepath.Join(repo, p), []byte(c), 0o644)
	}
	write("hub/openwebui/models.yaml", "vision:\n  - ornith\nhidden:\n  - embed\n")
	write("zaphod/llama-swap/config/config.yaml", "models:\n  coder:\n    name: \"Qwen3.8-27B, card 1\"\n  ornith:\n    name: Ornith\n  embed:\n    cmd: x\n  newone:\n    name: New\n")
	write("marvin/llama-swap/config/config.yaml", "models:\n  big:\n    cmd: y\n")

	pyCode := strings.Replace(py, `BASE = "http://127.0.0.1:3000/api/v1/models"`, `BASE = "`+srv.URL+`/api/v1/models"`, 1)
	for _, apply := range []string{"no", "yes"} {
		c := exec.Command("python3", "-")
		c.Dir = filepath.Join(repo, "hub")
		c.Env = append(os.Environ(), "APPLY="+apply, "OPENWEBUI_API_KEY=test")
		c.Stdin = strings.NewReader(pyCode)
		want, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("script python: %v\n%s", err, want)
		}
		owuiModels = srv.URL + "/api/v1/models"
		got := capture(t, func() error { return Models(repo, "test", apply == "yes") })
		if got != string(want) {
			t.Errorf("APPLY=%s\nscript:\n%s\nlab:\n%s", apply, want, got)
		}
	}
}

func capture(t *testing.T, f func() error) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	err := f()
	w.Close()
	os.Stdout = old
	var b bytes.Buffer
	io.Copy(&b, r)
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
