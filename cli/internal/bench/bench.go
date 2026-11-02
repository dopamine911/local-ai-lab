package bench

import (
	"fmt"
	"os/exec"
)

// ExitError carries a command's exit code up to main, which exits with it (like the script would).
type ExitError struct{ Code int }

func (e ExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

func exitErr(code int) error { return ExitError{code} }

// runQuiet runs c with stdout and stderr together and returns the output and the exit code.
func runQuiet(c *exec.Cmd) (string, int) {
	out, err := c.CombinedOutput()
	if err != nil {
		return string(out), exitCode(err)
	}
	return string(out), 0
}
