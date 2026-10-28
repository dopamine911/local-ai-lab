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
