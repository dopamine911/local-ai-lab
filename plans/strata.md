# Plan: test Strata (Qwen3.8-Flash-Next) on zaphod

Status: waiting. Files: /srv/models/strata/qwen3.8-flash-next-gsq-rco/IQ3_XXS (done), IQ3_S (downloading).
Image: ghcr.io/niko1221/strata:0.2.7-rocm-gfx120x (pulled). Needs both cards: llama-swap paused while it runs.

## Test rules
- Record free memory on both cards before every Strata start. UnbiasedStrata (an NVIDIA-only rewrite,
  github.com/robertkarl/UnbiasedStrata) reports that upstream Strata's output can depend on free VRAM at startup.
- Several seeds per test, not one (Ornith's spread between seeds was 89-93).
- Same tests as the other models: speed, tool-use full suite + hand sort, Terminal-Bench smoke (user runs it).
- Write the plan steps out for approval before pausing llama-swap.

## Not usable here
- UnbiasedStrata: CUDA only (Pascal or newer); no AMD/ROCm/Vulkan. Self-reported 10-18% faster on an RTX 3090.
