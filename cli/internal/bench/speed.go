package bench

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
	"github.com/dopamine911/local-ai-lab/cli/internal/hub"
)

// speedPy: tuning/speed's Python, word for word. It runs on the host's python3, as before: the 32K prompt is built
// from that Python's own standard library files, so another Python (or Go) would change the test text.
const speedPy = `import json, os, statistics, sysconfig, time, urllib.request, uuid
from pathlib import Path
URL, KEY, MODEL = os.environ["BASE_URL"] + "/chat/completions", os.environ["API_KEY"], os.environ["MODEL"]
H = {"Authorization": "Bearer " + KEY, "Content-Type": "application/json"}

def stream(messages, max_tokens, temperature=0):
    body = {"model": MODEL, "messages": messages, "max_tokens": max_tokens, "temperature": temperature, "seed": 1,
            "stream": True, "stream_options": {"include_usage": True}}
    t0 = time.time(); first = None; usage = {}; n_chunks = 0; timings = {}
    req = urllib.request.Request(URL, data=json.dumps(body).encode(), headers=H)
    with urllib.request.urlopen(req, timeout=1800) as r:
        for line in r:
            line = line.decode().strip()
            if not line.startswith("data:") or line == "data: [DONE]":
                continue
            d = json.loads(line[5:])
            ch = (d.get("choices") or [{}])[0].get("delta") or {}
            if (ch.get("content") or ch.get("reasoning_content")) and first is None:
                first = time.time()
            if ch.get("content") or ch.get("reasoning_content"):
                n_chunks += 1
            usage = d.get("usage") or usage; timings = d.get("timings") or timings
    end = time.time()
    return {"ttft": (first or end) - t0, "gen_s": end - (first or end), "usage": usage, "timings": timings, "chunks": n_chunks}

res = {"model": MODEL, "base_url": os.environ["BASE_URL"], "short": [], "long": []}
p = "Write a Python function that parses an Apache access log line into a dict, with a docstring and three test cases."
for i in range(3):
    r = stream([{"role": "user", "content": p}], 400)
    out = r["usage"].get("completion_tokens") or r["chunks"]
    tps = (out - 1) / r["gen_s"] if r["gen_s"] else 0   # measured here, the same way for every engine
    res["short"].append({"writing_tps": round(tps, 1), "tokens": out, "ttft_s": round(r["ttft"], 2),
                         "server_reported_tps": r["timings"].get("predicted_per_second")})
    print(f"short {i+1}: writing {tps:.1f} tok/s ({out} tokens)", flush=True)

lib = Path(sysconfig.get_paths()["stdlib"])
files = ["argparse.py", "inspect.py", "typing.py", "dataclasses.py", "pathlib/__init__.py", "subprocess.py",
         "tarfile.py", "zipfile/__init__.py", "logging/__init__.py", "json/decoder.py"]
text = ""
for f in files:
    if (lib / f).exists():
        text += f"\n# ==== {f}\n" + (lib / f).read_text(errors="replace")
text = text[:148_000]  # about 32K tokens of Python (4.6 characters per token, measured)
for i in range(3):
    msg = f"Run id {uuid.uuid4()}.\n{text}\n\nIn one sentence: what does the last function above do?"
    r = stream([{"role": "user", "content": msg}], 32)
    n = r["usage"].get("prompt_tokens") or 0
    tps = n / r["ttft"] if r["ttft"] else 0   # measured here (includes the network, ~ms), same for every engine
    res["long"].append({"prompt_tokens": n, "reading_tps": round(tps, 1), "ttft_s": round(r["ttft"], 1),
                        "server_reported_tps": r["timings"].get("prompt_per_second")})
    print(f"long {i+1}: {n} prompt tokens, reading {tps:.1f} tok/s, first token after {r['ttft']:.1f} s", flush=True)
res["speed_write_tps"] = statistics.median(x["writing_tps"] for x in res["short"])
res["speed_read_32k_tps"] = statistics.median(x["reading_tps"] for x in res["long"])
res["speed_ttft_32k_s"] = statistics.median(x["ttft_s"] for x in res["long"])
res["prompt_files"] = files; res["python_stdlib"] = str(lib)
json.dump(res, open(os.environ["OUT"], "w"), indent=1)
print(f"median: writing {res['speed_write_tps']} tok/s; 32K reading {res['speed_read_32k_tps']} tok/s "
      f"(first token after {res['speed_ttft_32k_s']} s)")
`

