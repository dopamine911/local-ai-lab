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
| next | Merge one PR at a time, deploy, run its check; record each | — | — |
