# marvin (Strix Halo): ways to more speed (review, 2026-11-09)

Daniel asked for a review while gpt-oss-120b downloaded. No changes made. Our own measurements come first; outside
numbers are other people's machines (web research 2026-11-09; links below), so read them as ranges. Each item: one
change, `lab speed` before and after, only when marvin is free.

## Ranked by expected gain

| # | Idea | Expected gain | Confidence | Cost / risk | Our evidence |
| --- | --- | --- | --- | --- | --- |
| 1 | Fix agent re-reading: find why `big` makes no context checkpoints | biggest in agent use: re-reads took 33 of 60 min in one hour | 0.6 | none to look; may need a flag change | plans/marvin-writing-speed.md theory 7 |
| 2 | Other engine (Strata / our fork) for `big` | 32K reading 315 → 822 (2.6x), writing 44 → 49.5-53.5 | 0.9 (measured here) | another engine to run; plans/strix-strata.md step 6 waiting | plans/strix-strata.md |
| 3 | Newer Mesa (pinned 26.0.8) | reading +17%, writing +3% (one report, 25.2 → 26.1) | 0.4 for us (we are already past 25.3's big reading fix) | image rebuild; rollback easy | none yet |
| 4 | `-ub 1024` / `2048` for `big` (default 512) | reading: unknown on our models; one 2025 report found 1024 best on RADV | 0.5 | memory for bigger buffers | CLAUDE.md next step 3 |
| 5 | Conversation memory q8_0 (`-ctk q8_0 -ctv q8_0`) | writing at 60K+ (less memory read per token) | 0.6 | small quality change: needs `lab quality` too | theory 4 |
| 6 | `-kvu` off (`-c 262144 -np 2` without it) | ~10% writing if the shared pool is the cause | 0.55 | same memory | theory 2 |
| 7 | `amd_iommu=off` (boot) | memory reads ~+6%, writing <2% | 0.6 | loses the NPU and VM passthrough; reboot | SETUP.md row 6 left it for later |
| 8 | `tuned` accelerator-performance profile | reading +5-8% | 0.4 (single reports, Fedora) | one package | none |

## Not worth it (for us)

- **ROCm instead of Vulkan in llama.cpp**: faster on short prompts but loses badly at long context (e.g. writing
  46 → 17 tok/s at 76K vs Vulkan 33 → 29). Our use is long agent conversations. hipBLASLt: ~0% in llama.cpp.
- **AMDVLK**: discontinued by AMD (2025-09-15); RADV is now faster at reading.
- **More GPU memory settings / BIOS UMA size**: they set how much memory the GPU may use, not speed. Ours are already right (512 MB UMA, 120 GB GTT).
- **Power**: already BIOS Performance (120 W sustained, 140 W for 2 min); memory clock is fixed. Keep the watchdog for the 55 W fault.
- **EAGLE3 for gpt-oss-120b**: on another machine (DGX Spark) it ran 0.83-1.08x (often slower); no Strix number. Cheap to try later, expect little.

## For judging the gpt-oss-120b result

Other people's Strix Halo, llama-bench, RADV: reading pp512 727-772, pp32768 478; writing tg128 ~55-56 tok/s
(2026, Mesa 26.0-26.2, b9049-b11146). Our `lab speed` is a server test (400-token answer, 32K prompt), so expect
somewhat lower; writing well under ~45 would be worth a look.

## Sources (confidence of each claim in the research notes)

- AMDVLK end: gamingonlinux.com/2025/09/amdvlk-has-been-discontinued-as-amd-are-throwing-their-full-support-behind-radv
- RADV vs AMDVLK, gpt-oss numbers: github.com/hogeheer499-commits/strix-halo-guide (README, BENCHMARKS.md)
- ROCm vs Vulkan at long context: blog-9hu.pages.dev/posts/strix-halo-full-context-decode-drops/ ; github.com/lhl/strix-halo-testing
- hipBLASLt, GTT settings: strixhalo.wiki/AI/llamacpp-performance , strixhalo.wiki/AI/AI_Capabilities_Overview
- amd_iommu: community.frame.work/t/tracking-will-the-ai-max-395-128gb-be-able-to-run-gpt-oss-120b/73280?page=2
- Mesa update: hardware-corner.net/llama-cpp-amd-radv-vulkan-driver-update/
- Bosgame M5 power modes: wccftech.com/review/bosgame-m5-review-the-cheapest-128-gb-strix-halo-mini-pc/
- EAGLE3 in llama.cpp: github.com/ggml-org/llama.cpp/pull/18039 (and #25794)
