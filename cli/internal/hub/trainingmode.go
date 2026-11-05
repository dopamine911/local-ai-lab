package hub

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Training mode: lend zaphod's two cards and marvin's GPU to model training.
//   on:     llama-swap profile "training" on both hosts, every model it pins unloaded, then wait until the memory is
//           free (zaphod cards <= 8 GB used each, marvin MemAvailable >= 100 GB) and print it
//   off:    profile off on both hosts; models load again on their next request
//   status: profile, loaded models and memory per host
// A training harness checks GET /api/profiles for active == "training" on each host it uses.
// A llama-swap config reload switches the profile off (and reloads the preloaded models): run `on` again then.

const (
	TrainingProfile   = "training"
	cardMaxUsedMB     = 8 * 1024   // per zaphod card, after unloading (helpers + screen stay)
	marvinMinAvailMB  = 100 * 1024 // MemAvailable on marvin
	trainingWaitLimit = 3 * time.Minute
)

type TrainingMode struct {
	Hosts     []LS     // zaphod first, then marvin
	CardPCIs  []string // zaphod card 1, card 2
	MarvinSSH string
	DryRun    bool
}

func NewTrainingMode(lsEnv, marvinURL, marvinSSH string, cardPCIs []string, dryRun bool) (*TrainingMode, error) {
	zk, err := EnvValue(lsEnv, "ZAPHOD_KEY")
	if err != nil {
		return nil, err
	}
	mk, err := EnvValue(lsEnv, "MARVIN_KEY")
	if err != nil {
		return nil, err
	}
	return &TrainingMode{
		Hosts:     []LS{{"zaphod", lsURL, zk}, {"marvin", marvinURL, mk}},
		CardPCIs:  cardPCIs,
		MarvinSSH: marvinSSH,
		DryRun:    dryRun,
	}, nil
}

// memory: zaphod card used MB (one per card; -1 unreadable) and marvin MemAvailable MB (-1 unreadable).
func (t *TrainingMode) memory() ([]int64, int64) {
	var cards []int64
	for _, pci := range t.CardPCIs {
		cards = append(cards, readMB("/sys/bus/pci/devices/"+pci+"/mem_info_vram_used"))
	}
	avail := int64(-1)
	out, err := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", t.MarvinSSH,
		"grep MemAvailable /proc/meminfo").Output()
	if err == nil {
		if f := strings.Fields(string(out)); len(f) >= 2 {
			if kb, err := strconv.ParseInt(f[1], 10, 64); err == nil {
				avail = kb / 1024
			}
		}
	}
	return cards, avail
}

func readMB(path string) int64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return -1
	}
	return n / 1048576
}

func memFree(cards []int64, avail int64) bool {
	for _, c := range cards {
		if c < 0 || c > cardMaxUsedMB {
			return false
		}
	}
	return avail >= marvinMinAvailMB
}

func gb(mb int64) string {
	if mb < 0 {
		return "?"
	}
	return fmt.Sprintf("%.1f GB", float64(mb)/1024)
}

func (t *TrainingMode) printMemory(cards []int64, avail int64) {
	mark := func(ok bool) string {
		if ok {
			return "ok"
		}
		return "NOT FREE"
	}
	for i, c := range cards {
		fmt.Printf("  zaphod card %d: %s used (want <= %s) %s\n", i+1, gb(c), gb(cardMaxUsedMB), mark(c >= 0 && c <= cardMaxUsedMB))
	}
	fmt.Printf("  marvin: %s available (want >= %s) %s\n", gb(avail), gb(marvinMinAvailMB), mark(avail >= marvinMinAvailMB))
}

func (t *TrainingMode) On() error {
	// Read every host's pins first: if the profile is missing anywhere, change nothing.
	pins := make([][]string, len(t.Hosts))
	for i, h := range t.Hosts {
		ms, err := h.PinnedModels(TrainingProfile)
		if err != nil {
			return fmt.Errorf("%w; nothing changed", err)
		}
		pins[i] = ms
	}
	if t.DryRun {
		for i, h := range t.Hosts {
			fmt.Printf("would: %s: set profile %s; unload %s\n", h.Name, TrainingProfile, strings.Join(pins[i], " "))
		}
		fmt.Println("would: wait up to 3 min for zaphod cards <= 8 GB used and marvin >= 100 GB available")
		return nil
	}
	var failed []string
	for i, h := range t.Hosts {
		if prev, err := h.ActiveProfile(); err == nil && prev != "none" && prev != TrainingProfile {
			fmt.Printf("%s: replacing profile %s (only one can be active)\n", h.Name, prev)
		}
		if err := h.SetProfile(TrainingProfile); err != nil {
			fmt.Printf("%s: could not set the profile: %v\n", h.Name, err)
			failed = append(failed, h.Name)
			continue
		}
		for _, m := range pins[i] {
			h.Unload(m)
		}
		fmt.Printf("%s: profile %s on; unloaded %s\n", h.Name, TrainingProfile, strings.Join(pins[i], " "))
	}
	fmt.Print("waiting for the memory to be free")
	deadline := time.Now().Add(trainingWaitLimit)
	cards, avail := t.memory()
	for !memFree(cards, avail) && time.Now().Before(deadline) {
		fmt.Print(".")
		time.Sleep(3 * time.Second)
		cards, avail = t.memory()
	}
	fmt.Println()
	t.printMemory(cards, avail)
	if ms, err := t.Hosts[0].Running(); err == nil {
		fmt.Printf("zaphod still loaded: %s\n", list(ms))
	}
	if len(failed) > 0 {
		return fmt.Errorf("profile not set on %s", strings.Join(failed, ", "))
	}
	if !memFree(cards, avail) {
		return fmt.Errorf("memory not free after %s: see `lab training-mode status`", trainingWaitLimit)
	}
	fmt.Println("training mode on: memory free")
	return nil
}

func (t *TrainingMode) Off() error {
	if t.DryRun {
		fmt.Println("would: set profile none on zaphod and marvin")
		return nil
	}
	var failed []string
	for _, h := range t.Hosts {
		prev, _ := h.ActiveProfile()
		if prev != "none" && prev != TrainingProfile && prev != "?" {
			fmt.Printf("%s: profile is %s, not %s: left as it is\n", h.Name, prev, TrainingProfile)
			continue
		}
		if err := h.SetProfile(""); err != nil {
			fmt.Printf("%s: could not switch the profile off: %v\n", h.Name, err)
			failed = append(failed, h.Name)
			continue
		}
		fmt.Printf("%s: profile off\n", h.Name)
	}
	if len(failed) > 0 {
		return fmt.Errorf("profile still on at %s", strings.Join(failed, ", "))
	}
	fmt.Println("Models load again on their next request (2026-11-05: coder 11 s, marvin's big 29 s; longer after a reboot).")
	return nil
}

func (t *TrainingMode) Status() error {
	for _, h := range t.Hosts {
		p, err := h.ActiveProfile()
		if err != nil {
			fmt.Printf("%s: not reachable (%v)\n", h.Name, err)
			continue
		}
		ms, _ := h.Running()
		fmt.Printf("%s: profile %s; loaded: %s\n", h.Name, p, list(ms))
	}
	cards, avail := t.memory()
	t.printMemory(cards, avail)
	return nil
}

func list(ms []string) string {
	if len(ms) == 0 {
		return "nothing"
	}
	return strings.Join(ms, " ")
}
