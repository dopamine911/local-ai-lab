#!/usr/bin/env bash
# Strata (upstream, 0.2.7, gfx1151 image) on marvin, sealed off from the network. Plan: plans/strix-strata.md, Phase 1.
# Runs ON marvin. Copy: scp marvin/strata-test/strata-test.sh marvin:strata-test/
#
#   ./strata-test.sh plan     dry run: what Strata would do with these settings (no GPU, no network, llama-swap untouched)
#   ./strata-test.sh start    stop llama-swap, start Strata sealed, wait until it answers (first start builds its pack)
#   ./strata-test.sh status   is it up; memory; can it reach the internet (must say NO)
#   ./strata-test.sh stop     stop Strata, start llama-swap again, wait for its health check   <- always end with this
#   PRESET=x PARALLEL=1 STRATA_SET="prefill=16384" ./strata-test.sh restart
#                             swap Strata for one with changed settings (llama-swap stays stopped). One change per
#                             experiment; each experiment gets its own PRESET name so nothing carries over from the last.
#
# Sealed: Strata sits only on an internal Docker network (no route out). A small relay (python, already on marvin) sits on
# that network and a normal one, and publishes Strata's API on marvin's Tailscale address :8099 only.
# Same model file as llama.cpp's `big` (the 3 UD-IQ4_XS shards, read-only). The MTP pieces were copied from zaphod's
# Strata volume beforehand (sha256 checked), so Strata has nothing to download.
# API key: made per start, kept in ~/strata-test/key (mode 600); never printed.
# Version exception (Daniel, 2026-10-29): image 0.2.7-rocm-gfx1151 was 2 days old; allowed for this test only, pinned by digest.
set -uo pipefail
IMG=ghcr.io/niko1221/strata@sha256:29ecd5cf8b0e21525dbb5c0191d7743d62fa356da646cb6aaf0546e5a31a8b3f   # 0.2.7-rocm-gfx1151
RELAY_IMG=python:3.13-slim-trixie
M=/srv/models; F=Qwen3.8-Flash-Next-UD-IQ4_XS
DIR=$HOME/strata-test; KEYF=$DIR/key; NET=strata-internal; PORT=8099
LS=$HOME/homelab/marvin/llama-swap
TS=$(tailscale ip -4)
mkdir -p "$DIR"; chmod 700 "$DIR"
log() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$DIR/log.txt"; }
mem() { echo "GPU memory used $(( $(cat /sys/class/drm/card*/device/mem_info_gtt_used | head -1) / 1073741824 )) GB (GTT); RAM: $(free -g | awk 'NR==2{print $3" GB used, "$7" GB available"}')"; }

SETTINGS=(-e MODEL=ud-iq4_xs -e FAMILY=unsloth -e MODEL_PATH=/models -e CONTEXT=128k -e PARALLEL="${PARALLEL:-2}" -e VISION=off
          -e TUNE=skip -e PRESET="${PRESET:-docker}")
# Defaults: 16K prompt chunks (+8% reading on a 32K prompt, measured) and thinking level medium, the same as
# llama.cpp's `big` (--reasoning-effort medium). Without it Strata's template thinks at xhigh (plans/strix-strata.md
# Phase 1 item 9). Check after start: the log says "thinking level: medium". STRATA_SET= turns both off.
STRATA_SET="${STRATA_SET-prefill=16384 reasoning-effort=medium}"
[ -n "$STRATA_SET" ] && SETTINGS+=(-e STRATA_SET="$STRATA_SET")
GPU=(--device /dev/kfd --device /dev/dri --group-add video --group-add "$(getent group render | cut -d: -f3)"
     --ulimit memlock=-1:-1 --shm-size 2g)
MODELS=(); for n in 1 2 3; do MODELS+=(-v "$M/$F-0000$n-of-00003.gguf:/models/$F-0000$n-of-00003.gguf:ro"); done

relay() {  # forwards TS:8099 -> strata:8080; nothing else
  docker rm -f strata-relay >/dev/null 2>&1
  docker run -d --name strata-relay --restart no -p "$TS:$PORT:$PORT" --read-only --cap-drop ALL --user 65534:65534 \
    "$RELAY_IMG" python3 -c '
import asyncio
async def pipe(r, w):
    try:
        while (d := await r.read(65536)): w.write(d); await w.drain()
    finally: w.close()
async def handle(cr, cw):
    try: sr, sw = await asyncio.open_connection("strata", 8080)
    except OSError: cw.close(); return
    await asyncio.gather(pipe(cr, sw), pipe(sr, cw), return_exceptions=True)
async def main():
    s = await asyncio.start_server(handle, "0.0.0.0", '"$PORT"'); await s.serve_forever()
asyncio.run(main())' >/dev/null
  docker network connect "$NET" strata-relay
}

