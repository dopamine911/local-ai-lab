# Enso in a container (zaphod)

Daniel's Enso harness around the pi coding agent, running the lab's local models, with the lab's tools through
MCPJungle, sealed off from the internet. Purpose: let local models work on Enso's own issues, to see where they
fall short. Plan: `plans/agent-infra.md` step 1.

```
            zaphod
  ┌─────────────────────────────── agents-internal (no internet, no host) ─┐
  │  enso container                  llama-swap :8080     mcpjungle :8080   │
  │  Enso + pi 1.1.0  ── models ──▶  coder, coder-tb,     search, browser   │
  │  /work/enso-harness  ── tools ─────────────────────▶  (token: enso)     │
  └──────────┬──────────────────────────────────────────────────────────────┘
             │ enso-relay (also on a normal network)
   127.0.0.1:5178 on zaphod  ◀── your browser (on zaphod, or ssh -L 5178:127.0.0.1:5178 zaphod)
```

## Use

1. `./enso.sh build` (once per Enso commit; ~3 min, 2.9 GB).
2. `./enso.sh setup` (keys and config; safe to rerun after editing `config/`).
3. `./enso.sh up`. Good result: `Enso: http://localhost:5178`.
4. `./enso.sh check`. Good result: `ok: models`, `ok: no internet`, `ok: tools (mcpjungle): connected, 36 tools` (browser 25, search 4, memory 2, recall 5).
5. Work: `./enso.sh tui --model lab/coder` (Enso's terminal on the workspace, with MCP tools), or the web page.
   One-shot: `docker exec -w /work/enso-harness enso bun /opt/enso/scripts/enso.ts -e builtin:mcp -e builtin:tool-search --model lab/coder -p "..."`.
6. `./enso.sh down` when done.

The agent's changes stay in `~/agents/work/enso-harness` (its own clone, commits as "enso-agent (local model)",
no push credentials). Review them there; nothing reaches GitHub unless Daniel pushes.

## What is where

| Path on zaphod | In the container | What |
| --- | --- | --- |
| `~/agents/enso-home/` (700) | `/home/ubuntu/.enso` (ENSO_HOME) | Enso's state: logs, sessions, permissions.json, config.json |
| `~/agents/enso-home/keys/` (600) | same | `zaphod-key` (llama-swap), `mcp-token` (MCPJungle login `enso`); copied by `setup`, never printed |
| `~/agents/enso-home/pi-agent/` | pi's agent dir | `models.json` and `mcp.json`, copied from `config/` by `setup` |
| `~/agents/work/enso-harness` | `/work/enso-harness` | the workspace (a clone) |
| image `/opt/enso` | | Enso at commit 537559a, with packages installed; pi = its lockfile's 1.1.0 |

Models (`config/models.json`, provider `lab`): `coder` (64K), `coder-tb` (256K, same model; swaps with `coder`),
`ornith` (256K, vision), `chat` (128K, vision), `marvin/big` (128K; offline while marvin runs other tests).

## Things learned setting it up (2026-10-30)

- **pi 1.1 has MCP built in** (`docs/mcp.md` in the pi package), but Enso starts pi with `--no-extensions`, which
  drops it, and Enso's permission modes deny every MCP server not approved by name. So: `-e builtin:mcp -e
  builtin:tool-search` (pi's documented way to load only those) and `"mcp": {"lab": "allow"}` in permissions.json
  (what `core/src/permission-modes.ts` describes). No Enso code change.
- **The web page has no MCP tools yet**: Enso's web host also turns discovery off (`noExtensions`), and there is no
  flag to pass there. That needs a change in Enso (Daniel's code). Use the terminal (`tui`) for MCP work.
- **Search tools are direct, browser tools deferred** (`toolExposure` in `config/mcp.json`). With all deferred,
  `coder` used Enso's own `web_search` (no backend for local models) and said so instead of searching.
- Enso removes secret-shaped environment variables before starting pi, so keys come from files via pi's
  `!command` values, inside ENSO_HOME, which Enso's guards keep the agent out of.
- `hub/mcp/setup.sh` stopped after the first client (stderr + pipefail); fixed, which also gave Open WebUI the
  `browser` tool its client file always listed.
- Telemetry: `PI_TELEMETRY=0`, and there is no internet.
