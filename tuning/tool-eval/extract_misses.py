#!/usr/bin/env python3
"""List every scenario that did not fully pass in a tool-eval-bench result, with what was
asked, which tools were called (with arguments) and the final answer, for manual sorting into:
  fabrication  - claimed something false or invented a result
  wrong        - wrong or malformed action (bad tool, bad arguments, skipped a required step)
  honest-stop  - stopped, asked, or stated a limitation instead of guessing
  server       - the serving stack failed before the model ran (not the model's fault)

Usage: extract_misses.py <result.json> [...]
"""
import json
import re
import sys

for path in sys.argv[1:]:
    d = json.load(open(path))
    print(f"\n######## {path}  score {d['scores']['final_score']}")
    for r in d["scores"]["scenario_results"]:
        if r["status"] == "pass":
            continue
        log = r.get("raw_log") or ""
        prompt = re.search(r"^prompt=(.*)$", log, re.M)
        calls = re.findall(r"^tool_call=(.*)$", log, re.M)
        final = re.search(r"^final_answer=(.*?)(?=^verdict=|\Z)", log, re.M | re.S)
        print(f"\n== {r['scenario_id']} [{r['status']}] {r['summary']}")
        print("asked:", (prompt.group(1) if prompt else "?")[:300])
        print("calls:", "; ".join(c[:120] for c in calls) or "none")
        print("final:", ((final.group(1) if final else "") or "(none)").strip().replace("\n", " ")[:400])
