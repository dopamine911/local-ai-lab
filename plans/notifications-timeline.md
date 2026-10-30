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
| next | Daniel: phone app set up (REBUILD.md "Phone notifications") | — | — |
