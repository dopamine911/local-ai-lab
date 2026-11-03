package hub

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Monitor is one Beszel network monitor on zaphod: target, protocol, port (TCP only), interval in seconds.
type Monitor struct {
	Target, Protocol string
	Port, Interval   int
}

// Monitors: the same list as hub/beszel-monitors.sh.
var Monitors = []Monitor{
	{"http://127.0.0.1:8080/health", "http", 0, 60},
	{"http://127.0.0.1:3000/health", "http", 0, 60},
	{"http://127.0.0.1:5000/health", "http", 0, 60},
	{"http://127.0.0.1:8888/healthz", "http", 0, 60},
	{"http://127.0.0.1:8000/health", "http", 0, 60},
	{"https://chat.ai.dopamine911.com/health", "http", 0, 60},
	{"1.1.1.1", "icmp", 0, 60},
}

const beszelHub = "http://127.0.0.1:8090"

// ErrLoginFailed: already reported ("Login failed."); the command exits 1 without another message.
var ErrLoginFailed = fmt.Errorf("login failed")

// AskLogin reads the Beszel email and password like the script's `read -rp` / `read -rsp`: prompts on stderr, the
// password not echoed on a terminal. The password is only sent to the local hub.
func AskLogin() (string, string, error) {
	in := bufio.NewReader(os.Stdin)
	st, _ := os.Stdin.Stat()
	term := st != nil && st.Mode()&os.ModeCharDevice != 0 // bash's read -p shows its prompt only on a terminal
	prompt := func(s string) {
		if term {
			fmt.Fprint(os.Stderr, s)
		}
	}
	prompt("Beszel email: ")
	email, err := in.ReadString('\n')
	if err != nil && email == "" {
		return "", "", fmt.Errorf("no email given")
	}
	prompt("Beszel password: ")
	echoOff := false
	if term {
		tty := exec.Command("stty", "-echo")
		tty.Stdin = os.Stdin
		echoOff = tty.Run() == nil
	}
	pass, _ := in.ReadString('\n')
	if echoOff {
		on := exec.Command("stty", "echo")
		on.Stdin = os.Stdin
		on.Run()
	}
	fmt.Println()
	return strings.TrimRight(email, "\r\n"), strings.TrimRight(pass, "\r\n"), nil
}

// AddMonitors is `lab monitors` (port of hub/beszel-monitors.sh): adds zaphod's network monitors to Beszel; existing
// targets are skipped, so it is safe to run again.
func AddMonitors(email, pass string) error {
	body, _ := json.Marshal(map[string]string{"identity": email, "password": pass})
	var auth struct {
		Token string `json:"token"`
	}
	if err := call("POST", "/api/collections/users/auth-with-password", "", body, &auth); err != nil || auth.Token == "" {
		fmt.Println("Login failed.")
		return ErrLoginFailed
	}
	var systems struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	q := url.QueryEscape(`name="zaphod"`)
	if err := call("GET", "/api/collections/systems/records?filter="+q, auth.Token, nil, &systems); err != nil {
		return err
	}
	if len(systems.Items) == 0 {
		return fmt.Errorf("no system named zaphod in Beszel (add it first: setup/REBUILD.md)")
	}
	var existing struct {
		Items []struct {
			Target string `json:"target"`
		} `json:"items"`
	}
	if err := call("GET", "/api/collections/network_monitors/records?perPage=200", auth.Token, nil, &existing); err != nil {
		return err
	}
	have := map[string]bool{}
	for _, m := range existing.Items {
		have[m.Target] = true
	}
	for _, m := range Monitors {
		if have[m.Target] {
			fmt.Printf("already there  %s\n", m.Target)
			continue
		}
		rec := map[string]any{"system": systems.Items[0].ID, "target": m.Target, "protocol": m.Protocol,
			"interval": m.Interval, "enabled": true}
		if m.Port != 0 {
			rec["port"] = m.Port
		}
		b, _ := json.Marshal(rec)
		if err := call("POST", "/api/collections/network_monitors/records", auth.Token, b, nil); err != nil {
			return err
		}
		fmt.Printf("added          %s (%s)\n", m.Target, m.Protocol)
	}
	return nil
}

func call(method, path, token string, body []byte, out any) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, beszelHub+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d", method, path, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
