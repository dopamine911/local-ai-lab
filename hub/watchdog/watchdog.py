#!/usr/bin/env python3
"""Lab watchdog (issue #12): small read-only checks every minute; one alert when a problem starts, one "resolved"
when it ends, through ntfy (hub/notify) with buttons from hub/actions/actiond.py.

  watchdog.py run [--interval 60]   the service (systemd unit lab-watchdog.service)
  watchdog.py once                  read everything now and print each check's verdict (no state, no notification)
  watchdog.py status                the state table (what is bad, since when, alerted, acked, snoozed)
  watchdog.py test                  the state machine with made-up readings and a fake clock; prints a pass table

Each check is ok / bad / unknown. Unknown (e.g. marvin unreachable, so its GPU can't be read) changes nothing: only
"marvin reachable" alerts then, not four checks for one Wi-Fi drop. A check must stay bad for its `for_s` before the
alert, and ok for `ok_after_s` before "resolved", so a model load or a reboot doesn't flap.

Buttons on an alert: Ack (no reminder: an unresolved alert reminds once after 4 h unless acked), Snooze 1 h (no alerts
for this check for an hour; if still bad after, it alerts again), Run check (the check's fixed, read-only first
command; its output comes back as a notification). Nothing from the phone chooses what runs.

State: ~/.local/state/lab-watchdog/state.db. Log: ~/.local/state/lab-watchdog/watchdog.log.
Good result of the service: `systemctl is-active lab-watchdog` and a "started" line in the log.
"""
import os, sqlite3, subprocess, sys, time

HUB = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
STATE = os.path.expanduser(os.environ.get("LAB_WATCHDOG_STATE", "~/.local/state/lab-watchdog"))
DB, LOG = os.path.join(STATE, "state.db"), os.path.join(STATE, "watchdog.log")
REMIND_S = 4 * 3600
MAX_PER_HOUR = 12  # all watchdog notifications together; more are logged, not sent
MARVIN_SSH = ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "marvin"]

# One read of marvin: key=value lines. Read-only (sysfs, curl to its own llama-swap).
MARVIN_READ = r'''
D=/sys/class/drm/card1/device; E=/sys/class/ec_su_axb35
echo busy=$(cat $D/gpu_busy_percent 2>/dev/null)
echo watts=$(( $(cat $D/hwmon/hwmon*/power1_average 2>/dev/null || echo 0) / 1000000 ))
echo edge=$(( $(cat $D/hwmon/hwmon*/temp1_input 2>/dev/null || echo 0) / 1000 ))
echo power_mode=$(cat $E/apu/power_mode 2>/dev/null)
echo health=$(curl -s -o /dev/null -m 5 -w '%{http_code}' http://127.0.0.1:8080/health)
'''


def read_all():
    """Everything the checks need, in one dict. Missing values are None (= unknown)."""
    r = {}
    try:
        p = subprocess.run(MARVIN_SSH + [MARVIN_READ], capture_output=True, text=True, timeout=20)
        r["marvin_reach"] = p.returncode == 0
        if p.returncode == 0:
            for line in p.stdout.splitlines():
                k, _, v = line.partition("=")
                r["marvin_" + k] = v.strip() or None
    except subprocess.TimeoutExpired:
        r["marvin_reach"] = False
    for name, pci in (("card1", "0000:06:00.0"), ("card2", "0000:0b:00.0")):  # CLAUDE.md: card 1 = 06:00.0
        try:
            h = f"/sys/bus/pci/devices/{pci}/hwmon"
            hw = os.path.join(h, os.listdir(h)[0])
            for t in os.listdir(hw):
                if t.endswith("_label") and open(os.path.join(hw, t)).read().strip() == "junction":
                    r[f"zaphod_{name}_junction"] = int(open(os.path.join(hw, t[:-6] + "_input")).read()) // 1000
        except (OSError, IndexError, ValueError):
            pass
    try:
        p = subprocess.run(["curl", "-s", "-o", "/dev/null", "-m", "5", "-w", "%{http_code}",
                            "http://127.0.0.1:8080/health"], capture_output=True, text=True, timeout=10)
        r["zaphod_health"] = p.stdout.strip()
    except subprocess.TimeoutExpired:
        r["zaphod_health"] = "timeout"
    return r


