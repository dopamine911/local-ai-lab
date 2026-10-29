#!/usr/bin/env bash
# Strata test on both cards (GPUS=0,1). Always gives the cards back to llama-swap at the end.
set -uo pipefail
S=$(mktemp -d)   # holds the one-time API key for this run
IMG=ghcr.io/niko1221/strata:0.2.7-rocm-gfx120x
PORT=8099
log() { echo "[$(date +%H:%M:%S)] $*"; }
vram() { for p in 06:00.0 0b:00.0; do d=/sys/bus/pci/devices/0000:$p; printf "%s used %s MB of %s MB; " $p $(( $(cat $d/mem_info_vram_used)/1048576 )) $(( $(cat $d/mem_info_vram_total)/1048576 )); done; echo; }

restore() {
  log "restore: stopping Strata, starting llama-swap"
  docker stop -t 120 strata >/dev/null 2>&1; docker rm strata >/dev/null 2>&1
  (cd ~/llama-swap && docker compose start >/dev/null 2>&1)
  for i in $(seq 1 120); do
    (cd ~/llama-swap && docker compose logs --since 10m 2>&1 | grep -q 'coder> Health check passed') && break; sleep 5; done
  log "llama-swap back: $(cd ~/llama-swap && docker compose logs --since 10m 2>&1 | grep -c 'Health check passed') health checks passed"; log "VRAM: $(vram)"
  log "STRATA-TEST-END"
}
trap restore EXIT

KEY=$(openssl rand -hex 24); echo "$KEY" > $S/strata-key; chmod 600 $S/strata-key
log "stopping llama-swap"; (cd ~/llama-swap && docker compose stop -t 30 >/dev/null 2>&1)
sleep 5; log "VRAM before Strata start (free memory matters, see plans/strata.md): $(vram)"
free -g | sed -n 2p | sed 's/^/    RAM: /'

docker run -d --name strata \
  --device /dev/kfd --device /dev/dri --group-add video --group-add 991 \
  -p 127.0.0.1:$PORT:8080 --ulimit memlock=-1:-1 --shm-size 2g --stop-timeout 120 \
  -v strata-data:/data \
  -v /srv/models/strata/qwen3.8-flash-next-gsq-rco/IQ3_XXS:/models:ro \
  -e MODEL_PATH=/models -e FAMILY=qwen -e MODEL=IQ3_XXS -e VISION=off -e GPUS=0,1 \
  -e API_KEY="$KEY" $IMG >/dev/null || { log "docker run failed"; exit 1; }
log "Strata started; waiting for it to be ready (first start prepares the pack)"
ready=0
for i in $(seq 1 720); do   # up to 2 hours
  if ! docker ps --filter name=^strata$ -q | grep -q .; then log "Strata container exited:"; docker logs --tail 40 strata 2>&1; exit 1; fi
  if curl -sf -H "Authorization: Bearer $KEY" http://127.0.0.1:$PORT/v1/models >/dev/null 2>&1; then ready=1; break; fi
  [ $((i % 30)) -eq 0 ] && log "still preparing: $(docker logs --tail 1 strata 2>&1 | cut -c1-160)"
  sleep 10
done
[ $ready = 1 ] || { log "not ready after 2 hours"; docker logs --tail 40 strata 2>&1; exit 1; }
MODEL=$(curl -s -H "Authorization: Bearer $KEY" http://127.0.0.1:$PORT/v1/models | python3 -c "import json,sys;print(json.load(sys.stdin)['data'][0]['id'])")
log "ready; model id: $MODEL"; log "VRAM with Strata loaded: $(vram)"
docker logs strata 2>&1 | grep -i -E 'gpu|card|resident|context|split|vram|expert' | tail -15 | sed 's/^/    /'

log "speed: 3 requests, same prompt as the coder test (600 tokens, temp 0)"
KEY=$KEY MODEL=$MODEL PORT=$PORT python3 - <<'EOF'
import json,os,time,urllib.request,statistics
H={"Authorization":"Bearer "+os.environ["KEY"],"Content-Type":"application/json"}
p="Write a Python function that parses an Apache access log line into a dict, with a docstring and three test cases."
r=[]
for i in range(3):
    b={"model":os.environ["MODEL"],"max_tokens":600,"temperature":0,"seed":1,"messages":[{"role":"user","content":p}]}
    t=time.time(); d=json.load(urllib.request.urlopen(urllib.request.Request("http://127.0.0.1:"+os.environ["PORT"]+"/v1/chat/completions",data=json.dumps(b).encode(),headers=H),timeout=900)); el=time.time()-t
    tim=d.get("timings") or {}; n=(d.get("usage") or {}).get("completion_tokens",0)
    tps=tim.get("predicted_per_second") or (n/el if el else 0)
    r.append(tps); print(f"    run {i+1}: {round(tps,1)} tok/s writing, {n} tokens, {round(el,1)} s total, prompt {round(tim.get('prompt_per_second',0),1)} tok/s")
print("    median writing:",round(statistics.median(r),1),"tok/s")
EOF

cd ~/homelab/tuning
for spec in "0 42"; do
  set -- $spec
  log "tool-eval full suite, temperature $1, seed $2"
  BASE_URL=http://127.0.0.1:$PORT/v1 API_KEY=$KEY ./quality "$MODEL" "strata 0.2.7 rocm, Qwen3.8-Flash-Next IQ3_XXS, GPUS=0,1 (two cards), temp $1 seed $2" --full --temperature $1 --seed $2 2>&1 | tail -3 | sed 's/^/    /'
done
log "tests finished"
