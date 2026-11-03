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
| 21:04 | Enso after reboot | left stopped (Daniel: not needed now). Start with `lab enso up` when needed; its compose has `restart: "no"`, so it never starts by itself after a boot | Daniel |
| 21:05 | issue #10 | closed (Daniel); comment: flags confirmed after the reboot on every running model, removal tracked in #34 | GitHub #10 |
| 21:08 | issue #11 | kept open (Daniel) until #3 (reservations) and #5 (research) add their buttons. Checked: Terminal-Bench smoke notification arrived 20:48:06 ("Done: north", MLflow link); watchdog buttons used today | ntfy cache.db, actions.log |
| 07:30 | north full run, mid-run check | 10/19 done after 10.7 h: 4 passed (nginx-request-logging, configure-git-webserver, regex-log, headless-terminal), 6 failed on the first attempt (fix-ocaml-gc agent timeout at 183 min; mailman 155 min; extract-elf; sparql-university; pypi-server and openssl-selfsigned-cert quick: agent said done, tests disagreed; openssl = test needs `cryptography`, same miss as Strata 10-30). Running: break-filter-js-from-html (2 h). Failed tasks get a second attempt at the end | ~/tbench-north-2026-11-02.log, job folder |
| 07:33 | "everything just started to fail"? (Daniel) | No time pattern: in run order pass, fail, pass, fail (timeout), fail, fail, pass, fail, fail, pass (passes at 04:22 and 05:13 too). north process not restarted (up since 20:42). Server errors in 10.7 h: 3x HTTP 500 "model produced output that does not match the expected peg-native format" (21:36, 21:40 in headless-terminal, which passed; 22:48 in fix-ocaml-gc) and 1x context full (262,166 > 262,144 tokens, 06:54, break-filter-js-from-html). The other failed tasks had no server errors: the model's own misses | docker logs llama-swap, job result.json |
| 07:36 | Is the north 500 the Qwen grammar bug, or caused by our changes? | **Different bug.** Qwen: 400 before writing (grammar built empty). north: 500 after writing: north emitted a Cohere tool call `<|START_ACTION|>[{"tool_call_id": "1", "tool_name": "function", "name": "grep", ...}]<|END_ACTION|>` (malformed: "tool_name" should be the tool, and grep is not a tool the agent offers) and llama.cpp's parser rejected it. Agent's next request 0.5 s later succeeded, so cost ~1 retry each; 3 of 2,372 requests. Not from #10 (cache flags change caching only) and the grammar bug got no server change. Unknown: whether b11870 (267) handled it differently; north tool-eval 10-28 on the old build had 0 errors, but those runs are short. Test after the run | docker logs llama-swap 03:36:50Z |
| 09:29 | north full run check | 12/19 done after 12.7 h, 6 passed: break-filter-js-from-html passed (146 min, despite the 06:54 context-full error), git-leak-recovery passed (4 min). Running: overfull-hbox (1 h 28 min). 1 more stray tool-call 500 at 07:57 (4 total). Background watcher from the earlier session had stopped; run itself unaffected (tmux) | ~/tbench-north-2026-11-02.log |
| 09:30 | old-vs-new build test for the north 500 | not wanted (Daniel) | |
| 09:30 | fail-then-pass pattern? (Daniel: interesting) | Compared north's 12 first attempts with 15 other full runs (ours + published). north fails tasks nearly every other model passes first try: mailman, pypi-server (15/15 others), openssl (14/15), fix-ocaml-gc, sparql-university (13/15). extract-elf is hard for all (6 of 15 fail first try). The late passes (regex-log, break-filter, git-leak-recovery) are tasks almost all models pass. So the streaks look like run order + task luck, not something changing; north is weaker on some easy tasks. Second attempts still to come | results/suites/*/results-*.json |
| 12:59 | north full run check | 17/19 first attempts done after 16.2 h: 8 passed. Since 09:29: overfull-hbox fail (agent timeout, 182 min), sqlite-with-gcov fail, llm-inference-batching-scheduler fail, cobol-modernization pass, fix-git pass. Running: build-cython-ext. **Stray tool-call 500s jumped**: 120 since 09:29 (4 before), 110 of them since 12:30 in build-cython-ext (110 errors vs 256 ok requests). north now wraps the agent's normal JSON commands (`{"keystrokes": ...}`) in Cohere's `<|START_ACTION|>` tokens; llama.cpp refuses them; the agent retries. Still making progress (running the task's tests). Not intervening mid-run | docker logs llama-swap |
| 14:27 | omp on a remote machine (Daniel) | agents/omp/models.yml + README: provider `lab` → zaphod:8080 over Tailscale, key from env ZAPHOD_KEY. Tested with omp 18.15.0 (separate test folder): `lab/marvin/big` answered "ready"; without the key 401 | this commit |
