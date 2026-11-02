// Package enso: `lab enso check`, port of `agents/enso/enso.sh check` (same three lines).
// Good result: "ok: models (llama-swap)", "ok: no internet", "ok: tools (mcpjungle): connected, N tools".
package enso

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Same script as enso.sh: the key is read inside the container and never leaves it.
const probe = `
    const fs = require("fs"); const key = fs.readFileSync(process.env.HOME + "/.enso/keys/zaphod-key", "utf8");
    const t = (u, h) => fetch(u, { headers: h, signal: AbortSignal.timeout(5000) });
    (async () => {
      try { const r = await t("http://llama-swap:8080/v1/models", { Authorization: "Bearer " + key });
            console.log(r.ok ? "ok: models (llama-swap)" : "FAIL: llama-swap " + r.status); } catch (e) { console.log("FAIL: llama-swap " + e.message); }
      try { await t("https://example.com"); console.log("FAIL: internet reachable"); } catch { console.log("ok: no internet"); }
    })()`

var toolsRe = regexp.MustCompile(`connected, [0-9]+ tools`)

// Check prints the lines and returns false if any check failed.
func Check(container string) bool {
	ok := true
	b, _ := exec.Command("docker", "exec", container, "node", "-e", probe).CombinedOutput()
	out := string(b)
	fmt.Print(out)
	if strings.Contains(out, "FAIL") || !strings.Contains(out, "ok: no internet") {
		ok = false
	}
	b, err := exec.Command("docker", "exec", "-e", "PI_CODING_AGENT_DIR=/home/ubuntu/.enso/pi-agent", container,
		"pi", "mcp", "list").CombinedOutput()
	if err != nil {
		fmt.Println("FAIL: mcp")
		fmt.Print(string(b))
		return false
	}
	fmt.Printf("ok: tools (mcpjungle): %s\n", strings.Join(toolsRe.FindAllString(string(b), -1), "\n"))
	return ok
}
