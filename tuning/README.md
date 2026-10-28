# Benchmarking and tuning

Change one setting at a time, test it the same way every time, and record the result before touching anything else. The records do the remembering: you look things up instead of recalling them.

| What | Where |
| --- | --- |
| Every test, with its exact settings | MLflow: https://mlflow.ai.dopamine911.com (experiment `llama-tuning`) |
| What changed, when and why | git history: `cd ~/llama-swap && git log --oneline` |
| Written summary and next ideas | `results.md` in this folder |
| What each setting does | "Settings reference" below |
| Full guide (all tools, the test ladder) | claude.ai doc "LLM Benchmarking and Tuning Guide" |

## The loop, every time

1. **Pick one setting.** Write down what you expect it to do.
2. **Measure before** (if there's no recent number for it in MLflow).
3. **Change only that setting** in `~/llama-swap/config/config.yaml`. llama-swap reloads by itself.
4. **Commit the change:**
   ```bash
   cd ~/llama-swap && git commit -am "coder: spec-draft-n-max 2 -> 3: faster writing?"
   ```
   Pattern: **what · from → to · why**.
5. **Run the test** that matches the setting (table below), and log it.
6. **Keep or revert.** Keep only if it beats the normal run-to-run wobble and nothing else got worse. To revert: `git revert HEAD`. Write "kept" or "reverted" in a final log note.

## The three commands

**`./bench`**: one card, llama-bench, 3 repeats, logged automatically. It stops llama-swap while it runs and starts it again after.

```bash
cd ~/homelab/tuning
./bench 1 "baseline card 1"
./bench 2 "ubatch sweep" -ub 512,1024,2048
MODEL=/models/flint-UD-Q4_K_XL.gguf ./bench 1 "flint first look"
```
Good: it prints lines like `pp512_tps = 1115.4` and `tg128_tps = 27.8`, then `Logged run ...`.
Note: llama-bench doesn't use speculative decoding (MTP). Test MTP with a server test.

**`./log`**: record any other result, e.g. a server test. It reads the model's current settings from llama-swap by itself.

```bash
./log zaphod coder "MTP depth 3 vs 2" server_tg_tps=61.2
./log <machine> <model> "<note>" [name=value ...] [files ...]
```

**`./quality`**: is the model still good at using tools? (tool-eval-bench 3.0.0, through llama-swap). Run it after any change that `./bench` says is faster, so faster never means worse.

```bash
./quality coder "baseline"                    # 15 core scenarios, about 1.5 minutes
./quality coder "MTP depth 3" --full          # all 69 scenarios, about 10 minutes
./quality coder "temp 0.6" --temperature 0.6  # test a sampling setting
```
Good: `tool-eval score = 97 (29/30 points)` or close to it, then `Logged run ...`.
It uses temperature 0 and seed 42, so the same settings give the same score every time; a change of even 1 point is real. Each run's full report (every conversation, what passed and failed) is in `tool-eval/runs/`, and attached to the run in MLflow.

**Server speed test** (uses MTP, the real setup), 3 runs, same answer each time:
```bash
source ~/llama-swap/.env
for i in 1 2 3; do curl -s http://localhost:8080/v1/chat/completions -H "Authorization: Bearer $ZAPHOD_KEY" -H 'Content-Type: application/json' -d '{"model":"coder","temperature":0,"seed":42,"messages":[{"role":"user","content":"Write a Python function that checks if a number is prime, with comments"}]}' | grep -oE '"predicted_per_second":[0-9.]+'; done
```
Use the middle of the three numbers in `./log`.

## Reading results in MLflow

Open the `llama-tuning` experiment, tick two or more runs, choose **Compare**. Settings that differ are shown side by side, and you can chart any setting against any result. Your note is on each run's page.

**Is a difference real?** Run each test 3 times. If the gap between two settings is smaller than the gap between your own repeats, treat them as the same. When two settings tie, keep the default: one less thing to remember.

**Things that skew results:** a second model busy on the same card, a chat in progress, heat (watch Beszel), and llama.cpp updates (re-test after updating the image).

## Settings reference

Which test to use depends on what kind of setting it is.

| Setting | What it does | Test with | Current → try |
| --- | --- | --- | --- |
| `-ub` (ubatch) | Prompt tokens the GPU handles per pass | `./bench` | 512 → 1024, 2048 |
| `-b` (batch) | Max prompt tokens per step; keep ≥ `-ub` | `./bench` | 2048 → keep |
| `-fa` | Flash attention | `./bench` | on → keep |
| `-c` | How much text the model holds in view | memory in llama-swap UI / Beszel | 65536 → raise to ~90% memory |
| `-np` / `-kvu` | Request slots / slots share one context pool | server test with 2 agents | 1 → 2 with `-kvu` |
| `-ctk` / `-ctv q8_0` | Halves context memory, small quality cost | `./bench`, then a quality test | f16 → q8_0 |
| `--spec-draft-n-max` | MTP: tokens drafted ahead | server test | 2 → 3, 4 |
| `--spec-draft-p-min` | MTP: only draft confident tokens | server test | 0 → 0.7 |
| `temperature`, `top_p`, `top_k`, `min_p` | How the next word is picked | quality test, 3 repeats | Qwen's values → leave unless testing quality |
| `--reasoning-effort` | How long it thinks first | quality test | medium → high |
| Quant (model file) | Q4 vs Q5/Q6: size vs accuracy | KL-divergence (full guide) | UD-Q4_K_XL → Q5_K_XL |

Other quality tests (KL-divergence for model file sizes, Terminal-Bench-Local overnight for finalists, a SWE-bench sample) are in the full guide; not set up yet.

## Files here

| File | What |
| --- | --- |
| `bench` | Benchmark one card and log it |
| `log` | Log any result (runs `log_run.py` inside the MLflow container; nothing to install) |
| `quality` | Tool-use quality test and log it |
| `log_run.py` | The logging script |
| `tool-eval/runs/` | Quality reports: one `.json` (scores) and one `.md` (every conversation) per run |
| `runs/` | Raw llama-bench output, one file per run |
| `results.md` | Written summary and next ideas |
