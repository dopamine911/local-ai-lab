# Homelab

What this lab does, feature by feature, with links to every detail: [FEATURES.md](FEATURES.md).

```
~/homelab/
  CLAUDE.md                  notes for Claude (link to ~/llama-swap/CLAUDE.md)
  hub/                       zaphod's web services: Caddy, Open WebUI, Beszel, Homepage
    compose.yaml
    .env.example             copy to .env and fill in
    caddy/Caddyfile          routes per subdomain
    caddy/Dockerfile         Caddy + Cloudflare DNS module (image caddy-cloudflare:2.11.7, already built)
    homepage/                start page links
    models-panel/            put index.html here (from the claude.ai project)
  marvin/llama-swap/         files to copy to marvin tonight
~/llama-swap/                zaphod's llama-swap (running; separate on purpose)
```

| Address | Service | Listens on |
| --- | --- | --- |
| `https://<DOMAIN>` | Homepage | 127.0.0.1:3001 |
| (internal) | SearXNG: private web search for Open WebUI | 127.0.0.1:8888 |
| (internal) | Open Terminal: sandboxed shell for the models, used by Open WebUI | 127.0.0.1:8000 |
| `https://chat.<DOMAIN>` | Open WebUI | 127.0.0.1:3000 |
| `https://monitor.<DOMAIN>` | Beszel | 127.0.0.1:8090 |
| `https://models.<DOMAIN>` | Models panel | files in hub/models-panel |
| `https://zaphod.<DOMAIN>` | zaphod llama-swap | 127.0.0.1:8080 |
| `https://marvin.<DOMAIN>` | marvin llama-swap | marvin:8080 over Tailscale |

Caddy listens only on zaphod's Tailscale address, so none of this is reachable from the home network or the internet.

## Open WebUI settings

**All Open WebUI settings are in `hub/compose.yaml`** (the `open-webui` environment block), not in the web page.
A rebuilt Open WebUI starts with them. Change a setting = edit compose.yaml, then `~/homelab/hub/openwebui/drift.sh --sync`.

- `hub/openwebui/drift.sh`: compares the running Open WebUI with compose.yaml (builds a throwaway copy, no network,
  about a minute). Good result: `0 setting(s) differ`. "web page only" lines = changed by hand: add them to compose or undo.
- `hub/openwebui/drift.sh --sync`: copies the compose-managed settings (`hub/openwebui/managed-keys.txt`) into the
  running Open WebUI; stops it for a few seconds; saves `data/open-webui/webui.db.before-sync-<date>` first.

What is set (2026-10-29, checked working end to end):
- **Accounts:** sign-up off; API keys on (`OPENWEBUI_API_KEY` in `hub/.env` = an admin key for scripts).
- **Web search:** SearXNG (`hub/searxng/settings.yml`), private, 3 results, asks before searching.
- **Documents:** embeddings `embed`, hybrid search on, re-ranking `rerank` (both llama-swap, card 1). After changing
  the embedding model, re-index (Admin Panel → Settings → Documents → "Reindex Knowledge Base Vectors").
- **Speech:** speech-to-text `stt` (Qwen3-ASR; `stt-whisper` for other languages), text-to-speech `tts` voice
  `af_heart` (Kokoro; or `tts-qwen` with `Ryan`/`Vivian`, which unloads `ornith`).
  Fixed 2026-10-29: before, the models were empty / `tts-1`, so text-to-speech and re-ranking failed (404).
- **Tools:** the MCP gateway (`homelab-tools`, next section) and Context7 (public library docs, hosted by Upstash:
  queries leave zaphod, no key).
- **Code runner:** Open Terminal (own container, own home folder `hub/data/open-terminal`, no Docker socket, no host files).
- **Images:** ComfyUI, Z-Image Turbo, 1024x1024, 8 steps (first image ~26 s incl. loading). **Only while image mode is
  on** (`~/homelab/hub/image-mode on`, then `off`): it borrows card 2. Workflow + node map are generated from the
  workflow file: `hub/openwebui/make-comfyui-env.sh` (or `... qwen` for Qwen-Image, better at text in images), then
  `drift.sh --sync`.
- **Models** (`hub/openwebui/models.yaml`, apply with `hub/openwebui/models.sh --apply`; without `--apply` it only checks):
  native function calling for all (also the default for new models), vision only for `ornith` and `chat`, helper models
  (embed, rerank, stt, tts, coder-tb) hidden from the model picker. Names come from the llama-swap configs.

## Agent tools: MCP gateway (MCPJungle)

One place for the tools agents use (plans/agent-infra.md, step 3). Runs on zaphod at `http://127.0.0.1:8085/mcp`.
Every agent has its own token and list of allowed tool servers; no token or a wrong one gets 401.

