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
  - series tag (SERIES=name ./quality ...): repeats of one setup, grouped in MLflow
  - behavior_* (tool-use): automatic checks for problems the grader misses (tool-eval/behavior.py)
  - benchmark tag: speed, tool-use, terminal-bench, kl-divergence, history or manual
  - env.* tags: machine, kernel, Mesa, GPUs, image, llama.cpp commit (from manifest.sh, via ./log)
  - model.* tags: the exact model file, Hugging Face repo, revision and sha256 (from setup/models-<machine>.txt)
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


def benchmark_kind(model, run_id):
    """Which kind of test a run is, from what was logged. Filter in MLflow: tags.benchmark = "tool-use"."""
    if model.startswith("history"):
        return "history"
    r = mlflow.get_run(run_id)
    tool, metrics = r.data.params.get("quality.tool", ""), r.data.metrics
    if tool == "tool-eval-bench":
        return "tool-use"
    if tool == "terminal-bench-local":
        return "terminal-bench"
    if tool == "kl-divergence":
        return "kl-divergence"
    if any(k.endswith("_tps") for k in metrics):
        return "speed"
    return "manual"


def manifest_tags():
    """env.<key> tags from the MANIFEST lines that ./log collected with manifest.sh."""
    tags = {}
    for line in os.environ.get("MANIFEST", "").splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            tags[f"env.{k.strip()}"] = v.strip()[:500]
    return tags


def model_tags(machine, cmd):
    """model.* tags: which exact file was served, looked up in the pinned list setup/models-<machine>.txt."""
    tags = {}
    args = shlex.split(cmd)
    files = {"model": None, "draft": None}
    for i, a in enumerate(args[:-1]):
        if a in ("-m", "--model"):
            files["model"] = args[i + 1]
        elif a in ("-md", "--model-draft"):
            files["draft"] = args[i + 1]
    pinned = {}
    try:
        with open(f"/setup/models-{machine}.txt") as f:
            for line in f:
                p = line.split()
                if len(p) == 6 and not line.startswith("#"):
                    pinned[p[3]] = {"repo": p[0], "rev": p[1][:10], "sha256": p[5][:16]}
    except OSError:
        pass
    for kind, path in files.items():
        if not path:
            continue
        rel = path.removeprefix("/models/")
        prefix = "model" if kind == "model" else "model.draft"
        tags[f"{prefix}.file"] = rel
        info = pinned.get(rel)
        if info:
            tags.update({f"{prefix}.repo": info["repo"], f"{prefix}.rev": info["rev"], f"{prefix}.sha256": info["sha256"]})
        else:
            tags[f"{prefix}.pinned"] = "no (not in setup/models-%s.txt)" % machine
    return tags


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


def read_terminal_bench(d):
    """Results from a Terminal-Bench-Local summary.json."""
    suite = d.get("suite", {})
    params = {
        "quality.tool": "terminal-bench-local",
        "tbench.suite": f"{suite.get('id', '')} {suite.get('version', '')}",
        "tbench.profile": d.get("inference_profile") or "",
        "tbench.quant": d.get("quant") or "",
    }
    tok = d.get("tokens", {})
    metrics = {
        "tbench_passed": d.get("passed_tasks", 0),
        "tbench_total": d.get("total_tasks", 0),
        "tbench_pass_rate_pct": round(100 * (d.get("pass_rate") or 0), 1),
        "tbench_hours": round((d.get("total_duration_ms") or 0) / 3.6e6, 2),
        "tbench_output_tokens": tok.get("output", 0),
    }
    return params, metrics


def read_tool_eval(path):
    """Settings and results from a tool-eval-bench --json-file result,
    or a Terminal-Bench-Local summary.json."""
    try:
        with open(path) as f:
            d = json.load(f)
    except (ValueError, OSError):
        return None, None
    if "passed_tasks" in d and "suite" in d:
        return read_terminal_bench(d)
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
    common = common_score(s)
    if common is not None:
        metrics["tooleval_score_common"] = common
    try:  # automatic behavior checks (tool-eval/behavior.py): problems the grader misses
        sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "tool-eval"))
        import behavior
        metrics.update(behavior.metrics(d))
    except Exception as e:
        print(f"note: behavior checks skipped ({e})")
    return params, metrics


# The 4 JSON-format scenarios. tool-eval leaves them out when the server rejects them (llama.cpp's grammar bug) but
# scores them when it doesn't (Strata), so raw scores aren't comparable. The common score always leaves them out.
COMMON_EXCLUDE = {"TC-65", "TC-66", "TC-67", "TC-69"}


