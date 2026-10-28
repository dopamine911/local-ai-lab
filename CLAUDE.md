# zaphod homelab notes

This repo is public. Personal notes live in ~/.claude/CLAUDE.md, not here. Never commit .env files or keys.

## Machines
- zaphod (this machine): Ubuntu 26.04 server, Ryzen 9 3900X, Gigabyte B550I AORUS PRO AX (mini-ITX), 2x Radeon AI PRO R9700 (32 GB each).
  - Card 1: PCI 06:00.0, /dev/dri/renderD128, on an OCuLink dock via an M.2 adapter in the back (chipset) M.2 slot, PCIe 3.0 x4.
  - Card 2: PCI 0b:00.0, /dev/dri/renderD129, in the x16 slot (links at x8), drives the screen.
- marvin: Bosgame M5, Ryzen AI Max+ 395, 128 GB. Not set up yet. Will join as a llama-swap peer (files in `marvin/llama-swap/`).

## Layout
| Path | What |
| --- | --- |
| `zaphod/llama-swap/` | zaphod's llama-swap: `compose.yaml`, `config/config.yaml`, `Dockerfile`; `.env` (ZAPHOD_KEY) and `data/` are local only. `~/llama-swap` is a link to here. |
| `hub/` | Caddy, Open WebUI, Beszel, Homepage, MLflow (see README.md) |
| `marvin/llama-swap/` | Files to copy to marvin |
| `tuning/` | Benchmarks and results (see tuning/README.md) |

## llama-swap on zaphod
- Image `llama-swap-zaphod:267`: official llama-swap 267 programs on Ubuntu 26.04's Mesa (built from the Dockerfile). Models in /srv/models.
- Models: `coder` (card 1) and `coder-2` (card 2): Qwen3.8-27B UD-Q4_K_XL, MTP speculative decoding, 64K context. `coder-tb`: card 1, 256K context, q8_0 KV, MTP, for Terminal-Bench; swaps with `coder`. `flint`: Recursive Flint UD-Q4_K_XL, card 2, 256K q8_0 context; swaps with `coder-2`. `coder-long`: both cards, 256K, not yet tested.
- Listens on 127.0.0.1:8080 only; `tailscale serve --bg --tcp=8080 tcp://localhost:8080` exposes it on the tailnet. Also https://zaphod.ai.dopamine911.com via Caddy.
- Config edits reload automatically. Check with `cd ~/llama-swap && docker compose logs --tail 20`.
- Commit every config change: "what · from → to · why". Benchmarks record the last commit that touched `zaphod/llama-swap/`.

## Do not undo
- `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1` in compose.yaml. Resizable BAR is off, and without this switch llama.cpp puts part of the model in system RAM: 3-10x slower.
- BIOS: CSM Support must stay Enabled, or Ubuntu won't boot (installed in legacy mode). This is also why Resizable BAR is off.

## Known issues
- llama.cpp b11870 (the build inside llama-swap 267) rejects Qwen3.8 and Ornith requests that combine tools with a required JSON format ("failed to parse grammar"). Flint is unaffected. Re-test after a llama.cpp update.

## Results
`tuning/results.md` (summary) and MLflow https://mlflow.ai.dopamine911.com (every run). Tools: `tuning/bench`, `quality`, `kl`, `tbench`, `compare-skill`; see tuning/README.md.

## Next steps
1. Terminal-Bench full run of `coder-tb`, with and without the-craft (overnight, in tmux).
2. Set up marvin, then add it as a peer in zaphod/llama-swap/config/config.yaml (block is there, commented out).
3. Tuning ideas in tuning/results.md.
4. Optional: RAM runs at 2666 instead of XMP 3600; card 2 links at x8 not x16.

Disk: root LVM volume expanded 2026-10-28 from 100 GB to the full 929 GB SSD.
Hub: DOMAIN=ai.dopamine911.com; Cloudflare A records ai + *.ai -> zaphod's Tailscale IP (DNS only). models-panel/index.html still needs copying from the claude.ai project.
