# Plan: Strata on marvin (Strix Halo), then port UnbiasedStrata to AMD and offer the PR

Status: plan, 2026-10-29. Nothing done yet. The llama.cpp baseline is complete (Terminal-Bench 19/19 on 2026-10-29); waiting on the decisions at the bottom.

## Why

marvin's weak spot is reading prompts: llama.cpp reads 463 tok/s in a short test and 230-280 tok/s in real agent use
(Flash-Next UD-IQ4_XS), so each agent step takes ~13 s (Ornith on zaphod: 5 s). Upstream Strata reports, on a
Ryzen AI Max+ 395 / 128 GB / Ubuntu 26.04 / kernel 7.0 (practically marvin), with the **same model file**:
~1,160-1,320 tok/s reading and ~51-54 tok/s writing (UD-IQ4_XS, 8K-128K). Their figures, one machine, marked
"experimental/unvalidated" for gfx1151. If they hold on marvin: 3-5x faster reading.

Daniel prefers **UnbiasedStrata** (github.com/robertkarl/UnbiasedStrata): a clean fork of PolyStrata v0.1.45-poly.1,
"no installer, no downloads at build or start, no Python between a request and a token", serves through llama.cpp's
server code, MIT. Upstream Strata has an installer, Python front end, and fetches files on its own (seen on zaphod:
~6.5 GB of MTP pieces downloaded from the original Qwen model by byte range at first start, unannounced).
UnbiasedStrata today: **NVIDIA only** (README: Pascal+, CUDA 12; `make build` is CUDA). It kept upstream's engines,
including the AMD (HIP) code, and upstream's docs/AMD_HIP.md + docs/STRIX_HALO.md (inherited, not wired into its build).

So: Phase 1 measures upstream Strata on marvin **sealed off from the network** (does the gain exist here?), Phase 3
ports UnbiasedStrata's clean build to AMD gfx1151 and measures it the same way, Phase 4 offers the port upstream.

## Baseline to compare against (llama.cpp on marvin, all recorded)

| Test | Result | Where |
| --- | --- | --- |
| Speed, no MTP (llama-bench pp512/tg128) | 463 / 24.9 tok/s | tuning/results.md "marvin" |
| Speed, MTP draft 3 (3 code requests) | 39.6 tok/s writing | same |
| Real-use speed during tool-eval | ~36 writing, ~246 reading, 72% MTP accepted | MLflow live_* |
| Tool use, full suite x3 (series marvin-flash-next-iq4xs-mtp3) | 90 / 93 / 92 (common score), 0 made up | `./series marvin` |
| Terminal-Bench Core-19 full | 19/19 (18 first attempt), 5.5 h, real use 33 writing / 254 reading tok/s | MLflow c145733e, tasks.json |
| Memory | 93.7 GB model + MTP 4.1 GB; ~11 GB RAM left during the benchmark | marvin/SETUP.md |

Add before Phase 1 (one change each, with llama.cpp, so both engines get the same tests):
1. **Long-prompt reading test**: one 32K-token prompt, 3 runs, median reading tok/s. Today's numbers are short prompts.

## Phase 1: upstream Strata on marvin, sealed (about 1 day, mostly waiting)

Rules: only marvin changes; zaphod untouched. marvin's llama-swap is stopped during the test (both need ~95 GB) and
always restarted at the end (script `trap`, like plans/strata-test/). `marvin/big` is offline meanwhile.

1. **Image**: `ghcr.io/niko1221/strata:<version>-rocm-gfx1151`, pinned by digest. Exception to the "2 weeks old" rule,
   for this test only (Strata releases near daily; gfx1151 fixes are recent). Recorded in the results.
