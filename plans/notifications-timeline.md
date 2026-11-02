# Notifications (issue #11): timeline

What was done and when. Times are Chicago time (CDT). Sources: git commits, `docker inspect`, file times. Design and
the next phases are in issue #11 (actions) and #12 (watchdog).

## 2026-10-30

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 18:47 | Read ntfy v2.35.0 docs and source | latest = v2.35.0; users/access/tokens can be declared; `upstream-base-url` default is empty (source, cmd/serve.go), so nothing goes to ntfy.sh unless set | — |
| 18:47 | Daniel: Android phone, Tailscale on | no upstream relay needed; set explicitly empty | — |
| 18:47 | Phone password, sender token, bcrypt hashes generated into hub/.env (mode 600) | a masked check printed the two hashes by mistake; all regenerated, none of the printed values in use | hub/.env (not in git) |
| 18:48 | ntfy started (hub/compose.yaml `ntfy`, 127.0.0.1:2586) | 8 access checks pass: anonymous 403, sender write-only on lab-*, daniel read-only, web app off (404) | this commit |
| 18:48 | Caddy route `notify.<DOMAIN>` | over Tailscale HTTPS: daniel 200, anonymous 403 | this commit |
| 18:49 | `hub/notify` sender | lab-done and lab-alert sent; unknown topic refused; token not in the process list | this commit |
| 18:50 | `tuning/log` notifies after every benchmark it logs (lab-alert on failure; `NOTIFY=0` off) | tested on a sample of the 17/19 run's output: title, result line, MLflow link (`/#/runs/<id>`, route checked in MLflow 3.17.2) | this commit |
| 18:53 | Checked the other branches: a failed run sends lab-alert with the error lines; `hub/notify` reports "failed (HTTP 000)" when ntfy is down (was exiting silently) | pass | next commit |
| ~18:58 | Daniel: Tailscale on the Android phone won't log in (button does nothing); generating an auth key also fails | cause: Tailscale incident "Coordination service issues" (status.tailscale.com); tailnet zenner.dev, Tailnet Lock off, zaphod ↔ marvin still fine (7 ms) | — |
| next | Daniel: phone joins the tailnet after the incident, then the app (REBUILD.md "Phone notifications") | — | — |

## 2026-10-31

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 20:22 | Daniel: Tailscale works on the phone | phone `pixel-11-pro` (100.64.122.73) on the tailnet; `tailscale ping` from zaphod 84 ms; ntfy up 26 h; anonymous HTTPS still 403 | — |

## 2026-11-01 (CST from today: daylight saving ended)

| Time | What | Result | Record |
| --- | --- | --- | --- |
| ~18:00 | Daniel: ntfy app installed, login added, 3 topics subscribed | first test (2026-10-31 20:30 CDT, re-sent today) not delivered: ntfy showed 0 subscribers | — |
| ~18:20 | Checked the path: phone browser opens `https://notify.<DOMAIN>/v1/health` (healthy); zaphod saw no stream from the phone | network fine; cause: topics were added without a server, so they went to the public ntfy.sh | — |
| ~18:40 | Daniel re-added the topics on our server; deleted the ntfy.sh ones | ntfy `subscribers=3`; the app's own test send is refused for `daniel` (read-only, as designed) | — |
| 18:41 | `hub/notify` to lab-done, lab-alert (high), lab-research | **all three arrived on the phone** | this commit |
| — | REBUILD.md "Phone notifications": set the app's default server first; check `subscribers=3`; the login fallback (`NTFY_ENABLE_LOGIN`) was not needed and is removed | — | this commit |

### Phase 2: buttons (issue #11)

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 18:43 | Design: a host service on zaphod (actions run the same scripts Daniel runs: docker, ssh, git), not a container; Python stdlib only; the button carries only a signed one-time token, the action is stored on zaphod; Rerun only for zaphod runs until reservations (#3) exist | advisor review agreed | issue #11 |
| 18:47 | `hub/actions/actiond.py` (serve / button / log), `ACTION_SECRET` in hub/.env, `hub/notify -a` (buttons go through the private file descriptor, not the process list) | 11 local checks pass: valid 200, again 403, bad signature 403, another button's signature 403, unknown 404, expired 403, GET 405, wrong path 404, 5 KB body 400, listens 127.0.0.1 only, LAN unreachable | this commit |
| 18:48 | Caddy route `act.<DOMAIN>` -> 127.0.0.1:2587, reloaded | https health ok; notify and chat pages still 200 | this commit |
| 18:53 | Phone test: notification with **Ack** + **Open MLflow** | Daniel tapped Ack; log: `100.64.122.73` (pixel-11-pro) ack 200; "Acknowledged" sent | `~/.local/state/lab-actions/actions.log` |
| 18:56 | Daniel installed the system service `lab-actions` (setup/host.sh step 6c) | active, runs as daniel; over HTTPS: valid 200, again 403, bad signature 403 | this commit |
| next | Step 2: **Rerun** and **Skip** on benchmark notifications (`tuning/log`); zaphod runs only | — | — |
