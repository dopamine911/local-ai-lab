# Engine launcher (option B): timeline

Decision 5.1 in plans/engines-and-frontend.md. Times are Chicago time, from `date` when written.

## Design
- `engines/lab-engined.py`: a small host service (systemd `lab-engined`, user daniel, group docker). Listens on a Unix
  socket in `/run/lab-engined/` (directory bind-mounted into llama-swap, kept across restarts). Starts only engines
  named in the machine's allowlist (`engines/<machine>.json`: image by digest, arguments, devices, read-only
  mounts). The caller sends only a name.
- Engines run as containers named `lab-engine-<name>`, labelled `lab.engine`, on the Docker network `lab-engines`
  (created `--internal`: no internet, no host ports). llama-swap joins that network and proxies to the container name.
- `engines/lab-engine`: llama-swap's side. `cmd: lab-engine run <name>` holds a streaming request (the engine's log);
  if that connection drops for any reason, the launcher stops the container. `cmdStop: lab-engine stop <name>`
  returns only when the container is gone.
- The launcher removes leftover `lab.engine` containers when it starts, and stops its engines when it stops.

## Steps
1. Standalone on marvin (foreground, as daniel), a no-GPU test container, driven with `curl` from the host. llama-swap
   untouched.
2. Failure tests, each ending with no engine container: client killed (`kill -9`), launcher restarted, unknown name
   refused, double start refused.
3. Wire into marvin's llama-swap (network + socket mount, one recreate). Idle check first; `lab speed marvin/big`
   before and after (the daily model's compose changes).
4. Through llama-swap: config reload, `docker compose restart llama-swap`, swap engine → `big` → engine.
5. GPU parity: `llama-server` from the `llama-swap-marvin:273` image as an engine, with a model marvin already serves
   natively (`chat-teacher`, 58.6 calls/min on 273). Good result: within ±3%.
6. systemd unit, host.sh step, `lab setup check` line, REBUILD.md, marvin/SETUP.md, FEATURES.md (Daniel installs the
   unit: sudo).
No engine image (Halogen, Gufo, DS4) is pulled in this work.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 10:39 | Step 1: launcher by hand on marvin (socket in ~/.local/state/lab-engined), allowlist `test-http` (no-GPU Python HTTP server, same pinned image as marvin's stats containers, nobody user, read-only) | network `lab-engines` created, internal=true; status `[]` | — |
| 10:39 | Start `test-http` through the socket; reach it; internet from that network; its log; normal stop | reachable by name ("ok"); **no internet** (name lookup fails); the caller sees the engine's log; stop answers "stopped", the caller ends, 0 containers | — |
| 10:40 | Step 2: client `kill -9` while running | container gone within 6 s | — |
| 10:40 | Unknown name; a path trick (`..%2Fetc`) | 404 "not in the allowlist"; 400 "bad path" | — |
| 10:40 | Double start | second gets 409 "already running", first keeps running; after it: 0 | — |
| 10:40 | Launcher SIGTERM while an engine runs | "stopping: engines down", container removed, 0 left (my `pgrep -f` also matched the ssh shell and cut the session; the launcher log shows the result) | — |
| 10:40 | Launcher `kill -9` while an engine runs, then started again | orphan stays while the launcher is down (the caller's connection drops, so llama-swap would see the model stop); the restart removes it: 0. Known gap: lasts until the launcher restarts (systemd `Restart=always`: seconds) | — |
| 10:42 | Step 3: marvin idle (`big` slots not processing). Before: `lab speed marvin/big` | 42.6 writing / 330.5 reading 32K (MLflow ab87d5ac); writing matches this morning's 42.4, below yesterday's 45.4 | — |
| 10:47 | Launcher restarted on the committed code (still by hand, same socket path as the service will use). marvin compose: socket folder mount, `engines/lab-engine` mount, networks `[default, lab-engines]` (external); `test-http` entry (cmd/cmdStop, checkEndpoint `/`, unloadTimeout 90; training pin). Files copied, sha256 = repo. `docker compose up -d llama-swap` | llama-swap on `lab-engines` + `llama-swap_default`; `/engines/lab-engine status` from inside → `[]` | 228df90 |
| 10:48 | After: `lab speed marvin/big` | **45.6 writing / 328.2 reading** (MLflow 81cb483f): reading −0.7%; writing +7% vs the before run but the same as yesterday's 45.4 (a freshly loaded `big` may write faster than one up for hours: unverified) | — |
| 10:53 | Step 4a: request `/upstream/test-http/` through marvin's llama-swap | "ok" in 1.0 s; `big` unloaded, test-http ready, 1 engine container | — |
| 10:53 | 4b: request `big` | answered; 28 s incl. load; engines 0 (cmdStop ran) | — |
| 10:54 | 4c: config reload (a comment line added, then removed) while test-http runs | engines 0 after the first reload; llama-swap preloads `big` | — |
| 10:55 | 4d: `docker compose restart llama-swap` while test-http runs | engines 0; launcher log: stop → container gone (llama-swap runs cmdStop on shutdown) | — |
| 10:56 | Step 5: `chat-teacher-engine` in the allowlist: `llama-server` from marvin's own image (`llama-swap-marvin@sha256:7b50eb02…`), `/dev/dri`, render+video groups, `/srv/models/gemma` read-only, exactly chat-teacher's flags. Entry in config (training pin). Smoke: one request | answered "ready", 24 s incl. load; 4 slots of 4K | — |
| 10:56-11:27 | Step 5 parity: `prompt_check.py --schema L1 --workers 4` (464 calls) from zaphod to marvin, native `chat-teacher` twice, then `chat-teacher-engine` twice; times from marvin's activity.sqlite | native: 60.0 (cold) / **61.7 calls/min (warm)**; engine: 57.5 (cold, incl. the swap) / **61.3 (warm), −0.7%**: within ±3%. 0 failures, 0 BAD in all four (yesterday's native on 273: 58.6). **A GPU engine started by the launcher runs as fast as the same llama-server run natively** | — |
| 11:29 | Parity entry `chat-teacher-engine` removed from allowlist and config (a duplicate of chat-teacher; its spec is in the row above). `test-http` stays as the launcher's test entry | | this commit |
