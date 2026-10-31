# Strix Halo port of UnbiasedStrata: timeline

What was done and when, for `plans/strix-strata.md` Phase 3 (and the steps after it). Times are Chicago time (CDT).
Sources: git commits (homelab, fork `~/src/UnbiasedStrata`), marvin's `~/strata-fork/results.txt` and
`~/strata-test/log.txt`, the build and download logs. Results and reasons are in the plan's Phase 3 log and in
`decisions.md`; this file is only the order of events.

## 2026-10-29 (evening): steps 1-5 on zaphod

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 20:58 | Port instructions written by the planning agent | `plans/strix-strata-port-instructions.md` | 5c6558f |
| 21:05 | Step 1: code map of gfx1151 in the fork | PR #19's AMD build leaves `STRATA_PREFILL_MMQ` off | a08ca53 |
| 21:13 | docker-buildx installed on zaphod (Daniel) | added to `setup/host.sh` | b027568 |
| 21:24 | Step 2: build container (ROCm 7.14.1, pinned) | built in 200 s; hipBLASLt 100401 | fork ed764de, bbe15dc |
| 21:30-21:36 | Step 3: engine build for gfx1151, no network | 4.5 min, no source patch | fork 63adfc3 |
| 21:41-21:47 | Steps 4-5: `make test`; slim runtime image | pass; 1.09 GB image | fork 0ac55a1, 1a44a8a |
| 21:49-21:53 | Tests image (GPU checks for marvin) | built | fork 4149027, c8cf6b8 |

