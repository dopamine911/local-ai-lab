# Watchdog (issue #12): timeline

What was done and when. Times are Chicago time (CST). Design: issue #12. Code: `hub/watchdog/watchdog.py`
(the CHECKS table holds every threshold, good value and first command). Buttons: `hub/actions/actiond.py`.

## 2026-11-02

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 11:14 | Decision (Daniel: yes): own small checker, not Uptime Kuma | Kuma can't express the custom rules (busy and under 100 W; temps; backup age) and can't attach our signed buttons; it would add a second web app and database | issue #12 |
| 11:15 | Checked first: the service can ssh to marvin (zaphod key has no passphrase, marvin in known_hosts); llama-swap `/health` needs no key (200 on both); sensors: zaphod cards have edge / junction / mem, marvin edge only | zaphod alerts on **junction > 100 C** (edge > 90 would never fire, junction > 90 is normal under load; the card slows itself near 110); marvin **edge > 90 C** | — |
| 11:18 | `watchdog.py` batch 1: 8 checks; ok / bad / unknown (marvin unreachable -> only the reachability alert); hold times before alert (2-5 min) and before resolved (1-2 min); reminder once after 4 h unless acked; snooze 1 h; cap 12 messages/hour | `watchdog.py test` with made-up readings and a fake clock: **17/17 pass** (no alert under the hold time, one alert, no repeat, flap, one resolved, unreachable, snooze silent then alerts again, reminder once, ack stops it, cap 12). `watchdog.py once`: all 8 ok | this commit |
| 11:20 | Buttons in actiond: `wd_ack`, `wd_snooze`, `wd_diag` (a button names a check, never a command) | on a test copy with scratch state: ack 200 (and "already ended" for an old incident), snooze 200 (60 min, alert reset), Run check 200 (marvin `curve.sh check` output came back); unknown check refused when making the button | — |
| 11:22 | `lab-watchdog.service`, `setup/host.sh` step 6d, REBUILD.md "Phone notifications" step 8, FEATURES | — | 4934245 + this commit |
| 11:23 | Daniel: restarted lab-actions (11:23:13), installed lab-watchdog (started 11:23:37); pushed | both active; from the 2nd loop (11:24:38) all 8 checks ok: marvin answers, performance, 0% / 9 W, 45 C; zaphod junctions 32 / 32 C; /health 200 on both | `watchdog.py status` |
| 11:25 | Two small bugs: the first loop's readings weren't stored (value saved before its row existed; alerts unaffected); `watchdog.py test` wrote its practice "rate limit" lines into the service log | fixed; test 17/17; loads at the next `sudo systemctl restart lab-watchdog` | this commit |
| 11:25 | Live test "before" (Daniel: `curve.sh check`) | performance, fan curve active, idle 45 C | — |
| 11:26 | Daniel: `echo quiet \| sudo tee /sys/class/ec_su_axb35/apu/power_mode` on marvin; I ran `tuning/speed marvin/big` (NOLOG) as load | marvin under load in quiet: **100% busy, 54 W, 2,085 MHz** = the "55 W problem" signature (CLAUDE.md Known issues) | — |
| 11:28:41 | Watchdog: **two alerts** (~2.5 min after the switch): "power mode not performance" and "GPU busy but under 100 W" | as designed; no repeats afterwards | watchdog.log |
| 11:30:39 | Daniel: **Run check** from the phone | 200, 814 bytes of `ec-module.sh check` + busy/watts came back | actions.log |
| 11:33 | Quiet-mode speed (one run): writing 38.8 tok/s, reading 32K 251.5 | vs performance (fan test B) 44.7 / ~313: **-13% writing, -20% reading** | — |
| 11:33:58 | Daniel: **Ack** from the phone (found in the app's lab-alert list; the shade had dropped both alerts after Run check, maybe grouped) | 200; acked_at set on marvin-low-power | actions.log |
| ~11:33 | Daniel: back to `performance` | — | — |
| 11:35:44 | Watchdog: **two "Resolved"** | the whole test sent exactly 4 messages (2 alerts, 2 resolved) | watchdog.log |
| 11:37 | Full-power check: `tuning/speed marvin/big` (NOLOG) | 132-133 W, ~2,880 MHz, performance, fan curve active (~4,140 rpm); writing 44.4, reading 320.0; no watchdog message | — |
| — | **Issue #12's good result met** (with the module's power-mode switch instead of a BIOS visit) | — | — |
| 11:45 | **Batch 2** (8 more checks): disk < 10% free and containers restarting/unhealthy on both machines; MCPJungle servers (hub/mcp/servers/*.json) without tools; nightly backup > 26 h (last `done:` line); memory sync last line not a success or log > 2.5 h old; certificate (`*.ai` wildcard, read over TLS) < 14 days | `watchdog.py test` **22/22** (5 new cases); `once`: all 16 ok (zaphod 28% free, marvin 92%; no bad containers; 4 MCP servers with tools; backup 8 h; sync 0.1 h; certificate 85 days); every new Run-check command ran | this commit |
| 11:52 | Daniel: `sudo systemctl restart lab-watchdog` | "started: 16 checks"; all 16 stored on the **first** loop (fix works), all ok; no alert sent | `watchdog.py status` |
| 11:53 | marvin checkout: hand-copied `curve.sh` restored, `git pull` (d77cd21) | clean; fan curve service active | — |
| 15:20 | Found by Renovate #25: mcp-proxy 0.13.3 broke the `memory` tool ("Invalid Host header") and no check noticed: `mcp-tools` only sees that servers are registered | fixed (plans/renovate-timeline.md) | — |
| 15:33 | Agent login `watchdog` (hub/mcp/clients/watchdog.json: memory + recall only; token `MCP_TOKEN_WATCHDOG` in hub/.env, Proton Pass) via `lab mcp setup` | 4 logins, `lab mcp status` matches | — |
| 15:25 | Check 17 `mcp-calls`: one real `memory` search and one `recall` per minute through MCPJungle (MCP initialize + tools/call, the agents' path); bad = an error or 0 results for 3 min | test **25/25** (3 new cases); live: memory 1 result, recall 1 result | this commit |
| 15:26 | Daniel restarted lab-watchdog | "started: 17 checks"; first `mcp-calls`: memory 1 result, recall 1 result; 0 problems | watchdog.log |
| next | Later checks (issue #12 list): `marvin/big` through the zaphod peer; reservations (#3) | — | — |
