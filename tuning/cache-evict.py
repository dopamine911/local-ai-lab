#!/usr/bin/env python3
# Leak check when a prompt cache must evict (plans/halogen-timeline.md): cache-switch.py stays inside a few cache
# entries; this one forces the cache to drop and reuse them, where a stale resume would show.
#   BASE_URL=http://marvin:8080/v1 API_KEY=... python3 -I tuning/cache-evict.py big-halogen [many|deep|both]
# many: 8 conversations of ~12K tokens, each with its own codeword, one after another; then a follow-up in each.
# deep: 5 conversations of ~120K tokens (together more than a 524,288-position pool); then a follow-up in the first.
# Every answer must name its own codeword and no other: one with another conversation's codeword is a LEAK.
# The follow-up's first-token time shows whether it resumed (fast) or read the conversation again (slow).
# Writes tuning/runs/<date>-cache-evict-<model>.json. Good result: leaks 0.
import json, os, sys, sysconfig, time, urllib.request, uuid
from pathlib import Path

URL = os.environ.get("BASE_URL", "http://127.0.0.1:8080/v1") + "/chat/completions"
KEY = os.environ["API_KEY"]
MODEL = sys.argv[1] if len(sys.argv) > 1 else "big-halogen"
MODE = sys.argv[2] if len(sys.argv) > 2 else "both"
CPT = 3.6  # characters per token on this text (tuning/depth.py)

lib = Path(sysconfig.get_paths()["stdlib"])
files = [f for f in sorted(lib.rglob("*.py")) if "test" not in f.parts and "idlelib" not in f.parts]
corpus = "".join(f"\n# ==== {f.relative_to(lib)}\n" + f.read_text(errors="replace") for f in files)


def piece(i, tokens):
    """A different stretch of the corpus for conversation i, so nothing is shared between conversations."""
    n = int(tokens * CPT)
    start = (i * n) % max(1, len(corpus) - n)
    return corpus[start:start + n]


def ask(messages):
    body = {"model": MODEL, "messages": messages, "max_tokens": 300, "temperature": 0, "stream": True,
            "reasoning_effort": "none"}
    req = urllib.request.Request(URL, json.dumps(body).encode(),
                                 {"Authorization": "Bearer " + KEY, "Content-Type": "application/json"})
    t0 = time.time(); first = None; text = ""
    with urllib.request.urlopen(req, timeout=3600) as r:
        for line in r:
            line = line.decode().strip()
            if not line.startswith("data:") or line == "data: [DONE]":
                continue
            ch = (json.loads(line[5:]).get("choices") or [{}])[0].get("delta") or {}
            if ch.get("content"):
                first = first or time.time()
                text += ch["content"]
    return text, (first or time.time()) - t0


def run(n, tokens, revisit):
    codes = [uuid.uuid4().hex[:8].upper() for _ in range(n)]
    convs, rows, leaks = [], [], 0
    for i in range(n):
        m = [{"role": "user", "content": f"Run id {uuid.uuid4()}. The codeword for this conversation is {codes[i]}.\n"
                                         f"{piece(i, tokens)}\n\nReply with only the codeword for this conversation."}]
        a, t = ask(m)
        convs.append(m + [{"role": "assistant", "content": a}])
        rows.append(check("first", i, a, t, codes))
    for i in revisit:
        a, t = ask(convs[i] + [{"role": "user", "content": "Repeat the codeword for this conversation, nothing else."}])
        rows.append(check("follow-up", i, a, t, codes))
    return rows


def check(kind, i, answer, ttft, codes):
    own = codes[i] in answer
    leak = any(c in answer for j, c in enumerate(codes) if j != i)
    print(f"  {kind:9} conversation {i + 1}: first token {ttft:6.1f} s, own codeword {own}, leak {leak}", flush=True)
    return {"kind": kind, "conversation": i + 1, "ttft_s": round(ttft, 2), "own": own, "leak": leak,
            "answer": answer[:80]}


res = {"model": MODEL}
if MODE in ("many", "both"):
    print("many: 8 conversations of ~12K tokens, then a follow-up in each")
    res["many"] = run(8, 12_000, range(8))
if MODE in ("deep", "both"):
    print("deep: 5 conversations of ~120K tokens, then a follow-up in the first")
    res["deep"] = run(5, 120_000, [0])
allrows = res.get("many", []) + res.get("deep", [])
res["leaks"] = sum(r["leak"] for r in allrows)
res["own"] = f"{sum(r['own'] for r in allrows)}/{len(allrows)}"
out = Path(__file__).parent / "runs" / f"{time.strftime('%Y-%m-%d-%H%M')}-cache-evict-{MODEL.replace('/', '-')}.json"
out.write_text(json.dumps(res, indent=1))
print(f"own codeword {res['own']}, leaks {res['leaks']}; saved {out.relative_to(Path(__file__).parent)}")
