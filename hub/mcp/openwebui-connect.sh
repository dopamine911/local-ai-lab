#!/usr/bin/env bash
# Add (or update) the MCP gateway as a tool connection in Open WebUI, through Open WebUI's admin API. Safe to run again:
# replaces only the connection with id "homelab-tools", keeps any others.
#
# Needs in hub/.env (both also in Proton Pass):
#   OPENWEBUI_API_KEY    an API key of your Open WebUI ADMIN account (Settings > Account > API keys; the admin switch
#                        "Admin Panel > Settings > General > Enable API Key" must be on)
#   MCP_TOKEN_OPENWEBUI  Open WebUI's gateway token (made by hand once; see hub/.env.example)
# Format follows Open WebUI v0.12.4 (backend routers/configs.py, AddToolServerModal.svelte). Re-check after upgrading.
set -euo pipefail
cd "$(dirname "$0")/.."
set -a; . ./.env; set +a
: "${OPENWEBUI_API_KEY:?add OPENWEBUI_API_KEY to hub/.env (see the top of this script)}"
: "${MCP_TOKEN_OPENWEBUI:?missing MCP_TOKEN_OPENWEBUI in hub/.env}"
python3 - <<'EOF'
import json, os, urllib.request
BASE = "http://127.0.0.1:3000/api/v1/configs/tool_servers"
H = {"Authorization": "Bearer " + os.environ["OPENWEBUI_API_KEY"], "Content-Type": "application/json"}

def call(method, body=None):
    req = urllib.request.Request(BASE, method=method, headers=H, data=json.dumps(body).encode() if body else None)
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.load(r)

conns = call("GET")["TOOL_SERVER_CONNECTIONS"] or []
ours = {
    "type": "mcp",
    "url": "http://127.0.0.1:8085/mcp",          # MCPJungle on zaphod (Open WebUI uses the host network)
    "spec_type": "url", "spec": "", "path": "openapi.json",
    "auth_type": "bearer",
    "key": os.environ["MCP_TOKEN_OPENWEBUI"],
    "config": {"enable": True, "function_name_filter_list": "", "access_grants": []},
    "info": {"id": "homelab-tools", "name": "Homelab tools (MCPJungle)",
             "description": "Agent tools from zaphod's MCP gateway: web search via SearXNG, reading public pages"},
}
others = [c for c in conns if (c.get("info") or {}).get("id") != "homelab-tools"]
saved = call("POST", {"TOOL_SERVER_CONNECTIONS": others + [ours]})["TOOL_SERVER_CONNECTIONS"]
mine = [c for c in saved if (c.get("info") or {}).get("id") == "homelab-tools"]
print(f"Open WebUI tool connections: {len(saved)} ({len(others)} others kept); homelab-tools "
      f"{'saved' if mine else 'NOT saved'}, type {mine[0]['type'] if mine else '-'}, url {mine[0]['url'] if mine else '-'}")
EOF