def num(r, k):
    try:
        return float(r[k])
    except (KeyError, TypeError, ValueError):
        return None


def marvin_known(r):
    return r.get("marvin_reach") is True


# --- the checks: one row each; alert text, button command and runbook come from the same row -----------------------
# verdict(r) -> ("ok" | "bad" | "unknown", value text)
def v_marvin_reach(r):
    return ("ok" if r.get("marvin_reach") else "bad"), ("answers" if r.get("marvin_reach") else "no answer over ssh")


def v_power_mode(r):
    if not marvin_known(r) or not r.get("marvin_power_mode"):
        return "unknown", "not readable (fan module not loaded?)"
    pm = r["marvin_power_mode"]
    return ("ok" if pm == "performance" else "bad"), f"power mode {pm}"


def v_low_power(r):
    busy, w = num(r, "marvin_busy"), num(r, "marvin_watts")
    if not marvin_known(r) or busy is None or w is None:
        return "unknown", "-"
    return ("bad" if busy > 80 and w < 100 else "ok"), f"busy {busy:.0f}%, {w:.0f} W"


def v_temp(key, limit, label):
    def v(r):
        t = num(r, key)
        if t is None or (key.startswith("marvin") and not marvin_known(r)):
            return "unknown", "-"
        return ("bad" if t > limit else "ok"), f"{label} {t:.0f} C"
    return v


def v_health(key, needs_marvin):
    def v(r):
        if needs_marvin and not marvin_known(r):
            return "unknown", "-"
        h = r.get(key)
        return ("ok" if h == "200" else "bad"), f"/health {h or 'no answer'}"
    return v


ZAPHOD_TEMPS = r'''for h in /sys/class/drm/card*/device/hwmon/hwmon*; do echo "$(basename $(readlink -f $h/../..)): $(for t in $h/temp*_label; do printf '%s=%s ' "$(cat $t)" "$(( $(cat ${t%_label}_input)/1000 ))"; done)busy=$(cat $h/../../gpu_busy_percent)% $(( $(cat $h/power1_average 2>/dev/null || echo 0)/1000000 )) W"; done'''

CHECKS = {
    "marvin-reach": dict(title="marvin unreachable", verdict=v_marvin_reach, for_s=180, ok_after_s=60,
                         good="answers over ssh (Tailscale)",
                         first=["bash", "-c", "tailscale ping -c 3 --timeout 5s marvin; tailscale status | grep -i marvin"]),
    "marvin-power-mode": dict(title="marvin power mode not performance", verdict=v_power_mode, for_s=120, ok_after_s=120,
                              good="performance (BIOS Quiet/Balanced hold the GPU at ~55 W)",
                              first=MARVIN_SSH + ["~/homelab/marvin/fan/ec-module.sh check"]),
    "marvin-low-power": dict(title="marvin GPU busy but under 100 W", verdict=v_low_power, for_s=120, ok_after_s=120,
                             good="busy GPU at ~120-140 W (CLAUDE.md Known issues: the 55 W problem)",
                             first=MARVIN_SSH + ["~/homelab/marvin/fan/ec-module.sh check; for i in 1 2 3; do "
                                                 "echo busy $(cat /sys/class/drm/card1/device/gpu_busy_percent)% "
                                                 "$(( $(cat /sys/class/drm/card1/device/hwmon/hwmon*/power1_average)/1000000 )) W; sleep 2; done"]),
    "marvin-temp": dict(title="marvin GPU over 90 C", verdict=v_temp("marvin_edge", 90, "GPU"), for_s=300, ok_after_s=120,
                        good="under 90 C (with the fan curve ~73 C median under load)",
                        first=MARVIN_SSH + ["~/homelab/marvin/fan/curve.sh check"]),
    "zaphod-card1-temp": dict(title="zaphod card 1 junction over 100 C", for_s=300, ok_after_s=120,
                              verdict=v_temp("zaphod_card1_junction", 100, "junction"),
                              good="under 100 C (the card slows itself near 110 C)", first=["bash", "-c", ZAPHOD_TEMPS]),
    "zaphod-card2-temp": dict(title="zaphod card 2 junction over 100 C", for_s=300, ok_after_s=120,
                              verdict=v_temp("zaphod_card2_junction", 100, "junction"),
                              good="under 100 C (the card slows itself near 110 C)", first=["bash", "-c", ZAPHOD_TEMPS]),
    "llama-zaphod": dict(title="zaphod llama-swap not healthy", verdict=v_health("zaphod_health", False), for_s=120,
                         ok_after_s=60, good="/health 200",
                         first=["bash", "-c", "curl -s -m 5 http://127.0.0.1:8080/health; echo; cd ~/llama-swap && "
                                "docker compose ps --format '{{.Name}} {{.Status}}' && docker compose logs --tail 12 2>&1 | cut -c1-160"]),
    "llama-marvin": dict(title="marvin llama-swap not healthy", verdict=v_health("marvin_health", True), for_s=120,
                         ok_after_s=60, good="/health 200",
                         first=MARVIN_SSH + ["curl -s -m 5 http://127.0.0.1:8080/health; echo; cd ~/homelab/marvin/llama-swap && "
                                             "docker compose ps --format '{{.Name}} {{.Status}}' && docker compose logs --tail 12 2>&1 | cut -c1-160"]),
}


