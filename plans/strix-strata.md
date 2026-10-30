# Plan: Strata on marvin (Strix Halo), then port UnbiasedStrata to AMD and offer the PR

Status 2026-10-30: Phase 1 measured (Strata reads 2.6x llama.cpp on marvin; Terminal-Bench on Strata running overnight).
Phase 3 steps 1-5 done on zaphod (gfx1151 image built, see Phase 3 log; pushed to dopamine911/UnbiasedStrata
`hip-gfx1151`); pack built (5e), step 6 script `marvin/strata-fork/fork.sh` and live-session draft ready (5c). **Step 6 check passed on marvin 2026-10-30** (Phase 3 log 6); speed/quality not yet measured. Step 6 (marvin): when marvin is free and not before 2026-11-04; pack: build with the fork's tool
(Daniel 2026-10-30, `decisions.md` "Daniel's answers"). Phase 4 (anything upstream): on hold until proven with
live sessions and plenty of stats.

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

**On hold (Daniel 2026-10-30):** no upstream submission of any kind (comment, issue, PR, contact with the PR #19
author) until the port is proven: fully tested with live sessions (real use through llama-swap, Open WebUI and
agents, not only benchmarks) and plenty of stats recorded (MLflow). Daniel decides when that bar is met.

1. Don't open a competing AMD PR. Post the Strix Halo (gfx1151) results as a comment on PR #19 and issue #6: first
   validation on a third AMD family, with the measurements table. That supports getting #19 merged.
