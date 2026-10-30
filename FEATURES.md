# Features

Everything this lab does, in one place, with links to the documents and files that hold the details.
Status words: **working** (in use, tested), **ready** (built and tested, used on demand), **planned**.
Last updated 2026-10-29.

Start here for how-to: [README.md](README.md). Rebuild a machine: [setup/REBUILD.md](setup/REBUILD.md).
Notes for Claude Code sessions: [CLAUDE.md](CLAUDE.md). Results of every test: [tuning/results.md](tuning/results.md).
Pictures of how it fits together (shows on GitHub): [docs/diagrams](docs/diagrams/README.md).

## 1. Machines

| Machine | What it is | Role | Details |
| --- | --- | --- | --- |
| zaphod | Ryzen 9 3900X, 60 GB RAM, 2x Radeon AI PRO R9700 32 GB | Main AI server + all web services (the hub) | [CLAUDE.md](CLAUDE.md) |
| marvin | Bosgame M5, Ryzen AI Max+ 395, 128 GB shared memory (120 GB usable by the GPU) | Big models | [marvin/SETUP.md](marvin/SETUP.md) (every setup step, with results) |

Device-level documentation (parts, serial numbers, network, cabling, problem log) is in a separate **private** repo,
not here.

## 2. Running models (llama-swap)

| Feature | Status | Details |
| --- | --- | --- |
| llama-swap on zaphod: one OpenAI-compatible API for all models, loads and swaps them on demand, per-card routing | working | [zaphod/llama-swap/config/config.yaml](zaphod/llama-swap/config/config.yaml), [README.md](README.md) |
| Coding models `coder` (card 1) and `coder-2` (card 2): Qwen3.8-27B with MTP speculative decoding (~60 tok/s) | working | [tuning/results.md](tuning/results.md) |
| `ornith` (best tool use so far, vision), `chat` (Gemma 4 31B, vision), `north`, `flint` (on hold), `coder-tb` (256K, benchmarks), `coder-long` (both cards) | working | [CLAUDE.md](CLAUDE.md) |
| Vision: send screenshots to `ornith` or `chat` | working | [plans/more-models.md](plans/more-models.md) |
| Document search models: `embed` (Qwen3-Embedding-4B) + `rerank` (Qwen3-Reranker), always loaded on card 1 | working | [plans/more-models.md](plans/more-models.md) |
| Speech-to-text `stt` (Qwen3-ASR), `stt-whisper` (99 languages); text-to-speech `tts` (Kokoro), `tts-qwen` | working | [zaphod/llama-swap/config/](zaphod/llama-swap/config/) |
| marvin's llama-swap: `big` = Qwen3.8-Flash-Next UD-IQ4_XS (93 GB), MTP draft 3 (24.7 → 39.6 tok/s) | working | [marvin/llama-swap/config/config.yaml](marvin/llama-swap/config/config.yaml), [marvin/SETUP.md](marvin/SETUP.md) |
| marvin linked to zaphod as a peer: appears everywhere as `marvin/big` | working | [README.md](README.md) |
| The slow-card fix (Resizable BAR off → `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1`, 3-10x faster) | working | [tuning/results.md](tuning/results.md) "The slow-card fix" |

## 3. Web services on zaphod (the hub)

All behind Caddy with real certificates at `*.ai.<domain>`, reachable only over Tailscale. Definition:
[hub/compose.yaml](hub/compose.yaml).

