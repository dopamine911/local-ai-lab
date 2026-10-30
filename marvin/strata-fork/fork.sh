#!/usr/bin/env bash
# UnbiasedStrata (our gfx1151 build, fork branch hip-gfx1151) on marvin. Plan: plans/strix-strata.md, Phase 3 step 6.
# Runs ON marvin. Copy: scp marvin/strata-fork/fork.sh marvin:strata-fork/
# Only when Daniel says marvin is free (llama-swap's `big` is offline while it runs).
#
#   ./fork.sh check    step 6 checks 1-4, in order, stops at the first failure (llama-swap stopped, started again after)
#   ./fork.sh start    stop llama-swap, serve the fork sealed on marvin's Tailscale address :8099 (like strata-test.sh)
#   ./fork.sh status   is it up; memory; can it reach the internet (must say NO)
#   ./fork.sh stop     stop the fork, start llama-swap again                                  <- always end with this
#   ./fork.sh args     dry run: print the engine arguments it would use (no GPU)
#
# Before the first run, from zaphod (images built there, fork commit in the plan):
#   docker save unbiased-strata:gfx1151 unbiased-strata-tests:gfx1151 | ssh marvin docker load
# The pack, built on zaphod (plan 5e), copied to $PACK; checked against pack-ud-iq4_xs.sha256 before every run:
#   rsync -a --mkpath /srv/models/strata-fork/pack-ud-iq4_xs/ marvin:/srv/models/strata-fork/pack-ud-iq4_xs/
# Engine settings: the Phase 1 upstream run's own arguments, read from marvin's strata-data volume (preset $PRESET),
# with the fork's pack and the same MTP pieces (strata-data/Strata-data/mtp/rt, sha256 = the fork's published set).
# Every container: normal user, only the GPU groups, read-only filesystem, no network (checks) or an internal one.
# Results: ~/strata-fork/results.txt (every line also on screen).
set -uo pipefail
# marvin only (stop/start run marvin's llama-swap compose); `args` is a dry run and works anywhere the images are
[ "${1:-}" = args ] || [ "$(hostname)" = marvin ] || { echo "run this on marvin (this is $(hostname))"; exit 1; }
RT=unbiased-strata:gfx1151; TESTS=unbiased-strata-tests:gfx1151; RELAY_IMG=python:3.13-slim-trixie
# config digests: fork cc2a9ca (Ubuntu resolute-20261024.1 + snapshot 2026-10-30), plans/strix-strata.md Phase 3 log 12
RT_CFG=sha256:6631f978e6b78160ff26c8170a76fc6009ee5f42218aa4b497f998aee2b4f474
TESTS_CFG=sha256:1f6f4be5b5b5735561598a5980930c9422e3249a8af8694bb94e2b6a6a574492
M=/srv/models; F=Qwen3.8-Flash-Next-UD-IQ4_XS
PACK=${PACK:-$M/strata-fork/pack-ud-iq4_xs}
PRESET=${PRESET:-p16m}; PARALLEL=${PARALLEL:-2}
DIR=$HOME/strata-fork; KEYF=$DIR/key; NET=strata-internal; PORT=8099; NAME=strata-fork
LS=$HOME/homelab/marvin/llama-swap
TS=$(tailscale ip -4 2>/dev/null)
mkdir -p "$DIR"; chmod 700 "$DIR"
# has PATTERN: does the input match. Not grep -q (or grep >/dev/null, which GNU grep treats the same): they stop
# reading at the first match, the writer dies of SIGPIPE, and pipefail calls the pipeline failed. -c reads it all.
has() { [ "$(grep -cE "$@")" -gt 0 ]; }
log() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$DIR/results.txt"; }
ok()  { log "GOOD: $*"; }
bad() { log "BAD: $*"; exit 1; }
mem() { echo "GPU memory used $(( $(cat /sys/class/drm/card*/device/mem_info_gtt_used | head -1) / 1073741824 )) GB (GTT); RAM: $(free -g | awk 'NR==2{print $3" GB used, "$7" GB available"}')"; }

# normal user + GPU groups only; HOME=/tmp because ROCm writes a code cache there (comgr) and the image's /root is not ours
USERRUN=(--user "$(id -u):$(id -g)" --group-add "$(getent group render | cut -d: -f3)" --group-add "$(getent group video | cut -d: -f3)"
         --device /dev/kfd --device /dev/dri --read-only --tmpfs /tmp:exec -e HOME=/tmp --ulimit memlock=-1:-1 --shm-size 2g)
