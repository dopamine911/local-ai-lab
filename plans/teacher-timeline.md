# Timeline: improving the pocket-terrarium trace teacher (2026-11-09)

Daniel: look at the original project (github.com/mediacutlet/pocket-tank) and see how to make the teacher faster
and better. Allowed: a Gemma 4 26B-A4B candidate and chat-teacher's server settings. Lab side only: the project code
(`~/pt-train`, a copy of the builder's project, not a git repo) is not changed; things it fixes (temperature 0.7,
48 tokens, thinking off) are findings. A better teacher means new labels and retraining: the builder's call.
Pilot result files (they hold the project's prompt) stay in `~/pt-train/lab-pilot/`, not in this public repo.
Times are Chicago.

## From the original project (docs/stats.md, retrain-v3/v4.md, pipeline.md)

- Teacher: gemma4:26b (Q4_K_M) via Ollama on a Mac Mini; 36-51 decisions/min with 2-3 workers (1.2-1.7 s each).
- Teacher agrees with itself (same 40 states twice): 82%, the "practical ceiling"; student vs teacher 72-75%.
- Biggest quality factor: the prompt ("prompt is DNA": one sentence made 45% of labels follow_friend);
  `prompt_check.py` (diagnostic panel, ~7 min) before any long run.
- Our lab teacher `chat-teacher` is Gemma 4 31B dense (not the 26B), 4 slots of 4K, thinking off, --swa-full.

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 18:20 | Orientation: `~/pt-train/probe-img/model` is the newer copy (gen_traces.py 11-05 13:04: an extra "solitary lizard" rule in the prompt); pilot.py, pilot_l1.jsonl, eval.py identical in both copies | | `cmp`, `diff` |
| 18:29 | Baseline: `pilot.py --model chat-teacher` (37 L1 edge states x k=3, 4 workers), twice | agreement **99.1% / 97.3%**; 0 failures; **103 / 156 decisions/min** (run 1 includes the model load); latency median 1.50 / 1.53 s. Self-agreement: goal **96.4% / 93.7%**, goal+urgency **75.7% / 82.9%**; prompt cache 92-95%. The pilot is near its ceiling: it can show a worse teacher, hardly a better one | `~/pt-train/lab-pilot/baseline-chat-teacher-{1,2}.json` |
| 18:31 | Gemma 4 26B-A4B QAT Q4_0 pinned (google rev d1c082b, 14.4 GB, no mmproj) and downloading; entry `chat-teacher-26b` prepared = chat-teacher with only the model changed; hidden in Open WebUI | | this commit |
| 18:33 | Baseline 2: `prompt_check.py --schema L1 --model chat-teacher --workers 4` (the original's ~7-min prompt/teacher panel; 464 calls) | 3 min 9 s (~147/min), 0 failures. **All acceptance checks OK except one: "max single goal, content" = explore 0.46 (want < 0.45)**, the attractor the original also fought. Report only: P(hunt, hunger 0-2, prey near) 0.58. Per-case mixes kept in the log | `~/pt-train/lab-pilot/promptcheck-chat-teacher.{log,json}` |
| 18:38 | Gemma 26B downloaded (`MODELS-OK`); `chat-teacher-26b` applied (reload), loaded | load 16 s; answers without thinking; card 2 19.6 GB used (chat-teacher: 28.4), no spill | 79a2e64 |
| 18:40 | pilot.py on chat-teacher-26b, twice | agreement **80.2% / 81.1%** (31B: 97-99%); 0 failures; **243 / 245 decisions/min**, latency 0.93-0.99 s (31B: ~150/min, 1.5 s). Self-agreement goal 91.0% (31B 94-96%), goal+urgency 58-64% (31B 76-83%) | `~/pt-train/lab-pilot/cand-26b-{1,2}.json` |
| 18:41 | prompt_check on chat-teacher-26b (464 calls, 2 min) | **5 BAD** (31B: 1): A7 sleep → rest/hide 0.62 (want ≥ 0.85); A11 bold 9 vs bold 0 +0.00 (want ≥ 0.15); r8 hatchling vs adult hide +0.00; r3 shed soon vs no +0.00; content attractor explore 0.87 (want < 0.45). The personality differences the student should learn are mostly gone. **Verdict: 1.6x faster, clearly worse; not a replacement** with this prompt (tuned on the 31B; a 26B-tuned prompt is project work) | `~/pt-train/lab-pilot/promptcheck-26b.{log,json}` |
| 18:43 | chat-teacher `-np 4 → 8` (same 16K total: 8 slots of 2K; n_slots 8 / n_ctx_slot 2048 confirmed), card 2 28.3 GB, no spill | | d1d03be |
| 18:44 | pilot k=6 (222 calls) at 4 and at 8 workers | 4 workers **154 / min**, latency 1.47 s, 99.1%; 8 workers **163 / min (+6%)**, latency 2.77 s, 100%. The card is the limit (31B dense). **Reverted** (+6% not worth losing the 4K room per request) | `~/pt-train/lab-pilot/np8-w{4,8}.json` |
| 18:47 | Gemma 4 31B MTP drafter ("-assistant"): only community GGUFs (no google / ggml-org / unsloth), llama.cpp support for its drafter type not confirmed: not tried | | HF search |

## Upstream fish line (what Daniel asked for; the rows above were the lizard line by mistake)

Daniel 18:50: "I was asking about the upstream" (github.com/mediacutlet/pocket-tank, the fish line). Upstream's
pipeline is Ollama-only (gen_traces / prompt_check, last changed 2026-09-15). The builder's copy
(`~/pt-train/probe-img/model`) has the same fish v2/v3/v4 prompts (text identical), the same fish panel cases
(the copy only adds L1 cases) and the same state encoder, plus `--api openai`; requests match upstream's
(temperature 0.7, 48 tokens, thinking off). So `prompt_check.py --schema 4 --api openai` = upstream's v4 check on the
lab. Only difference: upstream's gemma4:26b is Ollama's Q4_K_M, ours Google's QAT Q4_0. Downloaded upstream files
were compared as text only (running them was refused by the auto-mode check).

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 18:55-19:08 | prompt_check `--schema 4` (450 calls, seed 11) twice each on chat-teacher-26b and chat-teacher (31B) | **26B**: run 1 all checks OK but the content attractor (inspect_reef 0.53; upstream try 3: 0.50), run 2 follow|social 9 0.38 BAD (want ≥ 0.40), attractor OK 0.35: reproduces upstream, noisy at the bars. **31B**: both runs follow|social 9 0.88, bubbles kept 0.85, night rest 1.00, but **dart|bold 9 0.03 BAD** (want ≥ 0.10) both times; attractor 0.53 / 0.42. Speed: 26B 133-151 s (~180-200/min), 31B 208 s (~130/min); upstream Mac Mini ~49/min | `~/pt-train/lab-pilot/fish-v4-*.{log,json}` |
| 19:09 | Self-agreement (same states in both runs) | **26B 80.0%** (upstream measured 82% for its teacher: reproduced); **31B 92.7%**. 26B noisiest on social/bold/content cases (65-75%), 31B 78-100% | scratchpad fish_selfagree.py |

Reading: for upstream, the lab gives (1) **speed**: the same teacher 4x faster (~190/min vs 49; an 8 h, 20K-pair night
becomes ~2 h), the 31B ~2.7x; (2) **quality**: the 31B's labels are much steadier (93% vs 80% self-agreement; the
original's notes say teacher noise caps the student: 82% → 75%), at the cost of one prompt sentence to restore bold
fish playing (dart). Using either needs upstream to talk to an OpenAI-compatible server (the builder's copy already
has `--api openai`): an upstream change, the builder's call.

## pocket-tank issue #1: seeded 31B labels on the lab (dopamine911/pocket-tank PR #2)

Daniel 2026-11-09: goal = help upstream shrink the fish model (pocket-terrarium's methods), teacher the 31B over
llama.cpp, prompt unchanged, seeds added, several temperatures, everything replicable; all machines may be used.
Work in `~/pocket-tank` (fork, branch feat/training-harness-31b). Test outputs in `~/pocket-tank/model/out/` (git-ignored).

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 19:38 | Seeded v4 panel on chat-teacher (card 2), temperature 0.7, label seed 7, 4 workers, twice | same goal 98.7% (444/450), goal+urgency 95.3%; checks as before (dart bold 9 0.05/0.03 BAD, the rest OK) | `out/check-31b-t0.7-ls7-r{1,2}.json` |
| 19:55 | Same, 1 worker, n 8 (90 questions), twice | 100% / 100% (84 / 79 s) | `out/check-31b-t0.7-ls7-n8-w1-r{1,2}.json` |
| 20:21 | Same, 1 worker, full 450, twice | **100% (450/450) goal and urgency**; 1 worker vs 4 workers 98.9%. So parallel requests cause the ~1% differences; 1 worker repeats exactly. 395 s per run (~68/min) | `out/check-31b-t0.7-ls7-w1-r{1,2}.json` |
| 20:37 | `chat-teacher-c1` (twin on card 1, routing set `teachers`: coder off card 1, helpers stay) applied, loaded | load 21 s; card 1 32,235 of 32,624 MiB, GTT +110 MB (small spill, no speed cost) | 7ab3bf0 |
| 20:39 | Card 1 vs card 2: seeded n 8 panel, 1 worker | **100% (90/90) goal and urgency**, same time (84 s): either card can make any shard | `out/check-31b-card1-t0.7-ls7-n8-w1.json` |
| 20:40 | Gemma 4 31B file pinned for marvin, copied from zaphod over the cable (17.7 GB in 80 s, 211 MB/s), `lab models` sha256 OK; marvin `chat-teacher` entry (zaphod's settings, ttl 900, training pin) deployed while idle | loads in 9 s (unloads big) | 25aac1f |
| 20:46 | marvin vs zaphod card 2: seeded n 8 panel, 1 worker | 194 s (card: 84 s, **2.3x slower**, ~28/min); answers **differ slightly**: same goal 98.9% (89/90), goal+urgency 94.4% (other GPU type, other numbers) | `out/check-31b-marvin-t0.7-ls7-n8-w1.json` |
| 20:50 | marvin repeat (1 worker) and 4 workers | repeats exactly (90/90 goal and urgency, 192 s); 4 workers 99 s (~55/min). big restored afterwards | `out/check-31b-marvin-*` |

Capacity for phase 3 (31B, temperature 0.7): exact (1 worker each): card 1 + card 2 ~68/min each + marvin ~28 = ~164/min
(24,000 labels ~2.4 h); ~99% repeatable (4 workers each): ~135 + ~135 + ~55 = ~325/min (~1.2 h). A shard made on
marvin repeats only on marvin; zaphod's two cards are interchangeable.

### Night of 2026-11-09 (Daniel asleep; all three GPUs allocated; goal: a smaller fish model without losing quality)

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 21:31 | Preflight | training images on both hosts (pt-train-rocm gfx120X / gfx1151); pocket-tank has sim/ (needs SDL2 dev: not installed, so the sim census is optional); `~/pocket-tank` upstream push URL set to `no_push`; **watchdog still stopped since 16:35** (needs sudo) | |
| 21:35 | Decision rules posted before any result | temperature rule and tier A / tier B for "no quality loss" | pocket-tank issue #1 comment 6758187708 |
| 21:33 | Phase 2 started: `run_teacher_jobs.sh` queues in tmux p2-card1 / p2-card2: panels 0.7/0.85/1.0 x label seeds 7, 8 (0.7 s7 reused from 20:21), generator runs of 500 at each temperature, 26B calibration | | `~/pocket-tank/model/out/phase2/` |
| 21:36 | `ssh zaphod` from zaphod: "Host key verification failed"; zaphod's ed25519 host key added to `~/.ssh/known_hosts` after checking it equals `/etc/ssh/ssh_host_ed25519_key.pub` (the harness reaches zaphod over ssh) | ssh zaphod ok | |
| 21:38-21:44 | Phase 4 in a separate worktree `~/pocket-tank-port` (branch feat/harness-port; the running jobs use the frozen `~/pocket-tank`): harness.py, train.py (decision loss), prompt_check v4_table (+fedfood), student.py, probe_fish.py, sweep_report.py, TOML configs; llama2.c 350e04f copied from the 11-06 harness code | harness selftest OK; probe selftest OK; panel unchanged (sha256); smoke + smoke-decision through the whole chain on marvin (opportunistic: zaphod's cards serve teachers) | pocket-tank 9ab8506, 158b940, ee04ebf |
| 22:05 | Phase 2 done (0 failed jobs) | panels: F3 dart bold 9 BAD at every T (0.03-0.05), F13 0.40-0.53, the rest OK; self-agreement (seed 7 vs 8) goal 91.3 / 90.2 / 88.0%, goal+urgency 68 / 64 / 58% at T 0.7 / 0.85 / 1.0; same seed across T 97.6-98.9% (the seeded draw decides). Generator mix (500 states): **26B seek_food 34.8%** (= upstream's documented 35%), **31B 55.8 / 65.4 / 59.8%**; 31B eats at hunger 0-2 with food in view 0.37-0.45 (26B 0.13), follows 1.6-3.6% (26B 18%), explores 5-10% (26B 19%). **Chosen T = 0.7** (rule 5 then rule 2) | issue #1 comment 6758688788; `~/pocket-tank/model/out/phase2/` |
| 22:07 | Phase 3 started (tmux p3-card2, p3-card1, p3-marvin; marvin's `chat-teacher` replaces big): T 0.7, 1 worker per server; shards sim seed / label seed: card 2 101-104 / 1101-1104 x 3000, card 1 111-114 / 1111-1114 x 3000, marvin 121-122 / 1121-1122 x 2500 | first minute: card 2 67, card 1 64, marvin 22 labels (~65 / 65 / 22 per min): cards done ~01:15, marvin ~01:55 | `~/pocket-tank/model/out/data/fish31b/`; code b4284d2 |
| 23:40 | Phase 3 midpoint | 11,500 labels in 4 finished shards (cards ~45 min per 3,000 = ~67/min; marvin 2,500 in 92 min = ~27/min) + 3,000 in progress; all shards due ~01:12, so no shard cut. seek_food ~55% in the new shards (as phase 2). Card 1 GTT 1,670 MiB (steady). **One timeout: marvin shard 121's first request** (curl rc 28, 30 s: marvin was still loading the teacher after unloading big); the sim ticked on, so a rerun with the teacher loaded differs from that shard's 2nd state | `~/pocket-tank/model/out/data/fish31b/gen-chat-teacher-T0.7-ls1121-sim121-n2500.log` |
| 01:18 | Phase 3 done: 10 shards, **29,000 labels**, no failed job (the one timeout above) | cards 4 x 3,000 each (~45 min per shard), marvin 2 x 2,500 (~92 min each) | `~/pocket-tank/model/out/data/fish31b/` |
| 01:19 | Harness port merged into the PR branch (e6212a8); llama2.c 350e04f copied in (git-ignored); dataset folded: **28,993 pairs** (7 duplicates), sha256 ef2dd667, pinned. My first glob also caught the 10 `.meta.jsonl` files (dropped as malformed, data unaffected); narrowed to the shards, same sha | seek_food 55.1%, rest 24.8%, explore 10.7%, follow 3.7%, reef 3.6%, bubbles 1.5%, dart 0.6% | pocket-tank 0ef40b1 |
| 01:19 | `lab training-mode on` (both hosts): every teacher and chat model unloaded, helpers stay; zaphod cards 7.5 / 3.7 GB, marvin 120.5 GB free | | |
| 01:19 | Sweep started (tmux `sweep`, one driver, 14 runs x 3 seeds, reference fish-14m-all first) | 14M: 0.06-0.07 s/iter on the R9700s, 0.15 on the 8060S | `~/pocket-tank/model/out/sweep.log` |
| 01:33 | Reference fish-14m-all done (s1 marvin 792 s, s2/s3 cards ~300 s) | val 0.748-0.756. Rows (mean of 2 panels x 3 seeds): **F6 seek_food lonely, no food 0.11-0.37 FAIL** (teacher 0.00: the student over-learns the eating), **F14 fed + food near 0.89**, F3 dart 0.06-0.21, F10 0.40-0.68, F13 0.36-0.47, F1 0.73-0.86, others pass. Seed-to-seed spread is large, so tier A (vs the worst reference seed) is lenient | `~/pocket-tank/model/out/harness/fish-14m-all/report.md` |
| 02:33 | Sweep done: 14 runs x 3 seeds, 01:19-02:33, no failed job (14M ~14 min per run, marvin the slowest card; small sizes 2-4 min) | **Tier A (posted rule): no run passes, not even fish-14m-decision. The rule is flawed:** leave-one-out, each reference seed fails tier A against the other two (3 seeds: a model identical to the reference misses "no worse than the worst reference seed" ~1/4 of the time per row and seed). Tier B: the 14M reference misses upstream's F3, F6, F10, F13. Exploratory (means over 3 seeds x 2 panels, flag = worse than the reference mean by > 2 reference seed-sd): decision-only loss lowers goal NLL at every size (1.78M decision 0.2415 vs 14M reference 0.2549); smaller models drift on F7/F9 (boredom, ~0.02 -> ~0.12, bar 0.15) and F1 (0.78 -> ~0.6, bar 0.40); closest small: fish-d256-n6-decision (4.8M, 2.58 MB). Seed spread as large as the size effects | issue #1 comment 6762807666; `~/pocket-tank/model/out/harness/fish-sweep-{verdict,exploratory}.md` |
| 02:35 | Seeds 4-6 for fish-14m-all, fish-d256-n6-decision, fish-d256-n4-decision, fish-d192-n4-decision (tmux `sweep2`) | | `out/sweep2.log` |
| 02:56 | Seeds 4-6 done (02:35-02:56). Six-seed comparison (exploratory, Welch t) | **fish-d256-n6-decision (4.83M, 2.58 MB): no row worse than the 14M reference beyond seed noise, F6 better (0.06 vs 0.26), goal NLL 0.2433 vs 0.2529: the "smaller without losing quality" candidate.** d256-n4-decision (3.2M) loses the boredom response (F8 0.54 vs 0.79, F7 0.22 > 0.15); d192-n4-decision (1.78M) loses following (F1 0.49), dart (F3 0.03), F10 (0.36) | issue #1 comment; `out/harness/fish-sweep-6seeds.md` |
| 02:58 | Training mode off; PR branch pushed to the fork (d77df74) | | |
| 03:00-04:27 | Teacher tests (Daniel's ask): coder on zaphod card 1 / coder-2 card 2, big then gpt-oss-120b (`--max-tokens 512 --reasoning-effort low`, new options in the port worktree, defaults unchanged) on marvin; panel seeds 7 / 8 + 500 generator states, T 0.7, 1 worker; 0 failed answers | **No teacher passes all upstream checks; each makes its own fish.** 26B balanced and social (follows 18%); 31B hungry, steadiest (91%); **gpt-oss: food mix = 26B's (seek_food 35.4%, 0.00 at low hunger) but no following (F1 0.25-0.28), 89%, ~21/min**; big explorer (explore 27%, F10 0.03, F13 0.7), 78%; coder playful and erratic (timid fish dart, 65%) | issue #1 comment 6764994168; `~/pocket-tank-port/model/out/teachers/teacher-comparison.md` |
| 03:19 | Teacher-mix data: `chat-teacher-26b-c1` (26B twin on card 1) added (homelab 50b6d6f, reload while zaphod idle; marvin tests unaffected: direct); 26B shards sim 201-205 (card 2) and 211-215 (card 1) x 2,900, label seeds 2201-2215, T 0.7, 1 worker | ~110/min per card, done ~05:20 | `~/pocket-tank/model/out/data/fish26b/`; configs pocket-tank 5c8eb00 |
| 05:17 | 26B data done: 10 shards, 29,000 labels, no failures or warnings | ~110/min per card (1 worker) | `~/pocket-tank/model/out/data/fish26b/` |
| 05:18 | Datasets folded and pinned: fish-v4-26b 28,998 pairs (de3bbabe; seek_food 32.7%, follow 17.7%, explore 20.8%), fish-v4-mix 28,996 (5f55d52a; seek_food 44.9%, follow 10.5%) | | pocket-tank 1343903 |
| 05:18 | Training mode on; mix runs started (tmux `mix`): fish-14m-all and fish-d256-n6-decision on fish-v4-26b and fish-v4-mix, 3 seeds each | | `out/sweep-mix.log` |
| 05:56 | Mix runs done (05:18-05:56, no failures). 14M / d256-n6-decision on 31B / 50:50 / 26B labels | 31B: F1 0.78, F3 0.10, F6 0.22 (14M). **26B: F1 0.05, F3 0.02** (students go to visit_bubbles ~0.45 / inspect_reef ~0.30 on calm no-food states: upstream's attractor), F6 0.04, F14 0.28. Mix in between (F1 0.33). Cause found in the data: panel states have no food, the generator shows food in 96% of states, and neither teacher ever labels follow for social 7-9 without food in view; the 31B ties follow to social level (generalizes), the 26B follows by proximity (does not). Upstream (retrain-v4.md): the shipped cliffs came from the old v2/v3 labels. d256-n6-decision tracks the 14M on every dataset | issue #1 comment; `out/harness/fish-teacher-mix.md` |
| 05:57 | Port branch merged into the PR branch (bd25870: --max-tokens / --reasoning-effort), re-verified (dry-run identical, harness and probe selftests), pushed to the fork only (upstream push blocked: `no_push`) | | dopamine911/pocket-tank PR #2 |
| 05:58 | **State:** training mode off on both; zaphod: coder, coder-2, embed, rerank, stt, stt-whisper, tts ready; marvin: big ready, power mode performance; no tmux jobs left. **Watchdog still stopped since 16:35 (needs `sudo systemctl start lab-watchdog`).** Lab entries added tonight and kept: zaphod `chat-teacher-c1`, `chat-teacher-26b-c1`; marvin `chat-teacher` (+ its Gemma 31B file) | | |

### 2026-11-10 morning: coverage experiment (Daniel: "do 2")

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 07:50 | `gen_traces.py --no-food P` (clears the pellets before a sample with chance P; default 0, dry-run identical; 0.6 gives food none in 66% of states, default 3%); rule posted before running | | pocket-tank 0968ddb; issue #1 comment 6769121272 |
| 07:53 | Coverage data: 26B with NO_FOOD=0.6, shards sim 301-302 (card 2) and 311-312 (card 1) x 2,250, label seeds 2301-2312, T 0.7, 1 worker (coder / coder-2 off meanwhile) | ~105/min per card, done ~08:35 | `~/pocket-tank/model/out/data/fish26b-cov/` |
| 08:13 | Early look at the first 4,500 coverage labels (66% no food) | no food, friend in view, social 7-9: follow 0.17 (before: no such labels at all); bold + energetic without food: dart 0.01 (the 26B hardly darts on sim states) | |
| 08:30 | Coverage data done: 4 shards, 9,000 labels, no failures or warnings; fish-v4-26b-cov folded and pinned: 29,299 pairs, sha256 d06a7e28, food none 22.4% (was ~4%); seek_food 28.0%, follow 16.1%, explore 20.7%, dart 0.1% | | pocket-tank (datasets.toml) |
| 08:31 | Training mode on; runs fish-14m-all-26bcov and fish-d256-n6-decision-26bcov (tmux `cov`) | | `out/sweep-cov.log` |
| 08:50 | Coverage runs done (08:31-08:50, no failures). **Rule: does not fully work** | 14M: F1 0.05 -> **0.35** (bar 0.40; sd 0.15), F3 0.02 -> 0.00, nothing worse (F6 0.04 -> 0.01, F11 0.82 -> 0.91). 4.8M: F1 0.05 -> 0.28. Following was a coverage gap; dart is a teacher gap (the 26B darts on 0.1% of sim states) | issue #1 comment; `out/harness/fish-coverage.md` |
| 08:54 | Teacher mix with coverage (Daniel): rule posted; 31B coverage shards (NO_FOOD=0.6) sim 401 (card 2) and 411 (card 1) x 2,250, label seeds 1401 / 1411; dataset fish-v4-mix-cov configured (26B 201, 202, 211 + cov 301, 311; 31B 101, 102, 111 + cov 401, 411) | | issue #1 comment 6770586802; pocket-tank 1a2bb9f |
| 09:30 | 31B coverage shards done (2 x 2,250); one timeout on each shard's first request (the 31B still loading after pushing coder off; that state skipped). fish-v4-mix-cov folded and pinned: 26,698 pairs, sha256 d76fd64a, food none 24.5%, seek_food 36.5%, follow 10.8%, dart 0.3%; training mode on, runs started (tmux `mixcov`) | | `out/sweep-mixcov.log` |
| 09:52 | Mix-with-coverage runs done (09:31-09:52, no failures). **Rule: does not fully work** (14M F3 0.03; F6 flagged 0.04 vs 0.01, tiny) | **Best dataset so far:** 14M passes every upstream bar but F3 (F1 0.62); **4.8M passes every bar but F4 (0.03, at the bar) and passes F3 (0.13)**; seek_food 36.5% in the data, fed + food near 0.56 | issue #1 comment; `out/harness/fish-mixcov.md` |


### 2026-11-10: pendant test of the 4.8M (Daniel ran the commands on dark, option C)

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 10:03 | Bundle prepared: fish-d256-n6-decision-mixcov seed 2, `model_q4.bin` 2,579,888 B (sha256 72fa241c…) + `model_trailer.bin` (tag mixcov-d256n6), README with backup and restore steps; sent via Taildrop | runq4.c decisions sensible; the app's embedded tokenizer is byte-identical to schema 4 | `~/pocket-tank/model/out/pendant/` |
| ~10:35 | Daniel flashed (esptool 5.6.0, after a read-only backup of the model partition) and logged 12.5 min, 2-3 fish | 119 asks, 119 decisions, starve-ignored 0 | `~/pocket-tank/model/out/pendant/log/pendant-test.log` (894 lines) |
| 10:51 | Log received (Taildrop) and analysed | **1.47 s per decision** median (1.22-1.60), 29.8 tok/s; internal heap 62 KB and PSRAM 1,669 KB flat; all 7 goals used (seek_food 25 of 111, dart 6); fed + food in view: seek_food 16 of 56. vs a 14M on the same board (pocket-terrarium PR #9, other session): 4.05 s, 11.6 tok/s, 57 KB, so ~2.7x faster. 18 pairs of I2C "transaction failed" lines, often near touch taps, likely the board (0.75) | issue #1 comment 6773501901 |
| ~11:00 | Daniel restored the stock model from the backup (Daniel: "lets flash back the stock for a base compare") and logged it; log received 11:18 (second Taildrop send) | | `~/pocket-tank/model/out/pendant/log/pendant-stock.log` (1,125 lines) |
| 11:19 | Stock vs 4.8M, same board, same day | **stock 3,644 ms / 11.8 tok/s vs 4.8M 1,474 ms / 29.8 tok/s: 2.5x faster**; internal heap 35 vs 62 KB, PSRAM 1,227 vs 1,669 KB (all flat); asks/decisions stock 259/209 (19% folded while the model was busy) vs 119/119; I2C pairs 18 in both, so not the model. Fed with food in view: seek_food stock 2/38 vs 4.8M 16/56 (eats more when not hungry; not an upstream bar; uncontrolled) | issue #1 comment 6774149178; `~/pocket-tank/model/out/pendant/comparison.md`, `tools/pendant_stats.py` |
| 11:25 | Candidate committed in pocket-tank (Daniel: "commit the changes and the current version of the model"; his choice: own folder, stock model stays) | model/candidates/mixcov-d256n6/ + model/pendant_stats.py; trailer rebuilds byte for byte. **Correction:** this seed fails F3 on its own (0.077 / 0.047, bar 0.10); the 0.13 at 09:52 was the 3-seed mean | pocket-tank f62cc08 (not pushed); issue #1 correction comment |
| | Open (Daniel) | Is eating when fed acceptable, or rebalance the 31B share; which dataset for the PR; backup hash check output not seen | |

### 2026-11-10 midday: dart (F3) and the stock baseline (Daniel: "do both": push, then the training step)

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 11:27 | Pushed pocket-tank f62cc08 to the fork (PR #2 head f62cc08) | | |
| 11:29 | Dart labels counted in fish-v4-mix-cov (goal field) | 26B 13 of 13,200 (0.1%), 31B 73 of 13,500 (0.5%), all at bold 4+; bold 7+ energy 7+ day: 26B 0.8%, 31B 4.9%. Panel F3 of the 31B 0.03-0.05 (6 runs), gpt-oss 0.05, big 0.57-0.68 (fails F10, F13). Upstream docs/retrain-v4.md try 3: gemma4:26b on Ollama dart \| bold 9 = 0.50 | |
| 11:32 | Rules posted (step A 26B panel; step B seeds 4-9, confirm a pick on panel 37) | | issue #1 comment 6774445629 |
| 11:32-11:40 | Step A: 26B panel, label seeds 7 and 8 (chat-teacher-26b, T 0.7, 1 worker) | **F3 0.23 / 0.20: "between", no conclusion by the rule.** Seed 8 passes every bar; seed 7 all but F13 (inspect_reef 0.50, the same as upstream's try 3). Our 26B does dart on the panel; upstream's 0.50 not reproduced (n=40) | `~/pocket-tank/model/out/phase2/panel-chat-teacher-26b-T0.7-ls{7,8}.json` |
| 11:38 | Exploratory: training states like the panel's bold 9 case (bold 8+, energy 8+, hunger 0-2, no food, day) | only 44 (26B) + 52 (31B) of 26,698 (0.4%); there the 31B darts 13/52 (25%), the 26B 1/44: dart looks like a coverage gap of that context more than a teacher gap | |
| 11:35 | New `model/q4_to_ckpt.py`: a model_q4.bin back to a probe-able checkpoint. Checked on the candidate: q4 vs fp32 F1 0.665 / 0.665, F3 0.072 / 0.077, F14 0.708 / 0.737 | | pocket-tank (uncommitted) |
| 11:36 | **Stock baseline:** the shipped `model/out/model_q4.bin` (14.2M) probed on panels 11 / 23 / 37 | **fails F1 (0.36 / 0.35 / 0.38), F4 (0.10 / 0.11 / 0.11), F6 (0.08 / 0.08 / 0.07)**, F13 at the bar (0.454 / 0.459 / 0.438), F2 0.050 on p23; passes F3 (0.37). F14 (fed, food near: seek_food) **0.15 / 0.14 / 0.17 vs the candidate 0.71 / 0.67**: the candidate eats far more when not hungry (controlled) | `~/pocket-tank/model/out/q4probe/` |
| 11:39 | Daniel: "add it to the docker container dont use cpu its slow": new `model/docker/run_gpu.sh` (the harness's devices, one card) | a probe 5 s on card 1 (CPU ~60+ s) | pocket-tank (uncommitted) |
| 11:41 | Step B: training mode on, run fish-d256-n6-decision-mixcov-s4 (seeds 4-9, nothing else changed; tmux `seeds`) | | `out/sweep-mixcov-s4.log` |
| 11:48 | Step B done (11:41-11:48, no failures), training mode off | **2 of 9 seeds (6, 9) pass every row on panels 11 / 23; the pick, seed 9, fails confirmation on panel 37: F3 0.029 (seed 6 0.030).** No 4.8M passes every bar | issue #1 comment 6774853475; `out/harness/fish-d256-n6-decision-mixcov-s4/` |
| 11:49 | Why: the panel draws the bold9 case's hunger once per panel (p11 0, p23 2, p37 1). Dart by hunger, n=400: stock 0.35 / 0.36 / 0.39 (timid 0.11); 4.8M seeds 1-9 at hunger 1 0.009-0.064. Labels (bold 7+, energy 7+, day): hunger 0 dart 14.5%, **hunger 1 0.9%** (seek_food 169 of 343), hunger 2 3.3%: a data gap at hunger 1 | | `~/pocket-tank/model/out/q4probe/hunger_dart.py` |
| 11:51 | Pushed pocket-tank 738150d to the fork (the issue comment cites it) | | |
| 11:53 | Recount with no food in view (bold 7+, energy 7+, day, hunger 0 / 1 / 2): 31B dart 11/32, **3/42**, 4/36; 26B 1/32, 0/37, 1/42. Both a coverage gap (~110 such states per teacher, 0.4%) and the teachers (the 31B darts mostly at hunger 0; the 26B hardly on sim states). Every teacher panel so far (and upstream's 0.50) is panel seed 11: hunger 0 only. `model/dart_by_hunger.py` committed and pushed (d5c9bcc); issue comment corrected (the first version counted states with food in view) | | issue #1 comment 6774853475 (edited) |

### 2026-11-10 afternoon: upstream's acceptance checks (Daniel: "run upstream's acceptance checks on the 4.8M and stock")

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 11:59 | probe_dist.py (new `--device cuda`) on both q4-derived checkpoints, held-out 1,000 labels (26B + 31B, sim seed 1) | stock FAILS bubbles-kept at bored 0-1 (0.21 / 0.20, bar 0.4); 4.8M 0.89 / 0.85; both drop bubbles at bored 8-9, rest at night, eat when starving (0.97 / 0.95). Dart cliff: stock 0.03 -> 0.19, 4.8M 0.00 -> 0.01 (v3m 0.40); follow 0.05 -> 0.54 vs 0.01 -> 0.82 | pocket-tank `model/candidates/mixcov-d256n6/acceptance/` |
| ~12:01 | `docker/Dockerfile.sim` (ubuntu:26.04 by digest, SDL2, LVGL v9.2.2 7f07a129, curl added 12:05) builds sim/fishsim + model/runq4; no host change | | pocket-tank e5d5903 |
| 12:03 | Census 1 min x3 each | noisy: landmark stock 58/54/64, 4.8M 70/70/45; longest 59/54/59 vs 60/60/30 s | |
| 12:04-12:13 | Census 10 min x2 each | landmark 59/54 vs 54/54%; longest stretch **256/261 vs 82/82 s**; mean bored 4.5/4.7 vs 4.2 (both over 3.3) | |
| 12:06-12:20 | Teacher agreement (eval.py `--api openai --label-seed 7`, n=300, seed 1, runq4 greedy); 26B on card 2, 31B on card 1. First 4.8M runs lost to a script bug (`set --` overwrote the teacher name), rerun 12:14 | 26B: stock 68%, 4.8M 71% (bar 72; +-2.6); 31B: stock 58%, 4.8M 73%; 0 failed teacher calls | |
| 12:21 | Results committed and pushed (pocket-tank 3ddcbb7), posted | neither passes every check, the shipped model included | issue #1 comment |

### 2026-11-10 afternoon: dart fix (Daniel: "fix dart first, then rerun the checks")

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 12:31-12:46 | Teacher panels at panel seeds 23 (bold case hunger 2) and 37 (hunger 1): 26B card 2, 31B card 1, big via marvin peer; T 0.7, label seed 7, 1 worker | P(dart \| bold 9): 26B h0 0.23/0.20, h1 0.23, h2 0.10; 31B h0 0.03-0.05, h1 0.82, h2 0.12; big h0 0.57-0.68, h1 0.33, h2 0.55; timid 0.00-0.03 all. Both our teachers dart at every hunger: the student gap is mostly coverage | `~/pocket-tank/model/out/dartcheck/` |
| 12:45 | `gen_traces.py --play P` (rested, fed, no food, day, bold 0-9); default byte-identical (300-state dry run) | --play 0.3: 30% of 2,000 dry-run states in that context | pocket-tank (committed with the configs) |
| 12:47 | Rule posted; play shards started: 26B sim 501 ls 2501 (card 2), 31B sim 511 ls 1511 (card 1), 2,250 each, PLAY=0.6, tmux play-* | 26B ~130/min, 31B ~65/min | issue #1 comment 6776162754; `out/data/fish{26b,31b}-play/` |
| 13:03 / 13:22 | Play shards done (26B / 31B, 2,250 each, 0 warnings) | in the play context, bold 7+ dart: 26B 4/290 (1.4%), 31B 21/279 (7.5%; hunger 1 2/79); timid 0 | `out/data/fish{26b,31b}-play/` |
| 13:23 | **Correction:** at the teacher panel's n=40, panel seed 11 = hunger 1, 37 = hunger 0 (the students' n=400: 11 = 0, 37 = 1). Teacher dart h0/h1/h2: 26B 0.23 / 0.20-0.23 / 0.10, 31B 0.82 / 0.03-0.05 / 0.12, big 0.33 / 0.57-0.68 / 0.55 | issue comments 6774853475 and 6776162754 edited | |
| 13:23 | Panel vs sim: every panel state has `wall clear`, 299 of ~71,500 training states do (0.4%) | the panel's F3 asks about an open-water fish the sim hardly makes | |
| 13:24-13:28 | Run fish-d256-n6-decision-mixcovplay (dataset 31,198 pairs, cdaf7d4a), training mode on, then off | **Rule: does not work.** F3 mean 0.087 (was 0.127); hunger 1 0.008-0.024; panel 37 F3 0.009-0.021 all fail; timid 0.00-0.01 | `out/harness/fish-d256-n6-decision-mixcovplay/` |
| 13:29 | Dart on the 569 realistic play states (wall in view): teachers' labels 0.044; stock 0.059; candidate s2 0.026; mix-cov s9 0.060; play s2 0.042; at hunger 1 the students 0.002-0.003 vs stock 0.042 | acceptance checks not rerun (the rule) | pocket-tank ff216f4; issue #1 comment 6777146030 |
| 13:31 | Daniel asked which teachers dart more; from existing runs: big and coder (sim bold 7+ 9.0% / 9.4%; coder's timid fish dart on the panel 0.10). Daniel: "lets try that before acceptiong this" | | |
| 13:31-13:40 | Rule posted; big test: 300 samples, `--play 1.0`, sim 601, ls 3601 via marvin peer (first try failed: the slash in `marvin/big` in file names, fixed pocket-tank 1303d66; one timeout while big loaded) | **promising:** bold 7+ dart 15/64 (23%), hunger 1 5/19, timid 0/80; explore 110/212 | issue #1 comments 6777238403, 6777395919 |
| 13:40 | Full big play shard started: sim 611, ls 3611, 2,250, PLAY=0.6 (~60/min, ~38 min); configs fish-v4-mix-cov-bigplay + run mixcovbigplay | | `out/data/big-play/` |
| 14:17 | Big play shard done (13:40-14:17, 2,250, 0 warnings); training mode on; run mixcovbigplay (dataset 28,948 pairs, a52e8648; 14:17-14:21); training mode off | **Rule: works.** Seeds 1 and 2: dart >= 0.10 at hunger 0/1/2 (s1 0.36/0.24/0.41), timid <= 0.013, every row passes on panels 11/23; s1 also on panel 37. s3 fails timid dart (0.06). F10 0.57-0.61 (was 0.72-0.75); dart on realistic states 0.12 (stock 0.06) | `out/harness/fish-d256-n6-decision-mixcovbigplay/` |
| 14:22-14:28 | Acceptance on seed 1 (agreement 26B + 31B n=300, probe_dist, census 1 min x3 and 10 min x2) | agreement **74% / 77%** (stock 68 / 58); dart cliff **0.01 -> 0.46** (passes 0.40); bubbles kept bored 0/1 0.84 / **0.36**; census 1 min landmark 41/32/39, longest 60/37/37; 10 min 59%, 61 s, bored 3.3 | |
| 14:33 | Candidate 2 committed and pushed (pocket-tank 82d9d4c: `model/candidates/mixcovbigplay-d256n6/`), posted | misses 4 bars, all narrowly (bubbles bored 1, longest stretch, 10-min landmark 59% and bored 3.3 at the bars) vs the shipped model's 6 | issue #1 comment 6778569977 |
| 14:59 | Pendant bundle for candidate 2 (Daniel: "prepare the pendant bundle for candidate 2"): `model/out/pendant-c2/` + `pendant-c2.tar` (sha256 346309d0); runq4 on 4 states: bold fed -> dart_play, starving -> seek_food, night -> rest, social 9 -> follow_friend; app tokenizer identical | sending needs sudo (Daniel runs `sudo tailscale file cp`) | `~/pocket-tank/model/out/pendant-c2/README.txt` |
| ~15:04-15:16 | Candidate 2 on the pendant (Daniel flashed and logged on dark; "the fish seem more responsive but it could just be my bias"); log received 15:16 | 1,466 ms per decision (= candidate 1), heap 62 KB flat, PSRAM 1,591 KB; goal changes per fish-minute 2.17 (c1 2.10, stock 2.16); dart 9/149, 4 by timid fish | `~/pocket-tank/model/candidates/mixcovbigplay-d256n6/pendant/` |
| 15:20 | Timid darts checked: no teacher labels a timid dart (big 0/111, 26B+31B 0/86); replaying each session's states (new `pendant_replay.py`): candidate 2's timid P(dart) at or below stock's in every session (its own session: stock 0.050, c2 0.024) | the responsive feel is the darting, not more switching | pocket-tank f500e61, 9d6e376; issue #1 comment |

### 2026-11-10 afternoon: eating when not hungry (Daniel: "fix the eating when not hungry first")

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 15:20 | Labels: P(seek_food), hunger 1-3, food near, day: 31B 77%, 26B 34%, big 30%; by hunger x distance the 31B eats more at 1-3 everywhere, hunger 0 mixed (near 26B 20% vs 31B 6%) | 26B-only students F14 0.28 (fish-teacher-mix.md): the floor for this change, not stock's 0.15 | |
| 15:24 | Method changed from "less 31B" (that cuts following) to: the 26B relabels the 31B's pairs at hunger 1-3 with food in view (4,021 of 13,500; new `relabel.py`, ls 4701, both cards); rule posted | | issue #1 comment 6779643960 |
| 15:26 | Baselines: eat_by_hunger (food near, h0-h8): stock 0.01 0.23 0.05 0.14 0.33 0.50 0.72 0.98 0.99; candidate 2 0.12 0.77 0.46 0.62 0.77 0.84 0.90 0.97 0.98. Pendant states replayed, fed with food in view (182): stock 0.056, c1 0.258, c2 0.261 | | pocket-tank e888a5b |
| 15:29 | Relabel ~250/min both cards (slower while my probes shared card 1); chain started: counts, train relabel run, checks | | `out/dartcheck/chain-relabel.out` |
| 15:40 | Relabel done: 4,021 (0 failed, 1,757 changed goal), seek_food 17% (31B ~58%); kept 9,479 + 4,021 = 13,500 | dataset 28,948 pairs, f16bd193 | |
| 15:41-15:45 | Run relabel (seeds 1-3), training mode on, then off; checks | F14 0.27 (was 0.65), every panel row passes, dart and timid OK, panel 37 OK; **hunger 4/5 eating 0.62/0.74 vs candidate 2 0.77/0.84: the guard fails (rule not met)**; stock 0.33/0.50. Pendant replay fed eating 0.14 (c2 0.26, stock 0.06) | pocket-tank ba54349; issue #1 comment 6780102993 |
| ~15:50 | Daniel: "accept it and run the acceptance checks" (the hunger 4-5 drop accepted) | | |
| 15:52-16:03 | Acceptance on relabel seed 1 (the harness's pick) | agreement 75 / 74%; bubbles kept 0.59 / 0.51 (passes); follow cliff 0.55; **dart cliff 0.15** (seeds 2 / 3: 0.35 / 0.28; candidate 2 0.46); census 1 min landmark 30/43/31, longest 36/60/36; 10 min 58%, 87 s, bored 3.5. Misses 3 bars (c2 4, stock 6) | |
| 16:07 | Candidate 3 committed and pushed (pocket-tank 25044c4: `model/candidates/relabel-d256n6/`), posted | | issue #1 comment 6780585435 |
