# strix-llama.cpp vs `big` on marvin: plan and timeline

Daniel 2026-11-19: "write the plan and open an issue but skip gpt-oss". Issue: #66. Times are Chicago time, from `date`.
Nothing changed on any machine yet.

## What it is (read 2026-11-19 from https://github.com/halo-box/strix-llama.cpp)
- A community fork of llama.cpp (MIT) for AMD Strix Halo (`gfx1151`, marvin's chip). Everything not listed in its
  README is upstream llama.cpp; it last merged upstream up to 9048bad8d (2026-11-17, its PR #293).
- Very active: a build most days (release b1064 on 2026-11-19), 1,077 commits ahead of upstream, 59 behind.
  AI agents may open pull requests there (its AGENTS.md); its PRs are expected to carry before/after numbers.
- Much of its recent work is for `qwen4exp`, the architecture of `big` (Qwen3.8-Flash-Next): fused Vulkan kernels,
  MTP draft-vocabulary subset, faster prompt reading with an MTP draft loaded (PR #255, still open).
- Other additions that matter here: Vulkan batched mat-vec split into column counts that scale on RDNA 3.5 (it says
  3, 5 and 6 columns are several times slower than 1, 2 and 4, which hits speculative-decoding checks);
  `--spec-draft-adaptive` (draft length from measured acceptance); freed KV cells zeroed ("repeatable output at depth").
- Its own numbers (llama-bench, `-ub 2048`/`8192`, Flash-Next UD-Q4_K_XL, PR #269): reading 1,782 tok/s at 2K,
  1,465 at 32K depth; writing 28.5 without a draft; with MTP 36-38 (PR #255). Its "before" is its own older build,
  not upstream. `big` today (MLflow 2026-11-17, through the server, default `-ub`): reading 330 at 32K, writing ~44.
  The methods differ, so the gap is a question, not a result. Writing shows no sign of a gain.
- Open issue #157 there: "Numeric regression on qwen4exp" (2026-10-08); the maintainers asked for a retest after a
  correctness cleanup. Our KL step checks this.
- Our policy (CLAUDE.md): no long-lived fork as the main engine. This is a test model through the engine launcher,
  like `big-halogen`; daily use is Daniel's decision after the results.

## Pins
- Image `ghcr.io/halo-box/strix-llama.cpp:server-vulkan-b13570@sha256:89304dcfc1fe5c3bec8c7026ad161a8e368a085d287421aae3d8c06f6fae0d61`
  (built 2026-11-19 13:28 UTC, revision b9f3c0e878f82f80ea1528969f305017a909b73e; newest server-vulkan tag then).
  Ubuntu 26.04 base like ours, but its Mesa is not pinned: step 2 records which one it has (ours: 26.0.8).
- Model: `big`'s own files (`Qwen3.8-Flash-Next-UD-IQ4_XS`, `mtp-Qwen3.8-Flash-Next-Q8_0.gguf`), no download.

## Out of scope
- gpt-oss (Daniel 2026-11-19: skip it).
- zaphod: the fork's changes are for Strix Halo; zaphod's R9700s get upstream behaviour.
- ROCm/HIP builds (the fork notes a gfx1151 batched-output bug that needs `HIP_LAUNCH_BLOCKING=1`).

## Steps (one change at a time; each needs marvin idle and Daniel's go)
1. **#45 first** (official build, `big` `-ub 512 → 2048`). The fork's reading numbers use big batches; this tells how
   much of the gap is the setting and how much is the fork. `big-strix` then uses whatever `big` uses after #45.
   Good result: as in #45 (32K reading +10% or more, writing within ±3%, tool use within the seed spread).
2. **Image check.** Pull by digest on marvin; record `llama-server --version`, the Mesa version, and
   `llama-server --list-devices` (the GPU must show ~120 GB). Check the image has `llama-perplexity` (needed in step 5;
   if not, use the matching `full-vulkan-b13570` image for step 5 only).
3. **Entry.** `big-strix` in `engines/marvin.json` (image by digest, `/dev/dri`, groups video + render, `/srv/models`
   read-only, read-only root) and in marvin's llama-swap config: the same command as `big`, privacy switches
   (`--cache-ram 0 --no-cache-idle-slots`) included, swaps with `big`, pinned off in the training profile. No fork
   flags yet. Smoke: loads, one answer, `/health`, memory at load vs `big`.
4. **Speed.** `lab speed marvin/big-strix` and `lab speed marvin/big` the same day, 3 runs each.
   Good result: 32K reading clearly above `big`'s, writing not below `big`'s by more than 3%.
5. **KL against the official build** (same model file, same text, the official build's output as the reference;
   `lab kl` is zaphod-only today, so the commands go into this file). Good result: mean KL under 0.01, same top token
   97% or more. Stop if not (that would be the #157 regression).
6. **Tool use.** `lab quality marvin/big-strix --full --temperature 0.6`, seeds 1-3. Before (`big`, 2026-11-16):
   96 / 91 / 93. Good result: within that spread.
7. **Leaks.** `tuning/cache-evict.py big-strix both` and `tuning/cache-switch.py`. Good result: leaks 0. Stop if not.
8. **Fork flags, one at a time,** only if 4-7 pass: `--spec-draft-adaptive`, then anything else the README lists for
   this model. Speed before and after each; tool use again if writing changes.
9. **Decision (Daniel).** Terminal-Bench only if he wants it. MLflow runs get the engine tags (`engine`,
   image digest), since `lab manifest` describes the official llama-swap.

## Removing it
Take `big-strix` out of marvin's llama-swap config first, then out of `engines/marvin.json` (CLAUDE.md: both at once
left an engine running on 2026-11-17).

## Timeline
| When | What | Result |
| --- | --- | --- |
| 2026-11-19 | Repo read, plan written | Nothing run yet |
| 11:27 | Step 1 (#45): marvin idle (both slots free, no requests for 30 min); zaphod normal set | — |
| 11:27-11:32 | Before: `lab speed marvin/big` (`-ub 512`, default) | 43.3 writing / 329.4 reading 32K (first token 98.7 s); runs/2026-11-19-1127-speed-marvin-big.json, MLflow 25df9b86 |
| 11:33 | `big`: `-ub 2048` (commit 487c287), pulled on marvin, reloaded; ready 11:33:41 | Running with `-ub 2048`; marvin's available memory 16 → 11 GB |
| 11:34-11:38 | After: `lab speed marvin/big` | **45.4 writing / 477.0 reading 32K** (first token 68.1 s; the 3 long runs 505 / 439 / 477): reading **+44.8%**, writing +4.8% (normal spread: 42.6-45.6 on 11-16/17). runs/2026-11-19-1134-speed-marvin-big.json, MLflow 80a14c0f |
| 11:38-12:45 | Tool use: `lab quality marvin/big --full --temperature 0.6` seeds 1-3, series `big-ub2048-v1` | **92 / 89 / 91** (127, 123, 125 of 138; MLflow 2209d983, 403a5920, 9d913ff0) vs `-ub 512` on 2026-11-16 96 / 91 / 93 (series `big-ls273-v1`): each seed 2-4 lower, mean 90.7 vs 93.3; seed 2 below the old spread (91-96). Losses mostly in M (autonomous planning: 67 / 33 / 67 vs 83 / 100 / 67) and A (tool selection: 67 / 100 / 67 vs 100 ×3), both few-scenario categories; median turn 3.7 → 3.9 s. Not clearly within the spread: Daniel chose more seeds. `-ub 2048` still live |
| 12:48-13:59 | Seeds 4-6, same setup and series | **93 / 93 / 91** (128, 128, 125 of 138; MLflow 5a2bc59f, 27bf1817, e16cb2c5): all inside the old 91-96 spread, the bar set before the run. Six seeds 92 / 89 / 91 / 93 / 93 / 91, mean 91.5 (`-ub 512`, 3 seeds: 93.3; 269 series 93 / 95 / 91). Kept `-ub 2048`: reading +44.8%, tool use within the spread. `big-strix` (step 3 on) will use `-ub 2048` |
| 15:58 | Step 2: image pulled on marvin by digest (327 MB) | `llama-server --version`: 1.0.0-dev build 13570, commit b9f3c0e87. Mesa 26.0.8-1ubuntu0.3 and libvulkan1 1.4.341 = the same as `big`'s image. `--list-devices`: Vulkan0 Radeon 8060S, 123,392 MiB. Every flag `big` uses is accepted; the image has `llama-perplexity` built in (`/app/llama`, perplexity library) |
| 16:01 | Step 3: `big-strix` in engines/marvin.json + llama-swap (commit 70df16c); marvin idle; pulled, reloaded (big ready 16:02:30) | `lab-engined.py check` prints the expected `docker run` (user 65534, read-only root, /tmp tmpfs, groups video + render, models read-only, internal network) |
| 16:02 | Smoke: "17 x 23" to `big-strix` | Loaded and answered in 29.6 s: "391". Draft acceptance 0.88 on that short answer. Container healthy. marvin available memory 15 GB (`big` with `-ub 2048`: 11). Mesa's shader cache is off (read-only root, no home): costs only startup time |
| 16:03-16:05 | Step 4: `lab speed big-strix` (direct to marvin, MACHINE=marvin) | **50.8 writing / 971.5 reading 32K** (first token 33.4 s; the 3 long runs 972 / 974 / 968, very steady). runs/2026-11-19-1603-speed-big-strix.json, MLflow d892f651 (engine tags corrected: 6 × 200) |
| 16:05-16:10 | Same-session `big`: `lab speed marvin/big` (official 273, `-ub 2048`) | 45.5 / 461.4 (first token 70.4 s; first short run 10.9 included the swap back). MLflow ea969004. Fork vs `big`: writing **+11.6%**, 32K reading **2.1x** (vs `-ub 512` this morning: 2.95x) |
| 16:11 | Step 5 setup: the official image has no `llama-perplexity`; found `ghcr.io/ggml-org/llama.cpp:full-vulkan-b12407@sha256:a851d61fc17fa7dbec322d54e70723f5627f5af41aafb06160f81b8f849d627c` = release b12407 = commit e574a17 (big's exact llama.cpp), Mesa 26.0.8-1ubuntu0.3 (same) | Pulled on marvin |
| 16:12-16:15 | Step 5: llama-swap stopped (marvin idle), script `kl66.sh` (lab kl's text `tuning/kld/code-sample.txt`, checksum OK, `-c 512 --chunks 32 -b/-ub 512 -fa on`; big's IQ4_XS file). Reference = official b12407 (`/srv/models/kld/big-iq4xs-official-b12407.c512x32.kld`, 4.1 GB, PPL 1.7845). Control = official vs its own reference. Test = fork (`llama perplexity`). llama-swap started again 16:15:38, big ready 16:16:09 | Control: KL 0.000000, same top 100% (the reference is repeatable). **Fork: mean KL 0.0212 ± 0.0010, same top 96.8%**, 99th percentile 0.32, max 4.3; PPL 1.7770 (official 1.7845; ratio 0.997 ± 0.003: not worse). **Fails the bar** (KL < 0.01, same top ≥ 97%). For scale: `coder`'s whole Q4_K_XL quantization vs Q8_0 is KL 0.0088 / 97.7%. Outputs: tuning/kld/runs/2026-11-19-big-*.txt. Stopped here per the plan's stop rule: Daniel decides |
| 16:18 | Daniel: run tool use anyway (KL shows the outputs differ, not that they are worse; PPL is not worse) | 6 seeds, to compare with `big`'s 6 (`-ub 2048`) |
| 16:19-18:24 | Step 6: `lab quality big-strix --full --temperature 0.6` seeds 1-6 (direct to marvin), series `big-strix-b13570-v1` | **91 / 93 / 91 / 93 / 93 / 94**, mean **92.5** (`big` 92 / 89 / 91 / 93 / 93 / 91, mean 91.5): within `big`'s spread, slightly higher. Seeds 2, 4, 6 scored of 136 (the test left one 2-point scenario out). A (tool selection) 100 in all 6 (`big`: 67 ×3); M 67-100 (`big` 33-100). Behaviour checks the same order (looping 0-1, attacker echo 1-2). Live speed in the test: writing 46.0 vs 43.3 (+6%), reading 282 vs 220 (+28%; short prompts, so far below the 32K gain); median turn 3.6 vs 3.9 s. MLflow 992a3839, b2f1c73e, 2b5f34a8, 4e4b1969, 98b24559, 094b13c3 (engine tags corrected, 36 × 200). Passes the bar |
| 18:59-19:01 | Step 7a: `tuning/cache-switch.py big-strix` (direct to marvin), 3 rounds | **0 leaks**, own codeword 9/9. First token A 12.3 s, B 14.0 s, A again 0.48 s (read 41 tokens: A was still in its own slot, the other slot took B; 2 slots, as `big`). runs/2026-11-19-1901-cache-switch-big-strix.json |
| 19:01-19:17 | Step 7b: `tuning/cache-evict.py big-strix both` | **0 leaks**, own codeword 22/22. many: 8 × 12K, first token 11.1-12.4 s, follow-ups 11.8-13.1 s (read again: privacy switches working). deep: 5 × 120K (600K > the 256K pool), first token 111-133 s (~950 tok/s at 120K; `big` at 123K: 267 with `-ub 512`), follow-up in the first 127.0 s (read again). runs/2026-11-19-1916-cache-evict-big-strix.json. Passes |
| 21:15 | Daniel: step 8 (fork flags, one at a time). The fork's README lists one flag for this setup: `--spec-draft-adaptive` (draft length from measured acceptance). Test edits copied to marvin's engines/marvin.json only (not committed), engine unloaded between runs; marvin idle (both slots free) | Baseline = step 4: 50.3 / 50.8 / 50.8 writing, 971.5 reading 32K (runs/2026-11-19-1603) |
| 21:18-21:20 | 8a: + `--spec-draft-adaptive` (n-max 3) | **48.8 writing** (runs 49.2 / 48.8; first short run 11.0 is the load), 974.4 reading: writing **-3.5%** (both sets steady to ±0.5, so likely real), reading same. Acceptance 0.82, mean length 3.35. runs/2026-11-19-2118-speed-big-strix.json, MLflow c4b346f1 |
| 21:21-21:24 | 8b: adaptive + n-max 5 (adaptive is meant to make a longer cap cheap) | **44.0 writing** (-13%), 973.0 reading. Acceptance 0.68, mean length 3.66. runs/2026-11-19-2121-speed-big-strix.json, MLflow 685154fa. Engine tags corrected on both (14 × 200) |
| 21:25 | Back to the tested setup (n-max 3, no adaptive: the committed engines/marvin.json, marvin's copy clean again), engine restarted, smoke "17 x 23" → 391 | Step 8 done: **adaptive off** (slower in both). Tool use not re-run (setup unchanged). Note: the speed test's short prompt is code at temp 0 (high acceptance); adaptive might differ in agent use, untested. Next: step 9, Daniel's decision |
