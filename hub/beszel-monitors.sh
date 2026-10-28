#!/usr/bin/env bash
# Add zaphod's network monitors to Beszel (safe to run again: existing targets are skipped).
# Asks for your Beszel login; the password is only sent to the local hub.
set -euo pipefail
HUB=http://127.0.0.1:8090
SYSTEM_NAME=zaphod

# target | protocol | port (TCP only, else empty) | interval seconds
MONITORS='
http://127.0.0.1:8080/health|http||60
http://127.0.0.1:3000/health|http||60
http://127.0.0.1:5000/health|http||60
http://127.0.0.1:8888/healthz|http||60
http://127.0.0.1:8000/health|http||60
https://chat.ai.dopamine911.com/health|http||60
1.1.1.1|icmp||60
'

read -rp "Beszel email: " EMAIL
read -rsp "Beszel password: " PASS; echo
TOKEN=$(EMAIL="$EMAIL" PASS="$PASS" python3 -c '
import json, os, sys, urllib.request
req = urllib.request.Request(sys.argv[1] + "/api/collections/users/auth-with-password",
    data=json.dumps({"identity": os.environ["EMAIL"], "password": os.environ["PASS"]}).encode(),
    headers={"Content-Type": "application/json"})
print(json.load(urllib.request.urlopen(req))["token"])' "$HUB") || { echo "Login failed."; exit 1; }
unset PASS

TOKEN="$TOKEN" MONITORS="$MONITORS" python3 - "$HUB" "$SYSTEM_NAME" <<'EOF'
import json, os, sys, urllib.parse, urllib.request
hub, name = sys.argv[1], sys.argv[2]
H = {"Authorization": os.environ["TOKEN"], "Content-Type": "application/json"}

def call(method, path, body=None):
    req = urllib.request.Request(hub + path, method=method, headers=H,
                                 data=json.dumps(body).encode() if body else None)
    return json.load(urllib.request.urlopen(req))

q = urllib.parse.quote(f'name="{name}"')
system = call("GET", f"/api/collections/systems/records?filter={q}")["items"][0]["id"]
existing = {m["target"] for m in call("GET", "/api/collections/network_monitors/records?perPage=200")["items"]}

for line in os.environ["MONITORS"].split():
    target, proto, port, interval = line.split("|")
    if target in existing:
        print(f"already there  {target}")
        continue
    body = {"system": system, "target": target, "protocol": proto,
            "interval": int(interval), "enabled": True}
    if port:
        body["port"] = int(port)
    call("POST", "/api/collections/network_monitors/records", body)
    print(f"added          {target} ({proto})")
EOF