2. **No network for Strata**: its container sits on an internal Docker network (no route out). A tiny relay container
   (on the internal network + marvin's Tailscale IP, inbound only) publishes its API for zaphod's tests. Good result:
   `docker exec strata curl -m 5 https://huggingface.co` fails; zaphod reaches the API.
3. **Pre-load what it would download**: copy zaphod's already-built MTP folder (`strata-data:/Strata-data/mtp`, every
   piece has a sha256 in mtp-manifest.json; NOT the api-key/state folders) into marvin's Strata data volume.
   Check the checksums on marvin. If Strata still insists on downloading: stop and note it (that's a finding).
4. **Same model file as llama.cpp**: `/srv/models/Qwen3.8-Flash-Next-UD-IQ4_XS-*.gguf`, read-only. Strata builds its
   "pack" from it locally on first start (record how long and how much disk).
5. **Settings**: same context as llama.cpp's current `big` (128K) so results compare; API key generated per run.
6. **Tests**, same as the baseline, logged to MLflow with machine marvin, model strata-flash-next:
   1. Speed: 3 short requests (same prompt as before) + the 32K long-prompt test. Record free memory before start
      (Strata's output can depend on free GPU memory at start).
   2. Tool use: full suite x3 (temp 0 seed 42, temp 1.0 seeds 1 and 2) as `SERIES=strata-strix-iq4xs`.
      Note: Strata rejects the JSON-schema format; `tooleval_score_common` already compares fairly.
   3. Terminal-Bench full (19 tasks, ~5-10 h, in tmux) against Strata's endpoint directly: a new
      `TARGET=marvin-strata` in `tuning/tbench` (endpoint, platform label strixhalo-strata, context length).
   4. Determinism: same prompt at temp 0 after 3 restarts: identical answers? (upstream says ~1 in 10 differ)
7. Restore: Strata stopped, marvin llama-swap back; `marvin/big` answers through zaphod.

## Phase 2: decision (Daniel)

Go on to the port if Strata on marvin clearly wins, e.g. **reading at least 2x llama.cpp** in the 32K test and real use,
with tool-use and Terminal-Bench at least as good. If not: write down why and stop; the llama.cpp setup stays.

## Phase 3: port UnbiasedStrata to AMD gfx1151 (now likely hours: others did the HIP build part)

Existing work (checked 2026-10-29, both open, no maintainer reply yet):
- **PR #19** (firetussin, 2026-10-21, commit 115af49): `make build BACKEND=hip` (Makefile + README only), CUDA stays
  default. Tested on RX 6700 XT (gfx1031): full HIP build, `make test` passes, chat/streaming/tool calls work.
  https://github.com/robertkarl/UnbiasedStrata/pull/19
- **Issue #6** (axelgenus, 2026-10-14): built on RX 6900 XT (gfx1030) with ROCm 7.0 in Docker using 3 patches (stub
  files for missing HIP test/tool targets, `unbiased-strata` target enabled for HIP, a `__shared__ alignas(16)` fix
  clang-HIP rejects in src/kernels/cuda/iq_kernels.cu). Runs Flash-Next daily.
  https://github.com/robertkarl/UnbiasedStrata/issues/6
So: start from PR #19's commit (+ issue #6 patches only if our build needs them).
**Set up 2026-10-29:** fork github.com/dopamine911/UnbiasedStrata, clone at `~/src/UnbiasedStrata` on zaphod (remote
`upstream` = robertkarl), branch `hip-gfx1151` = PR #19 commit 115af49 (pushed to the fork). Upstream main is 8 commits
ahead of PR #19's base (2049dd2) and merging conflicts (Makefile, README; upstream also changed CMakeLists.txt by
+93 lines): first a working gfx1151 build on 115af49, then rebase onto upstream main with the build as reference. Our part is the gfx1151-specific
work: architecture flag, hipBLASLt tuning table for ROCm 7.14.1, unified-memory behavior, container recipe, tests.


Work in Daniel's fork (github.com/dopamine911/UnbiasedStrata), branch `hip-gfx1151` based on PR #19's commit;
one change per commit.
1. **Read first**: how `make build` drives CMake for CUDA; upstream's HIP CMake (`cmake/hip_backend.cmake`, flags
   `STRATA_ENABLE_HIP=ON`, `CMAKE_HIP_ARCHITECTURES=gfx1151`, `STRATA_PREFILL_MMQ=ON`) and the hipBLASLt tuning table
   (`tools/hip/gfx1151-hipblaslt-100401.txt`, needs ROCm 7.14.1 = hipBLASLt 100401).
2. **Build in a container, never on marvin's host**: Dockerfile with AMD TheRock ROCm 7.14.1 for gfx1151 (1.7 GB
   tarball, pinned URL + sha256), pinned CMake/Ninja, the fork at a commit, `make build BACKEND=hip` (new option).
   Output: an image with only the `unbiased-strata` binary and runtime libraries. Runs with no network.
3. **Keep the fork's rules**: no downloads at build or start (the MTP pieces come from a local, checksummed path),
   no Python in the request path, CUDA build unchanged (CI for CUDA must still pass).
4. **Tests**: the fork's `make test` (fake engine, no GPU); upstream's HIP parity tests
   (`hip_prefill_hcd_exact_parity`, `hip_prefill_hipblaslt_gemm`); then the full Phase 1 test list (speed, tool use x3,
   Terminal-Bench, determinism) as `SERIES=unbiased-strix-iq4xs`, so all three engines compare on one table.
