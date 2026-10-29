# Homelab

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

## Open WebUI tools

- **Web search:** SearXNG (`hub/searxng/settings.yml`), private, no API keys. Defaults set in compose; check Admin Panel → Settings → Web Search.
- **Open Terminal:** a sandboxed Linux shell and file area (own container, own home folder in `hub/data/open-terminal`, no Docker socket, no host files). Key: `OPEN_TERMINAL_API_KEY` in `hub/.env`. Connected in Admin Panel → Settings → Integrations → Open Terminal, URL `http://127.0.0.1:8000`.
- **Context7 (MCP):** current library documentation for coding questions. Hosted by Upstash (queries leave zaphod), no key. Admin Panel → Settings → Integrations → External Tool Servers, type MCP (Streamable HTTP), URL `https://mcp.context7.com/mcp`, auth None.
- **Document search (embeddings + reranker on zaphod):** Admin Panel → Settings → Documents.
  Embedding Model Engine: **OpenAI**, URL `http://127.0.0.1:8080/v1`, key: ZAPHOD_KEY, model `embed`.
  Hybrid Search: **on**. Reranking Engine: **External**, URL `http://127.0.0.1:8080/v1/rerank`, key: ZAPHOD_KEY, model `rerank`.
  After changing the embedding model, re-index existing documents (same page, "Reindex Knowledge Base Vectors").
- **Speech-to-text (on zaphod):** Admin Panel → Settings → Audio → Speech-to-Text Engine: **OpenAI**,
  URL `http://127.0.0.1:8080/v1`, key: ZAPHOD_KEY, model `stt` (Qwen3-ASR, ~30 languages).
  For other languages use model `stt-whisper` (Whisper, 99 languages).
- **Text-to-speech (on zaphod):** Admin Panel → Settings → Audio → Text-to-Speech Engine: **OpenAI**,
  URL `http://127.0.0.1:8080/v1`, key: ZAPHOD_KEY, model `tts`, voice `af_heart` (Kokoro; other voices such as
  `bf_emma`, `am_michael`, `bm_george`). For a more expressive voice or other languages: model `tts-qwen`,
  voice `Ryan` or `Vivian` (unloads `ornith` if it is loaded).
- **Image generation (ComfyUI, only while image mode is on):** Admin Panel → Settings → Images: engine **ComfyUI**,
  URL `http://127.0.0.1:8188`, workflow: upload `hub/comfyui/workflows/z-image-turbo-api.json`, then map the nodes:
  prompt → node `5` (text), model → `1`, width/height → `7`, steps and seed → `8`. Default size 1024x1024, steps 8.
  Alternative, better at text inside images: `hub/comfyui/workflows/qwen-image-2.1-turbo-api.json`, nodes:
  prompt → `5` (field `prompt`), model → `1`, width/height → `6`, seed → `7` (`noise_seed`). Steps are fixed (8).

## Agent tools: MCP gateway (MCPJungle)

One place for the tools agents use (plans/agent-infra.md, step 3). Runs on zaphod at `http://127.0.0.1:8085/mcp`.
Every agent has its own token and list of allowed tool servers; no token or a wrong one gets 401.

| Tool server | What | Allowed for |
| --- | --- | --- |
| `search` | Web search via our SearXNG, and reading public web pages (`web_url_read` refuses home-network and Tailscale addresses) | openwebui, claude-code |

Setup and rebuild: `hub/mcp/setup.sh` (servers in `hub/mcp/servers/`, agent logins in `hub/mcp/clients/`,
tokens in `hub/.env` as `MCP_TOKEN_<NAME>`). Admin: `mcpjungle list tools`, `mcpjungle list mcp-clients`.

Connect an agent:
- **Open WebUI:** `~/homelab/hub/mcp/openwebui-connect.sh` (adds or updates the connection through Open WebUI's admin
  API; needs `OPENWEBUI_API_KEY` in `hub/.env`, an API key of your admin account). By hand instead: Admin Panel →
  Settings → External Tools, type **MCP (Streamable HTTP)**, URL `http://127.0.0.1:8085/mcp`, auth **Bearer**, key
  `MCP_TOKEN_OPENWEBUI` from `hub/.env`.
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