sealed_check() {  # Strata must not reach the internet; good result: "blocked"
  docker exec strata /opt/strata/terminal/python/bin/python3 -c '
import urllib.request
try: urllib.request.urlopen("https://huggingface.co", timeout=5); print("REACHED THE INTERNET")
except Exception as e: print("blocked (" + type(e).__name__ + ")")'
}

case "${1:-}" in
plan)
  docker run --rm --network none "${GPU[@]}" -v strata-data:/data "${MODELS[@]}" --entrypoint /opt/strata/terminal/bin/strata "$IMG" \
    launch --no-tui --plan --model ud-iq4_xs --family unsloth --model-path /models --context 128k --parallel 2 --vision off \
    --data-dir /data/Strata-data 2>&1 | tee "$DIR/plan.txt" ;;
start|restart)
  docker network inspect "$NET" >/dev/null 2>&1 || docker network create --internal "$NET" >/dev/null
  if [ "$1" = start ]; then
    log "stopping marvin's llama-swap (marvin/big is offline until ./strata-test.sh stop)"
    (cd "$LS" && docker compose stop -t 30 llama-swap >/dev/null 2>&1); sleep 5
  else
    log "restart: stopping Strata"; docker stop -t 120 strata >/dev/null 2>&1; docker rm -f strata strata-relay >/dev/null 2>&1; sleep 5
  fi
  log "settings: preset ${PRESET:-docker}, parallel ${PARALLEL:-2}, STRATA_SET '$STRATA_SET'"
  log "before Strata start: $(mem)"
  ( umask 077; openssl rand -hex 24 > "$KEYF" )
  docker rm -f strata >/dev/null 2>&1
  docker run -d --name strata --network "$NET" --restart no --stop-timeout 120 "${GPU[@]}" \
    -v strata-data:/data "${MODELS[@]}" "${SETTINGS[@]}" -e API_KEY_FILE=/run/key -v "$KEYF:/run/key:ro" "$IMG" >/dev/null \
    || { log "docker run failed; run ./strata-test.sh stop"; exit 1; }
  relay
  log "Strata started; waiting until it answers (first start builds its pack from the GGUF)"
  t0=$(date +%s)
  for i in $(seq 1 720); do   # up to 2 hours
    docker ps -q --filter name=^strata$ | grep -q . || { log "Strata exited:"; docker logs --tail 40 strata 2>&1 | tee -a "$DIR/log.txt"; exit 1; }
    curl -sf -m 5 -H "Authorization: Bearer $(cat "$KEYF")" "http://$TS:$PORT/v1/models" >/dev/null 2>&1 && break
    [ $((i % 30)) -eq 0 ] && log "still preparing: $(docker logs --tail 1 strata 2>&1 | cut -c1-160)"
    sleep 10
  done
  curl -sf -m 5 -H "Authorization: Bearer $(cat "$KEYF")" "http://$TS:$PORT/v1/models" >/dev/null || { log "not ready after 2 hours"; exit 1; }
  log "ready after $(( ($(date +%s) - t0) / 60 )) min; $(mem)"
  log "internet from inside Strata: $(sealed_check)"
  log "Strata data volume: $(docker run --rm -v strata-data:/d:ro "$RELAY_IMG" du -sh /d/Strata-data | cut -f1)" ;;
status)
  docker ps --filter name=strata --format '{{.Names}} {{.Status}}'
  curl -sf -m 5 -H "Authorization: Bearer $(cat "$KEYF")" "http://$TS:$PORT/v1/models" >/dev/null && echo "API answers on $TS:$PORT" || echo "API not answering"
  echo "$(mem)"; echo "internet from inside Strata: $(sealed_check 2>/dev/null)" ;;
stop)
  log "stopping Strata and the relay"
  docker stop -t 120 strata >/dev/null 2>&1; docker rm strata strata-relay -f >/dev/null 2>&1
  docker network rm "$NET" >/dev/null 2>&1
  log "starting llama-swap"; (cd "$LS" && docker compose start llama-swap >/dev/null)
  for i in $(seq 1 60); do
    curl -sf -m 5 "http://127.0.0.1:8080/health" >/dev/null 2>&1 && break; sleep 5; done
  log "llama-swap: $(curl -s -m 5 http://127.0.0.1:8080/health || echo 'not answering yet')"; log "$(mem)" ;;
*) sed -n 2,10p "$0"; exit 1 ;;
esac
