#!/usr/bin/env bash
# Per-model settings in Open WebUI, from hub/openwebui/models.yaml + the llama-swap configs.
#   hub/openwebui/models.sh           show what differs (read-only)
#   hub/openwebui/models.sh --apply   write them (Open WebUI admin API "models/import": updates existing entries)
# Needs OPENWEBUI_API_KEY in hub/.env. Model entry format from Open WebUI v0.12.4 (backend/open_webui/models/models.py).
set -euo pipefail
HUB=$(cd "$(dirname "$0")/.." && pwd); cd "$HUB"
set -a; . ./.env; set +a
: "${OPENWEBUI_API_KEY:?add OPENWEBUI_API_KEY to hub/.env}"
APPLY=no; [ "${1:-}" = --apply ] && APPLY=yes
APPLY=$APPLY python3 - <<'PY'
import json, os, urllib.request, yaml
BASE = "http://127.0.0.1:3000/api/v1/models"
H = {"Authorization": "Bearer " + os.environ["OPENWEBUI_API_KEY"], "Content-Type": "application/json"}
def call(path, body=None):
    req = urllib.request.Request(BASE + path, method="POST" if body is not None else "GET", headers=H,
                                 data=json.dumps(body).encode() if body is not None else None)
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.load(r)
want_cfg = yaml.safe_load(open("openwebui/models.yaml"))
models = {}
for machine, path in (("zaphod", "../zaphod/llama-swap/config/config.yaml"), ("marvin", "../marvin/llama-swap/config/config.yaml")):
    for mid, m in (yaml.safe_load(open(path)).get("models") or {}).items():
        models[mid if machine == "zaphod" else f"marvin/{mid}"] = m.get("name") or mid
CAPS = ["file_context", "vision", "file_upload", "web_search", "image_generation", "code_interpreter",
        "terminal", "citations", "status_updates", "memory", "builtin_tools"]
current = {m["id"]: m for m in call("/base")}   # settings entries for llama-swap models ("/export" skips these)
entries, changes = [], []
for mid, name in models.items():
    caps = {c: True for c in CAPS}
    caps["vision"] = mid in want_cfg["vision"]
    active = mid not in want_cfg["hidden"]
    cur = current.get(mid)
    meta = dict((cur or {}).get("meta") or {})            # keep other fields (description, images) as they are
    meta["capabilities"] = caps
    params = dict((cur or {}).get("params") or {})
    params["function_calling"] = "native"
    e = {"id": mid, "base_model_id": None, "name": name, "meta": meta, "params": params, "is_active": active}
    entries.append(e)
    if not cur:
        changes.append(f"  {mid:12} new entry (vision {caps['vision']}, {'shown' if active else 'hidden'})")
    else:
        diff = []
        if cur.get("name") != name: diff.append(f"name '{cur.get('name')}' -> '{name}'")
        if (cur.get("params") or {}).get("function_calling") != "native": diff.append("function calling -> native")
        if ((cur.get("meta") or {}).get("capabilities") or {}).get("vision") != caps["vision"]: diff.append(f"vision -> {caps['vision']}")
        if bool(cur.get("is_active", True)) != active: diff.append("shown" if active else "hidden")
        if diff: changes.append(f"  {mid:12} " + ", ".join(diff))
print(f"{len(changes)} model(s) differ from models.yaml" + (":" if changes else ""))
print("\n".join(changes))
if os.environ["APPLY"] == "yes" and changes:
    ok = call("/import", {"models": entries})
    print("applied" if ok else "import returned false")
PY