MODELS=(); for n in 1 2 3; do MODELS+=(-v "$M/$F-0000$n-of-00003.gguf:/models/$F-0000$n-of-00003.gguf:ro"); done
DATA=(-v "$PACK:/pack:ro" -v strata-data:/sd:ro)

preflight() {
  [ "$(hostname)" = marvin ] || bad "run this on marvin"
  docker ps --format '{{.Names}}' | has -x strata && bad "upstream Strata is running: ~/strata-test/strata-test.sh stop first"
  for i in "$RT $RT_CFG" "$TESTS $TESTS_CFG"; do set -- $i
    docker image inspect "$1" --format '{{.Id}} {{json .Descriptor}}' 2>/dev/null | has "$2" \
      && ok "image $1 is the one built on zaphod ($2)" || bad "image $1 missing or not $2 (load it from zaphod)"; done
  [ -f "$PACK/native_experts.txt" ] && [ -f "$PACK/expert-profile.bin" ] || bad "no finished pack with expert-profile.bin in $PACK"
  (cd "$PACK" && sha256sum -c --quiet "$HOME/homelab/marvin/strata-fork/pack-ud-iq4_xs.sha256") \
    && ok "pack matches the one built on zaphod (pack-ud-iq4_xs.sha256)" || bad "pack differs from pack-ud-iq4_xs.sha256"
  for n in 1 2 3; do [ -f "$M/$F-0000$n-of-00003.gguf" ] || bad "missing $F-0000$n-of-00003.gguf"; done
}

engine_args() {  # Phase 1's arguments with the fork's paths; one per line. Source: the last "engine started:" line of
  # the preset's engine log (the real command, incl. what upstream's server adds at launch), else the preset's json.
  docker run --rm --network none -v strata-data:/sd:ro --entrypoint sh "$RT" -c \
    "grep -h 'engine started: ' /sd/state/strata/launch/$PRESET.engine.log 2>/dev/null | tail -1; echo ===; cat /sd/state/strata/launch/$PRESET.json" |
  python3 -I -c '
import json, sys
line, _, js = sys.stdin.read().partition("===\n")
if line.strip():
    a = line.split("engine started: ", 1)[1].split()[1:]          # drop the engine path
    print("source: engine log", file=sys.stderr)
else:
    try: d = json.loads(js); a = d.get("args_written") or d["args"]
    except (ValueError, KeyError): sys.exit("no engine log or preset json for this PRESET in strata-data (state/strata/launch/)")
    print("source: preset json", file=sys.stderr)
out = []; i = 0
swap = {"--pack": "/pack", "--expert-profile": "/pack/expert-profile.bin", "--mtp": "/sd/Strata-data/mtp/rt"}
drop = {"--reasoning-effort"}                      # upstream server option; the fork sets it per request / template kwargs
while i < len(a):
    k = a[i]
    if k == "--serve": i += 1; continue           # the front end adds it
    has_val = i + 1 < len(a) and not a[i + 1].startswith("--")
    v = a[i + 1] if has_val else None
    i += 2 if has_val else 1
    if k in drop: continue
    out += [k] + ([swap.get(k, v)] if has_val else [])
if "--batch" not in out: out += ["--batch", sys.argv[1]]
print("\n".join(out))' "$PARALLEL"
}

stop_llama_swap()  { log "stopping marvin's llama-swap (marvin/big offline until ./fork.sh stop)"; (cd "$LS" && docker compose stop -t 30 llama-swap >/dev/null 2>&1); sleep 5; }
start_llama_swap() { log "starting llama-swap"; (cd "$LS" && docker compose start llama-swap >/dev/null)
  for i in $(seq 1 60); do curl -sf -m 5 http://127.0.0.1:8080/health >/dev/null 2>&1 && break; sleep 5; done
  log "llama-swap: $(curl -s -m 5 http://127.0.0.1:8080/health || echo 'not answering yet'); $(mem)"; }