// Speed is `lab speed <model> <note>` (port of tuning/speed): writing and 32K reading speed, medians of 3, saved to
// tuning/runs/<date>-speed-<model>.json and logged (NOLOG=1 skips logging). Settings from the environment like the
// script: BASE_URL, API_KEY (else ZAPHOD_KEY from llama-swap's .env), MACHINE (who served it; default zaphod).
func Speed(cfg *config.Config, model, note string) error {
	if err := enterTuning(cfg, "speed", model, note); err != nil {
		return err
	}
	key := os.Getenv("API_KEY")
	if key == "" {
		k, err := zaphodKey(cfg)
		if err != nil {
			return err
		}
		key = k
	}
	out := "runs/" + time.Now().Format("2006-01-02-1504") + "-speed-" + strings.ReplaceAll(model, "/", "-") + ".json"
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8080/v1"
	}
	c := exec.Command("python3", "-I", "-")
	c.Stdin = strings.NewReader(speedPy)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	c.Env = append(os.Environ(), "BASE_URL="+base, "API_KEY="+key, "MODEL="+model, "OUT="+out)
	if err := c.Run(); err != nil {
		return exitErr(exitCode(err))
	}
	fmt.Println("saved " + out)
	if os.Getenv("NOLOG") == "1" {
		return nil
	}
	var res map[string]any
	b, err := os.ReadFile(out)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	var rest []string
	for _, k := range []string{"speed_write_tps", "speed_read_32k_tps", "speed_ttft_32k_s"} {
		rest = append(rest, k+"="+pyNum(res[k]))
	}
	rc, err := Log(cfg, machineEnv(), model, note, append(rest, out))
	if err != nil {
		return err
	}
	if rc != 0 {
		return exitErr(rc)
	}
	return nil
}

// pyNum prints a JSON number the way Python's print() does (54.0 stays "54.0"), as the script passed it to log.
func pyNum(v any) string {
	f, ok := v.(float64)
	if !ok {
		return fmt.Sprint(v)
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// machineEnv: MACHINE (who served the model), default zaphod.
func machineEnv() string {
	if m := os.Getenv("MACHINE"); m != "" {
		return m
	}
	return "zaphod"
}

// zaphodKey reads ZAPHOD_KEY from zaphod's llama-swap .env (never printed).
func zaphodKey(cfg *config.Config) (string, error) {
	p := config.Expand(cfg.Machines["zaphod"].LlamaSwapEnv)
	if p == "" {
		p = config.Expand("~/llama-swap/.env")
	}
	k, err := hub.EnvValue(p, "ZAPHOD_KEY")
	if err != nil || k == "" {
		return "", fmt.Errorf("no ZAPHOD_KEY in %s (set API_KEY for another server)", p)
	}
	return k, nil
}

// enterTuning moves into tuning/ (outputs and file arguments are relative to it, like the scripts' cd) and records
// how this run was started for the Rerun button (hub/actions/actiond.py): the script's name and arguments, NUL
// separated, base64. The outermost command wins (already set: kept). actiond accepts only the script names
// (speed, quality, bench, kl, tbench) run from tuning/, which re-run through the wrappers.
func enterTuning(cfg *config.Config, name string, args ...string) error {
	dir := cfg.Tuning()
	if err := os.Chdir(dir); err != nil {
		return err
	}
	if os.Getenv("LAB_RERUN") == "" {
		var b strings.Builder
		for _, a := range append([]string{name}, args...) {
			b.WriteString(a + "\x00")
		}
		os.Setenv("LAB_RERUN", base64.StdEncoding.EncodeToString([]byte(b.String())))
	}
	if os.Getenv("LAB_RERUN_CWD") == "" {
		os.Setenv("LAB_RERUN_CWD", dir)
	}
	return nil
}
