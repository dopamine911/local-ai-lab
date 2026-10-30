# Decisions made while Daniel was away

Newest last. Each: what, why, how to undo. Started 2026-10-30 (Phase 3 of `plans/strix-strata.md`, porting
UnbiasedStrata to gfx1151; instructions `plans/strix-strata-port-instructions.md`, code map `plans/strix-strata-port-notes.md`).

## Daniel's answers (2026-10-30)

1. **Step 6 on marvin: wait.** marvin is still busy. Nothing on marvin until Daniel says it is free.
2. **Fork pieces under 2 weeks old: wait** for the strict rule. Earliest step 6 date: **2026-11-04** (and marvin free).
3. **Pack: build** one with the fork's `tools/iq_pack.py --compat-bf16`, so the test is all-fork. Needs the Unsloth
   GGUF shards (only on marvin now, about 60 GB) and numpy (pinned, 2 weeks old). How to get the shards is decided
   then; not over marvin's Wi-Fi while it is busy.
4. **Makefile fix** (`make build BACKEND=hip` turns `STRATA_PREFILL_MMQ` on): **yes, later**, in the Phase 4
   follow-up PR. Still nothing posted upstream without Daniel's OK.
5. **Push:** Daniel asked to push to his fork `dopamine911/UnbiasedStrata`. Done: branch `hip-gfx1151`
   115af49 → 4149027 (the 4 docker commits; gitleaks clean). Nothing sent to robertkarl/UnbiasedStrata.

6. **Open, for Daniel: `hip-gfx1151` is 8 commits behind main** (found 2026-10-30 after Daniel asked). The branch is
   PR #19 (`115af49`, based on main `2049dd2`); main has since added the DeepSeek-V4.1 engine `strata-dsv41` (also
   built for HIP), tool-call healing (#17/#18) and a llama.cpp submodule bump (`44072a19`), all 2026-10-21..23.
   Main does **not** merge cleanly: conflicts in `Makefile` (both change the configure / TARGETS lines) and
   `README.md`. GitHub shows PR #19 "not mergeable" (dirty), last updated 2026-10-21. Options:
   a. **Test PR #19 as is on marvin (2026-11-04), rebase onto main later in Phase 4** (suggested): it is what was
      chosen to test, the images are built, and main's newest pieces pass the 2-week rule only on 2026-11-06.
   b. Rebase now: resolve the Makefile conflict by hand (our own edit of PR #19's build file), rebuild, and wait
      until 2026-11-06; the new engine adds an untested HIP build.
   Good answer: "a" or "b". Either way upstream PR #19 needs a rebase before it can merge; telling its author is
   upstream contact and needs Daniel's OK.
   **Daniel: later (a).** Test PR #19 as is on marvin; merge main in after testing, not before 2026-11-06.

7. **Upstream on hold** (Daniel 2026-10-30): no upstream submission of any kind (comment on PR #19 / issue #6, a PR,
   contact with the PR #19 author) until the port is proven: fully tested with live sessions and plenty of stats.
   Daniel decides when that bar is met. Covers items 4 and 6 above: the Makefile fix and the PR #19 rebase stay local.
   Recorded in the plan (status line, Phase 4).

Hard stops kept (not decided alone): no sudo, nothing on marvin, no push / PR / GitHub comment upstream, no version under
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
12. **Tests image added** (advisor's point): step 6's first checks (`strata-device`, the HIP parity tests) are not
    built by `make build`, so they get their own image (`--target tests`, runtime + 4 programs) instead of being
    found missing on marvin. Also `hip_prefill_mmq_parity`, beyond the two the instructions name: it checks the MMQ
    kernels PR #19's Makefile leaves out. Built on zaphod with no source patch.
13. **Image digests recorded both ways** (found on re-check). The plan listed the runtime image by its config digest
    (`8869cd2b…`) and the tests image by its manifest digest (`d427915b…`); zaphod's Docker (containerd store) shows
    the manifest digest as the ID, so `docker images` showed `3d785883…` and looked like a changed image. Nothing
    changed: `.Descriptor` shows manifest `3d785883…` with config `8869cd2b…`. The plan now lists both digests for
    both images, and step 5b.1 says how to compare on marvin. Side note: I ran `ssh marvin true` once
    (connection test only, no command on marvin, no GPU or Docker contact).
14. **Containers on marvin run as a normal user** (other agent's review). Both images default to root; step 6 runs
    them with `--user 1000:1000`, only the render + video groups, read-only filesystem, `--tmpfs /tmp -e HOME=/tmp`
    (plan step 5b). Not baked into the image (`USER` line): that would change the pushed Dockerfile and the image
    digests; the run command is where the Strata test sets it too. Checked on zaphod, no GPU: both start as user 1000.
    Also fixed: Daniel's answers were dated 2026-10-31; it is 2026-10-30 (I read "good morning" as the next day).
15. **Pack built on zaphod first** (Daniel: "build"; all 3 jobs). marvin's model files are downloaded again to zaphod
    from the same pinned revision with the same sha256 (`setup/models.sh marvin UD-IQ4_XS`), so marvin is not
    touched and its Wi-Fi is not used. On marvin later: copy the pack, or rebuild it there and compare hashes.
    The pack holds no experts (they are read from the GGUF), so it should be a few GB.
16. **MTP pieces reused, not downloaded**: identical to the fork's published set (plan 5d).
17. **Pack tools from the Ubuntu snapshot, not pip**: same pin as the rest of the image. The snapshot server is down
    (503) today; I wait rather than use the live archive (unpinned) or pip.
18. **Step 6 script reads Phase 1's engine command from marvin at run time** instead of a copy written now: it is
    the exact command upstream ran (incl. flags its server added), and I cannot read marvin today. The script prints
    and records the arguments before starting. `./fork.sh args` shows them without a GPU.
19. **Live sessions via a second zaphod peer** (`marvin-fork` → `big-fork`), drafted only: zaphod's llama-swap
    reloads its config on every edit, so the live config is not touched. Only one engine on marvin at a time
    (memory): `fork.sh start` stops marvin's llama-swap, `stop` starts it again, as in Phase 1.
