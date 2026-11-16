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
