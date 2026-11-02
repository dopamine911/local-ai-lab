# lab CLI (issue #6): timeline

What was done and when. Times are Chicago time (CST); "~" = estimated afterwards from commit times (corrected 14:30). Design and order: issue #6. Decision record: decisions.md 39.
Branch `cli-lab-skeleton` in its own worktree (`~/homelab-cli`), so benchmark runs on main are not affected.

## 2026-11-02

| Time | What | Result | Record |
| --- | --- | --- | --- |
| ~14:08 | Worktree `~/homelab-cli`, branch `cli-lab-skeleton` from main ea79308 | main untouched (marvin 269 before-runs reading it) | — |
| ~14:09 | Versions (pin latest): Go 1.27.3 (image by digest), cobra v1.10.2, bubbletea v2.1.1 (`charm.land/bubbletea/v2`), lipgloss v2.0.6, go.yaml.in/yaml/v3 v3.0.6 | Bubble Tea v2's API read from its v2.1.1 tutorial (KeyPressMsg, View returns tea.View) | go.mod, go.sum |
| ~14:15 | Skeleton: `lab` (menu on a terminal, help otherwise), `lab version`, `lab manifest [--json] [--host]`, `cli/lab.yaml`, `cli/build.sh` (vet, test, build in the container) | first build ok; parser tests pass (fixtures recorded from both machines) | this commit |
| ~14:17 | **Parity**: `diff <(tuning/manifest.sh) <(lab manifest)` on zaphod; same on marvin (binary in /tmp, removed after) | **identical on both** (12 lines each) | — |
| ~14:17 | No terminal: `lab </dev/null` | prints help, exit 0 (no hang) | — |
| ~14:18 | Found by the parity test: the official `unified-vulkan-N` llama-swap images are nightly builds; tag N carried llama-swap program N-1 (zaphod "269" = v268, marvin "267" = v266) | records on main corrected (d16060f), issue #1 comment | — |
| 14:19-14:26 | Skeleton committed 14:19 (ef31c2b); PR #17 opened (main pushed first so it holds only the CLI commit); Daniel merged it | `lab` installed on zaphod (`cli/build.sh install`): manifest still identical to the script | #17 |
| 14:27 | Step 2 branch `cli-step2-readonly` (worktree `~/homelab-cli`) | — | — |
| 14:29 | `lab mcp status` (new; the hub/mcp/status.sh to-do in plans/agent-infra.md): servers with enabled/total tools, agents and their servers, drift against hub/mcp/clients and servers/*.tools; exit 1 on drift; `--tools`, `--json` | live: browser 25/25, memory 2/2, recall 5/30, search 4/4; 3 agents; "Matches the repo". Tests: parsers on recorded output, drift found when a tool or an extra agent differs | this commit |
| 14:29 | `lab enso check` (port of `agents/enso/enso.sh check`) | **identical** to the script (3 lines: models, no internet, 36 tools) | this commit |
| 14:31 | `lab webui drift`: Go runs the throwaway Open WebUI and the steps; the comparison is drift.sh's Python, unchanged, in the pinned Python image (its JSON formatting is the output to match). Live database mounted read-only (the script mounts it writable for --sync) | **identical** to drift.sh ("0 setting(s) differ", ~30 s); no leftover volumes. Limit: no difference existed, so the difference-printing path ran only as the script's own unchanged Python | this commit |
| 14:31 | Found: drift.sh ran `python:3.13-slim-trixie` unpinned (missed by the 10-30 audit) | pinned to `python:3.13.18-slim-trixie@sha256:76baad…` (the one compose uses), in the script and the port | this commit |
| 14:34 | Daniel: no `lab versions audit`; issue #7 (Renovate) instead: it compares every pin with upstream and opens a PR per update | step 2 = mcp status, enso check, webui drift | issue #6 comment |
| 14:39 | PR #18 merged; `lab` reinstalled on zaphod | — | #18 |
| 15:00 | Step 3 branch `cli-step3-hub` (worktree `~/homelab-cli`); commands that change things get `--dry-run` | — | — |
| 15:02 | `lab memory sync [--install] [--dry-run]` (port of hub/memory/sync.sh) and `lab image-mode on/off/status [--dry-run]` (port of hub/image-mode); lab.yaml: zaphod's llama-swap key file and card 2's PCI address (defaults if missing) | `image-mode status`: **identical**; `memory sync` run right after the script: **identical** apart from the timestamp line (0 uploaded, 48 unchanged); dry runs list the steps; crontab untouched. First try failed: main's lab.yaml lacked the new keys (now defaults) | this commit |
| pending | Real `image-mode on` / `off` comparison: interrupts card 2's models for a few minutes (Daniel picks the time) | — | — |
| 15:04 | `lab webui models [--apply]` (full Go port of hub/openwebui/models.sh: model order from the configs, Python's formatting kept) | live: **identical** ("0 model(s) differ"). Parity test: the script's own Python and the Go port on the same made-up configs and a fake Open WebUI give the same output, read-only and --apply (run with python3-yaml added; in normal builds it skips itself: the Go image has no PyYAML) | this commit |
| next | `lab webui drift --sync`, `lab mcp setup`, `lab enso build/setup/up/tui/shell/down` | — | — |