| Feature | Status | Details |
| --- | --- | --- |
| Caddy: HTTPS for every service, Cloudflare DNS certificates | working | [hub/caddy/Caddyfile](hub/caddy/Caddyfile) |
| Open WebUI: chat with every model on both machines | working | [README.md](README.md) "Open WebUI settings" |
| Open WebUI settings kept as code (`compose.yaml`), drift check + sync | working | [hub/openwebui/drift.sh](hub/openwebui/drift.sh), [hub/openwebui/managed-keys.txt](hub/openwebui/managed-keys.txt) |
| Per-model settings as code: native function calling, vision only where it works, helper models hidden | working | [hub/openwebui/models.yaml](hub/openwebui/models.yaml), [hub/openwebui/models.sh](hub/openwebui/models.sh) |
| Open WebUI documents (embeddings + hybrid search + re-ranking), speech in and out, web search | working | [README.md](README.md) |
| Image generation: ComfyUI on card 2 (Z-Image Turbo 5 s, Qwen-Image 2.1 6 s), switched by "image mode" | ready | [hub/image-mode](hub/image-mode), [hub/comfyui/](hub/comfyui/), [hub/openwebui/make-comfyui-env.sh](hub/openwebui/make-comfyui-env.sh) |
| SearXNG: private web search (Bing, Yahoo, Brave, Google CSE + developer sources) | working | [hub/searxng/settings.yml](hub/searxng/settings.yml) |
| Open Terminal: sandboxed code runner for Open WebUI | working | [README.md](README.md) |
| Homepage start page: a row per machine (models loaded, system, GPU load/memory/temp/power), apps, NAS placeholder | working | [hub/homepage/](hub/homepage/), [hub/gpu-stats/gpu_stats.py](hub/gpu-stats/gpu_stats.py) |
| Agent stats on the start page, per machine: 24 h requests, tokens in/out, average/top speed; now: active, pending, slots, context per slot | working (pending: after `--metrics`) | [hub/agent-stats/agent_stats.py](hub/agent-stats/agent_stats.py) |
| Beszel monitoring: both machines, containers, disk health, systemd services, network monitors | working | [hub/compose.yaml](hub/compose.yaml), [hub/beszel-monitors.sh](hub/beszel-monitors.sh) |
| MLflow: every benchmark run with its full settings | working | section 5 below |
| Models panel (load/unload models from a page) | planned (file to copy in) | [hub/models-panel/README.txt](hub/models-panel/README.txt) |

## 4. Tools for agents (MCP gateway)

