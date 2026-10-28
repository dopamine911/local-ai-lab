#!/usr/bin/env python3
"""Log one test result to MLflow, with the exact settings.

Usage (use the ./log wrapper, which runs this inside the MLflow container):
  ./log <machine> <model> "<note>" [name=value ...] [result files ...]

Examples:
  ./log zaphod coder "MTP depth 3 vs 2" tg_tps=61.2
  ./log zaphod coder "baseline" runs/2026-10-28-1400.jsonl     (llama-bench output)

What it records:
  - settings: every flag of the command llama-swap is running for <model>
    (skipped if the model isn't loaded, e.g. a llama-bench run with llama-swap stopped)
  - settings + results from any llama-bench .jsonl file you attach
  - each name=value you give as a result, your note, and the files themselves
"""
import json
import os
import shlex
import sys
import urllib.request

import mlflow

LLAMA_SWAP = {
    "zaphod": ("http://127.0.0.1:8080", "ZAPHOD_KEY"),
    "marvin": ("http://marvin:8080", "MARVIN_KEY"),
}
MLFLOW_URL = "http://127.0.0.1:5000"
EXPERIMENT = "llama-tuning"
SKIP = {"port", "host"}  # change every run; not real settings
BENCH_KEYS = ["build_commit", "model_filename", "n_batch", "n_ubatch", "type_k", "type_v",
              "n_gpu_layers", "flash_attn", "split_mode", "devices", "n_threads", "backends"]


def server_cmd(machine, model):
    url, keyvar = LLAMA_SWAP[machine]
    req = urllib.request.Request(f"{url}/running")
    if os.environ.get(keyvar):
        req.add_header("Authorization", f"Bearer {os.environ[keyvar]}")
    try:
        with urllib.request.urlopen(req, timeout=10) as r:
            running = json.load(r).get("running", [])
    except Exception as e:
        print(f"note: couldn't ask llama-swap on {machine} ({e}); no server settings recorded")
        return None
    for m in running:
        if m.get("model") == model:
            return m.get("cmd", "")
    return None


def is_flag(token):
    if not token.startswith("-"):
        return False
    try:
        float(token)  # a negative number such as -1 is a value, not a flag
        return False
    except ValueError:
        return True


def flags_to_params(cmd):
    args = shlex.split(cmd)
    params = {}
    for i, a in enumerate(args):
        if is_flag(a) and a.lstrip("-") not in SKIP:
            nxt = args[i + 1] if i + 1 < len(args) else None
            params[a.lstrip("-")] = nxt if nxt is not None and not is_flag(nxt) else "on"
    return params


def read_bench(path):
    """Settings and results from llama-bench -o jsonl."""
    rows = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line.startswith("{"):
                rows.append(json.loads(line))
    if not rows or "avg_ts" not in rows[0]:
        return None, None
    params = {f"bench.{k}": str(rows[0][k]) for k in BENCH_KEYS if k in rows[0]}
    metrics = {}
    for r in rows:
        depth = f"@d{r['n_depth']}" if r.get("n_depth") else ""
        name = f"pp{r['n_prompt']}" if r.get("n_prompt") else f"tg{r['n_gen']}"
        if r.get("n_ubatch") and len({x.get("n_ubatch") for x in rows}) > 1:
            name += f"_ub{r['n_ubatch']}"
        metrics[f"{name}{depth}_tps"] = round(r["avg_ts"], 2)
        metrics[f"{name}{depth}_stddev"] = round(r.get("stddev_ts", 0), 2)
    return params, metrics


def read_tool_eval(path):
    """Settings and results from a tool-eval-bench --json-file result."""
    try:
        with open(path) as f:
            d = json.load(f)
    except (ValueError, OSError):
        return None, None
    if "tool_eval_bench_version" not in d or "scores" not in d:
        return None, None
    s, m = d["scores"], d.get("metadata", {})
    params = {
        "quality.tool": "tool-eval-bench",
        "quality.version": d["tool_eval_bench_version"],
        "quality.suite": m.get("scenario_selector", ""),
        "quality.temperature": m.get("temperature", ""),
        "quality.seed": m.get("seed", ""),
        "quality.thinking": m.get("thinking_enabled", ""),
        "quality.trials": m.get("trials", ""),
    }
    metrics = {
        "tooleval_score": s["final_score"],
        "tooleval_points": s["total_points"],
        "tooleval_max_points": s["max_points"],
        "tooleval_deployability": s.get("deployability") or 0,
        "tooleval_responsiveness": s.get("responsiveness") or 0,
        "tooleval_median_turn_ms": s.get("median_turn_ms") or 0,
        "tooleval_total_tokens": s.get("total_tokens") or 0,
    }
    for c in s.get("category_scores", []):
        metrics[f"tooleval_cat_{c['category']}"] = c["percent"]
    return params, metrics


def main():
    if len(sys.argv) < 4:
        sys.exit(__doc__)
    machine, model, note, *rest = sys.argv[1:]
    if machine not in LLAMA_SWAP:
        sys.exit(f"Unknown machine {machine}. Known: {', '.join(LLAMA_SWAP)}")
    cmd = server_cmd(machine, model)
    mlflow.set_tracking_uri(MLFLOW_URL)
    mlflow.set_experiment(EXPERIMENT)
    with mlflow.start_run(run_name=f"{machine}/{model}", description=note) as run:
        mlflow.set_tags({"machine": machine, "model": model, "mlflow.note.content": note,
                         "config_commit": os.environ.get("CONFIG_COMMIT", "unknown")})
        if cmd:
            mlflow.log_params(flags_to_params(cmd))
            mlflow.log_text(cmd, "server_cmd.txt")
        for item in rest:
            if os.path.isfile(item):
                mlflow.log_artifact(item)
                if item.endswith(".json"):
                    p, m = read_tool_eval(item)
                    if p:
                        mlflow.log_params(p)
                        mlflow.log_metrics(m)
                        print(f"  tool-eval score = {m['tooleval_score']} "
                              f"({m['tooleval_points']}/{m['tooleval_max_points']} points)")
                if item.endswith(".jsonl"):
                    p, m = read_bench(item)
                    if p:
                        mlflow.log_params(p)
                        mlflow.log_metrics(m)
                        for k, v in m.items():
                            if not k.endswith("stddev"):
                                print(f"  {k} = {v}")
            elif "=" in item:
                name, value = item.split("=", 1)
                mlflow.log_metric(name, float(value))
            else:
                sys.exit(f"Not a file or name=value: {item}")
        print(f"Logged run {run.info.run_id}  ->  https://mlflow.ai.dopamine911.com"
              f"  (config: {os.environ.get('CONFIG_COMMIT', 'unknown')})")


if __name__ == "__main__":
    main()
