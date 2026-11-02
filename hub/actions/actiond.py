#!/usr/bin/env python3
"""Lab actions: buttons on phone notifications (issue #11, phase 2).

A notification button (ntfy `http` action) POSTs a one-time token here; if the token is valid, the action stored
for it runs. Tokens are made by this same file (`button`), so a button can only ever do what was stored when the
notification was sent: no parameters come from the phone.

  actiond.py serve                         the service (systemd unit lab-actions.service; 127.0.0.1:2587, Caddy
                                           serves it as https://act.<DOMAIN> on the tailnet only)
  actiond.py button ack "Label" "subject"  store an action, print its ntfy button ("http, Label, <url>, body=...")
  actiond.py button skip "Label" <mlflow run id> "subject"
  actiond.py button rerun "Label" "subject"  (reads LAB_RERUN / LAB_RERUN_CWD and the settings in RERUN_ENV from the
                                           environment: tuning/log calls it right after a benchmark)
  actiond.py log [n]                       the last n lines of the action log (default 20)

Token: <id>.<sig>, sig = HMAC-SHA256(ACTION_SECRET, id|kind|expires), base64url. Checked in this order: known id,
signature, not expired (24 h, same as ntfy's message cache), not used (marked used in the same transaction).
Every request is logged with time, caller address (Caddy's X-Forwarded-For), id, kind and result.

Actions (KINDS below); only these exist:
  ack    log it and send "Acknowledged: <subject>" (lab-done). Proves the tap -> run -> notify path.
  skip   tag the MLflow run reviewed=skipped; "Marked reviewed".
  wd_ack     watchdog alert: acknowledged, so no 4 h reminder (hub/watchdog/watchdog.py)
  wd_snooze  watchdog alert: no alerts for this check for 1 h; if still bad after, it alerts again
  wd_diag    watchdog alert: run the check's fixed read-only first command, send its output back
  rerun  start the same benchmark again: one of RERUN_SCRIPTS in tuning/, same arguments and settings, zaphod only
         (until reservations, #3), one at a time (lock file; "busy" leaves the button usable). Runs detached, output in
         ~/.local/state/lab-actions/jobs/<id>.log; the benchmark's own notification reports the result.
         Restarting this service also stops a running rerun (same service cgroup).

Good result: `curl` of a fresh button's URL and body -> 200 "ok"; the same again -> 403 "already used".
Secret: ACTION_SECRET in hub/.env (mode 600, Proton Pass "lab actions"). Never printed.
"""
import base64, fcntl, hashlib, hmac, http.server, json, os, secrets, sqlite3, subprocess, sys, time, urllib.request

HUB = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
STATE = os.path.expanduser(os.environ.get("LAB_ACTIONS_STATE", "~/.local/state/lab-actions"))
DB, LOG = os.path.join(STATE, "actions.db"), os.path.join(STATE, "actions.log")
PORT = int(os.environ.get("LAB_ACTIONS_PORT", "2587"))
TTL = 24 * 3600
REPO = os.path.dirname(HUB)
TUNING = os.path.join(REPO, "tuning")
RERUN_SCRIPTS = {"speed", "quality", "bench", "kl", "tbench"}
# settings a rerun repeats (never keys: API_KEY etc. are re-read by the script itself from its .env file)
RERUN_ENV = ["MACHINE", "BASE_URL", "MODEL", "SUITE", "TARGET", "SERIES", "SKILL_NAME", "SKILL_SHA", "BACKEND",
             "BACKEND_VERSION", "CONTEXT_LENGTH", "ENDPOINT", "ENGINE", "MODEL_NAME", "PLATFORM", "PLATFORM_NAME",
             "PROFILE", "QUANT", "RERUN"]
LOCK = os.path.join(STATE, "rerun.lock")
MLFLOW = "http://127.0.0.1:5000"
MAX_BODY = 256


def env(name):
    with open(os.path.join(HUB, ".env")) as f:
        for line in f:
            if line.startswith(name + "="):
                return line.split("=", 1)[1].strip().strip("'\"")
    sys.exit(f"{name} missing in hub/.env")


def db():
    os.makedirs(STATE, mode=0o700, exist_ok=True)
    c = sqlite3.connect(DB, isolation_level=None, timeout=10)
    c.execute("""create table if not exists actions (id text primary key, kind text not null, args text not null,
                 created integer not null, expires integer not null, used integer, result text)""")
    return c


def sign(secret, id_, kind, expires):
    mac = hmac.new(secret.encode(), f"{id_}|{kind}|{expires}".encode(), hashlib.sha256).digest()
    return base64.urlsafe_b64encode(mac).decode().rstrip("=")


def log(line):
    os.makedirs(STATE, mode=0o700, exist_ok=True)
    with open(LOG, "a") as f:
        f.write(time.strftime("%Y-%m-%d %H:%M:%S %Z ") + line + "\n")


