package machines

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// Check is one step of setup/host.sh: is it done on this machine?
type Check struct {
	Step   string `json:"step"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"` // a command for Daniel (printed, never run)
}

const hostFix = "sudo ~/homelab/setup/host.sh"

func out(name string, args ...string) (string, error) {
	b, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(b)), err
}

// hostPackages reads the apt-get install line(s) of setup/host.sh step 1, so the list has one source.
func hostPackages(cfg *config.Config) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(cfg.Repo, "setup", "host.sh"))
	if err != nil {
		return nil, err
	}
	var pk []string
	in := false
	for _, l := range strings.Split(string(b), "\n") {
		if !in {
			_, rest, ok := strings.Cut(l, "apt-get install -y -q ")
			if !ok {
				continue
			}
			in, l = true, rest
		}
		l, _, _ = strings.Cut(l, "#") // comments
		cont := strings.HasSuffix(strings.TrimSpace(l), "\\")
		pk = append(pk, strings.Fields(strings.TrimSuffix(strings.TrimSpace(l), "\\"))...)
		if !cont {
			break
		}
	}
	if len(pk) == 0 {
		return nil, fmt.Errorf("no apt-get install line in setup/host.sh")
	}
	return pk, nil
}

// SetupCheck is `lab setup check`: every step of setup/host.sh, read-only, for this machine. Nothing is changed and
// nothing needs sudo; a step that is not done names the fix.
func SetupCheck(cfg *config.Config) []Check {
	host, _ := os.Hostname()
	u, _ := user.Current()
	var cs []Check
	add := func(step string, ok bool, detail, fix string) {
		if ok {
			fix = ""
		}
		cs = append(cs, Check{step, ok, detail, fix})
	}

	// 1. packages
	pk, err := hostPackages(cfg)
	if host == "marvin" {
		pk = append(pk, "tuned")
	}
	if err != nil {
		add("1 packages", false, err.Error(), hostFix)
	} else {
		var missing []string
		for _, p := range pk {
			if s, _ := out("dpkg-query", "-W", "-f=${Status}", p); s != "install ok installed" {
				missing = append(missing, p)
			}
		}
		add("1 packages", len(missing) == 0, fmt.Sprintf("%d installed", len(pk)-len(missing))+listIf(" missing: ", missing), hostFix)
	}

	// 2. groups
	g, _ := out("id", "-nG")
	var miss []string
	for _, want := range []string{"docker", "render", "video"} {
		if !hasWord(g, want) {
			miss = append(miss, want)
		}
	}
	add("2 groups (docker, render, video)", len(miss) == 0, "in: "+g+listIf("; missing: ", miss), hostFix+" (then log out and in)")

	// 3. SSH keys from GitHub
	ak, _ := os.ReadFile(filepath.Join(u.HomeDir, ".ssh", "authorized_keys"))
	n := strings.Count(string(ak), "ssh-import-id gh:dopamine911")
	add("3 SSH keys from GitHub", n > 0, fmt.Sprintf("%d key(s) from gh:dopamine911", n), "ssh-import-id gh:dopamine911")

	// 4. Tailscale, and llama-swap shared on the tailnet
	ts, err := out("tailscale", "status", "--self", "--peers=false")
	add("4 Tailscale up", err == nil, firstLine(ts, err), "sudo tailscale up")
	serve, _ := out("tailscale", "serve", "status")
	add("4 Tailscale serve tcp 8080 -> localhost:8080", strings.Contains(serve, "localhost:8080"),
		firstLine(serve, nil), "sudo tailscale serve --bg --tcp=8080 tcp://localhost:8080")

	// 5. model folder
	if st, err := os.Stat(ModelsDir); err != nil || !st.IsDir() {
		add("5 /srv/models", false, "missing", hostFix)
	} else {
		owner, _ := out("stat", "-c", "%U", ModelsDir)
		add("5 /srv/models", owner == u.Username, "owner "+owner, "sudo chown "+u.Username+" /srv/models")
	}

	// 6. ~/llama-swap link
	want := filepath.Join(cfg.Repo, host, "llama-swap")
	got, _ := os.Readlink(filepath.Join(u.HomeDir, "llama-swap"))
	add("6 ~/llama-swap link", got == want, "-> "+got, "ln -sfn "+want+" ~/llama-swap")

	// 6b. time zone
	tz, _ := out("timedatectl", "show", "-p", "Timezone", "--value")
	add("6b time zone America/Chicago", tz == "America/Chicago", tz, "sudo timedatectl set-timezone America/Chicago")

	if host == "zaphod" {
		for _, s := range []struct{ step, unit, src string }{
			{"6c notification buttons (lab-actions)", "lab-actions", "hub/actions/lab-actions.service"},
			{"6d watchdog (lab-watchdog)", "lab-watchdog", "hub/watchdog/lab-watchdog.service"},
		} {
			en, _ := out("systemctl", "is-enabled", s.unit)
			ac, _ := out("systemctl", "is-active", s.unit)
			same := sameFile(filepath.Join(cfg.Repo, s.src), "/etc/systemd/system/"+s.unit+".service")
			d := en + ", " + ac
			if !same {
				d += ", unit file differs from the repo"
			}
			add(s.step, en == "enabled" && ac == "active" && same, d, hostFix)
		}
	}

	if host == "marvin" {
		f := "/etc/default/grub.d/90-gpu-memory.cfg"
		wantG := `GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT amdgpu.gttsize=122880 ttm.pages_limit=31457280"`
		b, _ := os.ReadFile(f)
		cmd, _ := os.ReadFile("/proc/cmdline")
		set, live := strings.TrimSpace(string(b)) == wantG, strings.Contains(string(cmd), "amdgpu.gttsize=122880")
		d := "grub setting " + yes(set) + ", in use since boot " + yes(live)
		fix := hostFix
		if set && !live {
			fix = "sudo reboot"
		}
		add("7 GPU memory 120 GB (grub)", set && live, d, fix)

		prof, _ := out("tuned-adm", "active")
		prof = strings.TrimPrefix(prof, "Current active profile: ")
		add("8 tuned accelerator-performance", prof == "accelerator-performance", prof, hostFix)

		pm, err := os.ReadFile("/sys/class/ec_su_axb35/apu/power_mode")
		if err != nil {
			add("9 fan-chip module ec_su_axb35", false, "not loaded",
				"sudo ~/homelab/marvin/fan/ec-module.sh prepare, reboot (MOK screen), then ... install (host.sh step 9)")
		} else {
			add("9 fan-chip module ec_su_axb35", true, "loaded; BIOS power mode "+strings.TrimSpace(string(pm)), "")
			p := strings.TrimSpace(string(pm))
			add("9 BIOS power mode performance", p == "performance", p,
				"echo performance | sudo tee /sys/class/ec_su_axb35/apu/power_mode")
			fc, _ := out("systemctl", "is-enabled", "marvin-fan-curve")
			add("9 fan curve at boot (marvin-fan-curve)", fc == "enabled", fc, "sudo ~/homelab/marvin/fan/curve.sh boot on")
		}
	}

	// 10. lab CLI installed and as new as the repo
	inst := filepath.Join(u.HomeDir, ".local", "bin", "lab")
	v, err := out(inst, "version")
	v = strings.TrimPrefix(firstLine(v, nil), "lab ")
	head, _ := out("git", "-C", cfg.Repo, "rev-parse", "--short", "HEAD")
	ok := err == nil && strings.TrimSuffix(v, "-dirty") == head
	d := "installed " + v + ", repo at " + head
	if err != nil {
		d = "not installed"
	}
	add("10 lab CLI", ok, d, "~/homelab/cli/build.sh install")
	return cs
}

func listIf(prefix string, xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return prefix + strings.Join(xs, ", ")
}

func hasWord(s, w string) bool {
	for _, f := range strings.Fields(s) {
		if f == w {
			return true
		}
	}
	return false
}

func firstLine(s string, err error) string {
	if err != nil && s == "" {
		return err.Error()
	}
	l, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(l)
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func sameFile(a, b string) bool {
	x, err1 := os.ReadFile(a)
	y, err2 := os.ReadFile(b)
	return err1 == nil && err2 == nil && string(x) == string(y)
}
