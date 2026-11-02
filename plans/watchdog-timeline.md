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
| next | Live test (issue's good result): marvin power mode -> quiet via the fan module, one alert within ~2 min; back to performance, one "Resolved" | — | — |