def first_text(c):
    f = c["first"]
    return f[-1] if f[:1] in (["bash"], ["ssh"]) else " ".join(f)


# --- state -------------------------------------------------------------------------------------------------------
def db(path=None):
    path = path or DB
    os.makedirs(os.path.dirname(path), mode=0o700, exist_ok=True)
    c = sqlite3.connect(path, isolation_level=None, timeout=10)
    c.execute("""create table if not exists checks (id text primary key, bad_since integer, ok_since integer,
                 alerted_at integer, reminded_at integer, acked_at integer, snoozed_until integer, value text)""")
    c.execute("create table if not exists sent (time integer)")
    return c


def log(line):
    os.makedirs(STATE, mode=0o700, exist_ok=True)
    with open(LOG, "a") as f:
        f.write(time.strftime("%Y-%m-%d %H:%M:%S %Z ") + line + "\n")


class Watchdog:
    def __init__(self, conn, send, now=time.time, buttons=None):
        self.c, self.send, self.now, self.buttons = conn, send, now, buttons or (lambda cid, inc: "")

    def row(self, cid):
        self.c.execute("insert or ignore into checks (id) values (?)", (cid,))
        r = self.c.execute("select bad_since, ok_since, alerted_at, reminded_at, acked_at, snoozed_until from checks "
                           "where id=?", (cid,)).fetchone()
        return dict(zip(("bad_since", "ok_since", "alerted_at", "reminded_at", "acked_at", "snoozed_until"), r))

    def save(self, cid, **kv):
        sets = ", ".join(f"{k}=?" for k in kv)
        self.c.execute(f"update checks set {sets} where id=?", (*kv.values(), cid))

    def notify(self, kind, cid, text, actions=""):
        t = int(self.now())
        if self.c.execute("select count(*) from sent where time > ?", (t - 3600,)).fetchone()[0] >= MAX_PER_HOUR:
            log(f"rate limit: not sent: {kind} {cid}")
            return
        self.c.execute("insert into sent values (?)", (t,))
        self.send(kind, cid, text, actions)

    def step(self, readings):
        t = int(self.now())
        for cid, c in CHECKS.items():
            verdict, value = c["verdict"](readings)
            s = self.row(cid)  # creates the row first, so the first reading is kept too
            self.save(cid, value=value)
            snoozed = (s["snoozed_until"] or 0) > t
            if verdict == "unknown":
                continue
            if verdict == "bad":
                if s["bad_since"] is None:
                    self.save(cid, bad_since=t, ok_since=None)
                    s["bad_since"] = t
                elif s["ok_since"] is not None:
                    self.save(cid, ok_since=None)
                if snoozed:
                    continue
                since = time.strftime("%H:%M", time.localtime(s["bad_since"]))
                body = (f"Now: {value} (bad since {since})\nGood: {c['good']}\nFirst check: {first_text(c)}")
                if s["alerted_at"] is None and t - s["bad_since"] >= c["for_s"]:
                    self.save(cid, alerted_at=t)
                    self.notify("alert", cid, (c["title"], body), self.buttons(cid, s["bad_since"]))
                elif s["alerted_at"] is not None and s["acked_at"] is None and s["reminded_at"] is None \
                        and t - s["alerted_at"] >= REMIND_S:
                    self.save(cid, reminded_at=t)
                    self.notify("reminder", cid, ("Still: " + c["title"], body), self.buttons(cid, s["bad_since"]))
            else:  # ok
                if s["bad_since"] is None:
                    continue
                if s["ok_since"] is None:
                    self.save(cid, ok_since=t)
                elif t - s["ok_since"] >= c["ok_after_s"]:
                    if s["alerted_at"] is not None:
                        mins = (t - s["bad_since"]) // 60
                        self.notify("resolved", cid, ("Resolved: " + c["title"], f"Now: {value} (was bad {mins} min)"))
                    self.save(cid, bad_since=None, ok_since=None, alerted_at=None, reminded_at=None, acked_at=None)


