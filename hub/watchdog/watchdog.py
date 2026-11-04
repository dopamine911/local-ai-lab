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
import datetime, glob, json, os, re, socket, sqlite3, ssl, subprocess, sys, time, urllib.request

HUB = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
STATE = os.path.expanduser(os.environ.get("LAB_WATCHDOG_STATE", "~/.local/state/lab-watchdog"))
DB, LOG = os.path.join(STATE, "state.db"), os.path.join(STATE, "watchdog.log")
REMIND_S = 4 * 3600
MAX_PER_HOUR = 12  # all watchdog notifications together; more are logged, not sent
MJ, MJ_REG = os.path.expanduser("~/.local/bin/mcpjungle"), "http://127.0.0.1:8085"  # hub/mcp/setup.sh
CERT_HOST = "notify.ai.dopamine911.com"  # any *.ai name: Caddy holds one wildcard certificate
MARVIN_SSH = ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "marvin"]

# One read of marvin: key=value lines. Read-only (sysfs, curl to its own llama-swap).
MARVIN_READ = r'''
# The GPU's card number can change at boot (card1 until 2026-11-02, card0 since): find it by its busy file.
# Unreadable values stay empty (= unknown), never 0: a 0 looked like "GPU 0 C, ok" and hid that bug.
D=$(dirname "$(ls /sys/class/drm/card[0-9]*/device/gpu_busy_percent 2>/dev/null | head -1)"); E=/sys/class/ec_su_axb35
echo busy=$(cat $D/gpu_busy_percent 2>/dev/null)
W=$(cat $D/hwmon/hwmon*/power1_average 2>/dev/null); echo watts=${W:+$((W / 1000000))}
T=$(cat $D/hwmon/hwmon*/temp1_input 2>/dev/null); echo edge=${T:+$((T / 1000))}
echo power_mode=$(cat $E/apu/power_mode 2>/dev/null)
echo health=$(curl -s -o /dev/null -m 5 -w '%{http_code}' http://127.0.0.1:8080/health)
echo disk_free=$(df -P / | awk 'NR==2{print 100-$5}')
echo bad_containers=$(docker ps -a --format '{{.Names}}:{{.State}}:{{.Status}}' | grep -E ':restarting:|unhealthy' | cut -d: -f1 | paste -sd,)
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
    # batch 2: disk, containers, MCPJungle, backup and memory sync age, certificate
    st = os.statvfs("/")
    r["zaphod_disk_free"] = round(100 * st.f_bavail / st.f_blocks)
    try:
        p = subprocess.run(["docker", "ps", "-a", "--format", "{{.Names}}:{{.State}}:{{.Status}}"],
                           capture_output=True, text=True, timeout=20)
        if p.returncode == 0:
            r["zaphod_bad_containers"] = ",".join(l.split(":")[0] for l in p.stdout.splitlines()
                                                  if ":restarting:" in l or "unhealthy" in l)
    except subprocess.TimeoutExpired:
        pass
    try:
        p = subprocess.run([MJ, "--registry", MJ_REG, "list", "tools"], capture_output=True, text=True, timeout=30)
        have = set(re.findall(r"^\d+\. ([a-z0-9_-]+)__", p.stdout + p.stderr, re.M))
        want = {os.path.basename(f)[:-5] for f in glob.glob(os.path.join(HUB, "mcp", "servers", "*.json"))}
        r["mcp_missing"] = ",".join(sorted(want - have)) if have or p.returncode == 0 else "list tools failed"
    except (subprocess.TimeoutExpired, OSError):
        r["mcp_missing"] = "list tools failed"
    r["backup_age_h"] = log_age_h(os.path.expanduser("~/.cache/backup-to-marvin.log"),
                                  r"^== (\d{4}-\d\d-\d\d \d\d:\d\d:\d\d) done:")
    sync = os.path.expanduser("~/.cache/memory-sync.log")
    try:
        last = open(sync).read().strip().splitlines()[-1]
        r["sync_age_h"] = (time.time() - os.path.getmtime(sync)) / 3600
        r["sync_last_ok"] = bool(re.match(r"^sync: \d+ uploaded", last))
    except (OSError, IndexError):
        pass
    for name, (ok, detail) in mcp_calls().items():
        r[f"mcp_{name}_ok"], r[f"mcp_{name}_detail"] = ok, detail
    try:
        ctx = ssl.create_default_context()
        with socket.create_connection((CERT_HOST, 443), timeout=10) as sk, ctx.wrap_socket(sk, server_hostname=CERT_HOST) as t:
            end = ssl.cert_time_to_seconds(t.getpeercert()["notAfter"])
        r["cert_days"] = (end - time.time()) / 86400
    except (OSError, ssl.SSLError, KeyError, ValueError):
        r["cert_days"] = None
    try:
        p = subprocess.run(["curl", "-s", "-o", "/dev/null", "-m", "5", "-w", "%{http_code}",
                            "http://127.0.0.1:8080/health"], capture_output=True, text=True, timeout=10)
        r["zaphod_health"] = p.stdout.strip()
    except subprocess.TimeoutExpired:
        r["zaphod_health"] = "timeout"
    return r


MCP_URL = "http://127.0.0.1:8085/mcp"
MCP_CALLS = [("memory", "memory__search", {"query": "REBUILD", "max_results": 1}),
             ("recall", "recall__mnemosyne_recall", {"query": "where does a machine change have to be recorded", "limit": 1})]


def mcp_calls():
    """One real call per tool through MCPJungle, as the `watchdog` agent login (hub/mcp/clients/watchdog.json: memory
    and recall only). Returns {name: (ok, detail)}; {} when the token is missing (check unknown)."""
    try:
        tok = [l.split("=", 1)[1].strip().strip("'\"") for l in open(os.path.join(HUB, ".env"))
               if l.startswith("MCP_TOKEN_WATCHDOG=")][0]
    except (OSError, IndexError):
        return {}

    def post(body, sid=None):
        h = {"Authorization": "Bearer " + tok, "Content-Type": "application/json",
             "Accept": "application/json, text/event-stream"}
        if sid:
            h["Mcp-Session-Id"] = sid
        with urllib.request.urlopen(urllib.request.Request(MCP_URL, json.dumps(body).encode(), h), timeout=30) as r:
            raw, sid2 = r.read().decode(), r.headers.get("Mcp-Session-Id")
        data = None
        if raw.lstrip().startswith("{"):
            data = json.loads(raw)
        else:
            for line in raw.splitlines():
                if line.startswith("data:"):
                    data = json.loads(line[5:])
        return data, sid2

    out = {}
    try:
        init, sid = post({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
            "protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "lab-watchdog", "version": "1"}}})
        if not init or "result" not in init:
            return {n: (False, "gateway initialize failed") for n, _, _ in MCP_CALLS}
        post({"jsonrpc": "2.0", "method": "notifications/initialized"}, sid)
    except Exception as e:
        return {n: (False, f"gateway: {type(e).__name__}") for n, _, _ in MCP_CALLS}
    for i, (name, tool, args) in enumerate(MCP_CALLS, 2):
        try:
            r, _ = post({"jsonrpc": "2.0", "id": i, "method": "tools/call", "params": {"name": tool, "arguments": args}}, sid)
            res = (r or {}).get("result") or {}
            text = " ".join(c.get("text", "") for c in res.get("content", []) if c.get("type") == "text")
            m = re.search(r'"count":\s*(\d+)', text)
            n = int(m.group(1)) if m else 0
            if r and "error" in r:
                out[name] = (False, (r["error"].get("message") or "error")[:80])
            elif res.get("isError"):
                out[name] = (False, text[:80] or "tool error")
            else:
                out[name] = (n > 0, f"{n} result(s)")
        except Exception as e:
            out[name] = (False, f"{type(e).__name__}: {str(e)[:60]}")
    return out


def log_age_h(path, pattern):
    """Hours since the last line matching pattern (its timestamp, local time); None if none found."""
    try:
        stamps = re.findall(pattern, open(path).read(), re.M)
    except OSError:
        return None
    if not stamps:
        return None
    t = datetime.datetime.strptime(stamps[-1], "%Y-%m-%d %H:%M:%S").timestamp()
    return (time.time() - t) / 3600


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
    # Under 75 W, not 100: writing in a long conversation runs at ~80% busy and 89-125 W (2026-11-04), which
    # tripped 100 W 8 times in one night. The fault this is for sits at 54-55 W.
    return ("bad" if busy > 80 and w < 75 else "ok"), f"busy {busy:.0f}%, {w:.0f} W"


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


def v_disk(key, needs_marvin):
    def v(r):
        f = num(r, key)
        if f is None or (needs_marvin and not marvin_known(r)):
            return "unknown", "-"
        return ("bad" if f < 10 else "ok"), f"/ {f:.0f}% free"
    return v


def v_containers(key, needs_marvin):
    def v(r):
        if key not in r or (needs_marvin and not marvin_known(r)):
            return "unknown", "-"
        bad = r[key] or ""
        return ("bad" if bad else "ok"), (f"restarting or unhealthy: {bad}" if bad else "none restarting or unhealthy")
    return v


def v_mcp(r):
    m = r.get("mcp_missing")
    if m is None:
        return "unknown", "-"
    return ("bad" if m else "ok"), (f"no tools from: {m}" if m else "every server has tools")


def v_mcp_calls(r):
    names = [n for n, _, _ in MCP_CALLS]
    if not all(f"mcp_{n}_ok" in r for n in names):
        return "unknown", "no MCP_TOKEN_WATCHDOG in hub/.env"
    value = ", ".join(f"{n}: {r[f'mcp_{n}_detail']}" for n in names)
    return ("ok" if all(r[f"mcp_{n}_ok"] for n in names) else "bad"), value


def v_backup(r):
    a = num(r, "backup_age_h")
    if a is None:
        return "bad", "no successful backup in the log"
    return ("bad" if a > 26 else "ok"), f"last success {a:.0f} h ago"


def v_sync(r):
    a = num(r, "sync_age_h")
    if a is None:
        return "unknown", "-"
    if not r.get("sync_last_ok"):
        return "bad", f"last run failed ({a:.1f} h ago)"
    return ("bad" if a > 2.5 else "ok"), f"last success {a:.1f} h ago"


def v_cert(r):
    d = num(r, "cert_days")
    if d is None:
        return "bad", f"no certificate read from {CERT_HOST}"
    return ("bad" if d < 14 else "ok"), f"expires in {d:.0f} days"


ZAPHOD_TEMPS = r'''for h in /sys/class/drm/card*/device/hwmon/hwmon*; do echo "$(basename $(readlink -f $h/../..)): $(for t in $h/temp*_label; do printf '%s=%s ' "$(cat $t)" "$(( $(cat ${t%_label}_input)/1000 ))"; done)busy=$(cat $h/../../gpu_busy_percent)% $(( $(cat $h/power1_average 2>/dev/null || echo 0)/1000000 )) W"; done'''

CHECKS = {
    "marvin-reach": dict(title="marvin unreachable", verdict=v_marvin_reach, for_s=180, ok_after_s=60,
                         good="answers over ssh (Tailscale)",
                         first=["bash", "-c", "tailscale ping -c 3 --timeout 5s marvin; tailscale status | grep -i marvin"]),
    "marvin-power-mode": dict(title="marvin power mode not performance", verdict=v_power_mode, for_s=120, ok_after_s=120,
                              good="performance (BIOS Quiet/Balanced hold the GPU at ~55 W)",
                              first=MARVIN_SSH + ["~/homelab/marvin/fan/ec-module.sh check"]),
    "marvin-low-power": dict(title="marvin GPU busy but under 75 W", verdict=v_low_power, for_s=120, ok_after_s=120,
                             good="busy GPU at ~120-140 W (CLAUDE.md Known issues: the 55 W problem)",
                             first=MARVIN_SSH + ["~/homelab/marvin/fan/ec-module.sh check; D=$(dirname \"$(ls /sys/class/drm/card[0-9]*/device/gpu_busy_percent 2>/dev/null | head -1)\"); for i in 1 2 3; do "
                                                 "echo busy $(cat $D/gpu_busy_percent)% "
                                                 "$(( $(cat $D/hwmon/hwmon*/power1_average)/1000000 )) W; sleep 2; done"]),
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
    # batch 2
    "disk-zaphod": dict(title="zaphod disk under 10% free", verdict=v_disk("zaphod_disk_free", False), for_s=300,
                        ok_after_s=120, good="10% or more free on / (models, hub data, Docker)",
                        first=["bash", "-c", "df -h / /boot; docker system df"]),
    "disk-marvin": dict(title="marvin disk under 10% free", verdict=v_disk("marvin_disk_free", True), for_s=300,
                        ok_after_s=120, good="10% or more free on /",
                        first=MARVIN_SSH + ["df -h / /boot; docker system df"]),
    "containers-zaphod": dict(title="zaphod container restarting or unhealthy", for_s=180, ok_after_s=120,
                              verdict=v_containers("zaphod_bad_containers", False), good="none restarting or unhealthy",
                              first=["bash", "-c", "docker ps -a --format '{{.Names}} {{.Status}}' | grep -E 'Restarting|unhealthy' "
                                     "|| echo 'none restarting or unhealthy now'; "
                                     "for c in $(docker ps -a --filter status=restarting --format '{{.Names}}') "
                                     "$(docker ps --filter health=unhealthy --format '{{.Names}}'); do echo \"== $c\"; "
                                     "docker logs --tail 8 $c 2>&1 | cut -c1-160; done"]),
    "containers-marvin": dict(title="marvin container restarting or unhealthy", for_s=180, ok_after_s=120,
                              verdict=v_containers("marvin_bad_containers", True), good="none restarting or unhealthy",
                              first=MARVIN_SSH + ["docker ps -a --format '{{.Names}} {{.Status}}'; "
                                                  "for c in $(docker ps -a --filter status=restarting --format '{{.Names}}') "
                                                  "$(docker ps --filter health=unhealthy --format '{{.Names}}'); do echo \"== $c\"; "
                                                  "docker logs --tail 8 $c 2>&1 | cut -c1-160; done"]),
    "mcp-tools": dict(title="MCPJungle: a tool server has no tools", verdict=v_mcp, for_s=300, ok_after_s=120,
                      good="every server in hub/mcp/servers/ lists tools",
                      first=["bash", "-c", f"{MJ} --registry {MJ_REG} list servers | grep -E '^[0-9]+\\.'; "
                             "docker logs --tail 10 mcpjungle 2>&1 | cut -c1-160"]),
    "mcp-calls": dict(title="MCP tools don't answer (memory / recall)", verdict=v_mcp_calls, for_s=180, ok_after_s=60,
                      good="a real memory search and recall through MCPJungle each return a result (the path agents use)",
                      first=["bash", "-c", "~/.local/bin/lab mcp status | tail -3; for c in mcpjungle mcp-localrecall mnemosyne; "
                             "do echo \"== $c\"; docker logs --since 10m $c 2>&1 | grep -iE 'error|invalid|warn' | tail -4 | cut -c1-160; done"]),
    "backup": dict(title="nightly backup older than 26 h", verdict=v_backup, for_s=60, ok_after_s=60,
                   good="a success every night at 04:17 (setup/backup-to-marvin.sh)",
                   first=["bash", "-c", "tail -6 ~/.cache/backup-to-marvin.log; crontab -l | grep backup"]),
    "memory-sync": dict(title="memory sync failed or late", verdict=v_sync, for_s=60, ok_after_s=60,
                        good="a success every hour at :41 (hub/memory/sync.sh)",
                        first=["bash", "-c", "tail -6 ~/.cache/memory-sync.log; crontab -l | grep memory"]),
    "certificate": dict(title="HTTPS certificate expires within 14 days", verdict=v_cert, for_s=60, ok_after_s=60,
                        good="Caddy renews about 30 days before expiry",
                        first=["bash", "-c", f"echo | openssl s_client -connect {CERT_HOST}:443 -servername {CERT_HOST} "
                               "2>/dev/null | openssl x509 -noout -subject -enddate; "
                               "docker logs --since 72h caddy 2>&1 | grep -iE 'certificate|tls|error' | tail -6 | cut -c1-200"]),
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
            "zaphod_card2_junction": 40, "zaphod_health": "200", "marvin_disk_free": "90", "zaphod_disk_free": 30,
            "marvin_bad_containers": "", "zaphod_bad_containers": "", "mcp_missing": "", "backup_age_h": 5,
            "sync_age_h": 0.5, "sync_last_ok": True, "cert_days": 60,
            "mcp_memory_ok": True, "mcp_memory_detail": "1 result(s)", "mcp_recall_ok": True, "mcp_recall_detail": "1 result(s)"}
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
    tick(dict(good, marvin_busy="85", marvin_watts="90"), 5)  # long-context writing (2026-11-04)
    expect("busy 85% at 90 W for 5 min: normal, nothing sent", [])
    unreachable = dict(good, marvin_reach=False)  # every marvin_* reading unknown
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
    tick(dict(good, zaphod_disk_free=7), 6); expect("disk 7% free for 6 min: one alert", [("alert", "disk-zaphod")])
    tick(good, 3); expect("disk freed: resolved", [("resolved", "disk-zaphod")])
    tick(dict(good, backup_age_h=30, sync_last_ok=False), 2)
    expect("backup 30 h old and sync failed: two alerts", [("alert", "backup"), ("alert", "memory-sync")])
    tick(good, 2); expect("both fine again: two resolved", [("resolved", "backup"), ("resolved", "memory-sync")])
    tick(dict(good, zaphod_bad_containers="ntfy", mcp_missing="recall"), 6)
    expect("container restarting + MCP server without tools: two alerts", [("alert", "containers-zaphod"), ("alert", "mcp-tools")])
    tick(good, 3); sent.clear()
    clock[0] += 3601  # a new hour: the earlier made-up alerts filled the 12-per-hour cap
    broken = dict(good, mcp_memory_ok=False, mcp_memory_detail="gateway: HTTPError")
    tick(broken, 2); expect("memory call failing 2 min (< 3): nothing yet", [])
    tick(broken, 2); expect("memory call failing 4 min: one alert", [("alert", "mcp-calls")])
    tick(good, 2); expect("memory answers again: resolved", [("resolved", "mcp-calls")])
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
