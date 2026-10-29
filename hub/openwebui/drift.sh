#!/usr/bin/env bash
# Compare the running Open WebUI's settings with what compose.yaml would give a fresh install.
#   hub/openwebui/drift.sh          show differences (read-only)
#   hub/openwebui/drift.sh --sync   make the running Open WebUI match compose.yaml for the settings compose.yaml
#                                   manages (managed-keys.txt). Stops Open WebUI for a few seconds; saves a copy of
#                                   its database first (data/open-webui/webui.db.before-sync-<date>).
# How: starts a throwaway Open WebUI (same image, same environment, no network) on an empty Docker volume, reads
# the settings it writes, compares them with the live database, then deletes the throwaway container and volume.
# Secrets are never printed: settings whose name holds key/secret/token/password show as <set>.
# Output: "in compose" = managed here; "web page only" = changed by hand, not in compose.yaml: add it there or undo it.
set -euo pipefail
HUB=$(cd "$(dirname "$0")/.." && pwd); cd "$HUB"
SYNC=no; [ "${1:-}" = --sync ] && SYNC=yes
IMG=$(docker compose config --format json | python3 -c "import json,sys;print(json.load(sys.stdin)['services']['open-webui']['image'])")
ENVF=$(mktemp); chmod 600 "$ENVF"; VOL=owui-drift-$$
cleanup() { docker rm -f "$VOL" >/dev/null 2>&1 || true; docker volume rm "$VOL" >/dev/null 2>&1 || true; rm -f "$ENVF"; }
trap cleanup EXIT
# The exact environment compose.yaml gives open-webui (secrets included, so the comparison is exact; file mode 600)
docker compose config --format json | python3 -c "
import json,sys
for k,v in json.load(sys.stdin)['services']['open-webui']['environment'].items():
    if k not in ('PORT','HOST'): print(f'{k}={v}'.replace(chr(10),' '))" > "$ENVF"

echo "== building a fresh Open WebUI from compose.yaml ($IMG, no network, about a minute)"
docker volume create "$VOL" >/dev/null
docker run -d --name "$VOL" --network none --env-file "$ENVF" -e PORT=3099 -v "$VOL":/app/backend/data "$IMG" >/dev/null
n=0
for i in $(seq 1 60); do
  n=$(docker exec "$VOL" python3 -c "import sqlite3;print(sqlite3.connect('file:/app/backend/data/webui.db?mode=ro',uri=True).execute('select count(*) from config').fetchone()[0])" 2>/dev/null || echo 0)
  [ "$n" -gt 100 ] && break; sleep 5
done
[ "$n" -gt 100 ] || { echo "fresh Open WebUI did not write its settings"; exit 1; }
sleep 5; docker stop "$VOL" >/dev/null

[ $SYNC = yes ] && { echo "== stopping open-webui"; docker compose stop open-webui >/dev/null; }
docker run --rm -i --network none -e SYNC=$SYNC -e STAMP="$(date +%F-%H%M)" \
  -v "$HUB/data/open-webui":/live -v "$VOL":/fresh:ro -v "$HUB/openwebui/managed-keys.txt":/managed.txt:ro \
  python:3.13-slim-trixie python3 - <<'EOF'
import json, os, re, shutil, sqlite3
def load(p):
    c = sqlite3.connect(f"file:{p}?mode=ro", uri=True)
    rows = {}
    for k, v in c.execute("select key, value from config"):
        try: rows[k] = json.loads(v) if isinstance(v, (str, bytes)) else v
        except Exception: rows[k] = v
    c.close(); return rows
def norm(v):  # treat empty-ish lists and "300" vs 300 as equal
    if v in ([""], []): return []
    if isinstance(v, str) and v.isdigit(): return int(v)
    return v
live, fresh = load("/live/webui.db"), load("/fresh/webui.db")
managed = [l.strip() for l in open("/managed.txt") if l.strip() and not l.startswith("#")]
SEC = re.compile(r"(key|secret|token|password|passwd)", re.I)
def show(k, v):
    if SEC.search(k) and v not in (None, "", [], False, 0): return "<set>"
    return re.sub(r'"(key|api_key|token|secret|password|auth)": ?"[^"]+"', r'"\1": "<set>"', json.dumps(v))[:160]
diff = sorted(k for k in set(live) | set(fresh) if norm(live.get(k)) != norm(fresh.get(k)))
print(f"{len(diff)} setting(s) differ between the running Open WebUI and compose.yaml")
for k in diff:
    print(f"  {'in compose   ' if k in managed else 'web page only'}  {k}\n      running: {show(k, live.get(k))}\n      compose: {show(k, fresh.get(k))}")
if os.environ["SYNC"] == "yes":
    todo = [k for k in diff if k in managed]
    if todo:
        shutil.copy2("/live/webui.db", f"/live/webui.db.before-sync-{os.environ['STAMP']}")
        c = sqlite3.connect("/live/webui.db")
        for k in todo:
            c.execute("update config set value = ? where key = ?", (json.dumps(fresh[k]), k))
        c.commit(); c.close()
    print(f"== synced {len(todo)} managed setting(s); 'web page only' ones were left alone")
EOF
[ $SYNC = yes ] && { docker compose start open-webui >/dev/null; echo "== open-webui started"; }
exit 0
