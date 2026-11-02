// Package manifest ports tuning/manifest.sh: the machine and software facts that belong with every test result,
// as key=value lines (each becomes an "env.<key>" tag on the MLflow run). Output must match the script exactly:
// same keys, order and "unknown" fallbacks. Only stable facts; no secrets.
package manifest

import (
	"os"
	"strconv"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Field is one key=value line.
type Field struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Collect reads the facts of the machine it runs on. container is the llama-swap container name.
func Collect(container string) []Field {
	var f []Field
	add := func(k, v string) { f = append(f, Field{k, v}) }

	host, _ := os.Hostname()
	add("host", host)
	add("kernel", orUnknown(readFile("/proc/sys/kernel/osrelease")))
	add("mesa", orUnknown(run("dpkg-query", "-W", "-f=${Version}", "mesa-vulkan-drivers")))
	add("board", orUnknown(readFile("/sys/class/dmi/id/board_name"))+" BIOS "+orUnknown(readFile("/sys/class/dmi/id/bios_version")))
	add("cpu", orUnknown(cpuModel(readFile("/proc/cpuinfo"))))
	add("ram_gb", orUnknown(ramGB(readFile("/proc/meminfo"))))
	add("gpus", gpus())

	if _, ok := runOK("docker", "inspect", container); !ok {
		add("image", "unknown (llama-swap not running)")
		return f
	}
	add("image", trimNL(run("docker", "inspect", container, "--format", "{{.Config.Image}}")))
	add("image_id", imageID(run("docker", "inspect", container, "--format", "{{.Image}}")))
	add("llama_cpp_commit", orUnknown(LlamaCppCommit(runAll("docker", "exec", container, "llama-server", "--version"))))
	add("vulkan_devices", orUnknown(VulkanDevices(runAll("docker", "exec", container, "llama-server", "--list-devices"))))
	add("llama_swap_version", orUnknown(LlamaSwapVersion(runAll("docker", "exec", container, "llama-swap", "--version"))))
	return f
}

// --- parsers (tested with recorded outputs in manifest_test.go) ---------------------------------------------------

var commitRe = regexp.MustCompile(`commit ([0-9a-f]*)`)

// LlamaCppCommit: the hex after "commit " on the first line that has it
// (script: sed -n 's/.*commit \([0-9a-f]*\).*/\1/p' | head -1; sed's greedy .* takes the last match on a line).
func LlamaCppCommit(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if m := commitRe.FindAllStringSubmatch(line, -1); m != nil {
			return m[len(m)-1][1]
		}
	}
	return ""
}

var vulkanLine = regexp.MustCompile(`^ *Vulkan[0-9]+:`)
var mibFree = regexp.MustCompile(`, [0-9]+ MiB free`)
var leadingSpaces = regexp.MustCompile(`^ *`)

// VulkanDevices: lines "  VulkanN: ..." without leading spaces and without ", N MiB free", joined with "; ".
func VulkanDevices(out string) string {
	var keep []string
	for _, line := range strings.Split(out, "\n") {
		if vulkanLine.MatchString(line) {
			line = leadingSpaces.ReplaceAllString(line, "")
			line = mibFree.ReplaceAllString(line, "") // sed 's/…//' replaces the first only; one per line
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "; ")
}

// LlamaSwapVersion: the second word of the first line (script: awk '{print $2}' | head -1).
func LlamaSwapVersion(out string) string {
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		return ""
	}
	w := strings.Fields(lines[0])
	if len(w) < 2 {
		return ""
	}
	return w[1]
}

func cpuModel(cpuinfo string) string {
	for _, line := range strings.Split(cpuinfo, "\n") {
		if strings.Contains(line, "model name") {
			_, after, ok := strings.Cut(line, ":")
			if !ok {
				return ""
			}
			return strings.TrimLeft(after, " ")
		}
	}
	return ""
}

func ramGB(meminfo string) string {
	for _, line := range strings.Split(meminfo, "\n") {
		w := strings.Fields(line)
		if len(w) >= 2 && w[0] == "MemTotal:" {
			var kb int64
			for _, c := range w[1] {
				if c < '0' || c > '9' {
					return ""
				}
				kb = kb*10 + int64(c-'0')
			}
			return strconv.FormatInt(kb/1048576, 10)
		}
	}
	return ""
}

func imageID(out string) string {
	s := trimNL(out)
	if len(s) < 19 { // script: cut -c8-19
		if len(s) <= 7 {
			return ""
		}
		return s[7:]
	}
	return s[7:19]
}

// gpus: PCI address and VBIOS of every AMD GPU (vendor 0x1002), in glob order.
func gpus() string {
	dirs, _ := filepath.Glob("/sys/class/drm/card[0-9]*/device")
	var out []string
	for _, d := range dirs {
		v, err := os.ReadFile(filepath.Join(d, "vendor"))
		if err != nil || trimNL(string(v)) != "0x1002" {
			continue
		}
		real, err := filepath.EvalSymlinks(d)
		if err != nil {
			continue
		}
		pci := strings.TrimPrefix(filepath.Base(real), "0000:")
		out = append(out, pci+" VBIOS "+orUnknown(readFile(filepath.Join(d, "vbios_version"))))
	}
	if len(out) == 0 {
		return "unknown"
	}
	return strings.Join(out, "; ")
}

// --- small helpers mirroring the script's v() and $(…) ----------------------------------------------------------

// trimNL strips trailing newlines, like shell $(…).
func trimNL(s string) string { return strings.TrimRight(s, "\n") }

func orUnknown(s string) string {
	if s = trimNL(s); s == "" {
		return "unknown"
	}
	return s
}

func readFile(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return trimNL(string(b))
}

// run: stdout only, "" on failure (script: v cmd, stderr to /dev/null).
func run(name string, args ...string) string {
	out, ok := runOK(name, args...)
	if !ok {
		return ""
	}
	return out
}

func runOK(name string, args ...string) (string, bool) {
	b, err := exec.Command(name, args...).Output()
	return string(b), err == nil
}

// runAll: stdout and stderr together, whatever the exit code (script: cmd 2>&1 | …).
func runAll(name string, args ...string) string {
	b, _ := exec.Command(name, args...).CombinedOutput()
	return string(b)
}
