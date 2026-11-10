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

