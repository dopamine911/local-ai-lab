#!/usr/bin/env python3
# Reading and writing speed at growing context depth, any OpenAI-compatible server (plans/halogen-timeline.md step 6).
#   BASE_URL=http://marvin:8080/v1 API_KEY=... python3 -I tuning/depth.py big [depths in K, default 32 64 128]
# At each depth: one request of about that many tokens of Python source (the standard library, like `lab speed`),
# starting with a unique run id so no engine can answer from a cache, then 200 tokens written at temperature 0.
# Reading = prompt tokens / time to first token. Writing = tokens after the first / the time they took, measured here
# the same way for every engine. One request per depth (128K on llama.cpp takes about 10 minutes).
# Writes tuning/runs/<date>-depth-<model>.json. Good result: one line per depth, then "saved".
import json, os, sys, sysconfig, time, urllib.request, uuid
from pathlib import Path

URL = os.environ.get("BASE_URL", "http://127.0.0.1:8080/v1") + "/chat/completions"
KEY = os.environ["API_KEY"]
MODEL = sys.argv[1] if len(sys.argv) > 1 else "big"
DEPTHS = [int(x) for x in sys.argv[2:]] or [32, 64, 128]
CHARS_PER_TOKEN = 3.6  # measured 2026-11-17 on this text with big's tokenizer (lab speed's smaller file set: 4.6)

lib = Path(sysconfig.get_paths()["stdlib"])
text = ""
for f in sorted(lib.rglob("*.py")):
    if "test" in f.parts or "idlelib" in f.parts:
        continue
    text += f"\n# ==== {f.relative_to(lib)}\n" + f.read_text(errors="replace")
    if len(text) > max(DEPTHS) * 1024 * CHARS_PER_TOKEN * 1.1:
        break


def ask(content, max_tokens=200):
    body = {"model": MODEL, "messages": [{"role": "user", "content": content}], "max_tokens": max_tokens,
            "temperature": 0, "seed": 1, "stream": True, "stream_options": {"include_usage": True}}
    req = urllib.request.Request(URL, json.dumps(body).encode(),
                                 {"Authorization": "Bearer " + KEY, "Content-Type": "application/json"})
    t0 = time.time(); first = None; usage = {}; chunks = 0
    with urllib.request.urlopen(req, timeout=3600) as r:
        for line in r:
            line = line.decode().strip()
            if not line.startswith("data:") or line == "data: [DONE]":
                continue
            d = json.loads(line[5:])
            ch = (d.get("choices") or [{}])[0].get("delta") or {}
            if ch.get("content") or ch.get("reasoning_content"):
                chunks += 1
                if first is None:
                    first = time.time()
            usage = d.get("usage") or usage
    end = time.time()
    return (first or end) - t0, end - (first or end), usage, chunks


rows = []
for k in DEPTHS:
    chars = int(k * 1024 * CHARS_PER_TOKEN)
    msg = f"Run id {uuid.uuid4()}.\n{text[:chars]}\n\nIn one sentence: what does the last function above do?"
    ttft, gen, usage, chunks = ask(msg)
    n_in = usage.get("prompt_tokens", 0); n_out = usage.get("completion_tokens") or chunks
    read = n_in / ttft if ttft else 0; write = (n_out - 1) / gen if gen else 0
    rows.append({"depth_k": k, "prompt_tokens": n_in, "ttft_s": round(ttft, 1), "reading_tps": round(read, 1),
                 "tokens_out": n_out, "writing_tps": round(write, 1)})
    print(f"{k}K: {n_in} prompt tokens, reading {read:.1f} tok/s (first token {ttft:.1f} s), "
          f"writing {write:.1f} tok/s ({n_out} tokens)", flush=True)

out = Path(__file__).parent / "runs" / f"{time.strftime('%Y-%m-%d-%H%M')}-depth-{MODEL.replace('/', '-')}.json"
out.write_text(json.dumps({"model": MODEL, "base_url": URL, "rows": rows}, indent=1))
print(f"saved {out.relative_to(Path(__file__).parent)}")