2. Only gfx1151-specific fixes (tuning table, unified-memory notes, any build fix #19 lacks) go in a small follow-up PR
   on top of #19, after asking in issue #6 whether the maintainer wants it.
3. Expect a wait: neither #6 nor #19 had a maintainer reply after 1-2 weeks. marvin runs our own pinned build anyway.
4. Content of results/PR: the container build recipe, measurements (Phase 1 vs Phase 3 vs llama.cpp, one
   table, machine details from `tuning/manifest.sh`), and what was NOT validated (Windows, other AMD chips).
5. Small, reviewable commits; CUDA path untouched; no secrets or machine names beyond hardware in the PR.

## Later: fan control on marvin (researched 2026-10-30, nothing installed)

marvin facts that decide it: board Sixunited AXB35-02, fan chip (EC) ITE IT5570 firmware 1.08, BIOS 1.07,
**Secure Boot on, kernel lockdown = integrity**, kernel 7.0.0-42. Measured so far: GPU holds 2,800-2,890 MHz at
123-139 W while busy; peaks 93-97 C on 32K reads; no throttling seen yet. One user reports the factory curve reaches
80% fan only at 95 C and a steeper curve held 140 W at 79-80 C (Yamz-Labs/kyojin PR #95; single report, unverified).

| Option | How | Fits marvin? |
| --- | --- | --- |
| 1. BIOS fan setting | by hand | Works, nothing installed. But the one BIOS fan change (2026-10-30) coincided with the 55 W drop; re-check power after any change |
| 2. `ec_su_axb35` kernel module (github.com/cmetz/ec-su_axb35-linux, GPL-2.0) | kernel `ec_read`/`ec_write`; sysfs `/sys/class/ec_su_axb35/` (fan1-3 mode auto/fixed/curve, level 0-5, ramp curves, RPM; `apu/power_mode` read/write; temp) | **Best fit.** Works with Secure Boot if signed: repo has Ubuntu 26.04 DKMS + MOK scripts (`contrib/ubuntu/`). No releases: pin commit `f62c2c2` (2026-08-20). Costs: one MOK enrollment at the boot screen (Daniel at marvin), DKMS rebuild on kernel updates, no board check in the code. Settings not kept across boots: own systemd unit in `setup/` |
| 3. `strix-halo-fand` (github.com/nathanmarlor/strix-halo-fan-control, MIT) | root daemon writing EC registers through `ec_sys write_support=1` (debugfs) | **Very likely blocked** by lockdown (integrity mode blocks debugfs writes); would need Secure Boot off. No releases. Not recommended |
| 4. PTM7950 pad | physical | No software. Lowers temperature for any fan curve |
| 5. EC/BIOS firmware update (EC 1.06 "slightly quieter"; BIOS 1.09) | Windows flasher | Risk of bricking; wiki says don't flash a stable system. No |

Extra value of option 2 even without touching fans: `apu/power_mode` shows Quiet/Balanced/Performance from Linux,
so the "back to 55 W" problem (CLAUDE.md Known issues) can be checked directly instead of inferred from watts.
Note: setting power mode through the EC resets RyzenAdj limits (wiki); we don't use RyzenAdj.

Steps if Daniel chooses option 2, one at a time: (1) read-only first: build, sign, load; check `info.sh` RPM and
power mode match the BIOS; no writes. (2) Measure a long full load (32K reads back to back, 10 min) with factory
fans: clock, temp, watts. (3) Only if clocks drop (e.g. under ~2,600 MHz at 95 C+): a `curve` setting, same test
after. Sources: strixhalo.wiki Guides/Sixunited_AXB35/Power_Mode_and_Fan_Control, Hardware/Boards/Sixunited_AXB35/Firmware.

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
   **Final (Daniel: let it finish):** 19/19 (18 first attempt, extract-elf on the second), 13.3 h vs llama.cpp's
   5.5 h, MLflow db7b5ff4. Same score; the extra time is the xhigh thinking. Slowest: cobol-modernization 117 min
   (llama.cpp 17), headless-terminal 126 (8), mailman 106 (28), llm-inference-batching-scheduler 97 (77).
10. Thinking level medium (Daniel 2026-10-30: tool-use x3 first, cheaper; Terminal-Bench only if it passes).
   `strata-test.sh` default is now `STRATA_SET="prefill=16384 reasoning-effort=medium"`. Strata's launcher writes
   `--reasoning-effort medium` into the run config; the server takes it out of the engine arguments and uses it as the
   default level. Checked: `/data/state/strata/runtime/launch-p16m.server.log` says `thinking level: medium` (the
   engine log does not show it). tool-eval-bench sends no level of its own (only its warm-up turns thinking off).
   Preset `p16m`, series `strata-strix-iq4xs-medium`, same 3 runs as item 8. Decision rule: clearly below 91.7, or
   still more attacker echo than llama.cpp → stop Strata; else Terminal-Bench at medium (must be 19/19 in < 5.5 h).
   **Result (MLflow dd68dfba / d90f9a9f / 78c3f11b):** common score **93 / 96 / 93 (avg 94.0)** vs llama.cpp 91.7 and
   Strata at xhigh 89.3. Attacker echo **0 / 0 / 0** (llama.cpp 1.0, xhigh 2.7); looping 0, made-up query 0.
   16-19 min per run. Passes the rule: last night's lower score was the xhigh thinking, not the engine.
   Next: Terminal-Bench full at medium (profile `strata0.2.7-mtp-128k-np2-medium`).
   **Terminal-Bench at medium (MLflow 670c0c7a, 2026-10-30 11:16-15:40 CDT): 17/19 in 4.39 h.** Fails the rule
   (19/19 needed). Time: 4.39 h vs 13.3 h at xhigh; llama.cpp's 5.5 h was measured in BIOS Quiet mode, so it is not
   comparable (llama.cpp would be faster now). Failed both attempts:
   - `openssl-selfsigned-cert` (2.9 + 4.2 min): the agent's `/app/check_cert.py` imports `cryptography`, which the
     test's Python doesn't have (ModuleNotFoundError). Passed on llama.cpp and Strata xhigh.
   - `extract-elf` (27.9 + 35.0 min): output values differ from the reference. Also failed attempt 1 at xhigh (passed
     on retry there); passed first time on llama.cpp.
   Passed, first attempt: the other 17, incl. mteb-retrieve (16 min; llama.cpp needed a retry). Slowest: extract-elf,
   build-cython-ext 30, fix-ocaml-gc 26, break-filter-js-from-html 25. Tokens 2.35 M in, 0.28 M out (+ retries).
   Reading: 2 failures on 19 tasks is a small sample (one run each), so this does not prove Strata is worse; it also
   does not show it is better. Strata's clear wins stay speed (817 vs 315 reading, 55.5 vs 44.0 writing) and tool use
   (94.0 vs 91.7). Next is Daniel's call: the port (Phase 3) can still go ahead; a fair Terminal-Bench comparison needs
   llama.cpp re-run in Performance mode.

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
   Default `docker build` still gives the runtime image, ID unchanged. **Order on marvin, after Daniel's OK** (GPU
   memory free and power mode checked first). Every container runs as a normal user with only the GPU groups, like
   the relay container in the Strata test (both images default to root):
   `docker run --rm --network none --user 1000:1000 --group-add $(getent group render | cut -d: -f3) --group-add
   $(getent group video | cut -d: -f3) --device /dev/kfd --device /dev/dri --read-only --tmpfs /tmp -e HOME=/tmp ...`
   (group numbers read on marvin; HOME=/tmp because the image's /root is not writable for user 1000). Checked on
   zaphod without a GPU: both images start as user 1000 with a read-only filesystem. Also in the step 6 test script.
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
5c. Step 6 tooling, ready (2026-10-30, nothing run on marvin): `marvin/strata-fork/fork.sh` (runs on marvin):
   `check` = checks 1-4 above in order (preflight: date ≥ 2026-11-04, image digests, pack, GPU power limit; stops
   llama-swap and starts it again at the end; stops at the first failure; results in `~/strata-fork/results.txt`),
   plus one short answer (2+3) so the hipBLASLt line is seen even if the table loads at the first prompt. `start` /
   `status` / `stop` = the fork served sealed on marvin:8099 like `strata-test.sh`, with `big`'s sampling and thinking
   level medium (`--chat-template-kwargs`, this llama.cpp has no `--reasoning-effort`; check thinking length on
   first use). `args` = dry run. Engine arguments = Phase 1's own (preset `p16m`): read from the last "engine
   started:" line in marvin's strata-data (the real command; on zaphod that line had `--layer-split auto` and
   `--batch`, which the preset json lacks), fork paths swapped in. Checked on zaphod: syntax, the marvin-only guard
   (every command except `args` stops at once elsewhere), arguments read from zaphod's own upstream preset.
   `marvin/strata-fork/zaphod-peer.draft.yaml`: the zaphod llama-swap peer `marvin-fork` → `big-fork` for live
   sessions (Open WebUI, agents); a draft, not in the live config.
5d. Pack inputs (2026-10-30): MTP pieces need no download: zaphod's `strata-data/Strata-data/mtp/rt` (4 files) is
   byte-identical to the fork's published set (HF `robertkarl/UnbiasedStrataPacks` @ `cb35f7b7`: 3 by LFS sha256,
   `dense.txt` by git blob id), and marvin's copy came from zaphod with sha256 checked (Phase 1 decision 2). Expert
   profile: the fork's shipped `data/expert-profile.bin` (ranks all 24,576). Model shards: downloading marvin's
   pinned UD-IQ4_XS files to zaphod with `setup/models.sh marvin UD-IQ4_XS` (sha256 checked). Pack tool image
   (`--target pack-tools`, numpy + regex from the same Ubuntu snapshot): not built yet, snapshot.ubuntu.com answers
   500/502/503 today; retry, no fallback to the unpinned archive.
