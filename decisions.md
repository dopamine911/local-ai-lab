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