## 2026-10-30

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 01:59 | Image digests written both ways (manifest + config) | the marvin load check compares like with like | b799149 |
| 11:29 | Daniel's answers: wait (marvin busy), pack: build, Makefile fix later | recorded | ea19e39 |
| 11:33 | Containers on marvin to run as a normal user | added to the step 6 order | afcb7dc |
| 11:34 | Branch found 8 commits behind main, conflicts | question for Daniel | b2ab8ff |
| 11:36 | Upstream submissions on hold until proven | recorded | cacd851 |
| 13:08 | Daniel: catch up to main later | recorded | 4e640fc |
| 13:12 | Download of marvin's 3 model files to zaphod started (94 GB) | | models.sh log |
| 13:24 | Pack tools image: Ubuntu snapshot server down (503) | retried every 10 min | |
| 13:24 | Step 6 script `fork.sh` + live-session draft | committed, not run | a3b68fd |
| 13:37 | Pack tools image built (2nd try) | | |
| 15:29 | Download finished, checksums OK | `MODELS-OK`, 2 h 17 min | models.sh log |
| 15:30-15:31 | Pack built (first try failed: gguf-py missing; fixed) | 14 s, 1.4 GB, identical twice | fork 2bff35e, a479bec |
| 15:33-15:36 | Pack checksum check in `fork.sh`; device check; thinking level checked in code | | f512f58, b92f72c, 60266a1 |
| ~15:56 | Daniel: marvin free, copy now, run 11-04; push both | fork pushed (2bff35e) | |
| ~15:57-16:00 | Images (28 s) and pack (152 s) copied to marvin, checked | digests and checksums match | c2ec3ed |
| 16:05 | Daniel: run the test today (2-week exception) | recorded | 921b6cd |
| 16:05:59 | Upstream Strata stopped on marvin, llama-swap started | | strata-test log |
| 16:06:18 | marvin check run 1 | **BAD**: 19-switch line not found (script bug, line was there) | results.txt |
| 17:04 | Script keeps the engine log, judges step 3 at the end | | ff282d8 |
| 17:05:02 | marvin check run 2 | BAD again, same script bug; engine log shows all 19 switches | results.txt |
| 17:07 | Script fix (`grep -q` under pipefail) | reproduced and fixed on zaphod | f41d2f4 |
| 17:07:24 | marvin check run 3 | **passed** (GPU, 3 accuracy tests, 19 switches, table, 2+3 = 5) | results.txt, 323dbf4 |
| 17:11:01 | Fork served on marvin:8099 (medium) | API check raced the relay (BAD), answered seconds later | results.txt |
| 17:12 | Speed test, fork, medium | reading 821.6, writing 49.5 tok/s | MLflow 318a12f2 |
| 17:16 | `fork.sh`: thinking level setting; relay wait | | 6facd41, c83d3dd |
| 17:16:36 | Fork restarted at **xhigh** on marvin (part of the command Daniel rejected) | answered at 17:17:34; no test sent; replaced by the next stop | results.txt |
| 17:17 | Daniel: every test at medium | recorded | 403a931 |
| 17:17:27-17:19:34 | Fork stopped; llama-swap back | | results.txt |
| 17:19:34 | Upstream Strata started at medium (p16m) | ready 17:20:50 | strata-test log |
| 17:21 | Speed test, upstream, medium | reading 821.9, writing 53.5 tok/s | MLflow c05ca923 |
| 17:23:46 | Upstream stopped; llama-swap back (`big` online) | | strata-test log, 8e50682 |
| ~17:30 | Engine logs compared: upstream has `HC_Q8` on (answers can differ) | likely the writing gap | |
| ~17:32 | Engine audit started (Strata 0.1.45 → 0.1.54), code read only | | |
| 17:36 | main merged into the fork branch (Makefile, README conflicts) | | fork a185402 |
| 17:37-17:43 | Merged branch built for gfx1151; `make test` | 348 s; pass | |
| 17:39 | Fork branch pushed to GitHub by another session (not this one) | | reflog |
| 17:45-17:47 | Images rebuilt; pack rebuilt with merged tools | pack identical | |
| 17:48 | `fork.sh`: no date gate (rule withdrawn), new digests | | fc0337d |
| 17:49:31 | marvin check on the merged branch | **passed** | results.txt, 438f880 |
| ~17:51 | Engine audit finished | one answer-changing default (`HC_Q8`); 3 fixes worth taking | plan Phase 3 log 10 |
| 17:54-17:55 | 3 upstream fixes ported (hang, batch leak, memory sizing) | | fork 7a65497, a0acda7, 21bbb95, 02c9904 |
| 17:55-18:01 | Fixed engine built; unit tests | 344 s; 4,782 + all checks pass | |
| 18:04 | Images rebuilt, copied to marvin | | 461b642 |
| 18:05:01 | marvin check with the fixes | **passed** | results.txt |
| 18:06:34 | Fork served (medium); speed test | reading 811.4, writing 49.7 tok/s | MLflow 212e554d |
| 18:10:20 | Fork stopped; llama-swap back (`big` online) | | results.txt, b353ae4 |
| 18:10 | Fork branch pushed (02c9904) | | |
| 18:12-18:28 | Ubuntu base bump: `resolute-20261024.1` + snapshot 2026-10-30; full rebuild; `make test` | 15 min; pass; pack identical | fork cc2a9ca |
| 18:29 | New images copied to marvin | | |
| 18:30:06 | marvin check on the new base | **passed** | results.txt |
| 18:31:45 | Fork served (medium); speed test | reading **786.8** (falling: 821.6 → 811.4 → 786.8), writing 49.6 | MLflow 9fdeb14c |
| 18:35:20 | Fork stopped; llama-swap back (`big` online) | | results.txt |
| 18:37-18:40 | ROCm 10.1 found (Daniel's link): `stable.repo.amd.com/rocm/core/`, tarball has no published checksum | Daniel chose the signed apt packages | |
| 18:40 | AMD key fetched and checked (fingerprint D0F0…2107); repo signature good; gfx1151 set = 49 packages, all 10.1.0-3 | | |
| 18:41-18:49 | Trial install in a throwaway container (first try: key not readable by apt, fixed) | 8.1 GB; hipBLASLt 1.4.1 (= 100401); clang 23 → 24 | |
| 18:50 | Dockerfile: ROCm stage = signed packages, pinned; `.kpack` copy removed; full rebuild started | running | |
| 19:00:02 | Upstream Strata recheck (p16m, medium), while the ROCm 10.1 rebuild runs on zaphod | reading 822.1, writing 53.7: **marvin is not drifting**; the fork's drop is in our builds | MLflow bd592a58 |
| 19:03:55 | Upstream stopped; llama-swap back (`big` online) | | strata-test log |
| 18:50-19:04 | ROCm 10.1 rebuild; `make test` | 13 min; pass; clang 24, hipBLASLt 1.4.1 | fork fbfb29f |
| 19:05:44 | marvin check on ROCm 10.1 | **stopped**: HCD exact test skipped: 10.1's hipBLASLt numbers solutions differently (same version 100401); table entry → -1 | results.txt |
| 19:05:58 | llama-swap back (`big` online) | | results.txt |