def notify(*args):
    """Follow-up notification; a failure is logged, never raised."""
    r = subprocess.run([os.path.join(HUB, "notify"), *args], capture_output=True, text=True, timeout=30)
    if r.returncode:
        log(f"follow-up notification failed: {r.stdout.strip()} {r.stderr.strip()}")


# --- the actions -------------------------------------------------------------------------------------------------
def do_ack(args):
    notify("-g", "ok_hand", "Acknowledged", args["subject"])
    return "acknowledged"


class Busy(Exception):
    """The action can't run now; the button stays usable."""


def do_skip(args):
    body = json.dumps({"run_id": args["run_id"], "key": "reviewed", "value": "skipped"}).encode()
    req = urllib.request.Request(f"{MLFLOW}/api/2.0/mlflow/runs/set-tag", body, {"Content-Type": "application/json"})
    urllib.request.urlopen(req, timeout=15).read()
    notify("-g", "fast_forward", "Marked reviewed", args["subject"])
    return f"run {args['run_id'][:8]} tagged reviewed=skipped"


def rerun_argv(args):
    """The command to run, checked again here (the stored action came from tuning/log, but check anyway)."""
    argv, env = args["argv"], args["env"]
    if not argv or argv[0] not in RERUN_SCRIPTS:
        raise ValueError(f"not a rerunnable benchmark: {argv[:1]}")
    if os.path.realpath(args["cwd"]) != os.path.realpath(TUNING):
        raise ValueError("benchmarks run from tuning/")
    # zaphod only: marvin is reached directly (MACHINE, BASE_URL, TARGET, ...) or through zaphod's llama-swap peer
    # ("marvin/big" as the model argument or MODEL)
    looks_marvin = any("marvin" in str(v) for v in [*argv[1:2], *env.values()])
    if env.get("MACHINE", "zaphod") != "zaphod" or looks_marvin \
            or not env.get("BASE_URL", "http://127.0.0.1:8080").startswith("http://127.0.0.1:8080"):
        raise ValueError("zaphod runs only (until reservations, #3)")
    return [os.path.join(TUNING, argv[0]), *argv[1:]], env


def do_rerun(args):
    cmd, env = rerun_argv(args)
    os.makedirs(os.path.join(STATE, "jobs"), mode=0o700, exist_ok=True)
    with open(LOCK, "a") as lk:  # busy check; the job itself holds the lock through flock(1)
        try:
            fcntl.flock(lk, fcntl.LOCK_EX | fcntl.LOCK_NB)
            fcntl.flock(lk, fcntl.LOCK_UN)
        except BlockingIOError:
            raise Busy("busy: another rerun is still running")
    out = os.path.join(STATE, "jobs", f"{args['id']}.log")
    full_env = {k: v for k, v in os.environ.items() if k not in RERUN_ENV and k not in ("LAB_RERUN", "LAB_RERUN_CWD")}
    full_env.update(env)
    with open(out, "w") as o:  # setsid -f: detached, returns at once; flock -n holds the lock for the job's life
        subprocess.run(["setsid", "-f", "flock", "-n", LOCK, *cmd], cwd=TUNING, env=full_env, stdin=subprocess.DEVNULL,
                       stdout=o, stderr=subprocess.STDOUT, check=True, timeout=15)
    shown = " ".join([args["argv"][0], *args["argv"][1:]])[:150]
    notify("-g", "repeat", "Rerun started", f"{shown}\nlog: {out}")
    return f"started {shown}"


sys.path.insert(0, os.path.join(HUB, "watchdog"))
import watchdog  # noqa: E402  (the check table: buttons can only name a check, never a command)


def do_wd_ack(args):
    r = watchdog.ack(args["check"], args["incident"])
    notify("-g", "ok_hand", "Acknowledged", f"{watchdog.CHECKS[args['check']]['title']}: {r}")
    return r


def do_wd_snooze(args):
    r = watchdog.snooze(args["check"], 3600)
    notify("-g", "zzz", "Snoozed 1 h", watchdog.CHECKS[args["check"]]["title"])
    return r


def do_wd_diag(args):
    out = watchdog.diagnose(args["check"])
    notify("-g", "mag", "Check: " + watchdog.CHECKS[args["check"]]["title"], out)
    return f"ran first check ({len(out)} bytes back)"


KINDS = {"ack": do_ack, "skip": do_skip, "rerun": do_rerun,
         "wd_ack": do_wd_ack, "wd_snooze": do_wd_snooze, "wd_diag": do_wd_diag}


# --- making a button ---------------------------------------------------------------------------------------------
def button(kind, label, **args):
    if kind not in KINDS:
        sys.exit(f"unknown action {kind}; known: {', '.join(KINDS)}")
    secret, domain = env("ACTION_SECRET"), env("DOMAIN")
    id_, now = secrets.token_urlsafe(12), int(time.time())
    db().execute("insert into actions values (?,?,?,?,?,null,null)", (id_, kind, json.dumps(args), now, now + TTL))
    tok = f"{id_}.{sign(secret, id_, kind, now + TTL)}"
    # ntfy short form; label must not contain , or ; (separators)
    label = label.replace(",", " ").replace(";", " ")
    return f"http, {label}, https://act.{domain}/a, body={tok}, clear=true"


