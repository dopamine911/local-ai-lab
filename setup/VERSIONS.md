# Pinned versions

Rule (Daniel, 2026-10-30): pin every image, package and download to its **latest** version, exactly (digest, exact
version or sha256). No minimum age. Read a new major version's changelog before switching.

Last audit: **2026-10-30**. To audit: compare each row with its project's newest release; bump, rebuild or recreate
one service at a time, and run its check.

| What | Where pinned | Version | Check after a bump |
| --- | --- | --- | --- |
| Open WebUI | hub/compose.yaml | v0.12.5 (bumped from 0.12.4) | `hub/openwebui/drift.sh` and `models.sh`: 0 differ |
| Open Terminal | hub/compose.yaml | 0.14.7 (from 0.14.5) | container up |
| MCPJungle server + CLI | hub/compose.yaml, hub/mcp/setup.sh `V=` | 0.4.7 (from 0.4.6) | `hub/mcp/setup.sh`: every server and client line |
| mcp-searxng | hub/compose.yaml | 2.9.0 (from 2.6.0; 2.8.x fixed 4 CVEs) | a search through the `search` tool |
| SearXNG | hub/compose.yaml | 2026.10.30-ffa88e3ba (from 2026.10.25) | same |
| Squid (browser proxy) | hub/compose.yaml | 7.2-26.04_edge (from 6.6-24.04; `squid-gnutls -k parse` OK) | browser: public page loads, `http://192.168.1.1/` gets 403 |
| Playwright MCP | hub/compose.yaml | v0.0.83 (latest) | same |
| Beszel + agent | hub/compose.yaml, marvin/llama-swap/compose.yaml | 0.21.1 (latest; digest added) | Beszel page |
| MLflow | hub/compose.yaml | v3.17.2 (latest; digest added) | MLflow page |
| Homepage | hub/compose.yaml | v2.7.0 (latest; digest added) | start page |
| Caddy | hub/caddy/Dockerfile | 2.11.7 (latest; digest added: Docker republished the tag, takes effect at the next Caddy rebuild) | every `*.ai.<domain>` page |
| ComfyUI | hub/comfyui/Dockerfile | v0.43.0 (latest); Python base pinned 3.13.18 at next build | image mode |
| LocalRecall + Postgres | hub/compose.yaml | v0.6.6 (latest) | `memory` search |
| LocalRecall MCP / mcp-proxy | hub/memory/mcp-localrecall.Dockerfile | mcps v0.0.4, mcp-proxy 0.13.0 (latest); Python 3.12.16 | same |
| Mnemosyne | hub/memory/mnemosyne.Dockerfile | 4.0.0 (latest); Python 3.12.16 | `recall` finds the REBUILD rule |
| Python (relays, sync, stats) | compose files | 3.13.18-slim-trixie, 3.12.16-slim (were floating `3.13-slim-trixie` / a bare digest) | containers up |
| Enso image | agents/enso/Dockerfile | Ubuntu resolute-20261024.1, snapshot 2026-10-30, bun 1.4.4 (from 1.4.0), Enso 537559a, pi 1.1.0 | `agents/enso/enso.sh check` |
| Ubuntu base of llama-swap images | zaphod/ and marvin/llama-swap/Dockerfile | resolute-20261024.1 (was the floating `ubuntu:26.04`); takes effect at the next llama-swap build | — |

## Behind on purpose (each needs a measured before/after; Daniel decides when)

| What | Pinned | Latest | Why not bumped in the audit |
| --- | --- | --- | --- |
| llama-swap (and the llama.cpp build inside it), issue #1 | 267 (llama.cpp b11870) | v268 (2026-10-28) | Changes every speed and quality baseline in tuning/results.md; marvin is busy with the port. Bump as its own change: speed + tool-use before/after, re-test the grammar bug (CLAUDE.md Known issues) |
| harbor (Terminal-Bench runner), issue #2 | 0.20.0 | 0.25.0 (2026-10-25) | Results would no longer compare with the 19/19 and 17/19 runs; bump with a fresh baseline run |
| pi (inside Enso) | 1.1.0 | 1.4.0 | Enso supports only pi >=1.1.0 <1.2.0 (`SUPPORTED_PI_VERSIONS`); moves when Enso does |
| tool-eval-bench | 3.0.0 | 3.0.0 | current |
