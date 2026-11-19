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
