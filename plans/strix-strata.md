# Plan: Strata on marvin (Strix Halo), then port UnbiasedStrata to AMD and offer the PR

Status 2026-10-30: Phase 1 measured (Strata reads 2.6x llama.cpp on marvin; Terminal-Bench on Strata running overnight).
Phase 3 steps 1-5 done on zaphod (gfx1151 image built, see Phase 3 log); step 6 (marvin) waits for Daniel: see
`decisions.md` "Waiting on Daniel".

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

Instructions for the agent doing Phase 3: `plans/strix-strata-port-instructions.md`.

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

## Later: llama.cpp ROCm build as a third contender
strixhalo.wiki/AI/llamacpp-performance (Qwen3-30B-A3B, not our model): Vulkan RADV reads 755 tok/s at depth 0 but
17 at 128K depth; ROCm 41, lhl's rocm-wmma-tune branch 51. Agents work at long depth, so after the Strata tests:
build llama.cpp for HIP/gfx1151 in a container (same pinned tag as Vulkan), run `tuning/speed` + a depth sweep on
`big`, one change at a time. marvin `big` already has the wiki's flags (-ngl 999, -fa on, -lm dio, RADV).
Recipe (strixhalo.wiki/AI/llamacpp-with-ROCm, dated Aug 2025, ROCm 7.0.x era): `-DGGML_HIP=ON -DAMDGPU_TARGETS=gfx1151
-DCMAKE_BUILD_TYPE=Release`; do NOT use `-DGGML_HIP_ROCWMMA_FATTN=ON` (slower at depth on ROCm 7.0.2+, unmaintained);
runtime `ROCBLAS_USE_HIPBLASLT=1` (measure); use ROCm 7.14.1 like Strata. Known: GPU hangs (ROCm#5151), workaround
`amdgpu.cwsr_enable=0` (strixhaloguide). kyuz0's toolboxes and lhl's rocm-wmma-tune branch: reference only.

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
5. BIOS power mode Quiet → **Performance** (Daniel, 2026-10-29 23:28; marvin shipped on Quiet). Same 32K test, power sampled:
   | Engine | Quiet (GPU 54-55 W) | Performance (GPU 130-139 W, 2,850 MHz) |
   | --- | ---: | ---: |
   | Strata reading / writing | 599 / 50.9 | **758 / 55.7** (MLflow c5b6b5da) |
   | llama.cpp reading / writing | not measured | **315 / 44.0** (MLflow 0e1b8a5d) |
   Strata reads **2.4x** llama.cpp on the same test and power. Upstream's ~1,250 is still higher; their prompt length
   per row is not stated, so not directly comparable. Left to try, one at a time: prompt chunk 16384 (+8% in Quiet),
   `tuned`, `amd_iommu=off`. Heat: GPU peaked at 93-97 C in Performance (64 C in Quiet); limit is ~100 C, watch it.
6. `tuned` accelerator-performance (setup/host.sh step 8): Strata 32K reading 758 → 757, writing 55.7 → 55.1: no change (MLflow d7875016). Kept, harmless.
7. Prompt chunk 16384 (`STRATA_SET=prefill=16384`, now the script's default): 32K reading 757 → **817** (+8%), writing 55.5 (MLflow f807b742). **Settings for the quality tests:** Performance mode, tuned, Strata defaults + prefill 16384, preset `p16`. Strata reads 2.6x llama.cpp (315) on the same test.
8. Tool use, full suite x3 (series `strata-strix-iq4xs`, preset p16, MLflow d90fbf73 / de97f22f / 6d009a3f):
   common score **89 / 89 / 90 (avg 89.3)** vs llama.cpp on marvin 90 / 93 / 92 (avg 91.7). Strata's best equals
   llama.cpp's worst: about 2 points lower, small but consistent over 3 runs. Attacker echo (TC-57/58/60: repeats
   the attacker's address or key) avg 2.7 vs 1.0; looping 0, made-up query 0. Not yet hand-sorted.
9. Terminal-Bench full on Strata (started 2026-10-30 02:01, preset p16, job `20261030-020139-…-f19a97ac`).
   At 07:00, 5 h in: **4/4 passed, but only 4 of 19 done** (llama.cpp did all 19 in 5.5 h). Per task:
   | Task | llama.cpp min / steps / tokens written | Strata min / steps / tokens written |
   | --- | --- | --- |
   | nginx-request-logging | 3.4 / 4 / 2.1K | 4.9 / 9 / 4.8K |
   | sparql-university | 8.9 / 9 / 9.7K | 17.5 / 16 / 29K |
   | headless-terminal | 7.8 / 10 / 8.1K | 125.8 / 69 / 225K |
   | fix-ocaml-gc | 33.0 / 40 / 12.8K | 45.3 / 38 / 41K |
   | mailman | 27.7 / 27 / 36K | 100+ / 72+ / 176K+ (running) |
   Cause: not looping; long thinking (30-60K characters per step, then a short answer). **The two engines think at
   different levels:** llama.cpp's `big` runs `--reasoning-effort medium` (marvin config `qwen-sampling`); Strata
   sends no level, so its chat template uses its default **xhigh** (`serve/chat_template.jinja`:
   `reasoning_effort|default('xhigh')`). Sampling is otherwise the same (Strata's model-card set for thinking:
   temp 1.0, top-p 0.95, top-k 20). So this run, and the tool-use x3 in item 8, compare xhigh against medium.
   Strata can be set to medium: per request (`reasoning_effort` or `chat_template_kwargs`), or for every request with
   `"reasoning_effort"` in its run config (`[strata] thinking level: …` in the log confirms). Fair comparison = rerun
   tool-use x3 and Terminal-Bench at medium. Daniel decides whether to let this xhigh run finish first.

## Phase 3 log

Fork `~/src/UnbiasedStrata`, branch `hip-gfx1151`. Code map and findings: `plans/strix-strata-port-notes.md`.
Decisions made while Daniel was away: `decisions.md`.
1. Code map (2026-10-30). Main finding: PR #19's `make build BACKEND=hip` leaves `STRATA_PREFILL_MMQ` off (the prompt
   experts' MMQ kernels; STRIX_HALO.md turns it on). `make test` builds only the front end: it does not test HIP.
2. Toolchain image (fork commit `ed764de`, `docker/hip-gfx1151.Dockerfile --target toolchain`): base
   `ubuntu:resolute-20260927` by digest, Ubuntu snapshot 2026-10-10, ROCm 7.14.1 tarball by sha256. Built in 200 s,
   11.4 GB. Checked: ROCm 7.14.1, AMD clang 23.0.0git, HIP 7.14, hipBLASLt 1.4.1 (= 100401, matches the tuning table).
3. Build (fork `63adfc3`, `--target build`, network off): `make build BACKEND=hip HIP_ARCHS=gfx1151 PORTABLE=1` with
   STRIX_HALO.md's cmake flags incl. `STRATA_PREFILL_MMQ=ON` via `CMAKE_EXTRA`. **No source patch needed** (none of
   issue #6's patches). 4.5 min with 16 jobs. GPU code: gfx1151 only (strata 128 kernel images, strata-poly 48),
   byte-identical between two builds of the same tree. CUDA path: untouched (fork diff since `115af49` is `docker/` only).
4. Tests without a GPU: `make test` passes (engine_test 7 cases, literals, picture cache, e2e HTTP checks); it tests
   the front end only. Also `arch_defaults_test` (built by hand, no GPU): OK, the gfx1151 table has 19 switches.
5. Runtime image (fork `0ac55a1`, default target): `unbiased-strata:gfx1151`, 1.09 GB, config digest
   `sha256:8869cd2ba45592f85aef562d3875e7fa11224d9864d27050ae9ab13a7cbfd195`, manifest digest
   `sha256:3d785883812b2b4cc5c08d5b840a0bb622868cc435ef89517a717c9bed9f5f5c` (both stable with `--provenance=false`).
   zaphod's Docker uses the containerd image store, so `docker images` shows the **manifest** digest as the ID;
   a Docker with the classic store shows the **config** digest. Compare like with like.
   Holds the 3 programs, the 100401 tuning table (`STRATA_HIPBLASLT_TUNING` set), 619 MB of ROCm libraries, no
   compiler. Starts with `--network none`. Not yet tested on a GPU (zaphod has no gfx1151; its cards are in use).
   **Next: step 6 on marvin, waits for Daniel** (Terminal-Bench must be finished). Open before step 6: the pack
   (reuse upstream 0.2.7's from `strata-data`, or build one with the fork's `tools/iq_pack.py --compat-bf16`).
5b. Tests image (fork `4149027`, `--target tests`): `unbiased-strata-tests:gfx1151`, 1.18 GB, config digest
   `sha256:8902459072a9bd260ce2671569545efbf90353c73f47dc2af851c40402d33485`, manifest digest
   `sha256:d427915b29584e38c0cfa21036d174fc93fe38d5b697ff493935cb5344bc2583` = the runtime image + `strata-device`,
   `hip_prefill_hcd_exact_parity`, `hip_prefill_hipblaslt_gemm`, `hip_prefill_mmq_parity` (gfx1151 code only).
   Default `docker build` still gives the runtime image, ID unchanged. **Order on marvin, after Daniel's OK** (each
   with `--network none --device /dev/kfd --device /dev/dri --group-add` render + video; GPU memory free and power
   mode checked first):
   1. Copy both images: `docker save unbiased-strata:gfx1151 unbiased-strata-tests:gfx1151 | ssh marvin docker load`;
      check on marvin: `docker image inspect <image> --format '{{.Id}} {{json .Descriptor}}'`. Good = the config
      digest above appears (as `.Id` on the classic store, or as `config.digest` on the containerd store).
   2. `strata-device --list-devices` (tests image default): good = one gfx1151 GPU, "can run".
   3. The three parity tests: good = pass (exit 0), not skip (77). The hcd one needs the table (set in the image).
   4. Engine start with the Phase 1 settings + `--prefill 16384`: good = the start line lists 19 gfx1151 switches and
      `prefill gemm: hipBLASLt tuning enabled (… gfx1151, version 100401)`.
   5. **If the slim image fails at start** (a library or kernel file loaded at run time, which `ldd` cannot see): run
      the same command in `unbiased-strata-build:gfx1151` (full ROCm, 11+ GB). Works there = the slimming is at fault;
      fails there too = the engine or the port.