engine_start() {  # $1 = network; the engine in the background as $NAME
  mapfile -t EA < <(engine_args); [ ${#EA[@]} -gt 4 ] || bad "could not read preset $PRESET's arguments"
  log "engine arguments: ${EA[*]}"
  docker rm -f "$NAME" >/dev/null 2>&1
  docker run -d --name "$NAME" --network "$1" --restart no --stop-timeout 120 "${USERRUN[@]}" "${MODELS[@]}" "${DATA[@]}" \
    ${KEYF_MOUNT[@]+"${KEYF_MOUNT[@]}"} "$RT" --engine strata -m "/models/$F-00001-of-00003.gguf" \
    --host 0.0.0.0 --port 8080 --alias big-fork ${FRONT[@]+"${FRONT[@]}"} -- "${EA[@]}" >/dev/null || bad "docker run failed"
  t0=$(date +%s)
  for i in $(seq 1 360); do   # up to 1 hour
    docker ps -q --filter "name=^$NAME$" | has . || { docker logs --tail 40 "$NAME" 2>&1 | tee -a "$DIR/results.txt"; bad "the engine exited"; }
    docker logs "$NAME" 2>&1 | has "listening on" && break
    [ $((i % 30)) -eq 0 ] && log "still starting: $(docker logs --tail 1 "$NAME" 2>&1 | cut -c1-160)"
    sleep 10
  done
  docker logs "$NAME" 2>&1 | has "listening on" || bad "not ready after 1 hour"
  log "ready after $(( $(date +%s) - t0 )) s; $(mem)"
}

case "${1:-}" in
args) engine_args ;;
check)
  log "=== check $(date '+%F %T'), preset $PRESET, pack $PACK"
  preflight
  ELOG=$DIR/engine-$(date +%Y%m%d-%H%M%S).log
  stop_llama_swap; trap 'docker logs "$NAME" > "$ELOG" 2>&1 && log "engine log saved: $ELOG"; docker rm -f "$NAME" >/dev/null 2>&1; docker network rm "$NET" >/dev/null 2>&1; start_llama_swap' EXIT
  log "before: $(mem)"
  # 1. the GPU is seen and this build can run it
  out=$(docker run --rm --network none "${USERRUN[@]}" "$TESTS" strata-device --list-devices 2>&1); echo "$out" >> "$DIR/results.txt"
  echo "$out" | has "arch gfx1151" || bad "strata-device: no gfx1151 (output in results.txt)"
  echo "$out" | has "cannot run" && bad "strata-device: this build cannot run the GPU (output in results.txt)"
  ok "strata-device sees gfx1151 and can run it"
  # 2. the three HIP parity tests: pass = exit 0; 77 = skipped (counts as a failure here)
  for t in hip_prefill_hipblaslt_gemm hip_prefill_hcd_exact_parity hip_prefill_mmq_parity; do
    docker run --rm --network none "${USERRUN[@]}" "$TESTS" "$t" >> "$DIR/results.txt" 2>&1; rc=$?
    [ $rc = 0 ] && ok "$t passed" || bad "$t exit $rc (77 = skipped); output in results.txt"; done
  # 3. the engine starts with Phase 1's settings and turns on the gfx1151 defaults; one short answer (internal network
  #    only, asked from a throwaway container); the hipBLASLt table is loaded by then (at start or at the first prompt)
  docker network inspect "$NET" >/dev/null 2>&1 || docker network create --internal "$NET" >/dev/null
  FRONT=(); KEYF_MOUNT=(); engine_start "$NET"
  fail=0; soft() { log "BAD: $*"; fail=1; }
  docker logs "$NAME" 2>&1 | has "gfx1151.*19 exact speed switches" && ok "19 gfx1151 switches on" || soft "the start log does not list 19 gfx1151 switches"
  ans=$(docker run --rm --network "$NET" --read-only --cap-drop ALL --user 65534:65534 "$RELAY_IMG" python3 -c '
import json, urllib.request
r = urllib.request.Request("http://'"$NAME"':8080/v1/chat/completions", headers={"Content-Type": "application/json"},
    data=json.dumps({"messages": [{"role": "user", "content": "What is 2+3? Answer with one number."}],
                     "max_tokens": 400, "reasoning_effort": "none"}).encode())
print(json.load(urllib.request.urlopen(r, timeout=600))["choices"][0]["message"].get("content", "").strip()[:200])' 2>&1)
  log "answer to 2+3: $ans"; echo "$ans" | has 5 && ok "the engine answers (5)" || soft "no sensible answer (above)"
  log "GPU power now: $(awk '{print $1/1000000 " W"}' /sys/class/drm/card*/device/hwmon/hwmon*/power1_input 2>/dev/null | head -1)"
  L=$(docker logs "$NAME" 2>&1); echo "$L" | grep -iE "gfx1151|hipBLASLt|hipBLASEx|switch|memory guard|expert cache|error|warn" | head -40 >> "$DIR/results.txt"
  echo "$L" | has "hipBLASLt tuning enabled.*gfx1151.*100401" && ok "hipBLASLt table loaded (gfx1151, 100401)" || soft "hipBLASLt table not loaded"
  [ $fail = 0 ] || bad "step 3 failed (above); full engine log saved at the end"
  log "=== check passed. Next: ./fork.sh start, then the speed / quality tests (plan Phase 3 step 6)" ;;
start)
  preflight
  docker network inspect "$NET" >/dev/null 2>&1 || docker network create --internal "$NET" >/dev/null
  stop_llama_swap
  [ -s "$KEYF" ] || ( umask 077; openssl rand -hex 24 > "$KEYF" )   # kept between starts: zaphod's peer uses it
  # sampling and thinking level = llama.cpp's `big` (marvin config qwen-sampling), so the comparison is like for like.
  # This llama.cpp has no --reasoning-effort flag: the template's reasoning_effort is set as a default kwarg instead.
  # Check after start: a request without a level should think about as long as `big` does (the template default is xhigh).
  FRONT=(--api-key-file /run/key --temp 1.0 --top-p 0.95 --top-k 20 --min-p 0.0
         --chat-template-kwargs "{\"reasoning_effort\":\"${THINK:-medium}\"}")   # THINK=xhigh: upstream's default
  KEYF_MOUNT=(-v "$KEYF:/run/key:ro"); log "thinking level default: ${THINK:-medium}"; engine_start "$NET"
  docker rm -f strata-fork-relay >/dev/null 2>&1
  docker run -d --name strata-fork-relay --restart no -p "$TS:$PORT:$PORT" --read-only --cap-drop ALL --user 65534:65534 \
    "$RELAY_IMG" python3 -c '
