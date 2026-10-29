# Plan: test Strata (Qwen3.8-Flash-Next) on zaphod

Status: waiting. Files: /srv/models/strata/qwen3.8-flash-next-gsq-rco/IQ3_XXS (done), IQ3_S (downloading).
Image: ghcr.io/niko1221/strata:0.2.7-rocm-gfx120x (pulled). Needs both cards: llama-swap paused while it runs.

## Test rules
- Record free memory on both cards before every Strata start. UnbiasedStrata (an NVIDIA-only rewrite,
  github.com/robertkarl/UnbiasedStrata) reports that upstream Strata's output can depend on free VRAM at startup.
- Several seeds per test, not one (Ornith's spread between seeds was 89-93).
- Same tests as the other models: speed, tool-use full suite + hand sort, Terminal-Bench smoke (user runs it).
- Write the plan steps out for approval before pausing llama-swap.

## UnbiasedStrata (github.com/robertkarl/UnbiasedStrata)
- README covers NVIDIA only, but the code is a PolyStrata fork and keeps its AMD HIP backend: docs/AMD_HIP.md names
  the Radeon AI PRO R9700 (gfx1201) and multi-card splits (`--gpus 1,0`). Its own `make build` is CUDA-only, so using
  it here means wiring cmake/hip_backend.cmake into that build and testing on gfx1201 (estimate: a day or two).
- A Vulkan port would mean rewriting ~93 CUDA kernel files as Vulkan shaders: weeks, and no gain over HIP on R9700.
- Self-reported 10-18% faster than upstream on an RTX 3090 (Flash-Next). Try only after the upstream Strata test,
  with the same seeded tests.
