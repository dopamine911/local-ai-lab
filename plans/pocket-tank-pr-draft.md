# DRAFT: PR description for mediacutlet/pocket-tank (not posted; Daniel decides if and when)

Base: `mediacutlet/pocket-tank` `main` (952ef4d, v0.4.1). Head: `dopamine911/pocket-tank` `feat/training-harness-31b`.

Open questions for Daniel before posting are at the bottom, outside the PR text.

---

## Title

A 4.8M fish model (2.6 MB, 2.5x faster on the round pendant), the harness that made it, and the checks behind it

## Summary

This PR adds a smaller fish model as a **candidate**, the training harness and label tools that produced it, and the
measurements comparing it with the shipped model. **It does not replace `model/out/model_q4.bin`.** Shipping the
candidate is a separate, small step for you to decide on (see "Shipping it" at the end).

| | shipped 14M (`model/out/model_q4.bin`) | candidate (`model/candidates/relabel-d256n6/`) |
|---|---|---|
| shape | dim 384, 8 layers, 14.2M params | dim 256, 6 layers, 8 heads, 4.8M params |
| size at 4 bits | 7,557,640 B | 2,579,888 B |
| time per decision on the round 1.75C | 3.64 s (11.8 tok/s) | **1.47 s** (29.3 tok/s) |
| internal heap / PSRAM free during play (flat) | 35 / 1,227 KB | 62 / 1,675 KB |
| `docs/retrain-v4.md` acceptance bars missed | 6 | 3 (all narrowly) |

