# Decisions made while Daniel was away

Newest last. Each: what, why, how to undo. Started 2026-10-30 (Phase 3 of `plans/strix-strata.md`, porting
UnbiasedStrata to gfx1151; instructions `plans/strix-strata-port-instructions.md`, code map `plans/strix-strata-port-notes.md`).

Hard stops kept (not decided alone): no sudo, nothing on marvin, no push / PR / GitHub comment, no version under
2 weeks old, no change to the CUDA path.

## 2026-10-30

1. **docker-buildx on zaphod** (Daniel chose and installed it; 0.30.1-0ubuntu1). Added to `setup/host.sh` package list
   for both machines (marvin does not have it yet; it gets it on the next `host.sh` run). Why: the Dockerfile itself
   can then verify the ROCm download's sha256 (`ADD --checksum`) and block the network during the build
   (`RUN --network=none`). Checked: `docker builder ls` shows BuildKit v0.26.2.
2. **Base image `ubuntu:resolute-20260927`** (Ubuntu 26.04, digest `sha256:f144425f…59f7`, published 2026-10-04).
   Why: `ubuntu:26.04` today is 2026-10-25 (5 days, fails the 2-week rule); the next dated tag `resolute-20261006`
   was published 2026-10-17 (13 days, fails). Undo: change the `FROM` line.
3. **Ubuntu packages pinned by the archive snapshot of 2026-10-10** (`snapshot.ubuntu.com/ubuntu/20261010T000000Z`,
   every package at least 20 days old). Checked: git and curl come out one patch older than today's archive, so the
   pin works. The base image has no CA certificates and the snapshot server is HTTPS only, so the first apt step
   (ca-certificates alone) runs without the TLS check; apt still verifies GPG signatures (the same protection as the
   base image's default plain-HTTP archive). All later apt steps check TLS. Versions: cmake 4.2.3, ninja 1.13.2,
   g++ 15.2.0, git 2.53.0, python3 3.14.3. A `# syntax=` line is left out on purpose: it would pull an unpinned
   frontend image; BuildKit's built-in one has ADD --checksum and RUN --network.
4. **ROCm 7.14.1 tarball**: published 2026-08-31 (Last-Modified header), passes the 2-week rule; 1,713,467,716 bytes;
   sha256 checked by `ADD --checksum` in the Dockerfile.
5. **For Daniel to confirm: two pieces of the fork are under 2 weeks old.** PR #19's commit `115af49` (2026-10-21,
   you chose it) and its llama.cpp submodule `a6de9af3` (2026-10-19: ggml-org `3cf03257` from 2026-09-20 plus the fork
   author's two tool-call parser commits). They are the source under test, not a dependency choice, so I went on;
   nothing is deployed. If you want strict 2 weeks: wait until 2026-11-04 before anything runs on marvin.
6. **Build flags via `CMAKE_EXTRA`, Makefile unchanged.** The Dockerfile passes `-DSTRATA_PREFILL_MMQ=ON` and
   TheRock's paths. Fixing PR #19's Makefile (HIP branch turns MMQ on) is left for later: one change at a time, and
   first marvin should show the build works. Candidate for the Phase 4 follow-up PR (your call).
7. **Retries with the source mounted, final recipe in the Dockerfile.** `docker run --network=none -v fork:/src`
   kept the build folder between attempts; the Dockerfile's `build` stage runs the same command. Both builds give
   byte-identical GPU code; the CPU code differs in bytes but not in size (likely the build folder name and the git
   commit string llama.cpp embeds; not proven, confidence ~0.8).
8. **Runtime image keeps ROCm's own folder layout** (`/opt/rocm/lib`, `/opt/rocm/.kpack`) so rocBLAS / hipBLASLt find
   their kernel files without settings. Only `blas_lib_gfx1151.kpack` is included: rocBLAS, hipBLASLt and rocSOLVER
   name it; nothing linked names the other five archives (fft, hiptensor, rand, rccl, rocalution, 63 MB).
9. **No GPU test on zaphod.** zaphod has no gfx1151 (its cards would only give "no code for this card"), and both
   cards are serving llama-swap. First check on marvin instead: the engine finds the GPU and lists 19 switches.
10. **Image built with `--provenance=false`.** Without it BuildKit adds a per-build record and the image ID changes on
    every build, even fully cached (checked: two builds, same ID with the flag, different without).
11. **Stopped before step 6** (marvin): instructions say nothing on marvin until you say Terminal-Bench is finished.
