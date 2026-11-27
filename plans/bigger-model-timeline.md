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