The tokenizer is unchanged (schema 4, the app's embedded `firmware/main/tokenizer.bin`), so the candidate flashes
into the model partition with its trailer and nothing else.

## How it was made

**Teachers.** Your prompt, unchanged (`system_prompt()` sha256 `eb8c0717…`).
- The models: Gemma 4 26B-A4B and Gemma 4 31B (Google's QAT Q4_0 GGUFs), served by llama.cpp b12020 through its
  OpenAI-compatible API. That's the new `gen_traces.py --api openai`; the Ollama path is unchanged and still the default.
- Settings: T 0.7, one request at a time.
- Seeds: every label is seeded from the run's base seed, its position and the state
  (`label_seed = sha256(f"{base}|{index}|{state}")[:4] & 0x7fffffff`, sent as the request's seed). With one worker on
  the same GPU type, a rerun repeats every answer (450 / 450 measured); with 4 parallel workers or another GPU type
  about 1% of answers flip.
- Each run writes a `<out>.meta.jsonl` with the teacher, prompt hash, temperature and seeds.

**Data** (`model/harness/datasets.toml`, `fish-v4-mix-cov-bigplay-relabel`: 28,948 pairs, sha256 `f16bd193…`). Four
steps, each measured before the next:

1. **50/50 26B and 31B labels.** On their own they make different fish:
   - The 26B is balanced and social: it follows 18% of the time, and its students reach F14 0.28.
   - The 31B is steadier but eats with fed fish: 77% at hunger 1-3 with food near, against the 26B's 34%.
   - The 26B-only students fail following (F1 0.05), so the mix keeps both.
2. **No-food coverage** (`gen_traces.py --no-food 0.6`, new). The sim shows food in about 96% of states, so calm fish
   with no food in view were about 4% of the data. Coverage took them to 25% and following to 0.62.
3. **Dart states labelled by a third teacher** (`gen_traces.py --play 0.6`, new). The new states are rested, fed,
   daytime fish with no food in view, at every boldness.
   - On those states the 26B / 31B darted with 1.4% / 7.5% of bold fish. Training on their labels made dart worse.
   - Qwen3.8-Flash-Next (UD-IQ4_XS) darted with 23% of bold fish and 0 of 80 timid ones.
   - 2,250 of its labels fixed F3. No teacher ever labelled a timid fish darting.
4. **The 26B relabels the 31B's fed eating** (`relabel.py`, new).
   - Scope: the 31B's 4,021 pairs at hunger 1-3 with food in view, answered again by the 26B (label seed 4701).
   - seek_food in those pairs went from about 58% to 17%. Every other pair is unchanged and the pair count is the same.
   - Hunger 0 and hunger 4+ were left alone. At hunger 0 the two teachers disagree in both directions, and a hungry
     fish should eat.

**Training.**
- `train.py` (karpathy/llama2.c), with a new `--loss decision`: loss on the decision tokens only, from pocket-terrarium.
- 5,000 iterations, batch 64, lr 6e-4, 3 seeds; the harness picked seed 1, kept at iteration 4,000.
- Run in ROCm PyTorch containers (`model/docker/`) by `model/harness.py`: one job per GPU, manifests, reports.
  The record of this run is in `model/candidates/relabel-d256n6/run/`.

**Why this size.** A sweep (6 seeds per size, same data, iterations and learning rate) found the smallest shape
matching the 14M on every panel row. dim 256 x 6 layers with the decision loss was that shape; dim 256 x 4 layers
lost the boredom rows (F8, F9), and dim 192 x 4 layers lost F1 and F3.

## Results

### Upstream's acceptance (`docs/retrain-v4.md`), both models' 4-bit files

All results are in `model/candidates/relabel-d256n6/acceptance/` (raw outputs included). **Bold** = misses the bar.

| check (bar) | shipped 14M | candidate |
|---|---|---|
| teacher agreement, 26B (>= 72%), n=300 | **68%** | 75% |
| teacher agreement, 31B, n=300 | 58% | 74% |
| bubbles kept at bored 0 / 1 (>= 0.4) | **0.21 / 0.20** | 0.59 / 0.51 |
| ... dropped at bored 8-9 (<= 0.15), explore rises | 0.00, 1.00 | 0.00, 0.99-1.00 |
| social 8, bored 9 leaves the friend | follow 0.32, explore 0.61 | follow 0.01, explore 0.98 |
| night + bored 9 rests / starving + bored 9 eats (>= 0.85) | 0.99 / 0.97 | 1.00 / 0.96 |
| follow cliff, social 0 -> 9 (v3m 0.01 -> 0.52) | 0.05 -> 0.54 | 0.00 -> 0.55 |
| dart cliff, bold 0 -> 9 (v3m 0.02 -> 0.40) | 0.03 -> **0.19** | 0.00 -> **0.15** |
| `fishsim --selftest-llm`, 1 min x3: landmark (< 59%) | 58 / 54 / **64** | 30 / 43 / 31 |
| ... longest one-goal stretch (< 59 s) | **59** / 54 / **59** | 36 / **60** / 36 |
| ... zones per fish-minute (> 1.5) / mean bored (< 3.3) | 2.8-3.5 / 1.7-1.9 | 2.8-3.2 / 1.3-1.6 |
| `--selftest-llm 10` x2: landmark / longest / bored | **59%** / **256 s** / **4.5** | 58% / **87 s** / **3.5** |

How these were run:
- The student is the 4-bit file. Agreement uses `eval.py` (greedy, through `runq4`) with the new `--api openai` and
  `--label-seed` options. The probes run on a checkpoint rebuilt from the 4-bit file (`q4_to_ckpt.py`); on the
  first candidate that matched its fp32 checkpoint to within 0.03 on the rows checked (F1, F3, F4, F6, F14).
- Agreement is with our llama.cpp teachers, not your `gemma4:26b` on Ollama, and with n=300 rather than 60
  (about ±2.6 points).
- The dart cliff averages 6 hand-written states and swings by training seed: seeds 1 / 2 / 3 give 0.15 / 0.35 / 0.28.
  Seed 1 was kept because the harness picked it.
- The 1-minute census is noisy: three runs of one model differ by up to 25 points. The 10-minute runs are there
  for that reason; the bars were set for 1-minute runs.

### `prompt_check.py` v4 rows on the student (`probe_fish.py`, n=400 per case, panel seeds 11 / 23 / 37)

| row (bar) | shipped 14M | candidate |
|---|---|---|
| F1 follow, social 9 (>= 0.40) | **0.36 / 0.35 / 0.38** | 0.60 / 0.60 / pass |
| F3 dart, bold 9 (>= 0.10) | 0.37 / 0.37 / 0.37 | 0.37 / 0.25 / 0.24 |
| F4 timid dart (<= 0.03) | **0.10 / 0.11 / 0.11** | 0.007 / 0.006 / 0.003 |
| F6 seek_food, lonely, not hungry (<= 0.05) | **0.08 / 0.08 / 0.07** | pass |
| F13 max single goal, content (< 0.45) | **0.454 / 0.459** / 0.438 | 0.26 / 0.28 / pass |
| F14 fed, food near: seek_food (report) | 0.15 / 0.14 / 0.17 | 0.29 / 0.26 / 0.30 |
| every other row | pass | pass |

The F rows are your bars for the *teacher* (n=40). Applying them to a student is our addition, so treat the
shipped model's misses here as a measurement, not a verdict.

### On the pendant (round 1.75C, same board, one session per model)

| | shipped 14M | candidate |
|---|---|---|
| log | 16.1 min, 3 fish | 9.3 min, 3 fish |
| time per decision, median (range) | 3,644 ms (3,583-4,124) | 1,472 ms (1,268-1,876) |
| asks / decisions | 259 / 209 (50 asks folded while the model was busy) | 153 / 150 |
| goal changes per fish-minute | 2.16 | 2.07 |
| dart decisions | 4 / 210 | 12 / 147 (all bold 8) |
| fed (hunger 0-2), food in view: seek_food | 2 / 38 | 9 / 61 |
| `i2c.master: I2C transaction failed` pairs | 18 | 17 (in every session, with every model: the board) |

The logs are in `model/candidates/relabel-d256n6/pendant/` and `model/candidates/mixcov-d256n6/pendant/`.
`pendant_replay.py` replays a log's states through any checkpoint and reproduces the device's logged `p=`.

## Known differences from the shipped model

- **Eats more when not hungry.** F14 0.27 vs 0.15; on the pendant 9 / 61 vs 2 / 38. Food-near curve, hunger 0 -> 8
  (`eat_by_hunger.py`):
  - candidate: 0.07 0.54 0.10 0.20 0.62 0.74 0.85 0.95 0.98
  - shipped: 0.01 0.23 0.05 0.14 0.33 0.50 0.72 0.98 0.99

  It eats earlier, and hunger 1 stands out (the 26B's own labels do the same there).
- **Darts more with bold fish.** On 569 rested, fed, no-food sim states with a wall in view: 0.097 vs 0.059 (the
  teachers' own labels 0.044). Over the 470 replayed pendant states: 0.190 vs 0.178. Timid fish dart less: 0.020 vs
  0.021 there, and no timid dart in its pendant session.
- **The dart cliff** (0.15 for this seed) misses the v3m bar, as the shipped model's 0.19 does.
- **The 10-minute census:** longest stretch 87 s and mean bored 3.5 (shipped: 256 s, 4.5).

## Findings in the existing checks (independent of this model)

1. **The bold cases' hunger is drawn once per panel.** `add("bold9", 40, bold=9, energy=9, hunger=rng.randint(0, 2))`
   evaluates `randint` once, so all of a panel's bold9 (and bold1) states share one hunger value. Which value depends
   on the seed *and* `n`:
   - The teacher check (n=40, seed 11) sees hunger 1.
   - The student probe (n=400, seed 11) sees hunger 0.

   For the 31B, F3 is 0.03-0.05 at hunger 1 and 0.82 at hunger 0. `dart_by_hunger.py` measures all three values.
2. **Every panel state says `wall clear`;** 299 of about 71,500 sim states do (0.4%). F3 therefore asks about a
   fish in open water, which the sim hardly produces. On sim states with a wall in view, the teachers' own labels
   dart with 4.4% of bold, rested, fed fish (`dart_on_sim.py`).
3. **The shipped model misses F1, F4 and F6 on the student probe,** on all three panel seeds (table above).

## What this PR adds

All new files and options; every default keeps the old behaviour.

**Labels**
- `gen_traces.py`: `--api openai`, `--label-seed`, `--max-tokens`, `--reasoning-effort`, `--no-food`, `--play`,
  `--temperature`; the key comes from `$TEACHER_API_KEY`, never argv; the meta file.
- `prompt_check.py`: the same teacher options; `--out`, `--rescore`, `--extra-cases` (fedfood appended after your
  cases, so your 450 states are unchanged).
- `run_teacher_jobs.sh`; `relabel.py`; `teacher_compare.py`; `label_mix.py`.

**Training**
- `train.py --loss decision`.
- `harness.py`, with `harness/{models,datasets,hosts}.toml` and `harness/runs/`.
- `run_sweep.sh`, `sweep_report.py`, `docker/Dockerfile.rocm-gfx120X` and `-gfx1151`.

**Measuring**
- `student.py`, `probe_fish.py`, `q4_to_ckpt.py`.
- `dart_by_hunger.py`, `dart_on_sim.py`, `eat_by_hunger.py`.
- `pendant_stats.py`, `pendant_replay.py`.
- `eval.py --api openai --label-seed` and `probe_dist.py --device`.
- `docker/Dockerfile.sim` + `build_sim.sh` build `fishsim` (SDL2, LVGL v9.2.2) and `runq4` on Linux with nothing
  installed on the host; `docker/run_gpu.sh`.

**Candidates**
- `model/candidates/relabel-d256n6/` (this one), with the earlier `mixcov-d256n6/` and `mixcovbigplay-d256n6/`.
- Each has the model, trailer, checksums, the training record, acceptance outputs and pendant logs.

## Reproducing it

1. **Labels.** Regenerate them from the seeds in `datasets.toml` with `run_teacher_jobs.sh`. The label files (20 MB)
   are not in this PR.
2. **Train:** `python3 model/harness.py run fish-d256-n6-decision-relabel`. The harness checks the folded dataset
   against its pinned sha256 and stops on a mismatch.
3. **Check:** each candidate's README lists its exact commands, from the relabel step to the trailer
   (`tools/model_trailer.py --tag relabel-d256n6`).

The whole log of the work, with every rule posted before its run and every correction, is in
dopamine911/pocket-tank issue #1.

## Shipping it (not in this PR)

1. `cp model/candidates/relabel-d256n6/model_q4.bin model/out/model_q4.bin`
2. Set `PT_MODEL_TAG` in `common/version.h` to the new tag.
3. Update `docs/memory_budget.md` (7.56 MB -> 2.58 MB).
4. Flash with `tools/flash_round.sh` or the README's two `write_flash` lines.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

---

## Open questions for Daniel (not part of the PR text)

1. Post it as a draft PR on upstream, or open an issue there first to ask whether they want it?
2. Include the 20 MB of label files, so the run repeats without a teacher? It would make reproducing it exact.
3. Ship the candidate in this PR (the four steps above), or leave that to the maintainer as written?
4. Keep the two earlier candidates (`mixcov-d256n6`, `mixcovbigplay-d256n6`) in the PR, or only candidate 3?
5. The branch has 36 commits. Squash them, or keep the history?
