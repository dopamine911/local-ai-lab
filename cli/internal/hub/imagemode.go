package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Port of hub/image-mode: lend card 2 to ComfyUI, or give it back to llama-swap.
//   on:     llama-swap profile image-mode, card 2's chat models unloaded, ComfyUI started
//   off:    ComfyUI stopped and removed (frees its memory), profile off
//   status: profile, ComfyUI, card 2 memory
// Card 1 and the speech helpers on card 2 keep running. A llama-swap config reload switches the profile off.

const lsURL = "http://127.0.0.1:8080"

// The models unloaded when image mode goes on are the ones the image-mode profile pins, read from the running config
// (a fixed list here once missed chat-teacher).

type ImageMode struct {
	Repo, Key, Card2PCI string
	DryRun              bool
}

func NewImageMode(repo, lsEnv, card2PCI string, dryRun bool) (*ImageMode, error) {
	key, err := EnvValue(lsEnv, "ZAPHOD_KEY")
	if err != nil {
		return nil, err
	}
	return &ImageMode{Repo: repo, Key: key, Card2PCI: card2PCI, DryRun: dryRun}, nil
}

func (m *ImageMode) api(method, path, body string) ([]byte, error) {
	req, _ := http.NewRequest(method, lsURL+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+m.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 { // curl -f
		return b, fmt.Errorf("%s %s: HTTP %d", method, path, resp.StatusCode)
	}
	return b, nil
}

func (m *ImageMode) card2MB() string {
	b, err := os.ReadFile("/sys/bus/pci/devices/" + m.Card2PCI + "/mem_info_vram_used")
	if err != nil {
		return "?"
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	return strconv.FormatInt(n/1048576, 10)
}

func (m *ImageMode) On() error {
	hub := HubDir(m.Repo)
	pinned, err := LS{"zaphod", lsURL, m.Key}.PinnedModels("image-mode")
	if err != nil {
		return err
	}
	if m.DryRun {
		fmt.Println("would: set llama-swap profile image-mode; unload " + strings.Join(pinned, " ") +
			"; docker compose --profile images up -d comfyui (in " + hub + "); wait up to 4 min for ComfyUI on 127.0.0.1:8188")
		return nil
	}
	if _, err := m.api("PUT", "/api/profiles/active", `{"name":"image-mode"}`); err != nil {
		return err
	}
	for _, mod := range pinned {
		m.api("POST", "/api/models/unload/"+mod, "") // not loaded: fine (script: || true)
	}
	fmt.Printf("llama-swap: profile image-mode on, card 2 chat models unloaded (card 2 now %s MB used)\n", m.card2MB())
	if err := Compose(hub, "--profile", "images", "up", "-d", "comfyui"); err != nil {
		return err
	}
	fmt.Print("waiting for ComfyUI")
	ready := func() bool {
		r, err := (&http.Client{Timeout: 5 * time.Second}).Get("http://127.0.0.1:8188/")
		if err != nil {
			return false
		}
		r.Body.Close()
		return r.StatusCode < 400
	}
	for i := 0; i < 120 && !ready(); i++ {
		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}
	fmt.Println()
	if !ready() {
		return fmt.Errorf("ComfyUI did not start; see: docker logs --tail 30 comfyui")
	}
	domain, _ := EnvValue(hub+"/.env", "DOMAIN")
	fmt.Printf("ComfyUI ready: https://comfy.%s\n", domain)
	return nil
}

func (m *ImageMode) Off() error {
	hub := HubDir(m.Repo)
	if m.DryRun {
		fmt.Println("would: docker compose --profile images stop comfyui and rm -f comfyui (in " + hub + "); set llama-swap profile none")
		return nil
	}
	Quiet(hub, "--profile", "images", "stop", "comfyui")
	Quiet(hub, "--profile", "images", "rm", "-f", "comfyui")
	if _, err := m.api("PUT", "/api/profiles/active", `{"name":null}`); err != nil {
		return err
	}
	fmt.Printf("ComfyUI stopped, profile off, card 2 now %s MB used.\n", m.card2MB())
	fmt.Println("coder-2, ornith and chat load again on their next request.")
	return nil
}

func (m *ImageMode) Status() error {
	active := "?"
	if b, err := m.api("GET", "/api/profiles", ""); err == nil {
		var p struct {
			Active *string `json:"active"`
		}
		if json.Unmarshal(b, &p) == nil {
			active = "none"
			if p.Active != nil && *p.Active != "" {
				active = *p.Active
			}
		}
	}
	fmt.Printf("llama-swap profile: %s\n", active)
	st, _ := exec.Command("docker", "ps", "--filter", "name=^comfyui$", "--format", "{{.Status}}").Output()
	s := strings.TrimSpace(string(st))
	if s == "" {
		s = "stopped"
	}
	fmt.Printf("ComfyUI: %s\n", s)
	fmt.Printf("card 2: %s MB used\n", m.card2MB())
	return nil
}
