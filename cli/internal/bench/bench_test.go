package bench

import (
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

func TestPyNum(t *testing.T) {
	for in, want := range map[float64]string{54.0: "54.0", 44.3: "44.3", 102.1: "102.1", 0.5: "0.5", 318.4: "318.4"} {
		if got := pyNum(in); got != want {
			t.Errorf("pyNum(%v) = %q, want %q", in, got, want)
		}
	}
}

// The Rerun button's LAB_RERUN must be byte for byte what the scripts made: printf '%s\0' name "$@" | base64 -w0.
func TestRerunMatchesScript(t *testing.T) {
	args := []string{"marvin/big", "a note with spaces", "--full", "--seed", "3"}
	sh := exec.Command("bash", "-c", `printf '%s\0' quality "$@" | base64 -w0`, "x")
	sh.Args = append(sh.Args, args...)
	want, err := sh.Output()
	if err != nil {
		t.Skip("bash/base64 not available:", err)
	}
	dir := t.TempDir()
	os.Unsetenv("LAB_RERUN")
	os.Unsetenv("LAB_RERUN_CWD")
	if err := os.MkdirAll(dir+"/tuning", 0o755); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := enterTuning(&config.Config{Repo: dir}, "quality", args...); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("LAB_RERUN"); got != string(want) {
		b, _ := base64.StdEncoding.DecodeString(got)
		t.Errorf("LAB_RERUN = %q (%q), want %q", got, strings.ReplaceAll(string(b), "\x00", "|"), want)
	}
	if os.Getenv("LAB_RERUN_CWD") != dir+"/tuning" {
		t.Errorf("LAB_RERUN_CWD = %q", os.Getenv("LAB_RERUN_CWD"))
	}
}

// A kill (SIGTERM) during the benchmark stops the running command and still starts llama-swap again; exit 130.
func TestSwapRestartsOnSignal(t *testing.T) {
	var calls []string
	composeFn = func(args ...string) error { calls = append(calls, strings.Join(args, " ")); return nil }
	defer func() { composeFn = compose }()
	start := time.Now()
	go func() { time.Sleep(300 * time.Millisecond); syscall.Kill(os.Getpid(), syscall.SIGTERM) }()
	err := withSwapPaused(false, func(g *guard) error {
		return g.run(exec.Command("sleep", "30"))
	})
	if e, ok := err.(ExitError); !ok || e.Code != 130 {
		t.Errorf("err = %v, want exit 130", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Errorf("the running command was not stopped (took %v)", time.Since(start))
	}
	if strings.Join(calls, "; ") != "stop -t 30; start" {
		t.Errorf("compose calls = %q, want stop then start", calls)
	}
}
