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
