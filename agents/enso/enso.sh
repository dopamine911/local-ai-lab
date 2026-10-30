#!/usr/bin/env bash
# Enso in a container on zaphod. Plan: plans/agent-infra.md step 1. Details: agents/enso/README.md.
#
#   ./enso.sh build     build the image (network at build time only)
#   ./enso.sh setup     make ~/agents/enso-home (keys, pi config) and ~/agents/work (a clone of enso-harness); safe to rerun
#   ./enso.sh up        start Enso and the relay; web page on http://localhost:5178 (zaphod only)
#   ./enso.sh check     models and tools reachable from inside, internet not; good result: three "ok" lines
#   ./enso.sh tui [--model lab/coder]   Enso's terminal on the workspace, with the lab's MCP tools
#   ./enso.sh shell     a shell inside the container
#   ./enso.sh down      stop both
#
# Keys are copied from zaphod/llama-swap/.env (ZAPHOD_KEY) and hub/.env (MCP_TOKEN_ENSO) into
# ~/agents/enso-home/keys/ (mode 600); never printed. Enso's guards refuse reads of ENSO_HOME, so the agent can't
# read them through its tools.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
LAB=$(cd "$HERE/../.." && pwd)
EHOME=$HOME/agents/enso-home
WORK=$HOME/agents/work
IMG=enso-agent:537559a
# Enso denies every MCP server unless one is approved by name (core/src/permission-modes.ts, MCP_DENIED); approve
# ours. permissions.json is seeded by Enso's first start, so this runs after `up` too; it adds one key, keeps the rest.
approve_mcp() {
  [ -f "$EHOME/permissions.json" ] || return 0
  python3 -I -c 'import json,sys; p=sys.argv[1]; j=json.load(open(p)); j["permission"].setdefault("mcp", {})["lab"]="allow"; open(p,"w").write(json.dumps(j, indent=2)+"\n")' "$EHOME/permissions.json"
}
dc() { docker compose -f "$HERE/compose.yaml" "$@"; }

case "${1:-}" in
build)
  docker buildx build --load -t "$IMG" "$HERE" ;;
setup)
  mkdir -p "$EHOME/keys" "$EHOME/pi-agent" "$WORK"
  chmod 700 "$EHOME" "$EHOME/keys"
  ( umask 077
    grep '^ZAPHOD_KEY=' "$LAB/zaphod/llama-swap/.env" | cut -d= -f2- | tr -d '\n' > "$EHOME/keys/zaphod-key"
    grep '^MCP_TOKEN_ENSO=' "$LAB/hub/.env" | cut -d= -f2- | tr -d '\n' > "$EHOME/keys/mcp-token" )
  [ -s "$EHOME/keys/zaphod-key" ] && [ -s "$EHOME/keys/mcp-token" ] || { echo "a key is missing (see README)"; exit 1; }
  # pi's models and MCP servers, Enso's config: from the repo, overwritten on every setup (edit them in the repo)
  install -m 600 "$HERE/config/models.json" "$EHOME/pi-agent/models.json"
  install -m 600 "$HERE/config/mcp.json" "$EHOME/pi-agent/mcp.json"
  install -m 600 "$HERE/config/config.json" "$EHOME/config.json"
  approve_mcp
  # The workspace: its own clone, so the agent's commits never touch Daniel's checkout. No push credentials inside.
  [ -d "$WORK/enso-harness/.git" ] || git clone -q https://github.com/dopamine911/enso-harness.git "$WORK/enso-harness"
  git -C "$WORK/enso-harness" config user.name "enso-agent (local model)"
  git -C "$WORK/enso-harness" config user.email "enso-agent@zaphod.invalid"
  echo "setup done: $EHOME (keys: $(ls "$EHOME/keys" | tr '\n' ' ')), $WORK/enso-harness at $(git -C "$WORK/enso-harness" log -1 --format=%h)" ;;
up)
  dc up -d
  for i in $(seq 1 60); do curl -sf -m 3 -o /dev/null http://127.0.0.1:5178/ && break; sleep 2; done
  approve_mcp
  curl -sf -m 3 -o /dev/null http://127.0.0.1:5178/ && echo "Enso: http://localhost:5178" || { echo "page not answering yet:"; docker logs --tail 20 enso; } ;;
check)
  docker exec enso node -e '
    const fs = require("fs"); const key = fs.readFileSync(process.env.HOME + "/.enso/keys/zaphod-key", "utf8");
    const t = (u, h) => fetch(u, { headers: h, signal: AbortSignal.timeout(5000) });
    (async () => {
      try { const r = await t("http://llama-swap:8080/v1/models", { Authorization: "Bearer " + key });
            console.log(r.ok ? "ok: models (llama-swap)" : "FAIL: llama-swap " + r.status); } catch (e) { console.log("FAIL: llama-swap " + e.message); }
      try { await t("https://example.com"); console.log("FAIL: internet reachable"); } catch { console.log("ok: no internet"); }
    })()'
  docker exec -e PI_CODING_AGENT_DIR=/home/ubuntu/.enso/pi-agent enso pi mcp list >/tmp/enso-mcp.txt 2>&1 \
    && echo "ok: tools (mcpjungle): $(grep -oE 'connected, [0-9]+ tools' /tmp/enso-mcp.txt)" || { echo "FAIL: mcp"; cat /tmp/enso-mcp.txt; } ;;
tui)
  # Enso's terminal in the workspace. Enso starts pi with --no-extensions, which also drops pi's built-in MCP;
  # -e loads just MCP and tool search back (pi docs/cli.md). Extra arguments go to pi, e.g. --model lab/coder.
  shift; docker exec -it -w /work/enso-harness enso bun /opt/enso/scripts/enso.ts -e builtin:mcp -e builtin:tool-search "$@" ;;
shell)
  docker exec -it enso bash ;;
down)
  dc down ;;
*) sed -n 2,12p "$0"; exit 1 ;;
esac
