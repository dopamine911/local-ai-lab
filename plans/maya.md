# Plan: check Project Maya (bigger models on the same hardware)

Status: **to check later** (added 2026-10-30). Nothing installed or downloaded. Start after the UnbiasedStrata port's
marvin tests (plans/strix-strata.md), one change at a time.

## Why

The lab's goal is a fallback for frontier models. Today's best local model is Flash-Next (marvin `big`: tool use
91.7 on llama.cpp, 94.0 on Strata; Terminal-Bench 19/19). [Project Maya](https://github.com/mw00/project-maya) runs
much bigger models on hardware like ours, which could be a bigger step up in quality than another speed gain:

| Model | Size | Used per word | Smallest Maya file |
| --- | --- | --- | --- |
| GLM-5.3-Flash | 321B | 18B | Maya-S24, 94.7 GB (fits in marvin's 120 GB GPU memory) |
| MiniMax-M3.1-Flash | 428B | 23B | Maya-XS, 93 GB |
| DeepSeek-V4.1-Flash | 748B | not stated | Maya-XS, 277 GB (needs SSD streaming on marvin) |

## What it is (from its README and docs, 2026-10-30; not verified by us)

- Engine + server + browser dashboard; OpenAI- and Anthropic-compatible API (our test tools work unchanged).
- Engine started from Strata's and was rewritten per model. Keeps the most-used experts on the GPU, then RAM, then SSD.
- MIT licence. Repo created 2026-10-06, very active (pushed daily), 2.3k stars.
- Setup: `git clone` + `./setup.sh` builds locally (20-40 min) and downloads model files from Hugging Face.
  Needs **ROCm 7 on the host** for AMD (says it "installs nothing system-wide", but ROCm must already be there).
- Strix Halo (gfx1151) is supported as experimental. Their docs (`docs/AMD_MAYA.md`) record a Linux check on a
  Ryzen AI Max+ 395, 128 GB, **ROCm 7.14.1** (same as our port), Maya v1.1.5 `5403c222`, Maya-S24.

Reported speeds (users' numbers, GLM-5.3-Flash):

| Machine | Writing | Reading |
| --- | --- | --- |
| Strix Halo 395, 128 GB, Linux, ROCm 7.2.2, hand-tuned, Maya-S (their docs) | ~17.8 tok/s | ~210 tok/s (3-4K prompt) |
| Strix Halo 395, 128 GB (96 GB GPU), Windows, Maya-S | 18.3 | 320 |
| Gorgon Halo 495 (8065S), Windows, Maya-S, with MTP decode (on by default for gfx115x since v1.1.10) | 23-30 | 271-278 |
| 2x R9700, Ryzen 5 3600, 64 GB RAM, PCIe 4.0 x8 each, Maya-S24 (closest to zaphod) | 44.1 | 704 |

zaphod differs: card 1 is PCIe 3.0 x4 and RAM is 60 GB. A zaphod test also takes both cards from the daily models.

## Concerns (decide before starting)

1. **Host ROCm.** Our rule is no host toolchain changes: build in a container, as the port does
   (`docker/hip-gfx1151.Dockerfile` in the fork is a starting point: same ROCm 7.14.1).
2. **Version age.** The repo is under 4 weeks old; a commit at least 2 weeks old is very early. Pin a commit
   (e.g. v1.1.5 `5403c222` if old enough) or Daniel makes an exception, as for Strata 0.2.7.
3. **Model files** are Maya's own quantizations on one person's Hugging Face account, not signed. Download once into
   their own directory, record sha256 in `setup/models-marvin.txt`. Skip "Derisked" / "abliterated" variants (edited
   to refuse less).
4. **Thinking level.** Strata-derived: check the default thinking level and match llama.cpp's `big` (medium) before
   any Terminal-Bench run (the Strata lesson: xhigh took 13.3 h).
5. **Memory.** It pins most free RAM while running; on marvin nothing else runs then (same as the Strata test).
6. **Quality claim** ("keeps 97.7-99.2% of FP8 accuracy") is theirs; our tests decide.

## Steps (when started)

1. Read setup.sh and the engine's download/network code; confirm nothing downloads at start once files are local.
2. Container build for gfx1151 (pinned base, ROCm 7.14.1 by sha256, pinned Maya commit). Good: binary builds,
   `--check` passes inside the container.
3. Download GLM-5.3-Flash Maya-S24 (94.7 GB) on marvin, sha256 recorded.
4. Sealed run on marvin like `marvin/strata-test/strata-test.sh` (internal network + relay). Check power (130-140 W busy).
5. Measure, one at a time: `tuning/speed`, tool use x3 (same seeds), Terminal-Bench (Daniel starts long runs).
6. Record in this file, tuning/results.md, MLflow. Decide: keep, zaphod test, or drop.

## Could the UnbiasedStrata fork use it?

Probably as a **reference, not a merge**. Both are MIT, so copying code is allowed with the copyright notice kept.
But Maya's engine is rewritten for GLM/MiniMax/DeepSeek, while the fork runs Qwen (Flash-Next), so most of the
engine doesn't carry over. Parts worth comparing against our port (plans/strix-strata-port-notes.md):

| Maya piece | Why it may help the fork |
| --- | --- |
| `docs/AMD_UPSTREAM.md`: changes to the parts both inherited from Strata (`cmake/hip_backend.cmake`, `src/core/device.cu`, `include/strata/hip_compat/`, `intrinsics.hpp`, hipBLASLt loader) | Same files our HIP build uses; shows fixes already made on gfx1151/gfx1201 |
| Unified-memory sizing from `/proc/meminfo` MemAvailable, not HIP's GTT figure (112 GiB on a 128 GB box) | Our fork does the same in `device_free_bytes()` (6 GiB headroom); Maya uses 16 GiB headroom + 1 GiB reserve. Compare if our port hits out-of-memory |
| MTP decode on by default for gfx115x; fused prompt experts and `wmma2` attention on gfx1151 | Ideas to measure on the fork, one at a time, after the port matches upstream |
| `docs/AMD_EXPERT_KERNELS.md`: gfx1151 expert-kernel results (IQ3_XXS), derived from Strata's `moe_fused_iq.cu` | Same kernel family as the fork's |

Rule: take a piece only when a measured problem in our port needs it, one piece per commit, with Maya's licence
notice and the source commit in the message. Nothing upstream without Daniel's OK.