# --- the service -------------------------------------------------------------------------------------------------
class Handler(http.server.BaseHTTPRequestHandler):
    server_version = "lab-actions"
    sys_version = ""

    def reply(self, code, text):
        body = (text + "\n").encode()
        self.send_response(code)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):  # our own log only
        pass

    def do_GET(self):
        if self.path == "/health":
            return self.reply(200, "ok")
        self.reply(405, "POST only")

    def do_POST(self):
        who = self.headers.get("X-Forwarded-For", self.client_address[0])
        if self.path != "/a":
            return self.reply(404, "not found")
        n = int(self.headers.get("Content-Length") or 0)
        if n <= 0 or n > MAX_BODY:
            log(f"{who} - - refused: body size {n}")
            return self.reply(400, "bad request")
        tok = self.rfile.read(n).decode(errors="replace").strip()
        code, text, id_, kind = self.check_and_run(tok)
        log(f"{who} {id_ or '-'} {kind or '-'} {code} {text}")
        self.reply(code, text)

    def check_and_run(self, tok):
        id_, _, sig = tok.partition(".")
        c = self.server.db
        row = c.execute("select kind, args, expires, used from actions where id=?", (id_,)).fetchone()
        if not row:
            return 404, "unknown button", None, None
        kind, args, expires, used = row
        if not hmac.compare_digest(sig, sign(self.server.secret, id_, kind, expires)):
            return 403, "bad signature", id_, kind
        if time.time() > expires:
            return 403, "expired", id_, kind
        if used:
            return 403, "already used", id_, kind
        if kind not in KINDS:  # made by newer code than this running service: refuse without using the button up
            return 501, "unknown action: restart lab-actions to load the new code", id_, kind
        c.execute("begin immediate")
        claimed = c.execute("update actions set used=? where id=? and used is null", (int(time.time()), id_)).rowcount
        c.execute("commit")
        if claimed != 1:
            return 403, "already used", id_, kind
        try:
            a = json.loads(args)
            a["id"] = id_
            result = KINDS[kind](a)
        except Busy as e:  # not run: give the button back
            c.execute("update actions set used=null where id=?", (id_,))
            return 409, str(e), id_, kind
        except Exception as e:  # report, never crash the service
            result = f"failed: {type(e).__name__}: {e}"
            c.execute("update actions set result=? where id=?", (result, id_))
            return 500, result, id_, kind
        c.execute("update actions set result=? where id=?", (result, id_))
        return 200, f"ok: {result}", id_, kind


def serve():
    srv = http.server.HTTPServer(("127.0.0.1", PORT), Handler)  # one request at a time, on purpose
    srv.secret, srv.db = env("ACTION_SECRET"), db()
    srv.db.execute("delete from actions where expires < ?", (int(time.time()) - 7 * 86400,))  # keep a week
    log(f"service started on 127.0.0.1:{PORT}")
    print(f"lab-actions on 127.0.0.1:{PORT}, log {LOG}", flush=True)
    srv.serve_forever()


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[:1] == ["serve"]:
        serve()
    elif a[:1] == ["button"] and len(a) == 4 and a[1] == "ack":
        print(button("ack", a[2], subject=a[3]))
    elif a[:1] == ["button"] and len(a) == 5 and a[1] == "skip":
        print(button("skip", a[2], run_id=a[3], subject=a[4]))
    elif a[:1] == ["button"] and a[1:2] == ["wd_ack"] and len(a) == 5 and a[3] in watchdog.CHECKS:
        print(button("wd_ack", a[2], check=a[3], incident=int(a[4])))
    elif a[:1] == ["button"] and a[1:2] in (["wd_snooze"], ["wd_diag"]) and len(a) == 4 and a[3] in watchdog.CHECKS:
        print(button(a[1], a[2], check=a[3]))
    elif a[:1] == ["button"] and len(a) == 4 and a[1] == "rerun":
        raw = base64.b64decode(os.environ["LAB_RERUN"]).split(b"\0")
        argv = [x.decode() for x in raw[:-1]] if raw and raw[-1] == b"" else [x.decode() for x in raw]
        args = {"argv": argv, "cwd": os.environ.get("LAB_RERUN_CWD", ""),
                "env": {k: os.environ[k] for k in RERUN_ENV if os.environ.get(k)}, "subject": a[3]}
        rerun_argv(args)  # refuse to make a button that could never run
        print(button("rerun", a[2], **args))
    elif a[:1] == ["log"]:
        os.system(f"tail -n {int(a[1]) if len(a) > 1 else 20} {LOG}")
    else:
        print(__doc__)
        sys.exit(2)
