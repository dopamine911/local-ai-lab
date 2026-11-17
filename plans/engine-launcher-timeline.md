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