def common_score(scores):
    """Score out of 100 without the JSON scenarios: pass = 2 points, partial = 1, fail = 0 (tool-eval's scale)."""
    pts = {"pass": 2, "partial": 1, "fail": 0}
    keep = [r for r in scores.get("scenario_results", []) if r.get("scenario_id") not in COMMON_EXCLUDE]
    if not keep:
        return None
    return int(100 * sum(pts.get(r["status"], 0) for r in keep) / (2 * len(keep)) + 0.5)


KLD_LINES = {  # llama-perplexity --kl-divergence output label -> MLflow metric name
    r"Mean PPL\(Q\)\s*:": "kld_ppl_q",
    r"Mean PPL\(base\)\s*:": "kld_ppl_base",
    r"Mean PPL\(Q\)/PPL\(base\)\s*:": "kld_ppl_ratio",
    r"Mean\s+KLD:": "kld_mean",
    r"Maximum KLD:": "kld_max",
    r"99\.0%\s+KLD:": "kld_p99",
    r"Median\s+KLD:": "kld_median",
    r"RMS Δp\s*:": "kld_rms_dp_pct",
    r"Same top p:": "kld_same_top_p_pct",
}


def read_kld(path):
    """Results from llama-perplexity --kl-divergence output (saved by ./kl)."""
    import re
    with open(path, errors="replace") as f:
        text = f.read()
    if "KLD" not in text:
        return None, None
    metrics = {}
    for label, name in KLD_LINES.items():
        m = re.search(label + r"\s*(-?[0-9.]+)", text)
        if m:
            metrics[name] = float(m.group(1))
    first = text.splitlines()[0] if text else ""
    params = {"quality.tool": "kl-divergence", "kld.run": first.lstrip("# ")[:250]}
    return params, metrics


def main():
    if len(sys.argv) < 4:
        sys.exit(__doc__)
    machine, model, note, *rest = sys.argv[1:]
    # A peer model called through zaphod ("marvin/big") belongs to that machine: log it there.
    if "/" in model and model.split("/", 1)[0] in LLAMA_SWAP:
        machine, model = model.split("/", 1)
    if machine not in LLAMA_SWAP:
        sys.exit(f"Unknown machine {machine}. Known: {', '.join(LLAMA_SWAP)}")
    # kld:/history:/card labels are not llama-swap models, so there are no server settings to read
    cmd = None if model.split(":")[0] in ("kld", "history") or model.startswith("card") else server_cmd(machine, model)
    mlflow.set_tracking_uri(MLFLOW_URL)
    mlflow.set_experiment(EXPERIMENT)
    mlflow.set_experiment_tag("mlflow.experimentKind", "custom_model_development")  # open on the runs table
    with mlflow.start_run(run_name=f"{machine}/{model}", description=note) as run:
        mlflow.set_tags({"machine": machine, "model": model, "mlflow.note.content": note,
                         "config_commit": os.environ.get("CONFIG_COMMIT", "unknown")})
        if os.environ.get("SERIES"):  # repeats of one setup (seeds, temperatures): group by this tag in MLflow
            mlflow.set_tag("series", os.environ["SERIES"])
        if os.environ.get("SKILL_NAME"):
            mlflow.log_params({"skill": os.environ["SKILL_NAME"], "skill.sha": os.environ.get("SKILL_SHA", "")})
            mlflow.set_tag("skill", os.environ["SKILL_NAME"])
        mlflow.set_tags(manifest_tags())
        if cmd:
            mlflow.set_tags(model_tags(machine, cmd))
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
                        if "tbench_passed" in m:
                            print(f"  Terminal-Bench: {m['tbench_passed']}/{m['tbench_total']} tasks "
                                  f"({m['tbench_pass_rate_pct']}%) in {m['tbench_hours']} h")
                        else:
                            print(f"  tool-eval score = {m['tooleval_score']} "
                                  f"({m['tooleval_points']}/{m['tooleval_max_points']} points)")
                if item.endswith(".kld.txt"):
                    p, m = read_kld(item)
                    if p:
                        mlflow.log_params(p)
                        mlflow.log_metrics(m)
                        print(f"  mean KLD = {m.get('kld_mean')}   same top token = "
                              f"{m.get('kld_same_top_p_pct')}%   PPL ratio = {m.get('kld_ppl_ratio')}")
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
        mlflow.set_tag("benchmark", benchmark_kind(model, run.info.run_id))
        print(f"Logged run {run.info.run_id}  ->  https://mlflow.ai.dopamine911.com"
              f"  (config: {os.environ.get('CONFIG_COMMIT', 'unknown')})")


if __name__ == "__main__":
    main()
