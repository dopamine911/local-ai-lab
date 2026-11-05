package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// LS talks to one llama-swap (zaphod or marvin) with its API key.
type LS struct {
	Name, URL, Key string
}

func (l LS) API(method, path, body string) ([]byte, error) {
	req, err := http.NewRequest(method, l.URL+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+l.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 { // like curl -f
		return b, fmt.Errorf("%s %s %s: HTTP %d", l.Name, method, path, resp.StatusCode)
	}
	return b, nil
}

type profilesResp struct {
	Active   *string `json:"active"`
	Profiles []struct {
		ID   string            `json:"id"`
		Pins map[string]string `json:"pins"`
	} `json:"profiles"`
}

// ActiveProfile returns the active profile's name, or "none".
func (l LS) ActiveProfile() (string, error) {
	b, err := l.API("GET", "/api/profiles", "")
	if err != nil {
		return "?", err
	}
	var p profilesResp
	if err := json.Unmarshal(b, &p); err != nil {
		return "?", err
	}
	if p.Active == nil || *p.Active == "" {
		return "none", nil
	}
	return *p.Active, nil
}

// PinnedModels returns the models a profile pins (sorted), read from the running config, so the unload list always
// matches the profile. Error if the profile is not in the running config (config not deployed yet).
func (l LS) PinnedModels(profile string) ([]string, error) {
	b, err := l.API("GET", "/api/profiles", "")
	if err != nil {
		return nil, err
	}
	var p profilesResp
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	for _, pr := range p.Profiles {
		if pr.ID == profile {
			var ms []string
			for m := range pr.Pins {
				ms = append(ms, m)
			}
			sort.Strings(ms)
			return ms, nil
		}
	}
	return nil, fmt.Errorf("%s: profile %q is not in its running llama-swap config", l.Name, profile)
}

// SetProfile switches the active profile; "" switches it off.
func (l LS) SetProfile(name string) error {
	body := `{"name":null}`
	if name != "" {
		body = fmt.Sprintf(`{"name":%q}`, name)
	}
	_, err := l.API("PUT", "/api/profiles/active", body)
	return err
}

// Running returns the loaded models (local ones; peers are not listed).
func (l LS) Running() ([]string, error) {
	b, err := l.API("GET", "/running", "")
	if err != nil {
		return nil, err
	}
	var r struct {
		Running []struct {
			Model string `json:"model"`
		} `json:"running"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	var ms []string
	for _, m := range r.Running {
		ms = append(ms, m.Model)
	}
	return ms, nil
}

// Unload unloads one model. Not loaded is fine (returns no error worth reporting).
func (l LS) Unload(model string) { l.API("POST", "/api/models/unload/"+model, "") }
