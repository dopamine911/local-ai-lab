# lab CLI (issue #6): timeline

What was done and when. Times are Chicago time (CST). Design and order: issue #6. Decision record: decisions.md 39.
Branch `cli-lab-skeleton` in its own worktree (`~/homelab-cli`), so benchmark runs on main are not affected.

## 2026-11-02

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 14:20 | Worktree `~/homelab-cli`, branch `cli-lab-skeleton` from main ea79308 | main untouched (marvin 269 before-runs reading it) | — |
| 14:22 | Versions (pin latest): Go 1.27.3 (image by digest), cobra v1.10.2, bubbletea v2.1.1 (`charm.land/bubbletea/v2`), lipgloss v2.0.6, go.yaml.in/yaml/v3 v3.0.6 | Bubble Tea v2's API read from its v2.1.1 tutorial (KeyPressMsg, View returns tea.View) | go.mod, go.sum |
| 14:30 | Skeleton: `lab` (menu on a terminal, help otherwise), `lab version`, `lab manifest [--json] [--host]`, `cli/lab.yaml`, `cli/build.sh` (vet, test, build in the container) | first build ok; parser tests pass (fixtures recorded from both machines) | this commit |
| 14:35 | **Parity**: `diff <(tuning/manifest.sh) <(lab manifest)` on zaphod; same on marvin (binary in /tmp, removed after) | **identical on both** (12 lines each) | — |
| 14:35 | No terminal: `lab </dev/null` | prints help, exit 0 (no hang) | — |
| 14:36 | Found by the parity test: the official `unified-vulkan-N` llama-swap images are nightly builds; tag N carried llama-swap program N-1 (zaphod "269" = v268, marvin "267" = v266) | records on main corrected (d16060f), issue #1 comment | — |
| next | Daniel: try the menu (`cli/bin/lab` on a terminal); then step 2 (read-only commands: `mcp status`, `versions audit`, `webui drift`, `enso check`) | — | — |
