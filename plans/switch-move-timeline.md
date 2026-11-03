# Timeline: moving zaphod and marvin to the network switch (2026-11-02)

Issue #14 item 2. Checks from HANDOFF.md section 1. Times are Chicago, from `date` / logs.
Network details (switch ports, addresses) go in the private repo ~/full-homelab, not here.

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 19:34 | marvin shut down | | HANDOFF.md |
| 19:44 | zaphod shut down | | journal (previous boot) |
| 19:54 | zaphod booted | boots; CSM fine | `uptime` |
| 20:36 | 1-2. boot + Tailscale (zaphod side) | zaphod up. zaphod's cable port has **no link** (carrier 0), zaphod still on Wi-Fi as before the move. marvin offline (it had not been turned on yet) | `ip -br addr`, `tailscale status` |
| 20:37 | 3. `lab setup check` zaphod | all ok except "lab CLI installed 4f443ad, repo at c69451e": no `cli/` change in those 8 commits, so only the check is strict | |
| 20:37 | 4. llama-swap zaphod | up; coder, coder-2, embed, rerank, stt, stt-whisper, tts loaded | `docker compose ps`, `/running` |
| 20:37 | 5. hub | all 19 containers up | `docker compose ps` |
| 20:38 | 5. `lab enso check` | FAIL: Enso containers stopped. compose has `restart: "no"`, so Enso needs `lab enso up` after every boot (handoff missed this) | |
| 20:38 | 5. watchdog once | bad: marvin (off), mcp-calls (memory search 0 results). Rest ok; backup last success 16 h ago | |
| 20:39 | memory search cause | LocalRecall's database runs from a temporary folder in the container, so **its search index is lost on every restart** (since setup 10-30). The image's start script falls back to /tmp when `hub/data/localrecall-db` is not writable; the folder is owned by daniel, the database runs as uid 999. The 75 documents themselves are kept (`hub/data/localrecall`); only the index is empty. Not fixed yet: Daniel decides | `docker logs localrecall-db`, `show data_directory` |
| 20:39 | marvin powered on (Daniel) | Tailscale back via its old Wi-Fi address; still booting | `tailscale status` |
| 20:39 | marvin accepts ssh | up 2 min | |
| 20:40 | 1. marvin power mode | performance (good) | `/sys/class/ec_su_axb35/apu/power_mode` |
| 20:40 | 7. marvin link | **no link on the cable port either** (eno1 carrier 0); on Wi-Fi as before. Both machines without link: most likely the switch (power) or the cables, not the computers. Stopped here: needs Daniel at the switch. No "after" measurement yet | `ip -br addr`, `/sys/class/net/eno1/carrier` |
| 20:40 | 3. `lab setup check` marvin | all ok except the same strict "lab CLI" line (marvin's repo copy at d13ca9b; no `cli/` change) | |
| 20:40 | 4. llama-swap marvin | up; `big` ready | `/running` |
| 20:40 | 5. watchdog once | all ok except mcp-calls (memory search, see 20:39). marvin GPU read 0 C at 20:40:21, 33 C at 20:40:27 (sensor just after boot) | |
| 20:40 | 6. backup | not needed: last success 16 h ago (04:17 today, machines were on) | watchdog `backup` |
| 20:42 | 7. cause of no link (Daniel) | machines moved, network **not set up yet**: Daniel needs 2 cables. Expected, not a fault. Step 7 (cable + after measurement, full-homelab, CLAUDE.md, marvin/SETUP.md) waits for the cables | Daniel |

## Side task: Terminal-Bench on `north` (Daniel asked 20:40)

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 20:41 | `lab tbench doctor` with MODEL=north | passed (needs `CONTEXT_LENGTH=262144`; `CONTEXT` is ignored). Labels set by hand: with only MODEL=north the tool would log it as Qwen3.8-27B / mtp2-256k-kvq8 (coder's defaults) | settings: MODEL=north MODEL_NAME="North-Mini-Code-1.0 30B-A3B" QUANT=UD-Q4_K_XL PROFILE=256k-kvq8 CONTEXT_LENGTH=262144 |
| 20:41 | smoke started (1 task) | running; north on card 2, coder-2 unloaded | ~/tbench-north-smoke-2026-11-02.log |
| 20:48 | smoke finished | **1/1** (git-leak-recovery, first attempt), 0.11 h, 111K in / 15K out tokens, 0 errors | MLflow af013554; ~/tbench-north-smoke-2026-11-02.log |
| 20:48 | full run started (19 tasks, pass@2), Daniel: "it hasn't been run yet for this model" | running in tmux `tbench-north`; first task (nginx-request-logging) started 20:48 | ~/tbench-north-2026-11-02.log |

## Fix: memory search index lost on every restart (Daniel: "1", 20:58)

Before: `memory` search "REBUILD" 0 results; `documents_homelab` 0 rows; database in `/tmp/postgres-init-*`.

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 20:59 | Daniel: `sudo chown 999:999 ~/homelab/hub/data/localrecall-db` | folder owned by 999, empty | `ls -ldn` |
| 20:59 | restart localrecall-db, then localrecall | data folder now `/var/lib/postgresql/data` (the saved one) | `show data_directory` |
| 20:59 | re-index: copy `hub/data/memory-sync/sync.json` aside, set every hash to "reindex-2026-11-02", `lab memory sync` (each file deleted + uploaded) | 75 uploaded in 55 s; 1,120 rows; watchdog mcp-calls ok (memory 1 result) | |
| 21:00 | found a leftover entry: a tool-eval run transcript (prompt-injection test text; sync.py's SKIP_PATHS keeps these out). Uploaded before that rule, so the sync never removed it. Not in the new index (0 rows) | deleted through LocalRecall's entry/delete API; 75 entries = sync list | |
| 21:01 | **restart test**: restart both again | index kept (1,120 rows, no new-database setup); watchdog: all ok | |
| 21:02 | REBUILD.md: chown line before the hub starts and in "Restore" (the backup stores files as daniel) | | this commit |
