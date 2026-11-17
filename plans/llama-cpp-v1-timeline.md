# llama.cpp v1.0.0: timeline

Pinned now: llama-swap image `unified-vulkan-269` = llama.cpp b12020 (commit 0ec4027), both machines.

## 2026-11-16: checks (nothing changed on either machine)
1. llama.cpp v1.0.0 released 2026-11-13 = build b12359 (commit b1b770d), 339 commits after our b12020.
   Notes: https://github.com/ggml-org/llama.cpp/releases/tag/v1.0.0
2. Grammar bug (tools + required JSON format, upstream #31342): **fixed in 1.0** by PR #31956
   (merged 2026-11-10, same change as our closed #31395). Not yet tested on our models.
3. Cache leak #27148: **still open**. Keep `--cache-ram 0 --no-cache-idle-slots`.
4. Removed in 1.0: `--defrag-thold` and 35 old library functions. Our configs do not use them.
   Every llama-server flag in zaphod and marvin configs is still in v1.0.0 `common/arg.cpp`.
5. Newest llama-swap image: `unified-vulkan-273` (2026-11-15) = llama.cpp v1.0.0 + 48 commits
   (commit e574a17, 2026-11-15), so not exactly 1.0. It also moves the llama-swap program 269 -> 273
   (new "fit" selector, UI changes, 1xx pass-through fix).

## 2026-11-16: feature review, b12020 -> image 273 (read from PR descriptions and v1.0.0 source; nothing tested)
Covers v0.9.0 notes (the 90 commits after our build), v1.0.0 notes, and the 48 commits only in image 273.
The 1.0 blog post (huggingface.co/blog/ggml-org/llama-cpp-v1) is about versioning only; no new features in it.

### A. CPU offloading: the MoE expert cache
What it is: for MoE models whose experts sit in system RAM (`-cmoe` / `--n-cpu-moe N`), a GPU cache keeps the
most-used experts on the card. On a miss the expert is uploaded, or (new) the CPU computes it.
- Already in our b12020: `--moe-cache-mib N` (cache size, split over cards like `-ts`), old SLRU policy (#30637, 2026-10-19).
  So a cache baseline can be measured on 269 first: one change at a time.
- New in 1.0:
  - `--moe-cache-cpu-share F` (0..1, default 0): share of misses the CPU computes instead of uploading (#30966).
  - Better replacement policy, ClassLRU (#31233): author measured +6-7% writing vs SLRU (CUDA).
  - CPU no longer waits on cached experts (#31888): +7-9% writing on Gemma 4 26B-A4B (CUDA); no change on QFN.
  - The draft model no longer gets the cache (#31497): fixes a crash with `--moe-cache-mib` + a dense draft.
  - Tuning script `scripts/moe-cache-tune.py` (in the source, likely not in the image): tries cache sizes and
    shares, prints the best `--moe-cache-mib` / `--moe-cache-cpu-share`.
  - Auto mode (`--moe-cache-cpu-share auto`, #31286): closed, NOT merged.
- Limits (from the PRs):
  - Writing only: prefill skips the cache (#31233). It will not fix the slow reading we measured with experts
    in RAM (`--n-cpu-moe 8`: 180-366 tok/s reading, 2026-11-09).
  - Every benchmark in the PRs is CUDA. The cache itself runs on Vulkan (#31497 log), but CPU share on Vulkan
    is not shown anywhere. Expect nothing until measured.
- Where it could apply for us:
  - marvin: nothing. GPU and CPU share the same memory, so there is nowhere to offload to.
  - zaphod dense models (`coder`, `chat`, ...): nothing; they fit on the cards. (`--n-cpu-ffn` for dense models
    already exists in b12020.)
  - zaphod `gpt-oss` (test only) at 128K with f16 KV: it does not fit the cards, so some experts must sit in RAM.
    zaphod has 60 GB RAM, too little for all ~60 GB of experts, so it would be a partial `--n-cpu-moe N` plus
    `--moe-cache-mib`. Today's 128K options: `--fit` with a RAM part 54.5 writing / 1,423 reading, or all on
    cards with q8_0 KV 82.9 / 2,278 but tool use 90 (f16 gets 97). Good result: better than 54.5 writing
    with tool use 97. Reading will likely stay slow (see the limits above).
  - Upload links matter: card 1 is PCIe 3.0 x4 (slow uploads), card 2 x8. CPU is DDR4-2666 dual channel.

### B. Vulkan speed (both our GPU types: R9700 = RDNA4, marvin 8060S = RDNA3.5)
- In 1.0: RDNA3/4 split-K for small batches (#30146: up to +20% reading at small ubatch; MTP draft 3 unchanged,
  draft 8-16 +13%) and for large tiles (#30280: some matmuls +10-25%).
- Only in image 273: RDNA3/4 prompt-processing flash attention (#31323, tested by its author on Strix Halo and
  R9700, the same chips as ours) and narrow-N matmul (#31990, tuned on NVIDIA).
- #30146/#31990 change small-batch speed, so the best MTP draft count may move. Re-tune later as its own change.
- Vulkan tensor-parallel allreduce (#25051, 2 devices) for `-sm tensor` (still marked EXPERIMENTAL). Could matter
  for zaphod's two-card models (`coder-long`, `gpt-oss`). Never tested here.
- ReBAR detection (#31658): Vulkan now detects missing ReBAR and avoids the small BAR itself, and prints
  `rebar: 0/1` in the device line. Our `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1` is still read (ggml-vulkan.cpp
  line 4269). Keep it (Do not undo); the new device line is a free check.

### C. Multi-slot models
- `flash_attn_ext_rows` (#29510): faster unified KV with several slots, but CUDA only. Vulkan says "not supported"
  (ggml-vulkan.cpp line 16048), and llama.cpp then falls back by itself to today's path (load-time check in
  llama-context.cpp; log line "Flash Attention over K/V rows not supported, set to disabled"). So: no gain,
  no harm expected for `big` (2 slots), `chat-teacher` (4), `coder-long` (2). Check that log line once.
- Reference-counted checkpoints (#30137): less RAM and faster cache updates in llama-server. It does NOT fix the
  cache leak #27148.

### D. Correctness and chat fixes we care about
- Grammar bug fixed (#31956, see above). Also `response_format` honored in the gemma4 parser (#31915).
- Gemma 4 tool call inside an unclosed thought no longer swallowed (#32232, image 273 only): `chat`.
- gpt-oss: constraint after the final channel allowed (#31556).
- KV-shift fix (#31683), on-device state copy sizes for quantized KV fixed (#30541: we use q8_0 KV).
- MTP: hidden state saved in checkpoints (#30141), read from a device buffer (#31927).
- `--fit` NextN layer fix (#31569): this is about MTP models leaving layer 0 on the CPU, NOT the gpt-oss
  "part of 1 layer in RAM" we saw.
- Server: connection reuse after streamed responses (#30492, image 273 only).

### E. Behaviour changes to check after an upgrade
- Image URLs pointing at private/LAN addresses are now refused unless `--media-allow-private-urls` (#32115).
  Open WebUI normally sends images inline, but test one image request to `chat` or `ornith`.
- CORS / DNS-rebinding checks (#32098, #32113): only bite with `--cors-origins`, `--tools` or MCP; we use none.
- `--defrag-thold` gone; not used by us.

### F. Not useful for us
- FP8 KV cache (#28392, `-ctk/-ctv f8_e4m3`): no FP8 KV type in the Vulkan backend's flash attention (KV types
  allowed: f32, f16, bf16, q8_0, q5_x, q4_x, iq4_nl). Not found in Vulkan, untested.
- `--shard` / `--ple` (#31185): load PLE/engram parts from separate files; only for split downloads.
- New models (Kimi-K3.1, Ripple-65B, xing4_0, drama-3 TTS), CUDA/Metal/SYCL/Hexagon work, web UI redesign.

### G. llama-swap 269 -> 273 (program, comes with the image)
- v270: fixes memory growing across config reloads (#1297; we reload often); `/v1/realtime` proxy.
- v272: 1xx responses passed through, so upstream errors no longer arrive as 200 for big bodies (#1319);
  `/props?autoload=false`.
- v273: new `fit` selector (pick a model variant that fits free memory); UI changes.
- Not read for this review: anything about the 30 s peer reload cut (CLAUDE.md); assume still there.

## Possible next steps (need Daniel's go-ahead)
1. Upgrade test on zaphod first, `:269-media` kept for rollback; measure before/after:
   `tuning/grammar-repro.py`, `lab speed` (coder), tool-use quality, one image request, multi-slot speed
   (`chat-teacher`), and the "K/V rows ... disabled" log line.
2. Later, as its own test: MoE cache for `gpt-oss` at 128K f16 on zaphod. Baseline `--moe-cache-mib` on 269
   first (exists there), then 1.0's `--moe-cache-cpu-share`. Needs the cards alone (watchdog paused).
3. Later: re-tune MTP draft count after the upgrade (#30146 changes small-batch speed).

## 2026-11-16: step 1, zaphod 269 -> 273 (America/Chicago time)
Plan: before on 269 (fresh: the 2026-11-02 numbers predate the cache-leak flags), swap image, same tests after.
Tests: grammar-repro (coder), `lab speed coder`, `lab quality coder --full --temperature 0.6` seeds 1-3,
chat-teacher `prompt_check.py --schema L1 --workers 4` (multi-slot, was ~147 calls/min), one image request to `chat`.
Good result: speed within ±3%, tool use within the seed spread, grammar variants 3-5 OK on 273.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 12:50 | zaphod idle (only helpers loaded, no requests in 15 min); marvin no peer traffic | start | — |
| 12:50 | Before on 269: `tuning/grammar-repro.py coder` | variants 3, 4, 5 give 400 "failed to parse grammar"; 1, 2, 6, 7, 8 OK (bug present, as expected) | — |
| 12:51 | Before on 269: `lab speed coder` | 54.0 writing / 823.3 reading 32K (runs/2026-11-16-1251-speed-coder.json, MLflow 6e1bf84a) | — |
| 12:53-13:44 | Before on 269: `lab quality coder --full --temperature 0.6` seeds 1-3, series `coder-ls269-v1` | 96 / 94 / 92 (MLflow b959407a, 98b3d012, 2297d6be). Grammar 400s TC-65, 66, 67, 69 in every run (4 per run, as 2026-11-02); TC-14 partial and TC-68 fail in all 3 | — |
| 13:44 | Before on 269: chat-teacher `prompt_check.py --schema L1 --workers 4`, cold (model load included, via Tailscale) | 464 calls, 198 s, 0 failures (not the comparison number) | — |
| 13:48 | Same, warm, `--host http://127.0.0.1:8080/v1` | 464 calls, **190 s = 146.5 calls/min**, 0 failures (matches the earlier ~147) | — |
| 13:51 | Before on 269: one inline PNG (solid red, data URL) to `chat`, max_tokens 1000 | 200, "Red" (first try with max_tokens 50: empty, all thinking) | — |
| 13:52 | `docker compose build` of `:273-media` | **failed** at the library check: 273's whisper-server/whisper-cli now link Ubuntu 24.04's FFmpeg 6 (`libavformat.so.60` + ~90 more libs); Ubuntu 26.04 only has FFmpeg 8 (`libavformat62`) | — |
| 13:53 | Dockerfile: whisper-server + its libggml/libwhisper taken from the 269 image (stage `whisper269`); only whisper-server uses those shared libs (llama-server has ggml built in). Rebuilt | built. In the image: llama.cpp `1.0.0-dev, commit e574a17`; llama-swap program reports **v272 (802f1c4)** (nightly pattern, as "269" reported v268); whisper-server sha256 = the running 269 one (8bd9d1c7...) | — |
| 13:54 | No requests in 10 min. Committed, then `docker compose up -d` | helpers + coder + coder-2 preloaded by 13:55:27; tts preload 404 (known, harmless) | a1ac5fd |
| 13:55 | After on 273: `tuning/grammar-repro.py coder` | **all 8 variants OK** (3, 4, 5 were 400): grammar bug fixed. Note: variant 4 (json_object) returns the tool call written as JSON text, not a real tool call (JSON mode forces text) | — |
| 13:56 | After on 273: `lab speed coder` | 54.2 writing (+0.4%) / **937.0 reading 32K (+13.8%)** (runs/2026-11-16-1356-speed-coder.json, MLflow a932f051). Reading gain likely #31323 (RDNA4 FA prefill, image 273 only) | — |
| 13:58-15:25 | After on 273: `lab quality coder --full --temperature 0.6` seeds 1-3, series `coder-ls273-v1` | 96 / 93 / 92 (132, 128, 127 of **138** points: the 4 grammar scenarios now run and pass, 8 points; MLflow 8ff5e4c9, c116a4a9, c974bd2f). Without those 4: 124 / 120 / 119 of 130 vs 269's 125 / 122 / 119. TC-14 partial and TC-68 fail in all 6 runs | — |
| 15:25 | Why 273 runs took ~28 min (269: 14-23): **TC-45** (`tool_choice: required`, "What is 7 times 8?") | model answers "56" as text, never calls a tool, runs to the 16,384-token limit (892-1,065 s). Not new: 269 seed 3 did the same (598 s). 273: 3 of 3 runs, 269: 1 of 3. Watch (3 seeds is too few to say it got worse). The slower runaway on 273 is the text (varied lines vs a repeated `*(…)*` that MTP guesses well), not writing speed (fixed-prompt test: same) | — |
| 15:26 | `chat-teacher` load log | `kv_unified = 'false'`: the new K/V-rows attention never applies to it. The probe line is not printed at our log level | — |
| 15:26 | After on 273: chat-teacher `prompt_check.py --schema L1 --workers 4`, warm, localhost | 464 calls, **188 s = 148.1 calls/min** (+1%), 0 failures. Checks: 0 BAD (269 run: 1 BAD, A2 0.83 vs want 0.86; 273 0.96: 20 vs 23 of 24, read as noise) | — |
| 15:29 | After on 273: inline PNG to `chat` | 200, "Red" (same as 269) | — |
| 15:30 | **Result: kept on 273.** Speed within ±3% writing, reading +13.8%; tool use within the seed spread; grammar bug fixed; multi-slot same; images same. Rollback: `image: llama-swap-zaphod:269-media` in compose.yaml. marvin still on 269 (next, needs Daniel's go-ahead) | | this commit |

## 2026-11-16: step 2, marvin 269 -> 273 (America/Chicago time)
Same as zaphod: before on 269, swap, same tests after. Tests through zaphod's peer (`marvin/big`), as on 2026-11-02:
grammar-repro, `lab speed marvin/big`, `lab quality marvin/big --full --temperature 0.6` seeds 1-3, marvin's
`chat-teacher` (4 slots) `prompt_check.py --schema L1 --workers 4` direct to marvin:8080. No image test: marvin has no
vision model. marvin's image has no whisper-server, so zaphod's FFmpeg problem does not apply.
Good result: speed within ±3%, tool use within the seed spread, grammar variants 3-5 OK on 273.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 15:32 | marvin idle (nothing loaded, 0 requests in 30 min), power_mode performance; marvin's files = repo HEAD (its git copy is behind, files are copied) | start | — |
| 15:32 | Before on 269: `tuning/grammar-repro.py marvin/big` | variants 3, 4, 5 give 400; others OK (bug present) | — |
| 15:33 | Before on 269: `lab speed marvin/big` | 43.8 writing / 301.0 reading 32K (long runs 319, 301, 297: the known drop after the first long prompt with the cache-leak flags); runs/2026-11-16-1533-speed-marvin-big.json, MLflow 37062caf | — |
| 15:39-16:30 | Before on 269: `lab quality marvin/big --full --temperature 0.6` seeds 1-3, series `big-ls269-v1` | 93 / 95 / 91 (121/130, 123/130, 116/128; MLflow c1297d1d, 62203c79, f6eb7f3f). Grammar 400s TC-65, 66, 67, 69 in every run. Seed 3: TC-45 left out of the score by the test (could not confirm tool_choice enforcement), so 128 points | — |
| 16:30 | Before on 269: marvin `chat-teacher` `prompt_check.py --schema L1 --workers 4`, warm, direct to marvin:8080 | 464 calls, **462 s = 60.3 calls/min**, 0 failures; 1 BAD (A2 0.83, same as zaphod's 269 run) | — |
| 16:38 | Dockerfile + compose copied to marvin (scp), `docker compose build llama-swap` on marvin | built (no library problems: marvin's image has no whisper). In the image: llama.cpp `1.0.0-dev, commit e574a17`, llama-swap v272 (802f1c4); `--list-devices`: Radeon 8060S. `:269` kept | — |
| 16:39 | marvin idle (only my chat-teacher loaded, 0 requests in 5 min). Committed, `docker compose up -d llama-swap` (other containers untouched) | `big` preloaded, ready 16:40:07, `kv_unified = 'true'`, 2 slots; K/V-rows probe line not printed at our log level | a9ae0c3 |
| 16:40 | After on 273: `tuning/grammar-repro.py marvin/big` | **all 8 OK** (3, 4, 5 were 400): fixed | — |
| 16:40 | After on 273: `lab speed marvin/big` | **45.4 writing (+3.7%) / 329.3 reading 32K (+9.4%)** (long runs 355, 329, 326: same drop after the first long prompt); runs/2026-11-16-1640-speed-marvin-big.json, MLflow db14d348 | — |
| 16:45-17:43 | After on 273: `lab quality marvin/big --full --temperature 0.6` seeds 1-3, series `big-ls273-v1` | 96 / 91 / 93 (132, 125, 128 of **138**: grammar scenarios now pass, 8 points; MLflow e8ef58d0, 5290cc76, 845806e3). Without those 4: 124 / 117 / 120 of 130 vs 269's 121 / 123 / 116 (of 130, 130, 128). TC-45 runaway: 1 of 3 (662 s; 269: 0 of 2 scored, 1 left out). TC-14 partial and TC-68 fail in all 6 | — |
| 17:43 | After on 273: marvin `chat-teacher` `prompt_check.py --schema L1 --workers 4`, warm | 464 calls, **475 s = 58.6 calls/min (-2.7%)**, 0 failures, 0 BAD | — |
| 17:52 | `big` loaded again (chat-teacher had swapped it out), as before the tests | big ready | — |
| 17:55 | **Result: kept on 273.** Writing +3.7%, reading +9.4%, tool use within the seed spread, grammar fixed, chat-teacher -2.7% (inside ±3%). Rollback: `image: llama-swap-marvin:269` in marvin/llama-swap/compose.yaml, copy to marvin, `docker compose up -d llama-swap`. Both machines now on 273 | | this commit |

## 2026-11-16: step 3, zaphod `coder` MTP draft 2 -> 3 on 273 (America/Chicago time)
Daniel chose this over the MoE cache test (19:30). Draft 3 was never tested on `coder` (tuning/results.md idea 2).
#30146 (in 1.0) left draft 3 speed the same upstream, so expect little. One change: `--spec-draft-n-max 2` -> `3`
in the `mtp` macro (used by `coder`, `coder-2`, `coder-tb`). marvin `big` already uses draft 3; not changed.
Steps:
1. Check zaphod and marvin idle (a config reload cuts marvin peer requests).
2. Before: fresh `lab speed coder` on draft 2 (tool use before = today's 273 runs: 96 / 93 / 92).
3. Change draft to 3, wait for the reload, `lab speed coder`.
4. Only if writing is +3% or more: `lab quality coder --full --temperature 0.6` seeds 1-3.
Good result: writing +3% or more, reading within ±3%, tool use within the seed spread (92-96). Otherwise back to 2.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 19:33 | zaphod and marvin idle (0 requests in 15 min); loaded: coder, chat, embed, rerank, stt, stt-whisper, tts | start | — |
| 19:33 | Before, draft 2: `lab speed coder` | 54.2 writing / 938.5 reading 32K (same as 13:56); runs/2026-11-16-1933-speed-coder.json, MLflow ec1516a3 | — |
| 19:35 | Idle check: llama-swap does not log requests (not on 269 either: marvin's log has 0 `POST /v1` lines across today's quality runs), so `grep -c "POST /v1"` always gives 0, and the 15:32 "0 requests" check proved nothing (marvin had nothing loaded, so it was idle anyway). Used llama-server's `launch_slot_` lines (one per task) instead: marvin's last task 17:52 (my `big` reload), so the reload cut nothing. plans/gpt-oss-notes.md fixed | ok | — |
| 19:36 | `--spec-draft-n-max 2` -> `3`, reload | coder ready | 7ab65df |
| 19:36 | After, draft 3: `lab speed coder` | **55.3 writing (+2.0%) / 935.0 reading (-0.4%)**, 74% of drafted tokens accepted; runs/2026-11-16-1936-speed-coder.json, MLflow 5776c5a8 | — |
| 19:38 | **Result: under the +3% bar, back to draft 2.** Tool use not run (step 4 only on +3%). marvin unchanged (draft 3) | | this commit |

## Step 4 (skipped by Daniel 2026-11-17): MoE expert cache for `gpt-oss` at 128K f16 on zaphod
Skipped: small expected gain on a test-only model; 128K q8_0 on the cards is already faster. Plan kept for later.
Why only this model: it is the only one that does not fit the cards (128K f16 needs 69.9-75.7 GB, ~63.4 GB usable),
so `--fit` puts part of the experts in RAM: 54.5 writing / 1,423 reading on 269. The cache keeps the most-used of
those RAM experts on the cards. All on 273: the review's "baseline on 269 first" no longer applies (zaphod is on 273).
What it can win (my estimate, not measured): the RAM part costs ~6.5 ms per token (54.5 vs 85.3 tok/s when nothing is
in RAM). The cache takes card memory, so more experts move to RAM; at best maybe ~60-65 tok/s. That is still below
128K **q8_0** all on the cards (82.9 / 2,278). f16's only reason is tool use: 128K q8_0 scored 90 (TC-15 failed),
but that is one fixed-seed run, and 128K f16 was never tool-tested. Reading will not improve (prefill skips the cache).
Steps (one change each; a test entry `gpt-oss-128k` added, removed at the end; real `gpt-oss` untouched):
1. zaphod and marvin idle (`docker compose logs --since 10m | grep -c launch_slot_` = 0 on marvin).
2. Daniel stops the watchdog (sudo). Window: memory sync at :41 calls `embed`, which evicts gpt-oss.
3. A, baseline: `${server-fit}` 128K f16 -b/-ub 4096 -lm none (the 269 setup). `lab speed` + core tool use.
   **Stop here if A's tool use is under 97**: then 128K f16 has no advantage over q8_0 and the cache is moot.
4. Dry run with `-lv 4`: does fit count `--moe-cache-mib` in its plan (projected memory with and without)?
5. B: A + `--moe-cache-mib` (size from step 4). C: B + `--moe-cache-cpu-share 0.95`. `lab speed` each.
6. Remove the test entry, bring the 7 usual models back, Daniel restarts the watchdog.
Good result: writing ≥ 56.1 (+3% over A), reading within ±3%, tool use 97.
