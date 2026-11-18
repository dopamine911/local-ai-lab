# Radiance on zaphod's two R9700s: timeline

Daniel 2026-11-17: test Radiance (codeberg.org/StillDeadcode/radiance), after Wendell's Level1Techs guide "Ultimate 2x
R9700 guide for Qwen 3.8 27b" (2026-10-06). First pointed at marvin; Daniel: meant zaphod (Radiance is tuned for 2x R9700).
Times are Chicago time, from `date`.

## What it is (read 2026-11-17)
- A C++/HIP inference engine for AMD GPUs, Apache-2.0, open source, active (commits today). Kernel libraries: libr4d
  (RDNA4 = R9700), libr3d (RDNA3), libhalo (Strix Halo: port unfinished, ~65 stub ops). Image `stilldeadcode/radiance`
  (newest 2.1.0, 2026-11-15, sha256:c9affc489e438…). ROCm runtime inside the image; host needs /dev/kfd + /dev/dri.
- OpenAI chat/completions/Responses, Anthropic Messages and llama-server's API. Has its own prefix cache (leak tests apply).
- **The guide's models no longer load**: the guide used Radiance v1.2.3 with FP8 and MXFP4 `.rad` files; 2.1.0 refuses FP8
  and MXFP4 at startup ("each returns with an int8 container"). For our 27B now: `qwen3.8-27b-iq8r-dflash.rad` (rotated int8 +
  DFlash2 drafter, 29 GiB, tp 2) or IQR-V4 2.5-8 bits (IQ4R-L 17.4 GiB for one card, tp 1).
- Guide numbers (v1.2.3, 2x R9700, temp 0): 27B FP8 + DFlash2: 111 tok/s single (512 gen), 130 sustained, conc 8 596 agg;
  MXFP4: 133 single; Flash-Next: 125. Its own table also lists "2K prompt decode 23 tok/s" and "200K prompt decode 1.7 tok/s"
  for FP8 dense: writing after a long prompt is the thing to measure (`tuning/depth.py`). Ours: `coder` (llama.cpp Vulkan, one
  card, MTP 2) 54.2 writing, 938 reading 32K.
- **zaphod has Resizable BAR off** (CSM must stay on). Radiance's README: works without it, but kernel arguments then sit in
  pinned host memory and every launch reads them over the link (card 1: PCIe 3.0 x4). Expect less than the guide.