# --- called by the buttons (hub/actions/actiond.py) ----------------------------------------------------------------
def ack(cid, incident):
    c = db()
    r = c.execute("select bad_since from checks where id=?", (cid,)).fetchone()
    if not r or r[0] != incident:
        return "that problem has already ended"
    c.execute("update checks set acked_at=? where id=?", (int(time.time()), cid))
    return "acknowledged (no reminder)"


def snooze(cid, seconds=3600):
    c = db()
    c.execute("insert or ignore into checks (id) values (?)", (cid,))
    # alerted_at reset: if still bad when the snooze ends, it alerts again
    c.execute("update checks set snoozed_until=?, alerted_at=null, reminded_at=null where id=?",
              (int(time.time()) + seconds, cid))
    return f"snoozed {seconds // 60} min"


def diagnose(cid):
    p = subprocess.run(CHECKS[cid]["first"], capture_output=True, text=True, timeout=60)
    out = (p.stdout + p.stderr).strip() or "(no output)"
    return out[-3500:]  # ntfy's message limit is 4,096 bytes


# --- the service ---------------------------------------------------------------------------------------------------
def real_send(kind, cid, text, actions):
    title, body = text
    args = [os.path.join(HUB, "notify")]
    if kind in ("alert", "reminder"):
        args += ["-t", "lab-alert", "-p", "high", "-g", "rotating_light"]
    else:
        args += ["-g", "white_check_mark"]
    if actions:
        args += ["-a", actions]
    p = subprocess.run(args + [title, body], capture_output=True, text=True, timeout=30)
    log(f"{kind} {cid}: {title} -> {p.stdout.strip() or p.stderr.strip()}")


def real_buttons(cid, incident):
    a = os.path.join(HUB, "actions", "actiond.py")
    out = []
    for args in (["wd_ack", "Ack", cid, str(incident)], ["wd_snooze", "Snooze 1 h", cid], ["wd_diag", "Run check", cid]):
        p = subprocess.run([a, "button", *args], capture_output=True, text=True, timeout=15)
        if p.returncode == 0:
            out.append(p.stdout.strip())
        else:
            log(f"button {args[0]} for {cid} not made: {p.stderr.strip()[-200:]}")
    return ";".join(out)


def run(interval):
    w = Watchdog(db(), real_send, buttons=real_buttons)
    log(f"started: {len(CHECKS)} checks every {interval} s")
    print(f"lab-watchdog: {len(CHECKS)} checks every {interval} s, log {LOG}", flush=True)
    while True:
        try:
            w.step(read_all())
        except Exception as e:  # keep watching; report in the log
            log(f"loop error: {type(e).__name__}: {e}")
        time.sleep(interval)


def once():
    r = read_all()
    for cid, c in CHECKS.items():
        v, val = c["verdict"](r)
        print(f"{v:8s} {cid:20s} {val}")


def status():
    c = db()
    for row in c.execute("select id, bad_since, alerted_at, acked_at, snoozed_until, value from checks order by id"):
        f = lambda x: time.strftime("%m-%d %H:%M", time.localtime(x)) if x else "-"
        print(f"{row[0]:20s} bad since {f(row[1]):12s} alerted {f(row[2]):12s} acked {f(row[3]):12s} "
              f"snoozed until {f(row[4]):12s} {row[5]}")


