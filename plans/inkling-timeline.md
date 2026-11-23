# Inkling 2 Mini as `coder-2`? (#67)

Daniel 2026-11-21: look at replacing `coder-2` with Inkling 2 Mini. Plan written and step 1 started on his go.

## What it is
- Thinking Machines Inkling 2 Mini, released 2026-11-20, Apache 2.0 (https://huggingface.co/thinkingmachines/Inkling-2-Mini).
- Mixture of experts: 26.2B weights, 3.4B used per token (5.4B counting the loop: layers 6-17 run 3 times per token).
  Text and images in, up to 1M context. Recommended sampling: temperature 1.0, top-p 1.0, top-k off, min-p 0.
- Only 4 of its 24 layers keep the whole context (the rest look back 1,024 tokens). Estimate: about 4 GB of cache at
  128K 16-bit, so 128K should fit on card 2 next to the speech models. To check in step 3.
- Their own table against our `coder` model, Qwen3.8-27B (their numbers, not ours):

| | Inkling 2 Mini | Qwen3.8-27B |
| --- | --- | --- |
| SWE-bench Verified | 79.2 | 77.8 |
| Terminal Bench v2.1 (in pi) | 71.6 | 73.0 |
| AutomationBench (tool use) | 41.8 | 48.2 |
| JobBench | 49.6 | 33.4 |

- Their model card warns: with tools it "may take destructive actions, such as deleting or overwriting files, without
  confirmation, and may not report the outcome of its work accurately".

## What stands in the way
1. Our llama.cpp (e574a17, both machines) does not know the model. Support merged 2026-11-20 (#32694, in b12600);
   a fix for files that keep the original 4-bit experts merged 2026-11-21 11:10 UTC (#32704; without it the 2nd and
   3rd loop passes lose their scales: perplexity 130 instead of 7.65, gibberish). At 18:53 UTC no release had it yet
   (b12608 was built from a commit 12 minutes before it).
2. New and moving: open bugs on 2026-11-21: `reasoning_effort: none` ignored (#32753), raw grammar requests come back
   empty (#32745). llama.cpp reads its tool calls (`common/parsers/inkling.cpp`).
3. No speculative decoding: its draft method (DSpark) is not in official llama.cpp (#32218 open). `coder-2` has MTP.
4. Updating llama.cpp for every model is a separate, big change. So: run Inkling alone in its own container through
   the engine launcher (`engines/zaphod.json`), official llama.cpp Vulkan image pinned by digest, as `big-strix` and
   `coder-radiance`. Every other model stays on the pinned build (standing policy: official builds only).

## File
unsloth `Inkling-2-Mini-GGUF` rev ae9327c, `Mainline_Support/Inkling-2-Mini-UD-Q4_K_XL.gguf_file` (17.6 GB; saved as
`/srv/models/inkling/Inkling-2-Mini-UD-Q4_K_XL.gguf`) + `mmproj-F16.gguf` (0.64 GB, images). Same tensors as their root
files in the layout official llama.cpp reads; keeps the original 4-bit experts (their KL vs their 8-bit: 0.0107).
Not chosen: ggml-org IQ4_NL (17.6 GB, works without #32704 but re-quantizes the experts; KL 0.0095 vs 16-bit),
Q8_0 (27.8 GB, too big for card 2 next to speech).

## Steps
1. Download the file and `mmproj` (`lab models zaphod inkling`; lines in setup/models-zaphod.txt). Wait for a
   llama.cpp release with #32704 and its Vulkan server image. **Good:** both files pass the sha256 check; a release
   tag that contains commit 690b87d.
2. Engine `inkling` in `engines/zaphod.json` (image by digest, `/srv/models/inkling` read-only, card 2 only) and a
   test model `inkling` in llama-swap on card 2, in `coder-2`'s swap group. 128K 16-bit, 1 slot, its sampling,
   `--cache-ram 0 --no-cache-idle-slots` (llama.cpp#27148). Reload only when zaphod and marvin are idle.
3. Sanity: a few plain questions and one tool call; card 2 memory with speech loaded. **Good:** sensible answers
   (no gibberish), card 2 at least 1 GB free.
4. Speed: `lab speed inkling`, `tuning/depth.py inkling 32 56 120`. **Good:** writing above `coder-2`'s 58.5 tok/s,
   32K reading at least 946 (`coder-2`, 128K q8_0, 2026-11-20).
5. Tool use: `lab quality inkling --full --temperature 1.0` seeds 1-3. **Good:** 92 or more each (`coder` 93/93/93).
6. Terminal-Bench, all 19 tasks. No llama.cpp `coder` run on zaphod to compare (#59); references: `coder-radiance`
   17/19 in 3.05 h, marvin `big` 19/19.
7. Leak test (`tuning/cache-switch.py`). **Good:** 0 leaks.
8. Daniel decides: replace `coder-2` (the subagents' model), keep `inkling` as an extra model, or drop it.

Stop and ask Daniel if step 3 fails, or if speed or tool use misses its bar.

## Timeline
| When (CST) | What | Result |
| --- | --- | --- |
| 2026-11-21, before 12:53 | Model card, llama.cpp PRs, files checked; plan to Daniel | Daniel: write the plan, start step 1 |
| 12:53 | Step 1: plan, issue #67, file lines in setup/models-zaphod.txt (rev ae9327c); `lab models zaphod inkling` started | downloading |
| 12:54 | Official image: `ghcr.io/ggml-org/llama.cpp:server-vulkan-<build>` (Ubuntu 26.04 base), built once a day at 04:12 UTC (22:12 CST). Latest `server-vulkan-b12602` (built 2026-11-21 04:16 UTC from 418fe97): has #32694, **not #32704**. The 2026-11-19 and 11-20 nightly runs failed | first image with the fix: tonight's nightly, if it succeeds. Check with the image's `org.opencontainers.image.revision` containing 690b87d |
| 12:53-13:02 | Download done (`lab models zaphod inkling`) | **both files pass sha256** (MODELS-OK): model 17,629,760,736 bytes (6.8 min), mmproj 639,348,160. Step 1 waits only for the image now |
| 2026-11-23 09:13 | Image check | **`server-vulkan-b12648` has the fix**: built 2026-11-23 06:08 UTC from 44d5087 (37 commits after 690b87d), digest sha256:298c82bf5083…e4b0; nightlies 11-22 and 11-23 succeeded. **Step 1 done.** Fixed since 11-21: #32753 (`reasoning_effort: none`), #32745 (raw grammar). Still open and worth watching: #32778 (prompts over ~96 tokens give all-'?' output; seen with CPU-only or experts partly on the CPU, analysed as op-offload in #32710; we run all on the card, check in step 3 with a long prompt), #32707 (slow reading on CUDA; a community fork claims 4x, not for us: official builds only), #32836 (`--reasoning-budget` can think forever: we don't use it), #32739 (no tensor split: one card only, as planned), DSpark #32218 still open | next: step 2, on Daniel's go (it reloads llama-swap) |
| 09:14 | Daniel: yes (step 2) | — |
| 09:15-09:17 | Step 2: image pulled by digest (llama.cpp build 12648, 44d5087; Mesa 26.0.8 as ours); engine `inkling` in engines/zaphod.json (only /dev/dri/renderD129 = card 2, user nobody, read-only, `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1`; `lab-engined.py check` OK); test model `inkling` in llama-swap (swap sets `one_per_card` and `teachers`, off in `image-mode` and `training`). Waited for `coder`, `coder-2`, marvin `big` idle 5 checks in a row, saved 09:17:54 | Reload clean, all models back (stt/tts preload 404: known). Commit d1d24fc |
| 09:19-09:20 | Step 3: load + sanity (`coder-2` idle first) | Loaded in ~29 s (`coder-2` swapped out). "17 x 23" → 391 (thinking short and right). 3,292-token prompt (CLAUDE.md, #32778 check): correct two-sentence answer, no '?' output; reading 877 tok/s, writing 34.6. Tool call: `get_weather({city: Lyon, unit: C})`, then a correct answer from the result. Card 2 (sysfs, incl. speech + screen): 27.1 of 34.2 GB used vs 27.8 with `coder-2` (**7.1 GB free**). "Failed to allocate pinned memory" warning: also in our own llama-swap container (memlock 8 MB in both), so equal. Shader cache not saved (read-only, no home): slower first start only. **Passes.** Hint: writing 34.6 is well under `coder-2`'s 58.5; step 4 measures it properly |
| 09:21-09:27 | Step 4 (speed part): `lab speed inkling` | **41.1 writing / 462.9 reading "32K"** (shorts 40.3-41.1, longs 461-464), vs `coder-2` 58.5 / 946.6: **-30% / -51%, both bars missed.** Its tokenizer makes the same text 42,076 tokens (Qwen ~32K), so for the same text the first token comes after **90.9 s vs 34.3 s** (2.7x). No draft (DSpark not upstream); `-ub` default 512. runs/2026-11-23-0921-speed-inkling.json, MLflow 8476b64d (engine tags corrected: image b12648, 44d5087). Depth test not run. Upstream speed work still open: #32707 (slow reading, ~5,600 small kernels), PRs #32855, #32883, #32740. **Stopped: Daniel decides** |