5. **Run it like llama.cpp**: if it wins, a `big-strata` entry in marvin's llama-swap (llama-swap starts the binary;
   one engine loaded at a time), so Open WebUI, agents and zaphod see it like any other model.

## Phase 4: contribute upstream (results first, PR only for what is new)

1. Don't open a competing AMD PR. Post the Strix Halo (gfx1151) results as a comment on PR #19 and issue #6: first
   validation on a third AMD family, with the measurements table. That supports getting #19 merged.
2. Only gfx1151-specific fixes (tuning table, unified-memory notes, any build fix #19 lacks) go in a small follow-up PR
   on top of #19, after asking in issue #6 whether the maintainer wants it.
3. Expect a wait: neither #6 nor #19 had a maintainer reply after 1-2 weeks. marvin runs our own pinned build anyway.
4. Content of results/PR: the container build recipe, measurements (Phase 1 vs Phase 3 vs llama.cpp, one
   table, machine details from `tuning/manifest.sh`), and what was NOT validated (Windows, other AMD chips).
5. Small, reviewable commits; CUDA path untouched; no secrets or machine names beyond hardware in the PR.

## Risks

- Strix Halo support upstream is "experimental"; the speed claims may not hold on marvin.
- Memory: ~95 GB per engine; never two at once. Phase 1 leaves `marvin/big` offline for the day.
- Strata's offline start may fail without its downloads (step 3 handles it, or we learn it).
- The port may hit ROCm/hipBLASLt version issues (the tuning table is tied to ROCm 7.14.1).
- `amd_iommu=off` (used by upstream's measurements, reported 5-12% faster) is a separate marvin test, not part of this.

## Decisions for Daniel

1. Decided 2026-10-29: yes, `0.2.7-rocm-gfx1151` (built 2026-10-27; same Strata version as the zaphod test), test only,
   pinned by digest `sha256:29ecd5cf8b0e…`.
2. Decided 2026-10-29: yes. zaphod's MTP folder copied into marvin's `strata-data` volume (no lock file, no keys or
   state); 31 of 31 pieces match mtp-manifest.json, built draft files identical on both machines.
3. Phase 2 bar: is "reading at least 2x, quality same" the right threshold? (open)

## Phase 1 log

Script: `marvin/strata-test/strata-test.sh` (plan / start / status / stop; runs on marvin, copy at ~/strata-test/).
1. Dry run (`plan`): model family is `unsloth` (not `qwen`: Strata 0.2.7's qwen family only has the ISTA files).
   Strata recognises marvin's 3 UD-IQ4_XS shards (94 GB), "Nothing to download", drafting MTP, 128K, 2 parallel.
2. Start (2026-10-29 22:38): ready in 3 min (pack built locally), internet from inside Strata blocked. Memory: 65 GB GPU,
   106 GB RAM used. Setup chose the RAM-budget mode for Strix Halo (`--expert-cache 16806 --resident-budget-gib 55`,
   on purpose: Strata issue #1715, an out-of-memory kill on a 128 GB box); the engine still holds 98% of experts on the GPU.
3. Speed (`tuning/speed`, MLflow 9930e51a): writing **50.9** tok/s (upstream 50-54: same), 32K reading **599** tok/s
   (upstream ~1,200-1,300: half). Strata's own timer agrees with ours (604.6 vs 603.2).
4. Why reading is half (one change per experiment, 32K reading test, files in tuning/runs + scratch):
   | Exp | Change | 32K reading tok/s |
   | --- | --- | ---: |
   | E0 | none (fresh restart) | 590 |
   | E1 | `prefill=16384` (upstream's chunk) | **639 (+8%)** |
   | E2 | parallel 1 | 601 |
   | E3 | `kv-resident=off` (all context memory on the GPU) | 606 |
   | E4 | `resident-budget-gib=off` | does not start: out of memory allocating 47.5 GiB of expert cache |
   **Main cause found:** during a 32K read the GPU sits at **54-55 W** for the whole read, clock ~2,030 MHz (max
   2,900), 64 C max: a firmware power limit (marvin's BIOS power mode), not heat. Reading is compute-bound, so it
   scales with power; writing is memory-bound, so it matches upstream. llama.cpp's reading on marvin is capped the same way.
   Next, one change at a time on the same test: (1) BIOS power mode (Daniel), (2) `tuned` accelerator-performance
   (setup/host.sh step 8), (3) `amd_iommu=off` (+5-12% elsewhere). Tool-use x3 and Terminal-Bench wait until then.
