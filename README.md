# local-ai-lab

A two-machine home lab for running AI models locally: chat, coding agents, document search, speech and images, plus
the benchmarks that decide what runs. The goal is a setup that keeps working without hosted frontier models: open
models on our own hardware, measured before and after every change, and rebuildable from this repo.

Status (2026-11-02): both machines run, every service below is in daily use, and the lab is operated with one CLI,
`lab`. Feature-by-feature status: [FEATURES.md](FEATURES.md). Every test result: [tuning/results.md](tuning/results.md).

## Machines

| Machine | Hardware | Role |
| --- | --- | --- |
| **zaphod** | Ryzen 9 3900X, 2x Radeon AI PRO R9700 (32 GB each), Ubuntu 26.04 | Coding and chat models, and every web service (the hub) |
| **marvin** | Bosgame M5: Ryzen AI Max+ 395 (Strix Halo), 128 GB shared memory (120 GB for the GPU), Ubuntu 26.04 | The big model |

Both serve models with [llama-swap](https://github.com/mostlygeek/llama-swap) (one OpenAI-compatible API that loads and
swaps models on demand) on llama.cpp with Vulkan: image `unified-vulkan-269`, llama.cpp b12020. marvin is a llama-swap
peer of zaphod, so its models also answer on zaphod as `marvin/<name>`.

## Models

| Name | Machine | Model | Speed (measured) |
| --- | --- | --- | --- |
| `coder`, `coder-2` | zaphod card 1 / card 2 | Qwen3.8-27B UD-Q4_K_XL, MTP speculative decoding, 64K context | ~54 tok/s writing, ~765 tok/s reading a 32K prompt |
| `coder-tb` | zaphod card 1 | the same, 256K context (benchmarks) | |
| `ornith` | zaphod card 2 | Ornith-1.5 35B-A3B, vision; best tool-use score so far | ~90 tok/s writing |
| `chat` | zaphod card 2 | Gemma 4 31B (QAT Q4_0), vision, 128K | |
| `chat-teacher` | zaphod card 2 | Same Gemma file: 4 slots of 4K, full sliding-window cache, thinking off, no vision; many short parallel requests (unloads after 15 idle min) | |
| `embed`, `rerank` | zaphod card 1 | Qwen3-Embedding-4B, Qwen3-Reranker (document search) | always loaded |
| `stt`, `stt-whisper`, `tts`, `tts-qwen` | zaphod card 2 | Qwen3-ASR, Whisper, Kokoro, Qwen TTS | always loaded (`tts-qwen` on demand) |
| `big` (`marvin/big`) | marvin | Qwen3.8-Flash-Next UD-IQ4_XS (93 GB), MTP draft 3, 128K | ~44 tok/s writing, ~318 tok/s reading a 32K prompt |

Every model file is pinned to an exact revision and sha256 (`setup/models-<machine>.txt`).

## Services

All on zaphod behind Caddy with real certificates, reachable **only over Tailscale** (Caddy listens on zaphod's
Tailscale address; every service listens on 127.0.0.1). Nothing is open to the home network or the internet.

| Address | What |
| --- | --- |
| `https://<DOMAIN>` | Homepage: a start page with each machine's loaded models, GPU load, memory, temperature, power and agent stats |
| `https://chat.<DOMAIN>` | Open WebUI: chat with every model on both machines; documents, web search, speech, code runner, images, agent tools |
| `https://zaphod.<DOMAIN>`, `https://marvin.<DOMAIN>` | The llama-swap APIs (OpenAI-compatible, API key) |
| `https://monitor.<DOMAIN>` | Beszel: both machines, containers, disks, systemd services, network checks |
| `https://mlflow.<DOMAIN>` | MLflow: every benchmark run with its full settings |
| `https://notify.<DOMAIN>`, `https://act.<DOMAIN>` | ntfy (self-hosted push notifications) and its action buttons |
| `https://comfy.<DOMAIN>` | ComfyUI image generation, only while image mode lends it card 2 |
| internal | SearXNG (private web search), Open Terminal (sandboxed code runner), MCPJungle (agent tools), LocalRecall and Mnemosyne (memory) |

Open WebUI's settings live in `hub/compose.yaml`, not in its web page: `lab webui drift` shows anything changed by
hand, `lab webui drift --sync` puts compose's settings back. Per-model settings: `hub/openwebui/models.yaml`, applied
with `lab webui models --apply` (native function calling for all, vision only for `ornith` and `chat`, helper models
hidden). What is set:
- **Accounts:** sign-up off; API keys on (an admin key for scripts in `hub/.env`). `WEBUI_SECRET_KEY` is fixed in `.env`,
  so logins and tool connections survive updates.
