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
| next | `lab versions audit` (new: compare setup/VERSIONS.md with upstream); `--sync` waits for step 3 | — | — |
