# Features

Everything this lab does, in one place, with links to the documents and files that hold the details.
Status words: **working** (in use, tested), **ready** (built and tested, used on demand), **planned**.
Last updated 2026-11-02.

Start here for how-to: [README.md](README.md). Every command: [docs/cli/](docs/cli/README.md). Rebuild a machine: [setup/REBUILD.md](setup/REBUILD.md).
Notes for Claude Code sessions: [CLAUDE.md](CLAUDE.md). Results of every test: [tuning/results.md](tuning/results.md).
Pinned versions and the last audit: [setup/VERSIONS.md](setup/VERSIONS.md).
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
| Coding models `coder` (card 1) and `coder-2` (card 2): Qwen3.8-27B with MTP speculative decoding (~54 tok/s writing, ~765 tok/s reading a 32K prompt; `lab speed`) | working | [tuning/results.md](tuning/results.md) |
| `ornith` (best tool use so far, vision), `chat` (Gemma 4 31B, vision), `north`, `flint` (on hold), `coder-tb` (256K, benchmarks), `coder-long` (both cards) | working | [CLAUDE.md](CLAUDE.md) |
| Vision: send screenshots to `ornith` or `chat` | working | [plans/more-models.md](plans/more-models.md) |
| Document search models: `embed` (Qwen3-Embedding-4B) + `rerank` (Qwen3-Reranker), always loaded on card 1 | working | [plans/more-models.md](plans/more-models.md) |
| Speech-to-text `stt` (Qwen3-ASR), `stt-whisper` (99 languages); text-to-speech `tts` (Kokoro), `tts-qwen` | working | [zaphod/llama-swap/config/](zaphod/llama-swap/config/) |
| marvin's llama-swap: `big` = Qwen3.8-Flash-Next UD-IQ4_XS (93 GB), MTP draft 3 (24.7 → 39.6 tok/s; ~44 writing, ~318 reading a 32K prompt on BIOS Performance) | working | [marvin/llama-swap/config/config.yaml](marvin/llama-swap/config/config.yaml), [marvin/SETUP.md](marvin/SETUP.md) |
| marvin linked to zaphod as a peer: appears everywhere as `marvin/big` | working | [README.md](README.md) |
| Live test of our UnbiasedStrata fork on marvin: appears as `marvin-fork/big-fork` while `fork.sh start` runs (`marvin/big` offline then) | on hold (2026-11-01; was up 23 h, unused) | [plans/strix-strata.md](plans/strix-strata.md) |
| Both machines on llama-swap 269 (llama.cpp b12020), measured before and after on each: speed and tool use the same (#1, closed) | working | [tuning/results.md](tuning/results.md) "llama-swap 267 -> 269" |
| The slow-card fix (Resizable BAR off → `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1`, 3-10x faster) | working | [tuning/results.md](tuning/results.md) "The slow-card fix" |

## 3. Web services on zaphod (the hub)

All behind Caddy with real certificates at `*.ai.<domain>`, reachable only over Tailscale. Definition:
[hub/compose.yaml](hub/compose.yaml).

| Feature | Status | Details |
| --- | --- | --- |
| Caddy: HTTPS for every service, Cloudflare DNS certificates | working | [hub/caddy/Caddyfile](hub/caddy/Caddyfile) |
| Open WebUI: chat with every model on both machines | working | [README.md](README.md) "Open WebUI settings" |
| Open WebUI settings kept as code (`compose.yaml`), drift check + sync | working | [lab webui drift](docs/cli/webui.md), [hub/openwebui/managed-keys.txt](hub/openwebui/managed-keys.txt) |
| Per-model settings as code: native function calling, vision only where it works, helper models hidden | working | [hub/openwebui/models.yaml](hub/openwebui/models.yaml), [lab webui models](docs/cli/webui.md) |
| Open WebUI documents (embeddings + hybrid search + re-ranking), speech in and out, web search | working | [README.md](README.md) |
| Image generation: ComfyUI on card 2 (Z-Image Turbo 5 s, Qwen-Image 2.1 6 s), switched by "image mode" | ready | [lab image-mode](docs/cli/image-mode.md), [hub/comfyui/](hub/comfyui/), [lab webui comfyui-env](docs/cli/webui.md) |
| SearXNG: private web search (Bing, Yahoo, Brave, Google CSE + developer sources) | working | [hub/searxng/settings.yml](hub/searxng/settings.yml) |
| Open Terminal: sandboxed code runner for Open WebUI | working | [README.md](README.md) |
| Homepage start page: a row per machine (models loaded, system, GPU load/memory/temp/power), apps, NAS placeholder | working | [hub/homepage/](hub/homepage/), [hub/gpu-stats/gpu_stats.py](hub/gpu-stats/gpu_stats.py) |
| Agent stats on the start page, per machine: 24 h requests, tokens in/out, average/top speed; now: active, pending, slots, context per slot | working (pending: after `--metrics`) | [hub/agent-stats/agent_stats.py](hub/agent-stats/agent_stats.py) |
| Push notifications to the phone (ntfy 2.37.1, self-hosted, Tailscale only, nothing via ntfy.sh): topics lab-done / lab-alert / lab-research; `hub/notify` sends; every benchmark logged with `lab log` notifies with its result and MLflow link | working; buttons on notifications (`hub/actions`): Ack, and on benchmark results Rerun (zaphod only), Skip, Open | [hub/notify](hub/notify), [hub/actions](hub/actions/actiond.py), [hub/compose.yaml](hub/compose.yaml) |
| Watchdog: 17 read-only checks every minute (real `memory` and `recall` calls through MCPJungle; marvin reachable, power mode, busy under 100 W, GPU temp; zaphod card junctions; llama-swap, disks and containers on both; MCPJungle tools; backup and memory-sync age; certificate); one alert + one resolved, buttons Ack / Snooze 1 h / Run check | working (live test 2026-11-02: quiet power mode -> 2 alerts in 2.5 min, Run check and Ack from the phone, 2 resolved); batch 2 tested with made-up readings (#12) | [hub/watchdog](hub/watchdog/watchdog.py), [setup/REBUILD.md](setup/REBUILD.md) |
| `lab` CLI (Go, Cobra + Bubble Tea): one command for everything the lab does, on both machines; the old scripts are one-line wrappers that call it; `lab` on a terminal opens a menu; `lab manifest` (port of tuning/manifest.sh, identical output on both machines); `lab mcp status` (tool servers, who may use what, drift from the repo); `lab enso check` (port, identical); `lab webui drift` (port of the read-only drift.sh, identical); `lab memory sync`, `lab image-mode` (ports, `--dry-run`); `lab webui models [--apply]` (port, identical); `lab mcp setup` (port; fixed two bugs in setup.sh); `lab enso build/setup/up/tui/shell/down` (ports); benchmarks `lab log`, `series`, `speed`, `quality`, `bench`, `kl` (ports: same MLflow fields as the scripts; bench/kl always restart llama-swap, `--dry-run`); machines `lab models`, `lab backup [list|install]`, `lab tbench` (ports) and `lab setup check` (new: which host.sh steps are done, read-only); `lab monitors`, `lab webui comfyui-env`, `lab compare-skill`, `lab sort-log` (ports); `--json` for agents | working (steps 1-5 and the last ports merged: #26, #29, #31; step 6 `reserve`/`pool`/`research` waits on #3-#5) | [cli/](cli/main.go), [docs/cli/](docs/cli/README.md) (reference, generated from lab's help), issue #6 |
| Renovate: one PR per newer version of a pinned image, package or download (90 pins: compose, Dockerfiles, Go modules, pip, bun, git commits), with the check to run; held-back items (llama-swap, Python minor, bun, Enso, fan module) need approval on the dashboard | config validated + dry run; hosted Mend Renovate app (decisions.md 40), turned on after the config PR #7 | [renovate.json](renovate.json), [setup/VERSIONS.md](setup/VERSIONS.md) |
| Beszel monitoring: both machines, containers, disk health, systemd services, network monitors | working | [hub/compose.yaml](hub/compose.yaml), [lab monitors](docs/cli/monitors.md) |
| MLflow: every benchmark run with its full settings | working | section 5 below |
| Models panel (load/unload models from a page) | planned (file to copy in) | [hub/models-panel/README.txt](hub/models-panel/README.txt) |

## 4. Tools for agents (MCP gateway)

| Feature | Status | Details |
| --- | --- | --- |
| MCPJungle 0.4.7 gateway: tools set up once, one token + allowed-tool list per agent, enterprise (locked-down) mode | working | [README.md](README.md) "Agent tools", [lab mcp setup](docs/cli/mcp.md) |
| `search` tool: SearXNG search + public page reading (refuses home/Tailscale addresses) | working | [hub/mcp/servers/search.json](hub/mcp/servers/search.json) |
| `browser` tool: headless Chromium (Playwright MCP) on an internal network, out only through a filtering proxy | working | [hub/mcp/servers/browser.json](hub/mcp/servers/browser.json), [hub/browser-proxy/squid.conf](hub/browser-proxy/squid.conf) |
| Connected agents: Open WebUI, Claude Code, Enso (container) | working | [hub/mcp/clients/](hub/mcp/clients/) |
| Enso (Daniel's coding-agent harness, pi 1.1.0) in a container on zaphod: local models from llama-swap, MCPJungle tools, no internet, own workspace clone; for local models working on Enso issues | ready (2026-10-30; interactive use not yet tried) | [agents/enso/README.md](agents/enso/README.md) |
| Page fetching never reaches the lab (Open WebUI, search tool, browser: all tested) | working | [hub/compose.yaml](hub/compose.yaml) `ENABLE_LOCAL_WEB_FETCH` |
| `memory` tool: search the lab's own notes (~/homelab + private ~/full-homelab Markdown) with LocalRecall 0.6.6 (Postgres, keyword + vector, llama-swap `embed`); read-only; Enso and Claude Code only (private text: not Open WebUI); test transcripts and benchmark tasks kept out | working | [hub/compose.yaml](hub/compose.yaml), [hub/memory/](hub/memory/), [hub/mcp/servers/memory.json](hub/mcp/servers/memory.json) |
| `recall` tool: short memories shared by all agents (Mnemosyne 4.0.0): remember a fact/preference/decision, recall by meaning, mark wrong or confirmed; saving calls no model; 5 of its 30 tools on; Enso and Claude Code | working | [hub/memory/mnemosyne.Dockerfile](hub/memory/mnemosyne.Dockerfile), [hub/mcp/servers/recall.tools](hub/mcp/servers/recall.tools) |
| Session recording | planned | [plans/agent-infra.md](plans/agent-infra.md) |

## 5. Testing and measuring

How-to for all tools: [tuning/README.md](tuning/README.md). Every result: [tuning/results.md](tuning/results.md) and MLflow.

| Feature | Status | Details |
| --- | --- | --- |
| Speed test (llama-bench), per card | working | [lab bench](docs/cli/bench.md) |
| Tool-use test (tool-eval-bench, 69 scenarios), hand-sorted misses | working | [lab quality](docs/cli/quality.md), [tuning/tool-eval/classifications.json](tuning/tool-eval/classifications.json) |
| Terminal-Bench (19 real terminal tasks), zaphod or marvin (`TARGET=marvin`) | working | [lab tbench](docs/cli/tbench.md) |
| Quantization accuracy (KL-divergence vs Q8_0) | working | [lab kl](docs/cli/kl.md) |
| Skill A/B tests (the-craft, j5-craft) | working | [lab compare-skill](docs/cli/compare-skill.md), [tuning/instructions/](tuning/instructions/) |
| Every run records exactly what produced it: machine, BIOS, GPUs/VBIOS, kernel, Mesa, image, llama.cpp commit, exact model file + checksum | working | [lab manifest](docs/cli/manifest.md), [tuning/log_run.py](tuning/log_run.py) |
| Fair score across servers (`tooleval_score_common`) | working | [tuning/README.md](tuning/README.md) |
| Series of repeated runs: average, lowest, highest, spread | working | [lab series](docs/cli/series.md) |
| Automatic behavior checks the grader misses (Q4 claim, attacker echo, loops, made-up queries) | working | [tuning/tool-eval/behavior.py](tuning/tool-eval/behavior.py) |
| Real-use speed during every test (from llama-swap's request log) | working | [lab log](docs/cli/log.md) |
| Per-test and per-task tables in MLflow (compare one scenario across models) | working | [tuning/tool-eval/sort_to_mlflow.py](tuning/tool-eval/sort_to_mlflow.py) |

Main results so far (details in [tuning/results.md](tuning/results.md)):
- Tool use (full suite, temperature 0.6, 3 seeds): `coder` 96/93/94 on llama-swap 269, marvin `big` 94/91/88; common score
  (without the 4 scenarios llama.cpp's grammar bug removes): marvin Flash-Next on Strata (thinking medium) 94.0, Ornith 92.0,
  marvin Flash-Next 91.7, Strata Flash-Next (zaphod) 90.0, Flint 86.0. No model made anything up.
- Terminal-Bench on marvin, Flash-Next, full runs: llama.cpp **19/19** in 5.5 h (2026-10-29, BIOS Quiet) and **18/19** in
  5.37 h (2026-11-01, BIOS Performance); Strata 0.2.7 19/19 in 13.3 h (thinking xhigh) and 17/19 in 4.39 h (medium);
  our UnbiasedStrata fork **19/19** in 3.29 h (medium, 2026-10-31). One run each: Terminal-Bench varies run to run.
- Terminal-Bench's runner stays on harbor 0.20.0: the benchmark pins it, and 0.25.0 fails in its prompt hook (tested,
  [plans/harbor-0.25-notes.md](plans/harbor-0.25-notes.md); #2 waits for upstream).
- Strata (another inference engine) tested on zaphod: same quality, needs both cards: [plans/strata.md](plans/strata.md).

## 6. Rebuild, backup, safety

| Feature | Status | Details |
| --- | --- | --- |
| Rebuild either machine from the repo + secrets: BIOS/installer steps, host setup script, pinned model downloads | working (tested on both machines) | [setup/REBUILD.md](setup/REBUILD.md), [setup/host.sh](setup/host.sh), [lab models](docs/cli/models.md) |
| Model lists pinned to exact revisions with sha256 (zaphod 26 files, all verified; marvin 4) | working | [setup/models-zaphod.txt](setup/models-zaphod.txt), [setup/models-marvin.txt](setup/models-marvin.txt) |
| Nightly backup to marvin at 04:17 (Open WebUI, Beszel, MLflow, gateway, .env files), 14 snapshots | working (moves to the NAS: #30) | [setup/backup-to-marvin.sh](setup/backup-to-marvin.sh), [lab backup list](docs/cli/backup.md) |
| Secrets: only in `.env` files (mode 600) + Proton Pass, never in git; checked before commits | working | [setup/REBUILD.md](setup/REBUILD.md) section 5 |
| Services listen only on 127.0.0.1 or the Tailscale address; every image pinned (most by digest) | working | [hub/compose.yaml](hub/compose.yaml) |

## 7. Plans

| Plan | What | Status |
| --- | --- | --- |
| [plans/agent-infra.md](plans/agent-infra.md) | Infrastructure for all agents: Enso on local models, own-task test, gateway, memory, tracing, reliability, hardening | in progress (gateway done) |
| [plans/strix-strata.md](plans/strix-strata.md) | Strata on marvin (sealed off), then port UnbiasedStrata to AMD gfx1151 and contribute upstream | on hold (fork built and benchmarked: 19/19 Terminal-Bench) |
| [plans/maya.md](plans/maya.md) | Project Maya: much bigger models (GLM-5.3-Flash 321B) on marvin; ideas for the fork | to check (after the port) |
| [plans/strata.md](plans/strata.md) | Strata tests on zaphod | done |
| [plans/more-models.md](plans/more-models.md) | Vision, chat, document search, speech, images | done |
| [plans/morning-guide-2026-10-29.md](plans/morning-guide-2026-10-29.md) | Overnight work summary and steps for Daniel | done |
| [GitHub issues](https://github.com/dopamine911/local-ai-lab/issues) | Reservations (#3), card pool (#4), autoresearch (#5), conversation-leak fix (#10), backups to the NAS with a tested restore (#13, #30), ROCm llama.cpp (#16); README.md "What's next" | open |
| [CLAUDE.md](CLAUDE.md) "Next steps" | marvin 256K context, MTP tuning, Terminal-Bench full runs, llama.cpp update policy | ongoing |
