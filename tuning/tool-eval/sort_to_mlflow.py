#!/usr/bin/env python3
"""Add the manual miss sort (classifications.json) to the matching MLflow runs.

For each run: metrics sort_fabrication, sort_wrong, sort_honest_stop, sort_server,
sort_grader_miss, plus misses-sorted.md listing every scenario, its group and the reason,
plus scenarios.json: a table of ALL scenarios (status, points, hand-sort group, reason, seconds), so one scenario
can be compared across runs in MLflow (Compare > Artifacts > scenarios.json, or "Evaluation" tables).
Run with ./sort-log (inside the MLflow container). Safe to repeat: values are overwritten.
"""
import json

from mlflow.tracking import MlflowClient

BUCKETS = ["fabrication", "wrong", "honest-stop", "server", "grader-miss"]
client = MlflowClient("http://127.0.0.1:5000")
data = json.load(open("/tuning/tool-eval/classifications.json"))

for name, run in data.items():
    if name.startswith("_"):
        continue
    counts = {b: 0 for b in BUCKETS}
    lines = [f"# Misses sorted by hand: {name} (skill: {run['skill']})", "",
             "| Scenario | Group | What happened |", "| --- | --- | --- |"]
    for sid, (bucket, why) in sorted(run["misses"].items()):
        counts[bucket] += 1
        lines.append(f"| {sid} | {bucket} | {why} |")
    for b, n in counts.items():
        client.log_metric(run["mlflow_run"], "sort_" + b.replace("-", "_"), n)
    client.log_text(run["mlflow_run"], "\n".join(lines) + "\n", "misses-sorted.md")
    client.set_tag(run["mlflow_run"], "sorted_by_hand", "yes")
    try:  # every scenario of this run, with the hand-sort added (result file: tool-eval/runs/<name>.json)
        res = json.load(open(f"/tuning/tool-eval/runs/{name}.json"))["scores"]["scenario_results"]
        pts = {"pass": 2, "partial": 1, "fail": 0}
        table = {"scenario": [], "status": [], "points": [], "group": [], "reason": [], "seconds": []}
        for r in res:
            bucket, why = run["misses"].get(r["scenario_id"], ("pass" if r["status"] == "pass" else "", ""))
            table["scenario"].append(r["scenario_id"]); table["status"].append(r["status"])
            table["points"].append(pts.get(r["status"], 0)); table["group"].append(bucket)
            table["reason"].append(why or r.get("summary", "")[:160]); table["seconds"].append(round(r.get("duration_seconds") or 0, 1))
        client.log_table(run["mlflow_run"], table, "scenarios.json")
    except (OSError, KeyError) as e:
        print(f"  (no scenario table for {name}: {e})")
    print(f"{name:<24} {counts}")
