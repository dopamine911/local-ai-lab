#!/usr/bin/env python3
"""Lab actions: buttons on phone notifications (issue #11, phase 2).

A notification button (ntfy `http` action) POSTs a one-time token here; if the token is valid, the action stored
for it runs. Tokens are made by this same file (`button`), so a button can only ever do what was stored when the
notification was sent: no parameters come from the phone.

  actiond.py serve                         the service (systemd unit lab-actions.service; 127.0.0.1:2587, Caddy
                                           serves it as https://act.<DOMAIN> on the tailnet only)
  actiond.py button ack "Label" "subject"  store an action, print its ntfy button ("http, Label, <url>, body=...")
  actiond.py log [n]                       the last n lines of the action log (default 20)

Token: <id>.<sig>, sig = HMAC-SHA256(ACTION_SECRET, id|kind|expires), base64url. Checked in this order: known id,
signature, not expired (24 h, same as ntfy's message cache), not used (marked used in the same transaction).
Every request is logged with time, caller address (Caddy's X-Forwarded-For), id, kind and result.

Actions (KINDS below); only these exist:
  ack   log it and send "Acknowledged: <subject>" (lab-done). Proves the tap -> run -> notify path.

Good result: `curl` of a fresh button's URL and body -> 200 "ok"; the same again -> 403 "already used".
Secret: ACTION_SECRET in hub/.env (mode 600, Proton Pass "lab actions"). Never printed.
"""
import base64, hashlib, hmac, http.server, json, os, secrets, sqlite3, subprocess, sys, time

HUB = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
STATE = os.path.expanduser(os.environ.get("LAB_ACTIONS_STATE", "~/.local/state/lab-actions"))
DB, LOG = os.path.join(STATE, "actions.db"), os.path.join(STATE, "actions.log")
PORT = int(os.environ.get("LAB_ACTIONS_PORT", "2587"))
TTL = 24 * 3600
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


KINDS = {"ack": do_ack}


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
        c.execute("begin immediate")
        claimed = c.execute("update actions set used=? where id=? and used is null", (int(time.time()), id_)).rowcount
        c.execute("commit")
        if claimed != 1:
            return 403, "already used", id_, kind
        try:
            result = KINDS[kind](json.loads(args))
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
    elif a[:1] == ["log"]:
        os.system(f"tail -n {int(a[1]) if len(a) > 1 else 20} {LOG}")
    else:
        print(__doc__)
        sys.exit(2)
