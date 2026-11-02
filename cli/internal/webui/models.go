package webui

// `lab webui models [--apply]`: port of hub/openwebui/models.sh. Per-model settings in Open WebUI from
// hub/openwebui/models.yaml + the llama-swap configs (zaphod's models, marvin's as "marvin/<id>"): native function
// calling, every ability on except vision (only the models listed under `vision`), hidden models kept out of the
// picker. Read-only unless --apply (Open WebUI admin API "models/import": updates existing entries).
// Needs OPENWEBUI_API_KEY in hub/.env (read, never printed).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

var owuiModels = "http://127.0.0.1:3000/api/v1/models"

var caps = []string{"file_context", "vision", "file_upload", "web_search", "image_generation", "code_interpreter",
	"terminal", "citations", "status_updates", "memory", "builtin_tools"}

type model struct{ ID, Name string }

// orderedModels: the models of a llama-swap config in file order, with their display names.
func orderedModels(path, prefix string) ([]model, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	var out []model
	if len(doc.Content) == 0 {
		return out, nil
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "models" {
			continue
		}
		ms := root.Content[i+1]
		for j := 0; j+1 < len(ms.Content); j += 2 {
			id := ms.Content[j].Value
			name := id
			body := ms.Content[j+1]
			for k := 0; k+1 < len(body.Content); k += 2 {
				if body.Content[k].Value == "name" && body.Content[k+1].Value != "" {
					name = body.Content[k+1].Value
				}
			}
			out = append(out, model{prefix + id, name})
		}
	}
	return out, nil
}

// Plan works out the entries to import and the change lines (the script's output), from the current entries.
func Plan(models []model, vision, hidden map[string]bool, current map[string]map[string]any) ([]map[string]any, []string) {
	var entries []map[string]any
	var changes []string
	for _, m := range models {
		c := map[string]any{}
		for _, k := range caps {
			c[k] = true
		}
		c["vision"] = vision[m.ID]
		active := !hidden[m.ID]
		cur := current[m.ID]
		meta := copyMap(asMap(cur["meta"]))
		meta["capabilities"] = c
		params := copyMap(asMap(cur["params"]))
		params["function_calling"] = "native"
		entries = append(entries, map[string]any{"id": m.ID, "base_model_id": nil, "name": m.Name, "meta": meta,
			"params": params, "is_active": active})
		if cur == nil {
			shown := "shown"
			if !active {
				shown = "hidden"
			}
			changes = append(changes, fmt.Sprintf("  %-12s new entry (vision %s, %s)", m.ID, pyBool(vision[m.ID]), shown))
			continue
		}
		var diff []string
		if n, _ := cur["name"].(string); n != m.Name {
			diff = append(diff, fmt.Sprintf("name '%s' -> '%s'", pyStr(cur["name"]), m.Name))
		}
		if fc, _ := asMap(cur["params"])["function_calling"].(string); fc != "native" {
			diff = append(diff, "function calling -> native")
		}
		if v, ok := asMap(asMap(cur["meta"])["capabilities"])["vision"].(bool); !ok || v != vision[m.ID] {
			diff = append(diff, "vision -> "+pyBool(vision[m.ID]))
		}
		isActive := true
		if v, ok := cur["is_active"]; ok {
			isActive = truthy(v)
		}
		if isActive != active {
			if active {
				diff = append(diff, "shown")
			} else {
				diff = append(diff, "hidden")
			}
		}
		if len(diff) > 0 {
			line := fmt.Sprintf("  %-12s ", m.ID)
			for i, d := range diff {
				if i > 0 {
					line += ", "
				}
				line += d
			}
			changes = append(changes, line)
		}
	}
	return entries, changes
}

// Models prints the differences; with apply it imports the entries when something differs.
func Models(repo, apiKey string, apply bool) error {
	hub := filepath.Join(repo, "hub")
	var want struct {
		Vision []string `yaml:"vision"`
		Hidden []string `yaml:"hidden"`
	}
	b, err := os.ReadFile(filepath.Join(hub, "openwebui", "models.yaml"))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, &want); err != nil {
		return err
	}
	var models []model
	for _, src := range []struct{ path, prefix string }{
		{filepath.Join(repo, "zaphod", "llama-swap", "config", "config.yaml"), ""},
		{filepath.Join(repo, "marvin", "llama-swap", "config", "config.yaml"), "marvin/"},
	} {
		ms, err := orderedModels(src.path, src.prefix)
		if err != nil {
			return err
		}
		models = append(models, ms...)
	}
	var list []map[string]any
	if err := call(apiKey, "GET", "/base", nil, &list); err != nil {
		return err
	}
	current := map[string]map[string]any{}
	for _, m := range list {
		if id, ok := m["id"].(string); ok {
			current[id] = m
		}
	}
	entries, changes := Plan(models, set(want.Vision), set(want.Hidden), current)
	colon := ""
	if len(changes) > 0 {
		colon = ":"
	}
	fmt.Printf("%d model(s) differ from models.yaml%s\n", len(changes), colon)
	for i, c := range changes {
		if i > 0 {
			fmt.Print("\n")
		}
		fmt.Print(c)
	}
	fmt.Print("\n") // print("\n".join(changes)) prints an empty line when nothing differs
	if apply && len(changes) > 0 {
		var ok any
		if err := call(apiKey, "POST", "/import", map[string]any{"models": entries}, &ok); err != nil {
			return err
		}
		if truthy(ok) {
			fmt.Println("applied")
		} else {
			fmt.Println("import returned false")
		}
	}
	return nil
}

func call(key, method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, owuiModels+path, r)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Open WebUI %s %s: HTTP %d", method, path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func copyMap(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func set(l []string) map[string]bool {
	s := map[string]bool{}
	for _, x := range l {
		s[x] = true
	}
	return s
}

// Python's truthiness for JSON values.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// pyStr: how Python's f-string shows a JSON value inside '…' (None for a missing name).
func pyStr(v any) string {
	if v == nil {
		return "None"
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