| Tool server | What | Allowed for |
| --- | --- | --- |
| `search` | Web search via our SearXNG, and reading public web pages (`web_url_read` refuses home-network and Tailscale addresses) | openwebui, claude-code |
| `browser` | Headless Chromium (Playwright MCP 0.0.83): open pages, click, type, read, screenshot. On an internal network; its only way out is `browser-proxy` (Squid, `hub/browser-proxy/squid.conf`), which refuses home-network, Tailscale, Docker and zaphod addresses (tested). See blocked requests: `docker exec browser-proxy tail /var/log/squid/access.log` (TCP_DENIED) | openwebui, claude-code |
| `memory` | Search the lab's own notes (LocalRecall over ~/homelab and ~/full-homelab Markdown; synced hourly by `hub/memory/sync.sh`). Read-only. Raw tool-test transcripts and Terminal-Bench tasks are excluded (attacker text, benchmark contamination) | enso, claude-code |
| `recall` | Short memories shared by all agents (Mnemosyne 4.0.0, one SQLite file in hub/data/mnemosyne): remember, recall, get, invalidate, validate (the other 25 tools are off, `hub/mcp/servers/recall.tools`) | enso, claude-code |

Setup and rebuild: `hub/mcp/setup.sh` (servers in `hub/mcp/servers/`, agent logins in `hub/mcp/clients/`,
tokens in `hub/.env` as `MCP_TOKEN_<NAME>`). Admin: `mcpjungle list tools`, `mcpjungle list mcp-clients`.

Connect an agent:
- **Open WebUI:** already connected through `hub/compose.yaml` (`TOOL_SERVER_CONNECTIONS`, token `MCP_TOKEN_OPENWEBUI`).
- **Claude Code** on zaphod (one command; reads the token from the file, never typed):
  `claude mcp add --transport http --scope user homelab-tools http://127.0.0.1:8085/mcp --header "Authorization: Bearer $(grep ^MCP_TOKEN_CLAUDE_CODE= ~/homelab/hub/.env | cut -d= -f2)"`

## Images: ComfyUI on card 2

`hub/image-mode on` lends card 2 to ComfyUI (https://comfy.<DOMAIN>): it switches llama-swap to the `image-mode`
profile (coder-2 requests answered by coder on card 1; ornith, chat, north, flint, tts-qwen blocked), unloads card 2's
chat model and starts ComfyUI. `hub/image-mode off` stops ComfyUI and gives card 2 back. `hub/image-mode status` shows
the state. Document search and speech keep working throughout. ComfyUI has no login: anyone on the tailnet can use it
while it runs. Models live in /srv/models/comfyui; outputs in hub/data/comfyui/output.
Image: `comfyui-zaphod:v0.43.0-torch2.15.0-rocm7.14`, built from hub/comfyui/Dockerfile (30.6 GB; PyTorch ROCm wheels).

- Built in already (no setup): read a web page, current time, run code, memory, knowledge bases.
- Each model: Admin Panel → Settings → Models → (model) → Advanced Params → Function Calling: **Native**.
- `WEBUI_SECRET_KEY` is fixed in `.env` so logins and saved tool connections survive container updates.

## Set up or rebuild a machine

Follow `setup/REBUILD.md`: BIOS and installer choices, then `setup/host.sh` (host settings), `setup/models.sh`
(pinned model downloads), secrets, and `docker compose`. marvin's first setup is logged step by step in `marvin/SETUP.md`.

marvin is linked to zaphod as a llama-swap peer (`peers:` at the end of `zaphod/llama-swap/config/config.yaml`), so its
models show on zaphod as `marvin/<name>`, in Open WebUI too. Its key is `MARVIN_KEY` in zaphod's `~/llama-swap/.env`.

## Hub (after marvin, or any time)

Needs your real domain on Cloudflare and a Cloudflare API token.

1. `cd ~/homelab/hub && cp .env.example .env && chmod 600 .env`
2. Copy the key: `grep ZAPHOD_KEY ~/llama-swap/.env >> .env`, then `nano .env`: set `DOMAIN`, `CF_API_TOKEN`, `MARVIN_KEY`, and delete the `ZAPHOD_KEY=CHANGE_ME` line.
3. In Cloudflare DNS: `A` records for `<DOMAIN>` and `*.<DOMAIN>` → `100.117.73.97` (zaphod's Tailscale address), **DNS only** (grey cloud).
4. `docker compose up -d`
   Good: `docker compose logs caddy` says `certificate obtained successfully`.
5. Open `https://<DOMAIN>` from a device on Tailscale. Good: the Homepage start page.
6. Open `https://chat.<DOMAIN>` and create the admin account (first user becomes admin). Good: `coder`, `coder-2` (and `marvin/big`) in the model list.
7. **Monitoring:** open `https://monitor.<DOMAIN>`, create the account, **Add system**: name `zaphod`, host `127.0.0.1`, port `45876`. Copy the key it shows into `BESZEL_KEY` in `.env`, then `docker compose --profile agent up -d`. For marvin: host `marvin`, port `45876`, and the same steps in marvin's folder.
8. **Models panel:** copy `models-panel/index.html` from the claude.ai project into `hub/models-panel/`.