- **Documents:** embeddings `embed`, hybrid search, re-ranking `rerank`. After changing the embedding model, re-index
  (Admin Panel → Settings → Documents → "Reindex Knowledge Base Vectors").
- **Speech:** `stt` in (`stt-whisper` for other languages), `tts` out, voice `af_heart` (or `tts-qwen`, which unloads `ornith`).
- **Web search:** SearXNG, 3 results, asks before searching. **Code runner:** Open Terminal (own container and home folder,
  no Docker socket, no host files). **Tools:** the agent tool gateway below, and Context7 (public library docs, hosted:
  those queries leave zaphod).
- **Images:** ComfyUI with Z-Image Turbo (~5 s per image) or Qwen-Image 2.1 (better at text in images), only while
  `lab image-mode on` lends it card 2; the workflow comes from its file via `lab webui comfyui-env [qwen]`.

## Running the lab: `lab`

One command on both machines for everything the lab does (Go, in [cli/](cli/)). `lab` on a terminal opens a menu;
`lab <command> --help` says what a command does and what a good result looks like. Full reference, generated from that
help: **[docs/cli/](docs/cli/README.md)**.

| Task | Command |
| --- | --- |
| Is this machine set up as the repo says? | `lab setup check` |
| Download and verify the models | `lab models` |
| Open WebUI settings and models | `lab webui drift [--sync]`, `lab webui models [--apply]` |
| Lend card 2 to ComfyUI and back | `lab image-mode on\|off\|status` |
| Lend both zaphod cards and marvin to model training and back | `lab training-mode on\|off\|status` |
| Agent tool gateway | `lab mcp status`, `lab mcp setup` |
| Coding agent in a container | `lab enso build\|setup\|up\|check\|tui\|down` |
| Benchmarks (logged to MLflow, result on the phone) | `lab speed`, `lab quality`, `lab tbench`, `lab bench`, `lab kl`, `lab compare-skill` |
| Repeated runs: average and spread | `lab series` |
| Backups | `lab backup list` |

The old scripts (`tuning/speed`, `hub/image-mode`, ...) still work: they are one-line wrappers that call `lab`.

## Agents and their tools

Agents reach tools through one gateway, [MCPJungle](https://github.com/mcpjungle/MCPJungle) (`http://127.0.0.1:8085/mcp`
on zaphod). Every agent has its own token and its own list of allowed tools; set up with `lab mcp setup`
(`hub/mcp/servers/`, `hub/mcp/clients/`).

| Tool | What | Who may use it |
| --- | --- | --- |
| `search` | Web search via our SearXNG, and reading public pages (refuses home-network and Tailscale addresses) | Open WebUI, Claude Code |
| `browser` | Headless Chromium (Playwright MCP) on an internal network, out only through a filtering proxy that refuses home-network, Tailscale and Docker addresses (`docker exec browser-proxy tail /var/log/squid/access.log` shows what it blocked) | Open WebUI, Claude Code |
| `memory` | Search the lab's own notes (LocalRecall over the Markdown in this repo and a private one); read-only | Enso, Claude Code |
| `recall` | Short memories shared by all agents (Mnemosyne): remember, recall, mark wrong or confirmed | Enso, Claude Code |

Agents: Open WebUI, Claude Code, and [Enso](agents/enso/README.md), a coding-agent harness running in a container on
local models with these tools and no internet. Connect Claude Code on zaphod (the token is read from the file, never
typed):

```
claude mcp add --transport http --scope user homelab-tools http://127.0.0.1:8085/mcp --header "Authorization: Bearer $(grep ^MCP_TOKEN_CLAUDE_CODE= ~/homelab/hub/.env | cut -d= -f2)"
```

## Measuring

Nothing changes without a before-and-after measurement, one change at a time. How-to: [tuning/README.md](tuning/README.md).

- **Speed:** `lab speed` (writing, and reading a fixed 32K-token prompt, medians of 3) and `lab bench` (llama-bench).
- **Tool use:** `lab quality` (tool-eval-bench, 69 scenarios), repeated with different seeds as a series; misses sorted
  by hand, plus automatic checks for what the grader misses (made-up facts, attacker text repeated, loops).
- **Real terminal tasks:** `lab tbench` (Terminal-Bench-Local Core-19, 19 tasks, 2 attempts each).
- **Accuracy:** `lab kl` (KL-divergence against the Q8_0 original).
- Every run records what produced it (machine, BIOS, GPUs, kernel, Mesa, image, llama.cpp commit, model file checksum)
  and the real-use speed during the run, in MLflow.

Results so far ([tuning/results.md](tuning/results.md)): tool use 91-96 of 100 for `coder`, `ornith` and marvin's `big`,
with nothing made up; Terminal-Bench on marvin's `big`: 19/19 and 18/19 in two full runs; the Q4 coding model is
within 0.009 KL of full precision.

