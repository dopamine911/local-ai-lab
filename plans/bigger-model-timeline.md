# A model bigger than zaphod's cards, part of it read from the SSD?

Daniel 2026-11-25, after a video on tandem.cpp (a llama.cpp patch that keeps the most-used experts of a mixture-of-experts
model on the card, the rest in RAM and on the SSD): can we load a larger model and run a portion off the SSD?
Plain llama.cpp has the same kind of expert cache built in (`--moe-cache-mib`, `--moe-cache-cpu-share`; docs/moe-cache.md
upstream), and **our pinned build (e574a17) already has both flags** (`llama-server --help`, 2026-11-25).
Related, on hold: plans/maya.md (the same bigger models through the Maya engine, on marvin).

## Why zaphod and not marvin
- marvin: its 120 GB is one pool for the CPU and GPU, so there is no faster place for the hot experts; `big` already sits
  wholly in it. Nothing to gain.
- zaphod: 2 x 32 GB cards + 60 GB RAM (about 50 GB free) + PCIe 4.0 SSD (Sabrent Rocket Q4, 356 GB free on 2026-11-25).
  Roughly 105-110 GB of weights fit in cards + RAM (estimate: cards minus context and work buffers); the rest comes from
  the SSD.
- Expected weak spot: reading long prompts. The cache helps one word at a time; a big prompt batch needs nearly every
  expert, so everything off the cards comes over PCIe (card 1: 3.0 x4) or from the SSD. Probably fine for chat, slow
  for agents (30-100K prompts). Our estimate, to measure.
- While it runs it needs both cards: `coder`, `coder-2` and the helpers unload (as `gpt-oss`).

## Steps
1. Pick the model; check that our pinned llama.cpp knows its architecture. **Good:** a candidate clearly worth testing,
   known to our build (or to an official image through the launcher).
2. Download to zaphod. **Good:** sha256 OK, line in setup/models-zaphod.txt.
3. Run on zaphod alone: experts off the cards (`-cmoe`), `--moe-cache-mib` on both cards. **Good:** loads, sensible
   answers, the cache works on Vulkan (the docs do not say).
4. Measure from a cold start (file cache dropped): writing, 32K reading, time to first word. **Good for chat:** writing
   10 tok/s or more. **Good for agents:** a 32K prompt answered within about 2 min (`big` on marvin: ~70 s).
5. Only if step 4 passes: tool use x3 vs `big`.
6. Daniel decides.

## Step 1 result (2026-11-25)
Our build's architecture names come from the `llama-server` binary (e574a17, static); the models' from the
`architecture` field Hugging Face reads out of each GGUF. A name match means the code is there, not that it loads.