| Feature | Status | Details |
| --- | --- | --- |
| MCPJungle 0.4.6 gateway: tools set up once, one token + allowed-tool list per agent, enterprise (locked-down) mode | working | [README.md](README.md) "Agent tools", [hub/mcp/setup.sh](hub/mcp/setup.sh) |
| `search` tool: SearXNG search + public page reading (refuses home/Tailscale addresses) | working | [hub/mcp/servers/search.json](hub/mcp/servers/search.json) |
| `browser` tool: headless Chromium (Playwright MCP) on an internal network, out only through a filtering proxy | working | [hub/mcp/servers/browser.json](hub/mcp/servers/browser.json), [hub/browser-proxy/squid.conf](hub/browser-proxy/squid.conf) |
| Connected agents: Open WebUI, Claude Code, Enso (container) | working | [hub/mcp/clients/](hub/mcp/clients/) |
| Enso (Daniel's coding-agent harness, pi 1.1.0) in a container on zaphod: local models from llama-swap, MCPJungle tools, no internet, own workspace clone; for local models working on Enso issues | ready (2026-10-30; interactive use not yet tried) | [agents/enso/README.md](agents/enso/README.md) |
| Page fetching never reaches the lab (Open WebUI, search tool, browser: all tested) | working | [hub/compose.yaml](hub/compose.yaml) `ENABLE_LOCAL_WEB_FETCH` |
| `memory` tool: search the lab's own notes (~/homelab + private ~/full-homelab Markdown) with LocalRecall 0.6.6 (Postgres, keyword + vector, llama-swap `embed`); read-only; Enso and Claude Code only (private text: not Open WebUI); test transcripts and benchmark tasks kept out | working | [hub/compose.yaml](hub/compose.yaml), [hub/memory/](hub/memory/), [hub/mcp/servers/memory.json](hub/mcp/servers/memory.json) |
| Memory from conversations (mem0), session recording | planned | [plans/agent-infra.md](plans/agent-infra.md) |

## 5. Testing and measuring

How-to for all tools: [tuning/README.md](tuning/README.md). Every result: [tuning/results.md](tuning/results.md) and MLflow.

| Feature | Status | Details |
| --- | --- | --- |
| Speed test (llama-bench), per card | working | [tuning/bench](tuning/bench) |
| Tool-use test (tool-eval-bench, 69 scenarios), hand-sorted misses | working | [tuning/quality](tuning/quality), [tuning/tool-eval/classifications.json](tuning/tool-eval/classifications.json) |
| Terminal-Bench (19 real terminal tasks), zaphod or marvin (`TARGET=marvin`) | working | [tuning/tbench](tuning/tbench) |
| Quantization accuracy (KL-divergence vs Q8_0) | working | [tuning/kl](tuning/kl) |
| Skill A/B tests (the-craft, j5-craft) | working | [tuning/compare-skill](tuning/compare-skill), [tuning/instructions/](tuning/instructions/) |
| Every run records exactly what produced it: machine, BIOS, GPUs/VBIOS, kernel, Mesa, image, llama.cpp commit, exact model file + checksum | working | [tuning/manifest.sh](tuning/manifest.sh), [tuning/log_run.py](tuning/log_run.py) |
| Fair score across servers (`tooleval_score_common`) | working | [tuning/README.md](tuning/README.md) |
| Series of repeated runs: average, lowest, highest, spread | working | [tuning/series](tuning/series) |
| Automatic behavior checks the grader misses (Q4 claim, attacker echo, loops, made-up queries) | working | [tuning/tool-eval/behavior.py](tuning/tool-eval/behavior.py) |
| Real-use speed during every test (from llama-swap's request log) | working | [tuning/activity.sh](tuning/activity.sh) |
| Per-test and per-task tables in MLflow (compare one scenario across models) | working | [tuning/tool-eval/sort_to_mlflow.py](tuning/tool-eval/sort_to_mlflow.py) |

Main results so far (details in [tuning/results.md](tuning/results.md)):
- Tool use (common score): marvin Flash-Next on Strata (thinking medium) 94.0, Ornith 92.0, marvin Flash-Next 91.7, Strata Flash-Next (zaphod) 90.0, Flint 86.0. No model made anything up.
- Terminal-Bench on marvin (Flash-Next, no skill): **19/19** full run, 18 on the first attempt, 5.5 h (2026-10-29).
- Terminal-Bench on marvin, same model on Strata 0.2.7: thinking xhigh 19/19 in 13.3 h; thinking medium **17/19** in 4.39 h (2026-10-30; openssl-selfsigned-cert and extract-elf failed). llama.cpp's 5.5 h was in BIOS Quiet mode, so times are not comparable.
- Strata (another inference engine) tested on zaphod: same quality, needs both cards: [plans/strata.md](plans/strata.md).

## 6. Rebuild, backup, safety

| Feature | Status | Details |
| --- | --- | --- |
| Rebuild either machine from the repo + secrets: BIOS/installer steps, host setup script, pinned model downloads | working (tested on both machines) | [setup/REBUILD.md](setup/REBUILD.md), [setup/host.sh](setup/host.sh), [setup/models.sh](setup/models.sh) |
| Model lists pinned to exact revisions with sha256 (zaphod 26 files, all verified; marvin 4) | working | [setup/models-zaphod.txt](setup/models-zaphod.txt), [setup/models-marvin.txt](setup/models-marvin.txt) |
| Nightly backup to marvin at 04:17 (Open WebUI, Beszel, MLflow, gateway, .env files), 14 snapshots | working (moves to the NAS later) | [setup/backup-to-marvin.sh](setup/backup-to-marvin.sh) |
| Secrets: only in `.env` files (mode 600) + Proton Pass, never in git; checked before commits | working | [setup/REBUILD.md](setup/REBUILD.md) section 5 |
| Services listen only on 127.0.0.1 or the Tailscale address; every image pinned (most by digest) | working | [hub/compose.yaml](hub/compose.yaml) |

## 7. Plans

| Plan | What | Status |
| --- | --- | --- |
| [plans/agent-infra.md](plans/agent-infra.md) | Infrastructure for all agents: Enso on local models, own-task test, gateway, memory, tracing, reliability, hardening | in progress (gateway done) |
| [plans/strix-strata.md](plans/strix-strata.md) | Strata on marvin (sealed off), then port UnbiasedStrata to AMD gfx1151 and contribute upstream | planned (fork ready) |
| [plans/maya.md](plans/maya.md) | Project Maya: much bigger models (GLM-5.3-Flash 321B) on marvin; ideas for the fork | to check (after the port) |
| [plans/strata.md](plans/strata.md) | Strata tests on zaphod | done |
| [plans/more-models.md](plans/more-models.md) | Vision, chat, document search, speech, images | done |
| [plans/morning-guide-2026-10-29.md](plans/morning-guide-2026-10-29.md) | Overnight work summary and steps for Daniel | done |
| [CLAUDE.md](CLAUDE.md) "Next steps" | marvin 256K context, MTP tuning, Terminal-Bench full runs, llama.cpp update policy | ongoing |
