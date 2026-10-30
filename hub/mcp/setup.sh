#!/usr/bin/env bash
# Set up the MCP gateway (MCPJungle) after `docker compose up -d mcpjungle mcp-searxng`. Safe to run again.
#   1. installs the mcpjungle CLI (pinned, checksum checked) to ~/.local/bin
#   2. creates the admin account once (token in ~/.mcpjungle.conf, mode 600; copy it to Proton Pass "MCPJungle admin")
#   3. registers every tool server in mcp/servers/*.json
#   4. creates every agent login in mcp/clients/*.json (tokens come from hub/.env, MCP_TOKEN_<NAME>; never printed)
# Add a tool: a servers/<name>.json, its container in compose.yaml, its name in the clients' allowed_servers.
set -euo pipefail
V=0.4.6   # same version as the mcpjungle image in compose.yaml
HUB=$(cd "$(dirname "$0")/.." && pwd)
REG=http://127.0.0.1:8085
MJ=~/.local/bin/mcpjungle
cd "$HUB"

if ! "$MJ" version 2>/dev/null | grep -q "v$V"; then
  t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
  base=https://github.com/mcpjungle/MCPJungle/releases/download/$V
  curl -sSfL "$base/mcpjungle_Linux_x86_64.tar.gz" -o "$t/mcpjungle_Linux_x86_64.tar.gz"
  curl -sSfL "$base/checksums.txt" -o "$t/checksums.txt"
  (cd "$t" && grep ' mcpjungle_Linux_x86_64.tar.gz$' checksums.txt | sha256sum -c -)
  tar -xzf "$t/mcpjungle_Linux_x86_64.tar.gz" -C "$t" mcpjungle && install -m 755 "$t/mcpjungle" "$MJ"
fi
echo "== CLI $("$MJ" version 2>/dev/null | grep -m1 -o 'v[0-9.]*')"

curl -sf "$REG/health" >/dev/null || { echo "MCPJungle not answering on $REG: docker compose up -d mcpjungle"; exit 1; }
if [ ! -s ~/.mcpjungle.conf ]; then
  (umask 077; "$MJ" --registry "$REG" init-server >/dev/null)
  echo "== admin created; token in ~/.mcpjungle.conf: copy it to Proton Pass (entry 'MCPJungle admin')"
fi

set -a; . ./.env; set +a   # tokens for tool servers and clients
for f in mcp/servers/*.json; do
  "$MJ" register --force -c "$f" >/dev/null 2>&1 && echo "== tool server: $(basename "$f" .json)"
done
for f in mcp/clients/*.json; do
  n=$(basename "$f" .json)
  "$MJ" delete mcp-client "$n" >/dev/null 2>&1 || true
  # mcpjungle prints its result on stderr; without 2>&1 the grep found nothing and pipefail stopped the loop after the
  # first client (found 2026-10-30: enso and openwebui were never (re)created).
  out=$("$MJ" create mcp-client --conf "$f" 2>&1) || { echo "== client $n: FAILED"; echo "$out" | grep -iv token; exit 1; }
  echo "$out" | grep -i 'servers accessible' | sed "s/^/== client $n: /"
done
