# Pinned versions

Rule (Daniel, 2026-10-30): pin every image, package and download to its **latest** version, exactly (digest, exact
version or sha256). No minimum age. Read a new major version's changelog before switching.

Last audit: **2026-10-30** by hand. From issue #7: **Renovate** (`renovate.json`) watches 90 pins and opens one PR per
update, with this table's check in the PR body; held-back items wait for approval on its Dependency Dashboard issue.
Merge one at a time and run the check. Dry run (changes nothing): see plans/renovate-timeline.md.

**Not covered by Renovate (check by hand at each audit):** harbor (pinned in tuning/terminal-bench, not in git; #2);
Mesa (apt pin in both llama-swap Dockerfiles); Enso's Ubuntu snapshot date (`UBUNTU_SNAPSHOT`); ComfyUI's torch/ROCm
(`TORCH`, `ROCM`; torchvision unpinned); Caddy's Cloudflare plugin (`xcaddy --with`, unpinned); other apt packages
(take the Ubuntu archive of the build day); images we build ourselves (their inputs are covered).

| What | Where pinned | Version | Check after a bump |
| --- | --- | --- | --- |
| Open WebUI | hub/compose.yaml | v0.12.5 (bumped from 0.12.4) | `hub/openwebui/drift.sh` and `models.sh`: 0 differ |
| Open Terminal | hub/compose.yaml | 0.14.7 (from 0.14.5) | container up |
| MCPJungle server + CLI | hub/compose.yaml, hub/mcp/setup.sh `V=` | 0.4.7 (from 0.4.6) | `hub/mcp/setup.sh`: every server and client line |
| mcp-searxng | hub/compose.yaml | 2.9.0 (from 2.6.0; 2.8.x fixed 4 CVEs) | a search through the `search` tool |
| SearXNG | hub/compose.yaml | 2026.10.30-ffa88e3ba (from 2026.10.25) | same |
| Squid (browser proxy) | hub/compose.yaml | 7.2-26.04_edge (from 6.6-24.04; `squid-gnutls -k parse` OK) | browser: public page loads, `http://192.168.1.1/` gets 403 |
| ntfy | hub/compose.yaml | v2.35.0 (latest, added 2026-10-30) | `hub/notify "test" "x"` arrives on the phone (verified 2026-11-01) |
| Playwright MCP | hub/compose.yaml | v0.0.83 (latest) | same |
| Beszel + agent | hub/compose.yaml, marvin/llama-swap/compose.yaml | 0.21.1 (latest; digest added) | Beszel page |
| MLflow | hub/compose.yaml | v3.17.2 (latest; digest added) | MLflow page |
| Homepage | hub/compose.yaml | v2.7.0 (latest; digest added) | start page |
| Caddy | hub/caddy/Dockerfile | 2.11.7 (latest; digest added: Docker republished the tag, takes effect at the next Caddy rebuild) | every `*.ai.<domain>` page |
| ComfyUI | hub/comfyui/Dockerfile | v0.43.0 (latest); Python base pinned 3.13.18 at next build | image mode |
| LocalRecall + Postgres | hub/compose.yaml | v0.6.6 (latest) | `memory` search |
| LocalRecall MCP / mcp-proxy | hub/memory/mcp-localrecall.Dockerfile | mcps v0.0.4, mcp-proxy 0.13.3 (Renovate #25, 2026-11-02; needs `--allowed-host mcp-localrecall:8080` since 0.13.3); Python 3.12.16 | `memory` search through MCPJungle |
| Mnemosyne | hub/memory/mnemosyne.Dockerfile | 4.0.0 (latest); Python 3.12.16 | `recall` finds the REBUILD rule |
| Python (relays, sync, stats, webui drift) | compose files, hub/openwebui/drift.sh, cli/internal/webui | 3.13.18-slim-trixie, 3.12.16-slim (were floating `3.13-slim-trixie` / a bare digest) | containers up |
| Enso image | agents/enso/Dockerfile | Ubuntu resolute-20261024.1, snapshot 2026-10-30, bun 1.4.4 (from 1.4.0), Enso 537559a, pi 1.1.0 | `agents/enso/enso.sh check` |
| lab CLI: Go | cli/build.sh `IMG=` | golang:1.27.3 by digest (latest) | `cli/build.sh` (vet, tests, build) |
| lab CLI: libraries | cli/go.mod + go.sum | cobra v1.10.2, bubbletea v2.1.1, lipgloss v2.0.6, go.yaml.in/yaml v3.0.6 (latest stable; yaml v4 is still rc) | same + `lab manifest` diff against tuning/manifest.sh |
| Ubuntu base of llama-swap images | zaphod/ and marvin/llama-swap/Dockerfile | resolute-20261024.1 (was the floating `ubuntu:26.04`); in zaphod's 269-media build (2026-11-02; the comment on the FROM line had broken builds until then) | — |
| Mesa (Vulkan driver) in the llama-swap images | zaphod/llama-swap/Dockerfile | 26.0.8-1ubuntu0.3 (pinned 2026-11-02, = current Ubuntu 26.04; was unpinned) | `tuning/speed` before/after a bump |

## Behind on purpose (each needs a measured before/after; Daniel decides when)

| What | Pinned | Latest | Why not bumped in the audit |
| --- | --- | --- | --- |
| llama-swap (and the llama.cpp build inside it), issue #1 | **image `unified-vulkan-269` on both machines** (nightly; its llama-swap program reports v268; llama.cpp b12020; zaphod and marvin 2026-11-02, measured before/after: tuning/results.md) | v269 (2026-11-01) | — |
| harbor (Terminal-Bench runner), issue #2 | 0.20.0 | 0.25.0 (2026-10-25) | Results would no longer compare with the 19/19 and 17/19 runs; bump with a fresh baseline run |
| pi (inside Enso) | 1.1.0 | 1.4.0 | Enso supports only pi >=1.1.0 <1.2.0 (`SUPPORTED_PI_VERSIONS`); moves when Enso does |
| tool-eval-bench | 3.0.0 | 3.0.0 | current |
