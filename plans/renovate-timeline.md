# Renovate (issue #7): timeline

What was done and when. Times are Chicago time (CST), from `date` when written. Config: `renovate.json`.

## 2026-11-02

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 14:40 | Branch `renovate-config` (worktree `~/homelab-renovate`); Renovate 44.206.0 (latest, 2026-11-02), image pinned by digest for local runs | — | — |
| 14:41 | Inventory of pins: 22 compose images, 11 Dockerfile FROMs, Go modules, 2 pip pins, MCPJungle CLI `V=`, ComfyUI `ARG`, bun download, Enso commit, fan-module commit | gaps found: Caddy Cloudflare plugin and torchvision unpinned; harbor pin outside git; two FROMs had a digest but no tag (Enso's Ubuntu, mcps/localrecall): tags added (same digests) | this commit |
| 14:42 | `renovate.json`: config:recommended without grouping (one change per PR), pin digests, label `versions`, check notes from VERSIONS.md, 7 custom patterns, approval (dashboard) for llama-swap, bun, Enso, fan module | `renovate-config-validator`: valid | this commit |
| 14:43 | Dry run, lookups only (`--platform=local --dry-run=lookup`, GitHub token from `gh` in a mode-600 file, deleted after) | 90 pins found, 0 errors (first try failed only because a worktree's `.git` is a pointer; ran from a plain snapshot). Newer: ntfy 2.35.0 → **2.37.1**, Open Terminal 0.14.7 → **0.16.0**, mcp-proxy 0.13.0 → **0.13.3**, SearXNG nightly, Python 3.12/3.13 → **3.15.2**, new digests for Squid and Python, marvin llama-swap 267 → 269 | — |
| 14:45 | Two rules from the dry run: SearXNG only Monday mornings (it publishes almost daily); Python minor/major on approval | valid | this commit |
| 14:47 | Daniel: **hosted app** (decisions.md 40) | order: merge the config PR first, then install the app (else it opens a default onboarding PR) | REBUILD.md "Renovate" |
| 14:50 | Daniel merged #19 and installed the app (this repo only) | — | — |
| 14:54 | First run | Dependency Dashboard #22; PRs #20 (python 3.12.16 digest) and #21 (python 3.13.18 digest); pending approval: llama-swap (marvin 267 pin + 269), Python 3.15.2 x2; rate-limited: squid digest, mcp-proxy 0.13.3, ntfy 2.37.1, Open Terminal 0.16.0, SearXNG (check whether the Monday-morning schedule holds it back). Issue #7 closed | #22 |
| 14:57 | **#20** (python 3.12.16-slim digest → b18b261) merged by Daniel; rebuilt `mcp-localrecall` and `mnemosyne`, pulled the memory-sync image | before and after: `recall` finds the REBUILD rule (f539fa74), `memory` search finds the watchdog timeline (same 2 results); both on Python 3.12.16; `lab mcp status` matches the repo; watchdog quiet | — |
| 14:59 | **#21** (python 3.13.18-slim-trixie digest → ae14d4c) merged by Daniel; recreated zaphod's `gpu-stats`, `agent-stats` (hub) and `enso-relay` | same answers before/after (GPU tiles, agent totals, relay 200); `lab enso check` ok; all on image ae14d4ce8771. ComfyUI takes it at its next build | — |
| 15:23 | marvin's `gpu-stats` and `agent-stats` recreated on the new python 3.13.18 digest together with marvin's 269 switch (files copied to marvin's checkout; after the next push: `git checkout -- marvin/llama-swap && git pull` there) | both running on ae14d4ce8771 | — |
| 15:19 | **#24** (ubuntu/squid digest → 74a4935) merged; `browser-proxy` recreated | before/after through the proxy: public https://example.com 200, `http://192.168.1.1/` 403 (both unchanged); `squid-gnutls -k parse` exit 0 (warns "HTTP requires the use of Via": same version and config, very likely pre-existing, not recorded before) | — |
| 15:20 | **#25** (mcp-proxy 0.13.0 → 0.13.3) merged; image tag in compose still said proxy0.13.0: fixed; rebuilt `mcp-localrecall` | **memory search broke**: "Invalid Host header: mcp-localrecall:8080" (0.13.3 checks the Host header). Fix: `--allowed-host mcp-localrecall:8080`. After: memory search same result as before; other host names refused (421); Enso 36 tools; `lab mcp status` matches | this commit |
| 15:20 | renovate.json: mcp-proxy PRs get their own check note (rebuild, tag, memory search) | validated | this commit |
| 17:48 | **#27** (ntfy 2.35.0 → 2.37.1) merged; release notes 2.36/2.37: nothing we use removed; 2.36 re-checks the whole config at startup when base-url is set. `ntfy` recreated | before and after: healthy, the 10 access checks **10/10** (daniel reads lab-* and stats/auth; every other read/send 403; web app 404); startup log clean ("ntfy 2.37.1"); test notification sent 17:48: **arrived** (Daniel) | — |
| 17:48 | **#28** (mcp-proxy 0.13.3 → 0.14.0, no breaking changes in its notes) merged; compose image tag still said proxy0.13.3 (again): fixed; the Dockerfile's "0.13.0 (latest)" comment now points at the pin; `mcp-localrecall` rebuilt | before and after: watchdog mcp-calls ok (memory 1, recall 1 result), `lab enso check` 36 tools; mcp-proxy 0.14.0 in the container, log clean | this commit |
| next | Open Terminal 0.16.0, SearXNG (queued) | — | — |

## 2026-11-17: deploying the 8 updates Daniel merged (#32, #35-#39, #42, #43), issue #44

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 09:56 | Fresh backup before the updates (`lab backup`) | marvin:backups/zaphod/2026-11-17_0956 (263M) | — |
| 09:57 | **#36 Open WebUI 0.12.5 → 0.13.4**, before: `lab webui drift` on 0.12.5 (run from a worktree of 4d28628, LAB_REPO), `lab webui models` | drift **0 differ**; models: 7 differ (pre-existing: models.yaml entries never applied with `--apply`: chat-teacher name, chat-teacher-c1, -26b, -26b-c1, gpt-oss, marvin/gpt-oss, marvin/chat-teacher) | — |
| 09:57 | Release notes 0.13.0-0.13.4 read. For us: DB migrations (backup taken); image env vars (`IMAGE_GENERATION_ENGINE`, `COMFYUI_BASE_URL`, ...) are read only on the first start, then turned into an "image connection" stored in the DB; sub-agent limits default to none; Direct Connections removed (we had them off) | — | — |
| 09:58 | `docker compose pull open-webui && up -d open-webui` | healthy at 10:00:44; 6 schema migrations; log "Migrated image generation settings to 1 image connection(s)"; also warns `CORS_ALLOW_ORIGIN IS SET TO '*'` (not checked whether 0.12.5 warned too) | — |
| 10:01 | After: `lab webui drift`, `lab webui models` | models: same 7 as before. drift: **12 differ, all "web page only"** (none of compose's managed settings changed). Compared in full with a fresh 0.13.4 DB: `images.connections` identical except its random id (so drift will always flag it, also `image_generation.default_model` = `<id>:z_image_turbo_bf16.safetensors`); new 0.13.4 defaults the running DB kept old values for: `rag.chunk_size` 1000 (new 1024), `rag.top_k` 3 (10), `rag.top_k_reranker` 3 (10), `subagents.max_*` 20/20/30/30000 (new: no limit); `user.permissions` lacks the new `subagents`, `side_chat`, `video_generation` keys; `direct.*` left over from the removed feature. **Waiting on Daniel** for the RAG and sub-agent values; image generation not tested (needs image mode) | — |
| 10:03 | **#32 Open Terminal 0.14.7 → 0.17.20**, before: through its API (`/execute`): a command runs, Python works | ok; also: internet yes, router 192.168.1.1 no answer, **marvin's llama-swap (100.101.162.9:8080/health) 200** (pre-existing; a model-driven shell can reach the tailnet: noted for Daniel, not changed) | — |
| 10:05 | `pull` + `up -d open-terminal` | 0.17.20; same command and Python output; home folder kept; internet, router and marvin results the same; 401 without the key; no terminal errors in Open WebUI's log | — |
| 10:07 | **#35 squid digest → 42decc7**, before: from the browser-internal network through the proxy | example.com 200, 192.168.1.1 403, marvin:8080 403; `squid-gnutls -k parse` exit 0 | — |
| 10:07 | `pull` + `up -d browser-proxy` | same 4 results; browser tool (Playwright through MCPJungle) loads example.com ("Example Domain") | — |
| 10:08 | **#37 mcp-searxng 2.9.0 → 2.11.0**, before: `search` tool, "llama.cpp v1.0.0 release", 3 results | huggingface blog, GitHub releases, eesel | — |
| 10:08 | `pull` + `up -d mcp-searxng` | same 3 results; `lab mcp status`: search 4/4 tools, matches the repo | — |
| 10:09 | **#39 ntfy 2.37.1 → 2.38.1** (security release), before: the 10 access checks | 10/10 | — |
| 10:09 | `pull` + `up -d ntfy` | log "ntfy 2.38.1"; access checks 10/10; test notification "ntfy 2.38.1 test" sent: **arrived** on phone and desktop (Daniel, 10:12) | — |
| 10:09 | **#42 python 3.12.16-slim digest → 5e2bdb1**, before: `memory` search (2 results, IDs 3246, 778), `recall` (REBUILD rule f539fa74) | as listed | — |
| 10:10 | `build` + `up -d mcp-localrecall mnemosyne`; `pull memory-sync` | both Python 3.12.16 on new images; same memory results and recall answer; memory-sync takes it at its next hourly run | — |
| 10:10 | **#43 golang digest → e519d3b**, before: `lab manifest` 0 unknown | ok | — |
| 10:10 | `cli/build.sh install` | tests ok; `lab manifest` 0 unknown; `lab setup check` all done | — |
| 10:12 | **#38 ComfyUI 0.43.0 → 0.45.0**: the PR changed only the Dockerfile; compose's image tag still said v0.43.0 (same slip as mcp-proxy, 2026-11-02): fixed by hand. Build started | — | this commit |
| 10:13 | Daniel: take Open WebUI 0.13's new defaults. Written into hub/compose.yaml (`CHUNK_SIZE` 1024, `RAG_TOP_K` 10, `RAG_TOP_K_RERANKER` 10, `SUBAGENTS_MAX_*` empty = no limit) and the 7 keys added to hub/openwebui/managed-keys.txt | — | this commit |
| 10:14 | `lab webui drift --sync`, then `up -d open-webui` | 7 managed settings synced (DB copy saved first); healthy. drift now **5 differ, all "web page only"**: `direct.*` (left over from the removed feature), `images.connections` + `image_generation.default_model` (same content, random id: drift needs to ignore the id), `user.permissions` (lacks 0.13's new `subagents`, `side_chat`, `video_generation` keys). The new chunk size applies to documents processed from now on; existing knowledge is not re-chunked | — |
| 10:24 | ComfyUI 0.45.0 image built (30.7 GB; the 0.43.0 image kept for rollback, disk decision in #50) | built | — |
| 10:31 | zaphod idle (every loaded model's slots not processing); `lab image-mode on` | ComfyUI ready; `system_stats`: 0.45.0 on the R9700 | — |
| 10:32 | One image through ComfyUI's API with **Open WebUI's migrated image connection** (its workflow and node mapping from the DB), 1024x1024, 8 steps | first 28.1 s (model load), second **5.0 s** (FEATURES: Z-Image Turbo 5 s on 0.43); image correct (a red apple). Not tested: generating from inside Open WebUI's chat (needs a login); Daniel to try one when convenient | — |
| 10:33 | `lab image-mode off` | profile off, ComfyUI stopped, card 2 3.8 GB used | — |
| 10:34 | **All 8 deployed.** Gaps found on the way: Renovate changes ComfyUI's Dockerfile but not compose's image tag (third time for this pattern); VERSIONS.md was not updated with #27 | | this commit |
