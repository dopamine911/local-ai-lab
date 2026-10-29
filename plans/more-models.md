# Plan: more model types and ComfyUI on zaphod

Goal: besides the coding models, have vision, general chat, document search (embeddings + reranker),
speech-to-text, text-to-speech and image generation (ComfyUI) available, mostly through llama-swap
and Open WebUI.

Rules: one step at a time; each step has a check ("good result") and is committed before the next.
Pin every model to an exact Hugging Face commit and record its checksum. Exact model files are picked
and checked at each step, not guessed now.

## Starting point (2026-10-28)

- Each card: 32 GB, `coder` / `coder-2` use about 22 GB, so about 10 GB free per card.
- Disk: 609 GB free. RAM: 60 GB.
- Our image `llama-swap-zaphod:267` only has llama-server and llama-bench. The official
  `unified-vulkan-267` image also has whisper-server (speech-to-text), sd-server (images),
  audiocpp_server and llama-tts (speech).

## Steps

| # | Step | Where | Good result |
| --- | --- | --- | --- |
| 1 | Vision for Ornith: add its official `mmproj` file (same repo, same pinned commit) | llama-swap, `ornith` | A screenshot sent in Open WebUI is described correctly; tool-use score unchanged (93 at temp 0) |
| 2 | Image: also copy whisper-server, sd-server and the speech programs from the official image | `zaphod/llama-swap/Dockerfile` | `coder` speeds unchanged (llama-bench), all programs start (`--help`) |
| 3 | Embeddings + reranker, kept loaded next to the coding models | llama-swap; Open WebUI Documents settings | Open WebUI search over a test document finds the right passage; nothing else swaps out |
| 4 | Speech-to-text (Whisper) | llama-swap; Open WebUI Audio settings | Microphone input in Open WebUI is transcribed correctly |
| 5 | Text-to-speech | llama-swap; Open WebUI Audio settings | "Read aloud" in Open WebUI plays clear speech |
| 6 | General chat model (Gemma 4 31B or similar, checked first) | llama-swap, swaps like `north`/`ornith` | Loads, normal answers; speed and quick quality check recorded |
| 7 | ComfyUI: own container, ROCm build of PyTorch, pinned ComfyUI release, behind Caddy at `comfy.<DOMAIN>` | `hub/` | Generates a test image; Open WebUI can use it for image generation |
| 8 | Routing rules: small helpers (steps 3-5) stay loaded; big models swap as now | llama-swap `config.yaml` | Using voice or search never unloads `coder` |

## ComfyUI and card memory

ComfyUI uses ROCm (PyTorch), not Vulkan, and llama-swap cannot unload it. Two programs filling the
same card will run out of memory.

Decided 2026-10-28: **image mode switch.** ComfyUI uses card 2. One command unloads card 2's
llama-swap models (`coder-2`, `ornith`, ...) before generating images and switches back after;
`coder` on card 1 keeps running throughout.

## Progress

- 2026-10-28 Step 1 done: Ornith vision (mmproj BF16, sha256 1921a36a). Test image described correctly; tool-use still 93.
- 2026-10-28 Step 2 done: image `llama-swap-zaphod:267-media`; speeds unchanged.
- 2026-10-28 Step 3 models done: `embed` (Qwen3-Embedding-4B Q6_K) + `rerank` (Qwen3-Reranker-0.6B Q8_0), always loaded on card 1.
  Open WebUI connection: set by hand in Admin Panel (see README.md, Open WebUI tools).
- 2026-10-28 Found and fixed: llama.cpp's Vulkan0 is card 2; models were on the opposite card to their names.
- 2026-10-29 Step 4 done: `stt` = Qwen3-ASR-1.7B Q8_0 (audiocpp_server, audio.cpp build 4efa395 of 2026-10-24; newest release v0.9.3)
  and `stt-whisper` = Whisper large-v3-turbo Q8_0 (whisper.cpp v1.9.5, newest). Card 2. All three candidates
  (also Cohere Transcribe) transcribed jfk.wav exactly in 0.16-0.33 s; picked by Open ASR leaderboard accuracy.
  Helpers (embed, rerank, stt, stt-whisper) now preload at startup. Memory with all loaded: card 1 29.7 GB, card 2 25.5 GB.
  Open WebUI connection: set by hand (README.md, Open WebUI tools).
- 2026-10-29 Step 5 done: `tts` = Kokoro-82M Q8_0 (always loaded, 10 s of speech in 0.4 s) and `tts-qwen` = Qwen3-TTS-12Hz-1.7B
  CustomVoice Q8_0 (2.2 s per 10 s; runs only next to coder-2, set `voice`). Both on card 2, audio.cpp packages pinned to
  rev 874fcb41. Round-trip through stt: Kokoro said "4.5" as "four, five"; Qwen-TTS's "Zaphod" was heard as "Zefot".
  Memory: card 2 28.5 GB with coder-2 + all helpers + tts-qwen; 28.4 GB with ornith + helpers.
- 2026-10-29 Step 6 done: `chat` = Gemma 4 31B (google/gemma-4-31B-it-qat-q4_0-gguf rev 59dde245, + mmproj), card 2, 128K q8_0,
  Google's sampling (temp 1.0, top-p 0.95, top-k 64). 28.8 tok/s writing. Described the test image correctly.
  Card 2 with chat + stt + stt-whisper + tts: 29.2 GB. Gemma 4 is still Google's newest family (no 4.5 found).
- 2026-10-29 Step 7 done: ComfyUI v0.43.0 + PyTorch 2.15.0 + ROCm 7.14 (newest of each), own image, card 2 only,
  started by `hub/image-mode on` (llama-swap profile `image-mode` + unload), stopped by `off`. comfy.<DOMAIN> via Caddy.
  Z-Image Turbo (Comfy-Org rev 6fc90a3b: bf16 model, qwen_3_4b, ae) makes a 1024x1024 image in 5.0 s (15.8 s with load);
  test images matched their prompts, except "watercolor" came out photographic. 35 TFLOPS fp16 in a PyTorch test.
  Not yet: Qwen-Image-2.1 (newer, also edits images), Homepage button for image mode.
- Step 8 (routing rules) was done along the way: helpers set + preload, `voice` set, `image-mode` profile.
- 2026-10-29 Qwen-Image-2.1 Turbo added (Comfy-Org rev 0217efd1: turbo int8_convrot, qwen3vl_8b int8_convrot, VAE bf16;
  17.3 GB; checksums OK). The int8 files run on the R9700 through comfy_kitchen's HIP backend. 1024x1024 in 6.0 s
  (20 s with load). Wrote "ZAPHOD LAB" on a poster exactly. "Watercolor" became a watercolor picture inside a photo.
- 2026-10-29 audio.cpp v0.9.3 (2026-10-26) checked against our build 4efa395 (2026-10-24, 29 commits behind):
  changes are other TTS families (Drama 3, Ming3, BreezeTTS, Irodori), CUDA speed-ups and build fixes; nothing for
  qwen3_asr, kokoro_tts or qwen3_tts on Vulkan. Not updating; it comes with the next llama-swap image anyway.