- Its compose files use host networking, `seccomp=unconfined` by default (a shipped profile `seccomp-radiance.json` adds only
  io_uring and memory locking to Docker's default), memlock unlimited, ipc host. Our launcher: internal network, cap-drop ALL;
  it needs a field for a seccomp profile file.
- zaphod has no launcher yet: `lab-engined` there (sudo, Daniel), `engines/zaphod.json`, zaphod llama-swap compose (network +
  mount; recreate), entries that swap with `coder`/`coder-2` and the helpers (both cards for tp 2).

## Steps (after Daniel's choices)
1. Launcher on zaphod (+ seccomp-profile field), tested like marvin's.
2. Image by digest, model file pinned (setup/models-zaphod.txt).
3. Smoke, leak tests (cache-switch, cache-evict), `lab speed`, `tuning/depth.py`, `lab quality --full` x3 vs `coder`.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 13:52 | Daniel: current Radiance 2.1.0 + IQ8R, both cards (tp 2) | — | — |
| 13:53 | Image pulled by digest (`stilldeadcode/radiance:2.1.0@sha256:c9affc48…`, 0.24 GB; entrypoint `radiance`; 2.x takes a YAML `--config`, not the guide's flags). Model pinned in setup/models-zaphod.txt: `StillDeadcode/qwen3.8-27b-iq8r-dflash` rev c127a341, 30.9 GB, Apache-2.0; download started (log ~/.cache/radiance-download.log) | — | this commit |
| 13:54 | engines/radiance-zaphod.yaml (Radiance's own row for this model: tp 2, 32 seqs, 200K, i8 KV; coder's sampling, effort medium); `radiance --check`: "model coder-radiance = …, tp 2". engines/seccomp-radiance.json from Radiance v2.1.0 (2a2b756): Docker's default + io_uring and NUMA calls, instead of the guide's `seccomp=unconfined`; launcher field `seccomp`. engines/zaphod.json (test-http, coder-radiance). host.sh step 6f and `lab setup check` 6f now on both machines | — | this commit |
| 13:55 | zaphod llama-swap: compose (socket folder + engines/ mounts, network lab-engines) and config (test-http, coder-radiance entries; routing sets `radiance` (alone, helpers unload) and `launcher_test`; image-mode and training pins) | **Slip:** saving config.yaml reloaded zaphod's llama-swap at once (~/llama-swap links into the repo) without an idle check first. Checked after: no requests but `embed` in the 15 min before, Terminal-Bench goes to marvin directly: nothing cut; preload ran. From now on: idle check before saving zaphod's config | — |
| 14:06 | Model download done: MODELS-OK, 29 GB in /srv/models/radiance | — | — |
| 14:32 | Daniel installed lab-engined on zaphod | active; `lab setup check` all done (6f ok) | — |
| 14:33 | zaphod idle (no non-embed requests in 15 min, slots free). Before: `lab speed coder` (Terminal-Bench task containers busy on the CPU) | 54.3 / 938.1 (MLflow 8bf8b86d) | — |
| 14:35 | `docker compose up -d llama-swap` (launcher mounts, lab-engines network) | networks agents-internal, lab-engines, default; `/engines/lab-engine status` from inside → [] | 5bfdec5 |
| 14:37 | After: `lab speed coder` | 54.0 / 935.7 (MLflow 2a2fd8aa): same | — |
| 14:40 | `test-http` through zaphod's llama-swap, then `coder` and `coder-2` | "ok", 1 engine, helpers stayed (coder/coder-2 unloaded by the routing set); after: 0 engines, all 7 usual models ready; launcher: stop → gone | — |
| 14:45 | Waiting for Terminal-Bench (marvin) to finish before the Radiance tests (its task containers load zaphod's CPU); the tests need the watchdog paused (Radiance runs alone; the per-minute `embed` call would evict it) | | this commit |
| 18:22 | Daniel stopped the watchdog. Memory-sync cron line paused (commented out in Daniel's crontab; restore after the test). zaphod idle | — | — |
| 18:23 | Smoke: `coder-radiance` through zaphod's llama-swap | **load + answer 20 s** ("ready", 32 thinking tokens). Log: 2 x gfx1201; **Resizable BAR off on both cards** (kernel arguments read over the link) and **no peer-to-peer between the cards** (all-reduce through pinned host memory); DFlash2 drafter, depth 7; 25.2 GiB of weights, 13.9 GiB a card; KV cache 16.6 GiB a card (756,240 tokens; 12% of the 32 x 200K worst case); prefix cache in VRAM (finished conversations' KV kept until evicted; host tiers off); embeddings in pinned host RAM; cards 32.5 GiB used each | — |
| 18:23-18:25 | Leak test `tuning/cache-switch.py coder-radiance 3` | **0 leaks**, own codeword every time. First token A 12.28 s (12,118 tokens: ~990 tok/s), B 13.94 s, **A again 4.06 s** (3,963 tokens read: resumes at the last 2,048-token chunk boundary). runs/2026-11-17-1825-cache-switch-coder-radiance.json | — |
| 18:25-18:29 | 4 conversations at once (~12K each, own codewords), first turn + follow-up, 3 rounds | **24/24 own codeword, 0 leaks** | — |
| 18:29 | `DEEP_N=8 tuning/cache-evict.py coder-radiance` (8 x 12K, then 8 x ~120K = ~960K tokens, more than its 756K-token pool) | running | — |
| 18:29-18:50 | `DEEP_N=8 tuning/cache-evict.py coder-radiance` | **25/25 own codeword, 0 leaks.** 8 x 12K follow-ups resumed (2.7-4.2 s); 8 x ~120K (≈960K tokens > the 756K pool): the first conversation was evicted and **re-read correctly** (124.5 s). runs/2026-11-17-1850-cache-evict-coder-radiance.json | — |
| 18:50 | `lab speed coder-radiance` | **103.3 writing / 967.7 reading 32K** (first token 33.6 s). Same day: `coder` (card 1) 54.0 / 935.7, `coder-2` (card 2) 58.9 / 988.5: writing **1.9x / 1.75x**, reading the same. Uses both cards (they use one). MLflow a0d7e60f (engine tags to correct) | — |
| 18:52-18:56 | `tuning/depth.py coder-radiance 32 64 128` | reading **969 / 943 / 897**, writing **118 / 114 / 130** tok/s at 30K / 62K / 123K (123K first token 137.5 s). Writing does not fall with depth (the guide's "200K prompt decode 1.7 tok/s" on v1.2.3 does not show on 2.1.0). runs/2026-11-17-1856-depth-coder-radiance.json | — |
| 18:57 | Tool use started: `lab quality coder-radiance --full --temperature 0.6` seeds 1-3, series coder-radiance-v1 | running | — |
| 18:57-19:31 | Tool use: `lab quality coder-radiance --full --temperature 0.6` seeds 1-3 | **96 / 93 / 94** (132, 129, 130 of 138; MLflow be309a70, 652101e1, e2470d6d); `coder` on 273 (11-16): 96 / 93 / 92: same range. **TC-45** (`tool_choice: required` runaway): Radiance passes 3 of 3, `coder` fails 3 of 3. Behaviour checks: no attacker echo, no invented query, no loop (only the TC-62 Q4 claim, as `coder`). Each run ~11 min (`coder` 27-31 min) | — |
| 19:30 | MLflow engine tags corrected on a0d7e60f, be309a70, 652101e1, e2470d6d (`lab manifest` described llama-swap's llama.cpp) | 200 each | — |
| 19:31 | zaphod back: requests to coder, coder-2, embed, health of rerank, stt, stt-whisper, tts | all 7 ready; Radiance stopped cleanly (launcher: ended, gone; 0 engines); cards 29.8 / 25.9 GiB | — |
| 19:32-19:34 | Baseline `tuning/depth.py coder 32 56` (coder's context is 64K) | reading 944 / 869, writing 49.0 / 48.0 at 30K / 54K. Radiance at the same depths: reading 969 / 943, writing 118 / 114 | — |
| 19:34 | Memory-sync cron line restored (crontab = the copy saved before the test). Watchdog: Daniel restarts it (sudo) | — | — |
| 19:35 | **Result.** Radiance 2.1.0 on both R9700s serves Qwen3.8-27B (IQ8R + DFlash2) at **~1.9x `coder`'s writing (103 vs 54) and keeps ~115-130 tok/s to 123K**, reading the same as llama.cpp on one card (~970), tool use in `coder`'s range with TC-45 fixed, 0 leaks in all three tests with its prefix cache on. Costs: it takes both cards (coder, coder-2, chat, helpers unload); zaphod's ReBAR off and no peer-to-peer limit it (its guide's 130-169 tok/s was with ReBAR). **Daniel decides** what next (e.g. Terminal-Bench, or real use as a both-cards coder). | | this commit |
