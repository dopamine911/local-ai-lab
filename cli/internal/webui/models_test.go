package webui

import (
	"strings"
	"testing"
)

// Expected lines follow models.sh's Python formatting: f"  {mid:12} ...", True/False, 'None' for a missing name.
func TestPlan(t *testing.T) {
	models := []model{{"coder", "Qwen3.8-27B, card 1"}, {"ornith", "Ornith"}, {"embed", "embed"}, {"marvin/big", "big"}, {"newone", "New"}}
	vision := map[string]bool{"ornith": true}
	hidden := map[string]bool{"embed": true}
	cur := map[string]map[string]any{
		"coder":      {"name": "Qwen3.8-27B, card 1", "params": map[string]any{"function_calling": "native"}, "meta": map[string]any{"capabilities": map[string]any{"vision": false}}, "is_active": true},
		"ornith":     {"name": "Old name", "params": map[string]any{}, "meta": map[string]any{"capabilities": map[string]any{"vision": false}, "description": "keep me"}},
		"embed":      {"name": "embed", "params": map[string]any{"function_calling": "native"}, "meta": map[string]any{"capabilities": map[string]any{"vision": false}}, "is_active": true},
		"marvin/big": {"params": map[string]any{"function_calling": "native"}, "meta": map[string]any{}},
	}
	entries, changes := Plan(models, vision, hidden, cur)
	want := []string{
		"  ornith       name 'Old name' -> 'Ornith', function calling -> native, vision -> True",
		"  embed        hidden",
		"  marvin/big   name 'None' -> 'big', vision -> False",
		"  newone       new entry (vision False, shown)",
	}
	if strings.Join(changes, "\n") != strings.Join(want, "\n") {
		t.Errorf("changes:\n%s\nwant:\n%s", strings.Join(changes, "\n"), strings.Join(want, "\n"))
	}
	if len(entries) != 5 || asMap(entries[1]["meta"])["description"] != "keep me" {
		t.Errorf("entries should keep other meta fields: %+v", entries[1])
	}
}
