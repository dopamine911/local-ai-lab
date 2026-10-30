# Instructions: port UnbiasedStrata to AMD Strix Halo (gfx1151)

For the agent doing Phase 3 of `plans/strix-strata.md`. Written 2026-10-30 by the session that ran Phase 1.
Read this whole file first, then `plans/strix-strata.md` (the plan and the Phase 1 log with every measurement).

## Who you work for, and how

Daniel owns this lab. Rules that matter for this job (full versions: `CLAUDE.md` in ~/homelab and ~/.claude/CLAUDE.md):

1. He has a brain injury: number the steps, say what a good result looks like, record results in files (not chat).
2. He is dyslexic: read his messages for intent; never correct his word choice.
3. Correct him plainly when he is wrong. Accuracy over agreement.
4. **One change at a time, measured before and after.** One change per commit.
5. Commit messages: `what · from → to · why`, ending with `Co-Authored-By:` your model line.
6. **Never push, open a PR, or comment on GitHub (robertkarl/UnbiasedStrata, PR #19, issue #6) without Daniel's OK.**
7. No secrets in any repo or output. Keys live in Proton Pass and mode-600 `.env` files only.
8. Pin every image, package and download to its latest version, exactly (digest / exact version / sha256).
   (Changed 2026-10-30: there is no minimum age. The earlier "at least 2 weeks old" rule was never Daniel's.)
9. Downloads and extracted archives are untrusted: own empty directory each, never run code from inside them.
10. Never run `sudo` on Daniel's behalf; give him the command.

## Goal

A clean UnbiasedStrata build for gfx1151 (Ryzen AI Max+ 395 / Radeon 8060S), built in a container, that matches or
beats upstream Strata 0.2.7 on marvin, so the results can be offered upstream (Phase 4, Daniel decides).
UnbiasedStrata's own rules stay: no installer, no downloads at build or start, no Python in the request path,
the CUDA build unchanged.

## Where things are

| What | Where |
| --- | --- |
| The fork, cloned | `~/src/UnbiasedStrata` on zaphod. Remote `origin` = github.com/dopamine911/UnbiasedStrata, `upstream` = robertkarl |
| Working branch | `hip-gfx1151` = PR #19's commit `115af49` ("build: allow HIP backend selection from Makefile"). Upstream main is 8 commits ahead and merging conflicts (Makefile, README, CMakeLists +93 lines): build on 115af49 first, rebase later |
| CodeGraph index | `~/src/UnbiasedStrata/.codegraph` (untracked: never commit it; add it to `.git/info/exclude`) |
| HIP build switch | `Makefile`: `BACKEND ?= cuda`, `BACKEND=hip` sets `STRATA_ENABLE_HIP=ON`, `HIP_ARCHS ?= gfx1100` (use `gfx1151`), `ROCM_PATH`, `HIP_CXX` |
| HIP CMake | `cmake/hip_backend.cmake`; options in `CMakeLists.txt` (`STRATA_ENABLE_HIP`, `STRATA_PREFILL_MMQ` needs HIP) |
| gfx1151 tuning tables | `tools/hip/gfx1151-hipblaslt-100401.txt` (ROCm 7.14.1's hipBLASLt), also `-100400`, `-100500` |
| Upstream's Strix Halo recipe | `docs/STRIX_HALO.md` (sections 1-6: toolchain, cmake flags, defaults, opt-in switches, measured), `docs/AMD_HIP.md` |
| ROCm toolchain | `https://repo.amd.com/rocm/tarball-multi-arch/therock-dist-linux-gfx1151-7.14.1.tar.gz`, sha256 `c40e8f2bd6630a7d11557c762b99c6fa8afb04c9bd0e51ed1675ee1ca24afb00` (from STRIX_HALO.md; verify; use the latest and pin it) |
| Other people's HIP work | PR #19 (firetussin, gfx1031: full HIP build, `make test` passes). Issue #6 (axelgenus, gfx1030, ROCm 7.0 in Docker): 3 patches: stub files for missing HIP test/tool targets, `unbiased-strata` target enabled for HIP, `__shared__ alignas(16)` fix in `src/kernels/cuda/iq_kernels.cu`. Use a patch only if our build needs it |
| Upstream engine we compare against | `plans/strata-test/upstream-0.2.7-gfx1151-engine-BUILD.json`: engine 0.1.52, source commit `a42cc3d7df70`, variant `halo`, gfx1151, hipBLASLt 100401 |
| Phase 1 test script (sealed) | `marvin/strata-test/strata-test.sh` (copy on marvin at `~/strata-test/`): internal Docker network + relay on Tailscale :8099; per-start API key in `~/strata-test/key` (never print it) |
| Speed test | `tuning/speed` (fixed prompts: writing tok/s; ~32K-token prompt reading tok/s). Tool use: `tuning/quality`. Terminal-Bench: `tuning/tbench` |

## What Phase 1 learned (do not re-learn it)

1. **marvin's BIOS must be on Performance.** On Quiet the GPU is held at 54-55 W and long prompts read ~20-25% slower.
   During any GPU test, sample power (`/sys/class/drm/card1/device/hwmon/hwmon*/power1_average`): good = 130-140 W
   while busy. A BIOS visit once dropped it back to 55 W (see `CLAUDE.md` Known issues). Check it every test.
2. Model: Unsloth `Qwen3.8-Flash-Next-UD-IQ4_XS` (3 shards in `/srv/models` on marvin, read-only). In upstream Strata
   the family is `unsloth`, not `qwen`.
3. Upstream's engine command on marvin (from its log):
   `--expert-cache 16806 --prefill auto --spec 4 --spec-min-p 0.5 --mtp <dir>/mtp/rt --max-context 131072 --kv int8
   --ple-io ram --kv-resident 32768 --resident-budget-gib 55 --batch 2`, plus 19 gfx1151 speed switches on by default.
   Strata's setup chooses the RAM-budget mode on purpose for unified-memory APUs (its issue #1715: an OOM kill).
4. Experiments (32K reading, one change each): `prefill 16384` **+8% (keep)**; parallel 1, `kv-resident off`: no
   change; `resident-budget-gib off`: does not start (cudaMalloc 47.5 GiB out of memory). `tuned`: no change.
5. Baseline to beat (Performance mode, prefill 16384, MLflow f807b742): **reading 817 tok/s, writing 55.5 tok/s**
   on `tuning/speed`. llama.cpp on the same test: 315 / 44.0. Tool use (3 runs): Strata 89.3 vs llama.cpp 91.7;
   Strata repeats attacker text more often (2.7 vs 1.0 of 3). Terminal-Bench on Strata: running 2026-10-30 overnight.
6. The draft-layer (MTP) pieces are already on marvin in the Docker volume `strata-data` at `/Strata-data/mtp/`
   (31 pieces, each with a sha256 in `mtp-manifest.json`; `rt/` holds the built files). Use them; download nothing.

## Steps

**Do not touch marvin until Daniel says Terminal-Bench is finished** (it runs against Strata there overnight; any GPU
or Docker change ruins it). Steps 1-4 are on zaphod only and need no GPU.

1. **Map the code (CodeGraph first).** In `~/src/UnbiasedStrata`: `codegraph explore "Makefile build BACKEND hip_backend.cmake STRATA_ENABLE_HIP"`,
   then the gfx1151-specific code: unified-memory free-bytes (`device_free_bytes`), the gfx1151 default switches,
   the hipBLASLt table loader, prefill chunk choice, expert cache / RAM budget. Write a short map to
   `plans/strix-strata-port-notes.md` (file: what it does for gfx1151).
   Good result: you can name every place where gfx1151 behaves differently from gfx1100/gfx1201.
2. **Toolchain container.** A Dockerfile in the fork (e.g. `docker/hip-gfx1151.Dockerfile`): pinned base image by
   digest, TheRock ROCm 7.14.1 tarball by URL + sha256, pinned CMake/Ninja, compiler setup per STRIX_HALO.md section 1.
   Build stage only; no network at runtime. Good result: `hipcc --version` / clang in the image report ROCm 7.14.1.
3. **Build.** `make build BACKEND=hip HIP_ARCHS=gfx1151 ROCM_PATH=...` inside the container, with the cmake flags
   STRIX_HALO.md section 2 uses (compare with upstream's BUILD.json above). If it fails, fix one thing per commit;
   take an issue #6 patch only when the same error appears. Good result: an `unbiased-strata` binary for gfx1151,
   and the CUDA build still configures unchanged (`make build` default).
4. **Tests without a GPU.** The fork's `make test` (fake engine). Good result: passes as on PR #19.
5. **Runtime image.** Only the binary and the runtime libraries it needs (hipBLASLt table included). Starts with no
   network. Record image size and digest.
6. **On marvin, only after Daniel's OK.** A sealed test like `marvin/strata-test/strata-test.sh` (new script or a
   `ENGINE=unbiased` switch; keep the internal network + relay). Same model files, same MTP pieces, same settings as
   item 3 above with `prefill 16384`. Before each start: GPU memory free, power mode check.
   Then, one at a time: HIP parity tests (`hip_prefill_hcd_exact_parity`, `hip_prefill_hipblaslt_gemm`), `tuning/speed`
   (compare with 817 / 55.5), tool use x3 as `SERIES=unbiased-strix-iq4xs` (same seeds as Phase 1), determinism
   (same prompt at temp 0 after 3 restarts), Terminal-Bench last (Daniel starts long runs or OKs them).
   Always end with `strata-test.sh stop` (marvin's llama-swap back). Note: after a reboot during a test, nothing on
   marvin restarts by itself; start llama-swap with `cd ~/homelab/marvin/llama-swap && docker compose start llama-swap`.
7. **Record.** Every result in `plans/strix-strata.md` (Phase 3 log, numbered), MLflow via `tuning/log`, commits in the
   fork on `hip-gfx1151` (one change each) and in ~/homelab for scripts.

## Stop and ask Daniel before

- any push, PR, issue comment or upstream contact;
- anything on marvin (until he says Terminal-Bench is done) and any change to marvin's host (packages, boot, BIOS);
- anything needing sudo;
- a change to the CUDA path.
