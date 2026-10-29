# zaphod homelab notes

This repo is public. Personal notes live in ~/.claude/CLAUDE.md, not here. Never commit .env files or keys.

## Machines
- zaphod (this machine): Ubuntu 26.04 server, Ryzen 9 3900X, Gigabyte B550I AORUS PRO AX (mini-ITX), 2x Radeon AI PRO R9700 (32 GB each).
  - Card 1: PCI 06:00.0, /dev/dri/renderD128, on an OCuLink dock via an M.2 adapter in the back (chipset) M.2 slot, PCIe 3.0 x4.
  - Card 2: PCI 0b:00.0, /dev/dri/renderD129, in the x16 slot (links at x8), drives the screen.
  - llama.cpp numbers them the other way round: card 1 = `Vulkan1`, card 2 = `Vulkan0`. Check with `llama-server --list-devices` (free memory per card). Card 1 is ~10% slower with MTP (its PCIe 3.0 x4 link).
- marvin: Bosgame M5, Ryzen AI Max+ 395, 128 GB (120 GB usable by the GPU). Ubuntu 26.04, llama-swap image `llama-swap-marvin:267`. Model `big` = Qwen3.8-Flash-Next UD-IQ4_XS, 128K, ~25 tok/s writing. Linked to zaphod as a llama-swap peer: appears there as `marvin/big`. Setup log: `marvin/SETUP.md`. On Wi-Fi.

## Layout
| Path | What |
| --- | --- |
| `zaphod/llama-swap/` | zaphod's llama-swap: `compose.yaml`, `config/config.yaml`, `Dockerfile`; `.env` (ZAPHOD_KEY) and `data/` are local only. `~/llama-swap` is a link to here. |
| `hub/` | Caddy, Open WebUI, Beszel, Homepage, MLflow (see README.md) |
| `marvin/llama-swap/` | Files to copy to marvin |
| `tuning/` | Benchmarks and results (see tuning/README.md) |
| `setup/` | Rebuild a machine: REBUILD.md, host.sh, models.sh, models-<machine>.txt |

## llama-swap on zaphod
- Image `llama-swap-zaphod:267-media`: official llama-swap 267 programs (llama.cpp b11870, plus whisper-server, sd-server, audiocpp_server, llama-tts) on Ubuntu 26.04's Mesa, built from the Dockerfile. Previous image `:267` (text only) is kept for rollback. Models in /srv/models.
- Models: `coder` (card 1) and `coder-2` (card 2): Qwen3.8-27B UD-Q4_K_XL, MTP speculative decoding, 64K context. `coder-tb`: card 1, 256K context, q8_0 KV, MTP, for Terminal-Bench; swaps with `coder`. Card 2, swapping with `coder-2`, 256K q8_0 context: `ornith` (Ornith-1.5 35B-A3B Q4_K_M + vision; best tool-use score so far), `chat` (Gemma 4 31B, Google QAT Q4_0 + vision, 128K, general chat), `north`, `flint` (on hold). Speech: `stt` (Qwen3-ASR), `stt-whisper`, `tts` (Kokoro) stay loaded on card 2; `tts-qwen` only next to `coder-2`. Preload logs 404 for the audio.cpp models; harmless. `embed` + `rerank` (document search) stay loaded on card 1. `coder-long`: both cards, 256K, not yet tested.
- Listens on 127.0.0.1:8080 only; `tailscale serve --bg --tcp=8080 tcp://localhost:8080` exposes it on the tailnet. Also https://zaphod.ai.dopamine911.com via Caddy.
- Config edits reload automatically. Check with `cd ~/llama-swap && docker compose logs --tail 20`.
- Commit every config change: "what · from → to · why". Benchmarks record the last commit that touched `zaphod/llama-swap/`.

## Rebuild
`setup/REBUILD.md` + `setup/host.sh` + `setup/models.sh` rebuild either machine. Every change to a machine (package, boot setting, Tailscale, web-UI setting, model) goes into a script or REBUILD.md in the same commit.

Open WebUI: every setting is in `hub/compose.yaml` (open-webui environment). Never tell Daniel to change Open WebUI in the web page: edit compose, then `hub/openwebui/drift.sh --sync`; `drift.sh` alone checks for hand changes.

## Do not undo
- `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1` in compose.yaml. Resizable BAR is off, and without this switch llama.cpp puts part of the model in system RAM: 3-10x slower.
- BIOS: CSM Support must stay Enabled, or Ubuntu won't boot (installed in legacy mode). This is also why Resizable BAR is off.

## Known issues
- llama.cpp b11870 (the build inside llama-swap 267) rejects Qwen3.8 and Ornith requests that combine tools with a required JSON format ("failed to parse grammar"). Flint is unaffected. Re-test after a llama.cpp update.

## Results
`tuning/results.md` (summary) and MLflow https://mlflow.ai.dopamine911.com (every run). Tools: `tuning/bench`, `quality`, `kl`, `tbench`, `compare-skill`; see tuning/README.md.

## Next steps
1. Terminal-Bench full run of `coder-tb`, with and without the-craft (overnight, in tmux).
2. marvin: MTP speed-up for `big` (baseline 24.7 tok/s writing), then quality tests.
3. Tuning ideas in tuning/results.md.
4. llama.cpp: stay on the pinned build and watch for fixes (grammar bug above). If we need our own fixes: a small patch set on top of a pinned official tag, built from source, each patch offered upstream; not a long-lived fork. Separate the llama.cpp version from llama-swap's first (official llama.cpp image + llama-swap binary), changing one thing at a time.
5. Optional: RAM runs at 2666 instead of XMP 3600; card 2 links at x8 not x16.

Disk: root LVM volume expanded 2026-10-28 from 100 GB to the full 929 GB SSD.
Hub: DOMAIN=ai.dopamine911.com; Cloudflare A records ai + *.ai -> zaphod's Tailscale IP (DNS only). models-panel/index.html still needs copying from the claude.ai project.
