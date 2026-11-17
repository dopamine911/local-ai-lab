#!/usr/bin/env python3
"""lab-engined: starts and stops allowlisted engine containers for llama-swap (plans/engines-and-frontend.md, option B).

llama-swap runs in a container and cannot start sibling containers. This small host service can, but only the ones
named in this machine's allowlist (engines/<machine>.json): image by digest, arguments, devices, read-only mounts.
The caller sends a name and nothing else.

  lab-engined.py serve          the service (systemd unit engines/lab-engined.service, installed by setup/host.sh
                                step 6f); socket ~/.local/state/lab-engined/engined.sock (LAB_ENGINED_SOCK)
  lab-engined.py check          parse the allowlist and print the docker command each engine would get (no Docker)

API on the Unix socket (llama-swap's side is engines/lab-engine):
  POST /run/<name>    start lab-engine-<name>, then stream its log for as long as the caller stays connected. When the
                      caller goes away (llama-swap stopped the model, restarted, or the client was killed) or the
                      container exits, the container is stopped and removed.
  POST /stop/<name>   stop and remove it; answers only when it is gone (llama-swap's cmdStop).
  GET  /status        the running engine containers (JSON).

Engines run as containers named lab-engine-<name>, labelled lab.engine=<name>, on the Docker network lab-engines
(created --internal: no internet, no published ports). llama-swap joins that network and proxies to the name.
Every engine gets: --init, --cap-drop ALL, no-new-privileges, the allowlist's devices and read-only mounts only
(plus, per engine: groups, env, user, --ipc, --ulimit, a seccomp profile from engines/, a stop timeout).
On start the service removes leftover lab.engine containers; on SIGTERM it stops the ones it runs.

Good result: `lab-engined.py check` prints one docker command per engine; `curl --unix-socket <sock> http://x/status`
answers [] when nothing runs.
"""
import http.server, json, os, re, select, signal, socket, socketserver, subprocess, sys, threading, time

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC = os.environ.get("LAB_ENGINES", os.path.join(HERE, f"{socket.gethostname()}.json"))
# A folder in the home directory, not /run: llama-swap bind-mounts the folder (as /run/lab-engined), and a folder
# that outlives restarts keeps that mount valid when the launcher makes a new socket.
SOCK = os.environ.get("LAB_ENGINED_SOCK", os.path.expanduser("~/.local/state/lab-engined/engined.sock"))
NET, PREFIX, LABEL = "lab-engines", "lab-engine-", "lab.engine"
NAME = re.compile(r"^[a-z0-9][a-z0-9-]{0,40}$")
STOP_WAIT = 30  # seconds docker stop waits before killing (default; an engine can set "stop_timeout")

running = {}  # name -> True while a /run request holds it
lock = threading.Lock()


def log(msg):
    print(time.strftime("%Y-%m-%d %H:%M:%S"), msg, flush=True)