## Keeping it running

- **Notifications:** every benchmark sends its result to the phone (ntfy, self-hosted), with buttons to rerun it, mark
  it reviewed or open it in MLflow.
- **Watchdog:** 17 read-only checks every minute on both machines (models answer, memory and recall really answer,
  power mode, GPU temperatures, disks, containers, backups), alerting through the same notifications.
- **Versions:** every image, package and download is pinned (most by digest). Renovate opens one PR per newer version,
  with the check to run; updates that move benchmark baselines wait for approval ([dashboard](https://github.com/dopamine911/local-ai-lab/issues/22)).
- **Backups:** every night at 04:17, zaphod's data that isn't in git (chats, monitoring, MLflow, settings, `.env` files)
  goes to marvin as hard-linked snapshots, 14 kept. Moving to the NAS: [#30](https://github.com/dopamine911/local-ai-lab/issues/30).
- **Rebuild:** either machine can be rebuilt from this repo plus the secrets: [setup/REBUILD.md](setup/REBUILD.md)
  (BIOS, installer, `setup/host.sh`, models, secrets, services, first-time web settings). marvin's own setup is logged
  step by step in [marvin/SETUP.md](marvin/SETUP.md).

## Known issues

- llama.cpp rejects requests to the Qwen-family models (`coder`, `big`, `ornith`) that send tools **and** a required JSON
  format ("failed to parse grammar"). Cause found (an empty grammar in llama.cpp's Qwen chat handler); workaround: send
  `tool_choice: "required"` or `"none"` with the format. Details and re-test: [plans/grammar-bug-timeline.md](plans/grammar-bug-timeline.md).
- llama.cpp's RAM prompt cache can put text from one conversation into another (llama.cpp#27148). Mitigated on both
  machines ([#10](https://github.com/dopamine911/local-ai-lab/issues/10)): the cache is off. Cost: returning to a long
  conversation after someone else used the model reads it again (~11 s for 12K tokens on `coder`); marvin's long prompts
  are ~7% slower.
- More, with what to watch for: [CLAUDE.md](CLAUDE.md) "Known issues".

## What's next

| Issue | Plan |
| --- | --- |
| [#3](https://github.com/dopamine911/local-ai-lab/issues/3) | Reservations: book a GPU for a person, benchmark or agent; scripts respect them (`lab reserve`) |
| [#4](https://github.com/dopamine911/local-ai-lab/issues/4) | Use zaphod's two cards as one pool for the coding model (`lab pool`) |
| [#5](https://github.com/dopamine911/local-ai-lab/issues/5) | Autoresearch: agents test performance ideas in idle time and report them for review (`lab research`) |
| [#34](https://github.com/dopamine911/local-ai-lab/issues/34) | Turn llama.cpp's prompt cache back on once it fixes the conversation leak (the workaround, #10, is in place), measured |
| [#13](https://github.com/dopamine911/local-ai-lab/issues/13), [#30](https://github.com/dopamine911/local-ai-lab/issues/30) | Database dumps, a tested restore, and backups on the NAS |
| [#16](https://github.com/dopamine911/local-ai-lab/issues/16) | A ROCm build of llama.cpp next to Vulkan, same llama.cpp commit |
| [#2](https://github.com/dopamine911/local-ai-lab/issues/2) | Terminal-Bench's runner (harbor): follow when the benchmark moves to a newer version |

All open work: [issues](https://github.com/dopamine911/local-ai-lab/issues). Longer plans: [plans/](plans/).

## Repository layout

```
cli/            the lab CLI (Go); config in cli/lab.yaml (no secrets)
docs/cli/       command reference, generated from lab's help
docs/diagrams/  how the pieces fit together
zaphod/         zaphod's llama-swap: compose, config, Dockerfile (~/llama-swap links here)
marvin/         marvin's llama-swap, fan control, setup log, Strata experiments
hub/            zaphod's web services (compose.yaml) and their config: Caddy, Open WebUI, MCP gateway, memory, ...
agents/enso/    the Enso coding agent in a container
tuning/         benchmarks, results, MLflow logging
setup/          rebuild a machine: REBUILD.md, host.sh, model lists, version pins
plans/          plans and timelines of the work, step by step
```

## Ground rules

- **Public repo:** no secrets in git. Keys live in `.env` files (mode 600, never committed) and a password manager;
  scripts read them and never print them. Device-level details are in a separate private repo.
- **Rebuildable:** every change to a machine goes into a script or `setup/REBUILD.md` in the same commit.
- **Measured:** one change at a time, before and after, recorded in files and MLflow.
- **Private by design:** services listen on 127.0.0.1 or the Tailscale address only.
