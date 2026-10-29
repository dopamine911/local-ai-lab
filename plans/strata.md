# Plan: test Strata (Qwen3.8-Flash-Next) on zaphod

Status: first test done 2026-10-29 (results: tuning/results.md, "Strata" section). Files:
/srv/models/strata/qwen3.8-flash-next-gsq-rco/IQ3_XXS (71 GB, tested), IQ3_S (downloaded, not tested).
Image: ghcr.io/niko1221/strata:0.2.7-rocm-gfx120x. Needs both cards: llama-swap paused while it runs.

## Results 2026-10-29 (IQ3_XXS, free VRAM before each start: 57 MB used on both cards)
| Setting | Cards used | Writing tok/s | Tool-use score | Full suite |
| --- | --- | ---: | --- | ---: |
| `GPUS=all` | card 1 only (engine's choice) | 65 | 86 / 92 / 86 of 138 = **88 / 95 / 87** on the 130-point basis the llama.cpp models use | 33 min |
| `GPUS=0,1` | both, main = card 2 | 85 | 90 of 138 = **92** on the 130-point basis | 12 min |
Same model on marvin (llama.cpp, IQ4_XS): 90 / 93 / 92, so quality is the same on both servers.
Always set `GPUS=0,1`; `GPUS=all` silently uses one card. Strata rejects the JSON-schema request format (about 4 points
per run). Test scripts: copies kept in plans/strata-test/ (they stop llama-swap and always restart it at the end).

## Possible next tests (one at a time)
1. Two cards, two more seeds at temp 1.0 (is 90 steady?).
2. `GPUS=1,0` (main card = card 1) to split the speed gain between "second card" and "main card on x16".
3. IQ3_S (bigger, better quality) on two cards.
4. A long-prompt reading-speed test.

## Test rules
- Record free memory on both cards before every Strata start. UnbiasedStrata (an NVIDIA-only rewrite,
  github.com/robertkarl/UnbiasedStrata) reports that upstream Strata's output can depend on free VRAM at startup.
- Several seeds per test, not one (Ornith's spread between seeds was 89-93).
- Same tests as the other models: speed, tool-use full suite + hand sort, Terminal-Bench smoke (user runs it).
- Write the plan steps out for approval before pausing llama-swap.

## UnbiasedStrata (github.com/robertkarl/UnbiasedStrata)
Next: plans/strix-strata.md (Strata on marvin sealed off, then port UnbiasedStrata to AMD gfx1151, PR upstream).
Correction 2026-10-29: its AMD/Strix Halo docs are inherited from upstream, not wired into its build; its README
is NVIDIA-only, and it does not claim a determinism fix (output can still depend on free VRAM at start).
- README covers NVIDIA only, but the code is a PolyStrata fork and keeps its AMD HIP backend: docs/AMD_HIP.md names
  the Radeon AI PRO R9700 (gfx1201) and multi-card splits (`--gpus 1,0`). Its own `make build` is CUDA-only, so using
  it here means wiring cmake/hip_backend.cmake into that build and testing on gfx1201 (estimate: a day or two).
- A Vulkan port would mean rewriting ~93 CUDA kernel files as Vulkan shaders: weeks, and no gain over HIP on R9700.
- Self-reported 10-18% faster than upstream on an RTX 3090 (Flash-Next). Try only after the upstream Strata test,
  with the same seeded tests.