def docker(*args, timeout=120):
    """Never raises on a slow Docker (it can take long while a big model loads): a timeout comes back as code 124."""
    try:
        return subprocess.run(["docker", *args], capture_output=True, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        log(f"docker {args[0]} timed out after {timeout} s")
        return subprocess.CompletedProcess(["docker", *args], 124, "", "timed out")


def load_spec():
    with open(SPEC) as f:
        spec = json.load(f)
    return spec.get("engines", {})


def run_args(name, e):
    """The docker run command for one allowlisted engine. Everything comes from the allowlist, nothing from the caller."""
    image = e["image"]
    if "@sha256:" not in image:
        raise ValueError(f"{name}: image must be pinned by digest")
    a = ["run", "-d", "--rm", "--init", "--name", PREFIX + name, "--label", f"{LABEL}={name}", "--network", NET,
         "--cap-drop", "ALL", "--security-opt", "no-new-privileges"]
    for d in e.get("devices", []):
        a += ["--device", d]
    for g in e.get("groups", []):
        a += ["--group-add", str(g)]
    for v in e.get("volumes", []):
        if not v.endswith(":ro"):
            raise ValueError(f"{name}: mount {v} must be read-only (:ro)")
        a += ["-v", v]
    for k, v in e.get("env", {}).items():
        a += ["-e", f"{k}={v}"]
    if e.get("user"):
        a += ["--user", str(e["user"])]
    if e.get("shm_size"):
        a += ["--shm-size", str(e["shm_size"])]
    if e.get("read_only"):
        a += ["--read-only"]
    if e.get("ipc"):  # e.g. "host": Halogen Flash asks for it (shares the host's IPC namespace)
        a += ["--ipc", str(e["ipc"])]
    for u in e.get("ulimits", []):  # e.g. "memlock=-1:-1"
        a += ["--ulimit", str(u)]
    if e.get("seccomp"):  # a seccomp profile file in engines/ (e.g. Radiance's: Docker's default + io_uring, NUMA)
        f = os.path.join(HERE, os.path.basename(e["seccomp"]))
        if not os.path.isfile(f):
            raise ValueError(f"{name}: seccomp profile {f} not found")
        a += ["--security-opt", f"seccomp={f}"]
    if e.get("stop_timeout"):
        a += ["--stop-timeout", str(int(e["stop_timeout"]))]
    for t in e.get("tmpfs", []):
        a += ["--tmpfs", t]
    if e.get("entrypoint"):
        a += ["--entrypoint", e["entrypoint"]]
    return a + [image] + [str(x) for x in e.get("args", [])]


def exists(cname):
    """True if the container exists. A timed-out answer counts as "exists" (the safe guess when stopping)."""
    r = docker("inspect", "-f", "{{.State.Status}}", cname, timeout=60)
    return r.returncode in (0, 124)


def stop_wait(name):
    """The engine's own stop timeout from the allowlist, else STOP_WAIT (also for a name no longer listed)."""
    try:
        return int(load_spec().get(name, {}).get("stop_timeout", STOP_WAIT))
    except Exception:
        return STOP_WAIT


def stop(name):
    """Stop and remove lab-engine-<name>; return when it is gone."""
    cname = PREFIX + name
    wait = stop_wait(name)
    if exists(cname):
        docker("stop", "-t", str(wait), cname, timeout=wait + 60)
    end = time.time() + 180
    while time.time() < end:
        if not exists(cname):
            return True
        docker("rm", "-f", cname, timeout=60)
        time.sleep(1)
    return False


def leftovers():
    out = docker("ps", "-aq", "--filter", f"label={LABEL}", timeout=30).stdout.split()
    if out:
        docker("rm", "-f", *out, timeout=120)
        log(f"removed {len(out)} leftover engine container(s)")


def ensure_network():
    if docker("network", "inspect", NET, timeout=30).returncode != 0:
        r = docker("network", "create", "--internal", NET, timeout=60)
        log(f"network {NET} created (--internal)" if r.returncode == 0 else f"network {NET}: {r.stderr.strip()}")


def client_gone(conn):
    """True when the caller closed its end (readable with no data)."""
    r, _, _ = select.select([conn], [], [], 0)
    if not r:
        return False
    try:
        return conn.recv(1, socket.MSG_PEEK) == b""
    except OSError:
        return True


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.0"  # the streamed log ends when the connection closes

    def log_message(self, fmt, *args):  # the client address of a Unix socket is empty
        log("http " + (fmt % args))

    def reply(self, code, body):
        b = (body if body.endswith("\n") else body + "\n").encode()
        self.send_response(code)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def target(self, verb):
        parts = self.path.strip("/").split("/")
        if len(parts) != 2 or parts[0] != verb or not NAME.match(parts[1]):
            return None, "bad path"
        try:
            spec = load_spec()
        except Exception as e:
            return None, f"allowlist unreadable: {e}"
        if parts[1] not in spec:
            return None, f"{parts[1]}: not in the allowlist"
        return (parts[1], spec[parts[1]]), None

    def do_GET(self):
        if self.path.rstrip("/") != "/status":
            return self.reply(404, "unknown")
        out = docker("ps", "--filter", f"label={LABEL}", "--format", "{{json .}}", timeout=30).stdout
        rows = [json.loads(l) for l in out.splitlines() if l.strip()]
        self.reply(200, json.dumps([{"name": r.get("Names"), "status": r.get("Status"), "image": r.get("Image")}
                                    for r in rows]))

    def do_POST(self):
        verb = self.path.strip("/").split("/")[0]
        if verb == "stop":
            # Stopping is always allowed for a valid name, also one no longer in the allowlist (removing an engine
            # from the allowlist and llama-swap's config at the same time must not leave it running: 2026-11-17).
            parts = self.path.strip("/").split("/")
            if len(parts) != 2 or not NAME.match(parts[1]):
                return self.reply(400, "bad path")
            ok = stop(parts[1])
            log(f"stop {parts[1]}: {'gone' if ok else 'STILL THERE'}")
            return self.reply(200 if ok else 500, "stopped" if ok else "container still there")
        if verb != "run":
            return self.reply(404, "unknown")
        t, err = self.target("run")
        if err:
            log(f"refused run: {err}")
            return self.reply(404 if "allowlist" in err else 400, err)
        name, e = t
        with lock:
            if running.get(name):
                log(f"refused run {name}: already running")
                return self.reply(409, f"{name}: already running")
            running[name] = True
        try:
            self.run_engine(name, e)
        except Exception as err:  # a broken caller connection or a Docker error: the cleanup below still runs
            log(f"run {name}: {type(err).__name__}: {err}")
        finally:
            ok = False
            try:
                ok = stop(name)
            except Exception as err:
                log(f"run {name}: stop failed: {type(err).__name__}: {err}")
            with lock:
                running.pop(name, None)
            log(f"run {name} ended; container {'gone' if ok else 'STILL THERE'}")

    def run_engine(self, name, e):
        cname = PREFIX + name
        if exists(cname):  # left over from a crash: never two of the same
            stop(name)
        try:
            args = run_args(name, e)
        except ValueError as err:
            return self.reply(400, str(err))
        r = docker(*args, timeout=600)
        if r.returncode != 0:
            log(f"run {name}: docker run failed: {r.stderr.strip()[:300]}")
            return self.reply(500, f"docker run failed: {r.stderr.strip()[:300]}")
        log(f"run {name}: started {cname}")
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.end_headers()
        logs = subprocess.Popen(["docker", "logs", "-f", cname], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        conn = self.connection
        try:
            while True:
                if client_gone(conn):
                    log(f"run {name}: caller went away")
                    return
                r, _, _ = select.select([logs.stdout], [], [], 1.0)
                if r:
                    line = logs.stdout.readline()
                    if not line:  # docker logs ended: the container exited
                        log(f"run {name}: container exited")
                        return
                    try:
                        self.wfile.write(line)
                        self.wfile.flush()
                    except OSError:
                        log(f"run {name}: caller went away (write)")
                        return
        finally:
            logs.kill()


class Server(socketserver.ThreadingMixIn, socketserver.UnixStreamServer):
    daemon_threads = True


def serve():
    ensure_network()
    leftovers()
    d = os.path.dirname(SOCK)
    os.makedirs(d, exist_ok=True)
    if os.path.exists(SOCK):
        os.unlink(SOCK)
    srv = Server(SOCK, Handler)
    os.chmod(SOCK, 0o660)

    def bye(*_):
        log("stopping: engines down")
        threading.Thread(target=srv.shutdown, daemon=True).start()
    signal.signal(signal.SIGTERM, bye)
    signal.signal(signal.SIGINT, bye)
    log(f"lab-engined: socket {SOCK}, allowlist {SPEC} ({', '.join(load_spec()) or 'empty'})")
    srv.serve_forever()
    leftovers()
    try:
        os.unlink(SOCK)
    except OSError:
        pass


def check():
    for name, e in load_spec().items():
        if not NAME.match(name):
            print(f"{name}: bad name"); continue
        try:
            print(f"{name}: docker {' '.join(run_args(name, e))}")
        except (KeyError, ValueError) as err:
            print(f"{name}: INVALID: {err}")


if __name__ == "__main__":
    {"serve": serve, "check": check}.get(sys.argv[1] if len(sys.argv) > 1 else "", lambda: sys.exit(__doc__))()
