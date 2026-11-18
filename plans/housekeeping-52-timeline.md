# Issue #52 housekeeping: timeline

The audit's small items (plans/audit-2026-11-17.md item 10; GitHub #52), each on its own. Times are Chicago time, from `date`.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 2026-11-18 09:10 | Read #52: 1 wrappers, 2 lab on marvin, 3 result-file rule, 4 per-minute `embed` call, 5 memory-sync cron (`lab memory sync` or `sync.sh`) | — | — |
| 09:12 | Item 2: `lab version`: zaphod 9093b76, marvin bfbd763 (the issue said 4f443ad: reinstalled since). `cli/build.sh install` on marvin (disk 20% used), then on zaphod too, so both show the same commit | **both `lab 40b1a46`**, tests ok | — |
| 09:16 | Item 4: who makes the per-minute `embed` call? llama-swap's request log, `embed` per hour, last 30 h | ~60 an hour while the watchdog ran (11-17 03-17); **0** from 00:00 to 08:00 today with the watchdog stopped but `embed` loaded; ~1 a minute again since its restart (08:49). Nothing else calls `embed` once a minute: it is the watchdog's `mcp-calls` check (a real memory search). Memory sync at :41 makes no `embed` request through llama-swap | — |
| 09:18 | Item 1: who calls the 18 wrappers? Call log `~/.cache/lab-wrapper-calls.log`: empty since 2026-11-02 18:54. crontab and systemd units: none. Scripts in git: only comments ("port of ..."). The one live caller: actiond's Rerun ran `tuning/<name>` | — | — |
| 09:20 | Item 1a: actiond's Rerun → `~/.local/bin/lab <name> <args>` (full path: the service has no ~/.local/bin on PATH); same allowlist, same tuning/ and zaphod-only checks. Checked the builder offline: `speed coder "note x"` → `[~/.local/bin/lab, speed, coder, note x]`; `rm` refused. Live after `sudo systemctl restart lab-actions` (Daniel) | — | this commit |
| 09:25 | Daniel chose: item 3 commit only the result files a results.md row or timeline cites; item 4 the watchdog's memory check skips (unknown) while a run-alone model is loaded; item 5 switch the cron to lab | — | — |
| 09:26 | Item 5: `lab memory sync --install` still wrote a line that ran sync.sh. Changed lab: the line is `41 * * * * ~/.local/bin/lab memory sync >> ~/.cache/memory-sync.log 2>&1` (full path: cron has no ~/.local/bin); `--install` replaces the old sync.sh line or an earlier lab line, keeps the rest; new test `TestMemorySyncLine`. Crontab before saved in the session scratchpad | tests ok | — |
| 09:30 | The new line run by hand with cron's bare environment (HOME, PATH=/usr/bin:/bin), then installed | exit 0; log ends "sync: 1 uploaded, 0 removed, 94 unchanged" (what the watchdog reads); crontab: backup, prune, the lab line; watchdog `memory-sync` ok, 0.0 h. `hub/memory/sync.sh` removed; REBUILD.md, agent-infra.md, compose comment, watchdog text updated | this commit |
