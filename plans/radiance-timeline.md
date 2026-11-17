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