import asyncio
async def pipe(r, w):
    try:
        while (d := await r.read(65536)): w.write(d); await w.drain()
    finally: w.close()
async def handle(cr, cw):
    try: sr, sw = await asyncio.open_connection("'"$NAME"'", 8080)
    except OSError: cw.close(); return
    await asyncio.gather(pipe(cr, sw), pipe(sr, cw), return_exceptions=True)
async def main():
    s = await asyncio.start_server(handle, "0.0.0.0", '"$PORT"'); await s.serve_forever()
asyncio.run(main())' >/dev/null
  docker network connect "$NET" strata-fork-relay
  for i in $(seq 1 15); do curl -sf -m 2 -o /dev/null http://$TS:$PORT/health && break; sleep 1; done   # relay start-up
  curl -sf -m 5 -H "Authorization: Bearer $(cat "$KEYF")" "http://$TS:$PORT/v1/models" >/dev/null && ok "API answers on $TS:$PORT" || bad "API not answering on $TS:$PORT"
  log "internet from inside: $(docker run --rm --network "$NET" "$RELAY_IMG" python3 -c '
import urllib.request
try: urllib.request.urlopen("https://huggingface.co", timeout=5); print("REACHED THE INTERNET")
except Exception as e: print("blocked (" + type(e).__name__ + ")")')" ;;
status)
  docker ps --filter "name=strata-fork" --format '{{.Names}} {{.Status}}'
  curl -sf -m 5 -H "Authorization: Bearer $(cat "$KEYF")" "http://$TS:$PORT/v1/models" >/dev/null && echo "API answers on $TS:$PORT" || echo "API not answering"
  echo "$(mem)" ;;
stop)
  log "stopping the fork and the relay"
  docker stop -t 120 "$NAME" >/dev/null 2>&1; docker rm -f "$NAME" strata-fork-relay >/dev/null 2>&1
  docker network rm "$NET" >/dev/null 2>&1
  start_llama_swap ;;
*) sed -n 2,12p "$0"; exit 1 ;;
esac
