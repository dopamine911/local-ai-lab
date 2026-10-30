# Notes: UnbiasedStrata on gfx1151, code map (Phase 3 step 1)

Written 2026-10-30. Fork `~/src/UnbiasedStrata`, branch `hip-gfx1151` at `115af49` (PR #19). Engines are PolyStrata
`v0.1.45-poly.1`; upstream Strata 0.2.7 (Phase 1) ran engine **0.1.52**. That gap is a risk: if we miss 817 tok/s,
some of the cause may be engine changes since 0.1.45, not the port.

## Findings that change the plan (most important first)

1. **PR #19's `make build BACKEND=hip` never turns on `STRATA_PREFILL_MMQ`.** STRIX_HALO.md section 2 says the prompt
   experts need it (`-DSTRATA_PREFILL_MMQ=ON`). Without it the `elseif(STRATA_PREFILL_MMQ)` branch
   (CMakeLists.txt:1737) never runs: no ggml MMQ prompt kernels, no `STRATA_PREFILL_FUSED`. Likely a big loss in reading
   speed. First build: pass it via `CMAKE_EXTRA`. A Makefile fix is a separate commit and a Phase 4 candidate.
2. ~~Compiler path~~ (checked 2026-10-30, not a problem): TheRock 7.14.1 has clang at both `$SDK/llvm/bin/clang++`
   (the Makefile's default) and `$SDK/lib/llvm/bin/clang++` (STRIX_HALO.md). AMD clang 23.0.0git.
3. **Build with `PORTABLE=1`.** Upstream's BUILD.json: `"portable": true`. Without it ggml-cpu is built
   `-march=native` for zaphod's Zen 2 (no AVX-512) and then runs on marvin's Zen 5. Strata's own AVX-512 kernels
   choose at run time either way (CMakeLists.txt:30-33).
4. **`make test` says nothing about HIP.** It builds a separate front-end-only tree (`build-frontend`, CUDA and HIP
   off) and runs it against `frontend/tests/fake-engine.py`. A pass in step 4 only shows the front end works.
   HIP is checked by the parity tests on marvin (step 6), or the HIP build's own ctest targets.
5. **Use the `strata` engine, not `strata-poly`.** Only `strata` has the Phase 1 flags (`--expert-cache --prefill --spec
   --spec-min-p --mtp --ple-io --kv-resident --resident-budget-gib`). Run as `unbiased-strata --engine strata -m <shard1>
   ... -- <engine flags>` (README "Qwen3.8-Flash-Next" example).
6. **`strata` needs a pack directory** (README "Preparing a Flash-Next pack"): `python tools/iq_pack.py --gguf <shard1>
   --out <pack> --compat-bf16` (Unsloth files), plus `--native <shard1>`, `--ple-gguf <last shard>`,
   `--expert-profile <pack>/expert-profile.bin`. One-time and offline (Python, but not in the request path, so the
   fork's rules allow it). Open question for step 6: reuse upstream 0.2.7's pack from marvin's `strata-data` volume
   (format may differ between 0.1.45 and 0.1.52), or build one with the fork's tool (needs ~disk; time it). The MTP
   `rt/` folder: the fork's README publishes byte-identical MTP files; compare hashes with marvin's 31 pieces before use.
7. **The llama.cpp submodule is not checked out** (`git submodule status`: `-a6de9af…`). `git submodule update --init`
   is a fetch, pinned to commit a6de9af3. Do it before the container build, so the build itself needs no network.

## Where gfx1151 behaves differently (file: what it does)

| File | gfx1151 behavior |
| --- | --- |
| `Makefile` | `BACKEND=hip`: CUDA off, HIP on, `HIP_ARCHS` (default gfx1100), `ROCM_PATH`, `HIP_CXX`. No PREFILL_MMQ (finding 1). |
| `cmake/hip_backend.cmake` | gfx1151 in the "unvalidated" list (build warns, does not stop). Any gfx11 arch defines `STRATA_WMMA_GFX11=1`. Finds hip, hipblas, rocblas (quiet), hipblaslt (quiet; without it no tuning table). Force-includes `hip_compat/cuda_runtime.h`. |
| `CMakeLists.txt` | Options: `STRATA_PREFILL_MMQ` (HIP only, default OFF), `STRATA_PORTABLE`, `STRATA_FRONTEND` (unbiased-strata, ON). `strata-poly`'s MMQ (line 1985+) is built for HIP too. |
| `src/core/arch_defaults.cpp` (+ `.hpp`, `_test.cpp`) | The 19 "exact" speed switches set at start on gfx1151 (8 prompt, 11 decode); `STRATA_GFX1151_DEFAULTS=0` turns them off; a user's setting wins. Same 19 as STRIX_HALO.md section 4. Check at step 6: the start line lists all 19. |
| `src/program/generate.cpp:2642` | Calls `apply_arch_defaults(gcnArchName)` for device 0 before any switch is read. |
| `src/program/generate.cpp:5745` | APU only (`integrated`): warns if the expert cache is bigger than available RAM less 4 GiB. Warning, not a cap. |
| `src/program/generate.cpp:5465` | APU + `STRATA_IO_PREFETCH=1` + adaptive tier: GPU-reset warning (780M). Not our settings. |
| `src/core/expert_cache.cpp:31` `device_free_bytes()` | Integrated GPU on Linux: free = `MemAvailable` − 6 GiB (`STRATA_UMA_HEADROOM_GIB`) if larger than `hipMemGetInfo`'s. Used by `--expert-cache auto` and the cache's allocation check (line 493). We pass a number (upstream used 16806), so only the check matters. |
| `include/strata/hip_compat/cuda_runtime.h:101` | `mem_get_info`: the sysfs VRAM figure is skipped on an integrated GPU (it would only count the BIOS carve-out). |
| `src/prefill/hipblaslt_tuning.hpp` | Loads `STRATA_HIPBLASLT_TUNING`. Header must say the exact arch (gfx1151 has no family fallback, unlike RDNA3/4) and the exact hipBLASLt version (100401 = ROCm 7.14.1), else refused. |
| `src/prefill/gemm.cu:791` `bf16_hcd_exact` | `STRATA_HCD_EXACT`: only when the table picks hipBLASLt solution 1176/1177 for N 320 K 10240; else prints once and uses hipBLAS. |
| `tools/hip/gfx1151-hipblaslt-100401.txt` | The table for ROCm 7.14.1 (also `-100400`, setup's wheels; `-100500`). Ship 100401 in the runtime image. |
| `src/prefill/wmma_gemm.cu`, `moe_fused.cu`, `moe_fused_iq.cu`, `kernels.cu`, `moe_f16_common.cuh`, `qsa_select.cu`, `qsa_prompt_attn.cu`, `hip_compat/intrinsics.hpp` | gfx11 WMMA kernels compiled for `__gfx1151__` (same list as gfx1100-1103/1150). `moe_fused_iq.cu:777`: capped kernels only for gfx1151 on **clang 23** (ROCm 7.14.1's); another compiler takes the default kernels. |
| `src/core/verify.cpp:236` | `STRATA_SH_STREAM` reason (set by the gfx1151 table). |
| `tests/hip/prefill_hcd_exact_parity.cpp`, `prefill_mmq_parity.cpp`, `gdn_chunked.cpp` | HIP tests; hcd parity needs the table, exits 77 (skip) without it. |
| `docs/STRIX_HALO.md` | Upstream's recipe: section 1 toolchain, 2 cmake flags, 3 table, 4 defaults, 5 opt-in (bit-changing) switches. Note: its section 2 says CMake fetches ggml; in this fork ggml comes from the submodule (UPSTREAM.md). |

Not touched by gfx1151: prefill chunk choice (`--prefill auto` gets a gfx12-only bump to 16384 at generate.cpp:2650;
gfx1151 gets nothing: we pass `--prefill 16384` like Phase 1).

## Build recipe to try first (step 3), from STRIX_HALO.md section 2

```sh
make build BACKEND=hip HIP_ARCHS=gfx1151 ROCM_PATH=$SDK PORTABLE=1 \
  CMAKE_EXTRA="-DSTRATA_PREFILL_MMQ=ON -DCMAKE_HIP_COMPILER_ROCM_ROOT=$SDK \
    '-DCMAKE_PREFIX_PATH=$SDK;$SDK/lib/rocm_sysdeps;$SDK/lib/llvm' \
    '-DCMAKE_HIP_FLAGS=--rocm-path=$SDK --rocm-device-lib-path=$SDK/lib/llvm/amdgcn/bitcode'"
```

`CMAKE_EXTRA` comes last on the cmake line, so its `CMAKE_PREFIX_PATH` overrides the Makefile's.

## Step 2 checks (toolchain container)

1. Tarball: check `Last-Modified` (`curl -sI`) against the 2-week rule before the 1.7 GB download; verify sha256
   `c40e8f2b…afb00`.
2. Good result: clang reports ROCm 7.14.1's clang 23, and hipBLASLt's version header says **100401** (the table is
   refused otherwise, and the only sign at run time is one start-up line).
3. Downloads before the build only: `ADD --checksum=sha256:…` for the tarball, build steps with `RUN --network=none`
   (check the Docker version supports both). apt packages pinned (snapshot.ubuntu.com by date, if it works for 26.04).
