"""Upload the lab's Markdown docs to LocalRecall (collection `homelab`), changed files only. Runs in the memory-sync
container (hub/compose.yaml, profile sync) with the repos mounted read-only at /src/<repo>; state in /state.
Good result: a line per change, then `sync: N uploaded, M removed, K unchanged`.
A file is named by its path, `<repo>__<dir>__<file>.md`, so two README.md files don't collide and a search result
says where it came from."""
import hashlib, json, os, sys, time, urllib.request, urllib.error, uuid
from pathlib import Path

URL = "http://localrecall:8080/api/collections"
COL = "homelab"
KEY = os.environ["LOCALRECALL_API_KEY"]
STATE = Path("/state/sync.json")
SKIP_DIRS = {".git", "node_modules", "data", "jobs", ".runner", "results", ".codegraph", "__pycache__"}
MAX_BYTES = 512 * 1024
# Never in agents' memory: raw tool-test transcripts (they hold the prompt-injection test cases' attacker text) and
# benchmark tasks (an agent that later runs the benchmark would have the tasks in memory).
SKIP_PATHS = ("homelab/tuning/tool-eval", "homelab/tuning/terminal-bench/tasks")


def call(method, path, body=None, ctype="application/json"):
    req = urllib.request.Request(URL + path, data=body, method=method,
                                 headers={"Authorization": f"Bearer {KEY}", "Content-Type": ctype})
    with urllib.request.urlopen(req, timeout=600) as r:
        return r.read()


def files():
    for repo in sorted(Path("/src").iterdir()):
        for root, dirs, names in os.walk(repo):
            dirs[:] = sorted(d for d in dirs if d not in SKIP_DIRS and not d.startswith("."))
            for n in sorted(names):
                p = Path(root, n)
                rel = p.relative_to(Path("/src"))
                if str(rel).startswith(SKIP_PATHS):
                    continue
                if p.suffix == ".md" and p.stat().st_size <= MAX_BYTES:
                    yield "__".join(rel.parts), p


def upload(name, data):
    b = uuid.uuid4().hex
    body = (f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{name}"\r\n'
            f"Content-Type: text/markdown\r\n\r\n").encode() + data + f"\r\n--{b}--\r\n".encode()
    call("POST", f"/{COL}/upload", body, f"multipart/form-data; boundary={b}")


def delete(name):
    try:
        call("DELETE", f"/{COL}/entry/delete", json.dumps({"entry": name}).encode())
    except urllib.error.HTTPError as e:
        if e.code != 404:
            raise


def main():
    for i in range(30):  # LocalRecall may still be starting
        try:
            cols = json.loads(call("GET", ""))
            break
        except (urllib.error.URLError, ConnectionError):
            time.sleep(2)
    else:
        sys.exit("sync: LocalRecall not answering")
    names = (cols.get("data") or {}).get("collections", []) if isinstance(cols, dict) else cols
    if COL not in names:
        call("POST", "", json.dumps({"name": COL}).encode())
        print(f"sync: created collection {COL}")
    state = json.loads(STATE.read_text()) if STATE.exists() else {}
    seen, up, same = set(), 0, 0
    for name, p in files():
        seen.add(name)
        data = p.read_bytes()
        h = hashlib.sha256(data).hexdigest()
        if state.get(name) == h:
            same += 1
            continue
        if name in state:
            delete(name)
        upload(name, data)
        state[name] = h
        up += 1
        print(f"sync: uploaded {name}", flush=True)
        STATE.write_text(json.dumps(state, indent=0, sort_keys=True))
    gone = [n for n in state if n not in seen]
    for n in gone:
        delete(n)
        del state[n]
        print(f"sync: removed {n}")
    STATE.write_text(json.dumps(state, indent=0, sort_keys=True))
    print(f"sync: {up} uploaded, {len(gone)} removed, {same} unchanged")


main()
