#!/usr/bin/env python3
"""Side-by-side table of two tool-eval-bench result files (without vs with a skill)."""
import json
import sys

model, base_path, with_path, name = sys.argv[1:5]
a, b = json.load(open(base_path)), json.load(open(with_path))
sa, sb = a["scores"], b["scores"]


def row(label, x, y, unit=""):
    d = y - x
    sign = "+" if d > 0 else ""
    return f"| {label} | {x}{unit} | {y}{unit} | {sign}{round(d, 1)}{unit} |"


print(f"\n## {model}\n")
print(f"| | Without | With {name} | Change |")
print("| --- | ---: | ---: | ---: |")
print(row("Score (of 100)", sa["final_score"], sb["final_score"]))
print(row("Points", sa["total_points"], sb["total_points"]))
print(row("Median turn time (ms)", round(sa.get("median_turn_ms") or 0), round(sb.get("median_turn_ms") or 0)))
print(row("Total tokens", sa.get("total_tokens") or 0, sb.get("total_tokens") or 0))
if a["config"].get("scenario_count") != b["config"].get("scenario_count"):
    print(f"\nWarning: different suites ({a['config'].get('scenario_count')} vs "
          f"{b['config'].get('scenario_count')} scenarios); scores are not comparable.\n")
ca = {c["category"]: c for c in sa["category_scores"]}
for c in sb["category_scores"]:
    if c["category"] not in ca:
        continue
    x, y = ca[c["category"]]["percent"], c["percent"]
    if x != y:
        print(row(f"{c['category']} {c['label']}", x, y, "%"))

ra = {r["scenario_id"]: r for r in sa["scenario_results"]}
changed = [(r["scenario_id"], ra[r["scenario_id"]]["status"], r["status"], r["summary"])
           for r in sb["scenario_results"]
           if r["scenario_id"] in ra and ra[r["scenario_id"]]["status"] != r["status"]]
if changed:
    print("\nScenarios that changed:\n")
    for sid, x, y, summary in changed:
        print(f"- {sid}: {x} → {y}. {summary[:120]}")
else:
    print("\nNo scenario changed result.")