5e. **Pack built** (2026-10-30, fork `2bff35e`, image `unbiased-strata-pack:gfx1151`): shards downloaded to zaphod
   (94 GB, 2 h 17 min, `MODELS-OK`); `iq_pack.py --compat-bf16` in the container (no network, user 1000, read-only,
   only the 3 shards mounted): **14 s, 1.4 GB** at `/srv/models/strata-fork/pack-ud-iq4_xs` + the fork's
   `expert-profile.bin`. 1,079 tensors (303 served from the GGUF), 460 converted (264 exact, 196 rounded, max |err|
   0.0144). Pack format v4 (layer 14's gate/up/down span two shards); the fork engine reads up to v4. A second run gave
   byte-identical files: checksums in `marvin/strata-fork/pack-ud-iq4_xs.sha256`. First run failed: the image lacked
   the vendored gguf-py and python3-yaml (README says gguf-py comes from the submodule; my miss), fixed.
   On marvin (after Daniel's OK): copy the 1.4 GB pack (small enough for Wi-Fi), `sha256sum -c` against the file.
5f. **Copied to marvin** (2026-10-30, Daniel: marvin free, copy now, run 2026-11-04): marvin's homelab fast-forwarded
   (750eedf → 40a80d4; nothing under `marvin/llama-swap/` changed). Images by `docker save | ssh marvin docker load`
   (28 s): config digests `8869cd2b…` / `8902459…` = the zaphod builds. Pack by rsync (152 s): `sha256sum -c` OK. MTP
   rt on marvin = zaphod's = the fork's published set. `./fork.sh args` on marvin (dry run, no GPU) reads preset
   `p16m` from the engine log: `--pack /pack --native <shard 1> --expert-profile /pack/expert-profile.bin
   --expert-cache 16806 --spec 4 --spec-min-p 0.5 --mtp /sd/Strata-data/mtp/rt --max-context 131072 --kv int8
   --ple-io ram --kv-resident 32768 --resident-budget-gib 55 --prefill 16384 --batch 2` (no `--ple-gguf`: with
   `--native` the engine finds the PLE shard itself). Upstream Strata (`strata`, Phase 1) was still running; the
   check refuses to start while it runs. **On 2026-11-04:** `~/strata-test/strata-test.sh stop`, then
   `~/homelab/marvin/strata-fork/fork.sh check`.
6. **Step 6 check passed on marvin** (2026-10-30 17:07, run 3, exception decision 25; `~/strata-fork/check-run3.log`,
   engine log `~/strata-fork/engine-20261030-170725.log`): images = zaphod builds, pack sha256 OK; `strata-device`:
   gfx1151, can run; `hip_prefill_hipblaslt_gemm`, `hip_prefill_hcd_exact_parity`, `hip_prefill_mmq_parity`: pass
   (exit 0, not skipped); engine with Phase 1's `p16m` settings ready in **61 s**, GPU 65 GB, RAM 110 GB used / 12 GB
   available (Phase 1 upstream: 65 / 106); start log lists all **19 gfx1151 switches**; `hipBLASLt tuning enabled
   (90 rows, gfx1151, version 100401)`; 2+3 → "5". The slim image needed no fallback. Runs 1 and 2 failed only on
   my script (19-switch line present; `grep -q` under pipefail gave a false BAD), fixed in `f41d2f4`.
   Not yet measured: speed, quality, tool use (next: `./fork.sh start`, then `tuning/speed` against marvin:8099).