# --- test: made-up readings, fake clock, notifications collected instead of sent ------------------------------------
def test():
    clock = [1_000_000.0]
    sent = []
    good = {"marvin_reach": True, "marvin_busy": "0", "marvin_watts": "9", "marvin_edge": "45",
            "marvin_power_mode": "performance", "marvin_health": "200", "zaphod_card1_junction": 40,
            "zaphod_card2_junction": 40, "zaphod_health": "200"}
    global LOG
    LOG = os.path.join(STATE, "test.log")  # the test's lines stay out of the service log
    path = os.path.join(STATE, "test.db")
    if os.path.exists(path):
        os.remove(path)
    w = Watchdog(db(path), lambda k, cid, text, a: sent.append((k, cid, text[0])), now=lambda: clock[0],
                 buttons=lambda cid, inc: f"buttons:{cid}:{inc}")
    results = []

    def tick(readings, minutes=1):
        for _ in range(minutes):
            w.step(readings)
            clock[0] += 60

    def expect(name, kinds):
        got = [(k, cid) for k, cid, _ in sent]
        ok = got == kinds
        results.append((name, ok, got))
        sent.clear()

    quiet = dict(good, marvin_power_mode="quiet", marvin_busy="100", marvin_watts="54")
    tick(good, 3); expect("all good: nothing sent", [])
    tick(quiet, 1); expect("bad for 1 min (< 2): nothing yet", [])
    tick(quiet, 2); expect("bad for 3 min: one alert per check (power mode, low power)",
                          [("alert", "marvin-power-mode"), ("alert", "marvin-low-power")])
    tick(quiet, 30); expect("stays bad 30 min: no repeat", [])
    tick(good, 1); expect("good for 1 min (< 2): not resolved yet", [])
    tick(quiet, 1); tick(good, 1); expect("flaps back: still no resolved, no new alert", [])
    tick(good, 3); expect("good 3 min: one resolved per check",
                          [("resolved", "marvin-power-mode"), ("resolved", "marvin-low-power")])
    unreachable = dict(good, marvin_reach=False)
    for k in list(unreachable):
        if k.startswith("marvin_") and k != "marvin_reach":
            unreachable[k] = None
    tick(unreachable, 4); expect("marvin unreachable 4 min: only the reachability alert", [("alert", "marvin-reach")])
    tick(good, 2); expect("back: resolved", [("resolved", "marvin-reach")])
    hot = dict(good, zaphod_card2_junction=104)
    tick(hot, 6); expect("card 2 at 104 C for 6 min: one alert", [("alert", "zaphod-card2-temp")])
    w.c.execute("update checks set snoozed_until=?, alerted_at=null where id='zaphod-card2-temp'", (int(clock[0]) + 3600,))
    tick(hot, 59); expect("snoozed 1 h, still hot: silent", [])
    tick(hot, 2); expect("snooze over, still hot: alerts again", [("alert", "zaphod-card2-temp")])
    tick(hot, 240); expect("4 h unresolved, not acked: one reminder", [("reminder", "zaphod-card2-temp")])
    tick(hot, 300); expect("after the reminder: nothing more", [])
    tick(good, 3); expect("cools: resolved", [("resolved", "zaphod-card2-temp")])
    hot1 = dict(good, zaphod_card1_junction=104)
    tick(hot1, 6); sent.clear()
    w.c.execute("update checks set acked_at=? where id='zaphod-card1-temp'", (int(clock[0]),))
    tick(hot1, 300); expect("acked: no reminder", [])
    tick(good, 3); sent.clear()
    clock[0] += 3601  # an hour after the last message, so the cap starts empty
    for i in range(20):
        w.notify("alert", f"x{i}", ("t", "b"))
    n = len(sent); sent.clear()
    results.append(("rate limit: 20 alerts in a minute -> 12 sent", n == 12, n))
    os.remove(path)
    if os.path.exists(LOG):
        os.remove(LOG)
    width = max(len(r[0]) for r in results)
    for name, ok, got in results:
        print(f"{'PASS' if ok else 'FAIL'}  {name:{width}s}" + ("" if ok else f"   got {got}"))
    print(f"{sum(r[1] for r in results)}/{len(results)} pass")
    return all(r[1] for r in results)


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[:1] == ["run"]:
        run(int(a[2]) if a[1:2] == ["--interval"] else 60)
    elif a[:1] == ["once"]:
        once()
    elif a[:1] == ["status"]:
        status()
    elif a[:1] == ["test"]:
        sys.exit(0 if test() else 1)
    else:
        print(__doc__)
        sys.exit(2)
