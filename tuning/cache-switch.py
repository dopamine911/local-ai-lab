#!/usr/bin/env python3
# Switch test and leak check for llama.cpp's RAM prompt cache (issue #10, llama.cpp#27148).
#   (set -a; . ~/llama-swap/.env; set +a; python3 -I tuning/cache-switch.py coder [rounds])
#   BASE_URL=... API_KEY=... python3 -I tuning/cache-switch.py marvin/big
# Each round: conversation A (a long prompt with its own codeword), then B (another long prompt, another codeword),
# then A again (A's prompt + its answer + a follow-up question). Measures the time to first token of each, and checks
# every answer names its own codeword: an answer with the other conversation's codeword is a LEAK.
# The second A shows the cost of the fix: with the RAM cache, A's prompt comes back from RAM after B took the slot;
# with --cache-ram 0 --no-cache-idle-slots it is read again in full.
# Writes tuning/runs/<date>-cache-switch-<model>.json. Good result after the fix: 0 leaks.
import json, os, statistics, sys, sysconfig, time, urllib.request, uuid
from pathlib import Path

URL = os.environ.get("BASE_URL", "http://127.0.0.1:8080/v1") + "/chat/completions"
KEY = os.environ.get("API_KEY") or os.environ["ZAPHOD_KEY"]
MODEL = sys.argv[1] if len(sys.argv) > 1 else "coder"
ROUNDS = int(sys.argv[2]) if len(sys.argv) > 2 else 3

lib = Path(sysconfig.get_paths()["stdlib"])
def text(files, chars):
    s = ""
    for f in files:
        if (lib / f).exists():
            s += f"\n# ==== {f}\n" + (lib / f).read_text(errors="replace")
    return s[:chars]
# About 12K tokens each (upstream saw the leak at ~12K-token prompts); different files, so nothing is shared.
TEXT_A = text(["argparse.py", "inspect.py"], 55_000)
TEXT_B = text(["typing.py", "dataclasses.py"], 55_000)

def ask(messages):
    body = {"model": MODEL, "messages": messages, "max_tokens": 600, "temperature": 0, "seed": 1, "stream": True,
            "stream_options": {"include_usage": True}}
    req = urllib.request.Request(URL, json.dumps(body).encode(),
                                 {"Authorization": "Bearer " + KEY, "Content-Type": "application/json"})
    t0 = time.time(); first = None; content = ""; usage = {}; timings = {}
    with urllib.request.urlopen(req, timeout=1800) as r:
        for line in r:
            line = line.decode().strip()
            if not line.startswith("data:") or line == "data: [DONE]":
                continue
            d = json.loads(line[5:])
            delta = (d.get("choices") or [{}])[0].get("delta") or {}
            if (delta.get("content") or delta.get("reasoning_content")) and first is None:
                first = time.time()
            content += delta.get("content") or ""
            usage = d.get("usage") or usage; timings = d.get("timings") or timings
    return {"ttft_s": round((first or time.time()) - t0, 2), "answer": content.strip(),
            "prompt_tokens": usage.get("prompt_tokens"),
            "cached_tokens": (usage.get("prompt_tokens_details") or {}).get("cached_tokens"),
            "prompt_n": timings.get("prompt_n"), "cache_n": timings.get("cache_n")}

def opening(code, body):
    return (f"Codeword for this conversation: {code}\n\nHere is some Python source code:\n{body}\n\n"
            f"Question: what is the codeword at the top of this message? Answer with the codeword only.")

rounds, leaks = [], 0
for i in range(ROUNDS):
    a, b = "ALPHA-" + uuid.uuid4().hex[:8].upper(), "BRAVO-" + uuid.uuid4().hex[:8].upper()
    conv_a = [{"role": "user", "content": opening(a, TEXT_A)}]
    r_a1 = ask(conv_a)
    r_b1 = ask([{"role": "user", "content": opening(b, TEXT_B)}])
    conv_a += [{"role": "assistant", "content": r_a1["answer"]},
               {"role": "user", "content": "Repeat the codeword of this conversation once more, nothing else."}]
    r_a2 = ask(conv_a)
    row = {"round": i + 1, "codeword_a": a, "codeword_b": b, "a1": r_a1, "b1": r_b1, "a2": r_a2}
    for name, r, own, other in (("A", r_a1, a, b), ("B", r_b1, b, a), ("A again", r_a2, a, b)):
        r["own_codeword"] = own in r["answer"]
        r["leak"] = other in r["answer"]
        leaks += r["leak"]
    rounds.append(row)
    print(f"round {i+1}: first token A {r_a1['ttft_s']} s, B {r_b1['ttft_s']} s, A again {r_a2['ttft_s']} s "
          f"(prompt tokens read: A {r_a1['prompt_n']}, B {r_b1['prompt_n']}, A again {r_a2['prompt_n']}); "
          f"own codeword: {r_a1['own_codeword']}/{r_b1['own_codeword']}/{r_a2['own_codeword']}; "
          f"leaks: {sum(x['leak'] for x in (r_a1, r_b1, r_a2))}", flush=True)

res = {"model": MODEL, "base_url": URL, "rounds": rounds, "leaks": leaks,
       "median_ttft_a_again_s": statistics.median(r["a2"]["ttft_s"] for r in rounds),
       "median_ttft_a_s": statistics.median(r["a1"]["ttft_s"] for r in rounds),
       "median_prompt_read_a_again": statistics.median((r["a2"]["prompt_n"] or 0) for r in rounds)}
out = Path(__file__).parent / "runs" / f"{time.strftime('%Y-%m-%d-%H%M')}-cache-switch-{MODEL.replace('/', '-')}.json"
out.write_text(json.dumps(res, indent=1))
print(f"median first token: A {res['median_ttft_a_s']} s, A again {res['median_ttft_a_again_s']} s "
      f"(read {res['median_prompt_read_a_again']} prompt tokens); leaks: {leaks}; saved {out.name}")