| Model | Total / per word | Licence | Our build | Files (unsloth unless noted) | On zaphod |
| --- | --- | --- | --- | --- | --- |
| **GLM-5.3-Flash** (Z.ai) | 321B / 18B | MIT | `glm5-next`: yes | UD-IQ3_XXS 120 GB, **UD-IQ4_XS 157 GB**, UD-Q4_K_XL 200 GB | IQ4_XS: ~50 GB from the SSD. Fits the disk |
| DeepSeek-V4.1-Flash-1017 | 748B in the file (552B + Engram tables) / 8B reading, 16B writing | MIT | `deepseek41`: yes | ISTA IQ2_XS 155 GB + Engram 111 GB = 266 GB; unsloth smallest UD-IQ1_S 233 GB | Mostly from the SSD; leaves ~90 GB disk free. 2-bit costs a lot: KL 0.15, top-1 88% (tandem.cpp's measurement) |
| MiniMax-M3.1-Flash | 426B / 23B | own ("other") | `minimax-m3`: yes | UD-IQ1_M 129 GB ... UD-Q4_K_XL 249 GB | Unsloth calls the M3 GGUF experimental (text only, without its sparse attention) |
| Kimi K3 | 2.8T / 104B | modified MIT | `kimi-k3`: yes | smallest GGUFs 466-594 GB | Does not fit the disk |
| Hy4-Flash (Tencent) | 220B / 11B | | **no** (`hy_v4_next_dfly`) | | Not in our build |

Quality vs `big` (Qwen3.8-Flash-Next), from what others publish (not ours):
- GLM-5.3-Flash: Z.ai's own Terminal-Bench 2.1 84.3, DeepSWE 63.4 (Qwen did not report those); Qwen leads on its own
  SWE-bench Pro 62.5. The only like-for-like independent number found (Artificial Analysis tau3-Banking): 47 vs 45.
  So probably better at agent tasks, not clearly smarter overall.
- DeepSeek-V4.1-Flash: stronger at full size, but at the 2-bit that fits it loses a lot (above); a r/LocalLLaMA test
  title says Qwen3.8-Flash-Next beat a 2.6-bit DeepSeek.
- From plans/maya.md (a user's report, Maya engine, not llama.cpp): GLM-5.3-Flash Maya-S24 (95 GB) on 2 x R9700 with
  64 GB RAM at 44.1 tok/s writing, 704 reading.

**Pick: GLM-5.3-Flash UD-IQ4_XS (157 GB).** Known to our build, MIT, 4-bit (good quality), and about 50 GB of it really
comes from the SSD, which is the question. UD-IQ3_XXS (120 GB) would mostly fit in memory and test less. 18B per word
(3x `big`'s 6B) makes it slower per token than `big`. Next: step 2 (157 GB download), on Daniel's go.

Side finding: the pinned build also has `inkling2` / `inkling2v` (not `inkling2_loop`, which b12648 has). Not tested;
plans/inkling-timeline.md assumed it did not know the model. The launcher route still stood: the loop-scale fix
(#32704) is newer than e574a17.

## Timeline
| When (CST) | What | Result |
| --- | --- | --- |
| 2026-11-25 12:59 | Video checked (title, description, Daniel's pasted transcript) against tandem.cpp's README and llama.cpp's docs | Numbers are the author's; built-in cache in our build; prompt reading is the weak spot |
| by 13:06 | Step 1 (architecture names, file sizes, published benchmarks) | GLM-5.3-Flash UD-IQ4_XS picked (above). Waiting on Daniel for step 2 |
| 13:07 | Daniel: yes, start the download | — |
| 13:08 | Step 2: 5 lines in setup/models-zaphod.txt (unsloth rev a38483c, sha256 from Hugging Face), `lab models zaphod glm-5.3-flash` in tmux `glm-dl`. Disk before: 356 GB free | downloading at ~45 MB/s, about 1 h |
| 2026-11-25 13:08-14:23 | Step 2 done (`lab models zaphod glm-5.3-flash`) | **All 5 files pass sha256 (MODELS-OK)**, 157 GB in 75 min (~35 MB/s). Disk 210 GB free after. Step 3 (first run, needs both cards) waits for Daniel |
| 2026-11-27 11:27 | Daniel: start step 3 now | — |
| 11:28 | Memory-sync cron line paused (its `embed` call would unload a run-alone model; crontab saved first; restore after the test). Test model `glm-flash` added (both cards, card 2 first, `-cmoe --moe-cache-mib`, 32K, 1 slot, temp 1.0 / top-p 0.95 as Z.ai's evals; runs alone; off in `image-mode` and `training`). Waited for `coder`, `coder-2`, marvin `big` idle 5 checks, saved 11:28:59 | Reload clean |
| 11:29-11:37 | Load tries with `--moe-cache-mib` 32768, then 24576 | **Both failed** (after ~2.5 min each): "failed to allocate buffer for MoE cache", one Vulkan buffer of 4.64 / 4.30 GB over the card's ~4 GB limit. Cause (pinned source, src/llama-moe-cache.cpp, + the file's tensor list): the cache keeps each expert weight kind in one block per card; `ffn_down` (IQ4_XS, 4.46 MB per expert, 39 of 43 layers) is the biggest. File: 147.1 GB of experts, 9.7 GB rest; 288 experts, 8 used per word, 46 layers |
| 11:37-11:45 | `--moe-cache-mib 16384` (~2.9 GB blocks) | **Loads** in 2 min 45 s. "17 x 23": **391, correct** (short sensible thinking). Cold start: **0.72 tok/s writing** (177 tokens in 245 s), 33-token prompt read in 64 s. RAM page cache 56 GB |
| 11:45 | GLM swapped out right after the answer | **My own fault**: my busy check asks `/upstream/<model>/slots`, and llama-swap loads a model that is not running to answer it (also the `coder`/`coder-2` starts between the failed tries). The watchdog's skip works (no `embed` call while GLM was loaded). Busy checks from now: only models in `/running` |
| 11:46-12:32 | Warm test: 3 short questions in a row (the cache and the RAM page cache filling) | **0.68 tok/s** (prompt 33 tokens in 67 s), then **0.94 tok/s** (14 tokens in 25.5 s); the 3rd (a Python function, up to 1,500 tokens) passed my 30-min client limit and was cancelled. Answers correct and sensible (391; Seine/Paris, Loire/Orléans, Rhône/Lyon). Cards 13.6 + 18.2 GB used (~30 GB free), page cache 55 GB. **Step 3: loads and answers, the cache works on Vulkan up to ~16 GB; step 4's bar (10 tok/s) is out of reach: ~1 tok/s, `big` 40.** Why (estimate): each word needs 8 experts in each of 43 layers, ~4 GB of weights; the 16 GB cache holds ~11% of the 147 GB of experts, the page cache ~37%, so every word still reads GBs from the SSD and over PCIe |
| 12:33 | Back to normal: `coder`, `coder-2`, `embed` answer; memory-sync cron line restored (crontab = saved copy). `glm-flash` stays in the config (unused) until Daniel decides | **Stopped: Daniel decides** |
| 12:34-12:41 | Daniel asked why the cards were half empty, whether we use tandem.cpp (no: official pinned llama.cpp, its built-in cache), and DeepSeek vs GLM (not yet: same wall, 2-bit, does not fit the disk next to GLM). Then: tune GLM first | — |
| 12:41-12:51 | Memory-sync cron line paused again (the 12:41 run had started; waited for it to finish, 12:51) | — |
| 12:51-13:13 | Tuning 1: the experts of 8 whole layers on the cards (`-ot`: 3-6 on card 2, 42-45 on card 1, the rest CPU), cache 16384 unchanged. Busy check now asks only models in `/running` | Loads in ~3 min, cards **56.1 of 68.4 GB** (card 2 31.0 of 34.2: near full). Same 3 questions: **0.92 / 1.39 / 1.14 tok/s** (prompt 33 tokens in 98 / 48 s), was 0.68 / 0.94: about +40%. Answers correct |
| 13:13 | Tuning 2: `--moe-cache-cpu-share 0.5` (half the misses computed on the CPU from RAM), nothing else changed. Speech helpers were loaded when I saved (13:13; GLM had been swapped out by a speech request); reload anyway | Cards 55.5 GB. **1.32 / 1.90 / 1.38 tok/s** (prompt 33 tokens in 42 / 40 s): +20-40% over tuning 1, best of the day. Answers correct |
| 13:32 | Back to normal: `coder`, `coder-2`, `embed` answer; memory-sync cron line restored (crontab = saved copy). `glm-flash` stays in the config with both tunings (unused) | **Tuned: ~1.3-1.9 tok/s, about 2x the first setting but 5x under the 10 tok/s bar and ~25x slower than `big` (40). Daniel decides** |

| 13:33 | Daniel: drop it, push the commits | — |
| 13:34-13:36 | `glm-flash` out of the llama-swap config first (now identical to the config before the test; idle 5 checks, saved 13:34:27, reload clean, not in `/v1/models`), then the 5 files deleted (no process had them open) and their 6 lines out of setup/models-zaphod.txt | Disk 210 → **356 GB free**. **Dropped.** Learned: llama.cpp's expert cache works on our Vulkan cards up to ~16 GB (4 GB buffer limit per block); a 157 GB model with 18B per word runs at 1.3-1.9 tok/s on zaphod. Worth retrying only for a model that needs far less per word (the video's case: 6B per word at 3 bits) |
