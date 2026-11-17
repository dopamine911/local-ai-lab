# gpt-oss-120b: how to run it (notes, 2026-11-09)

A **test model only, not for daily use** (Daniel, 2026-11-09). Nothing loads it by itself: it starts only when a
request names `gpt-oss`. Every step with times and records: `plans/gpt-oss-timeline.md`.

## Best results

| Where | Context | Writing tok/s | Reading 32K tok/s | First word (30K prompt) | Tool use, 15 core | Tool use, full 69 |
| --- | --- | --- | --- | --- | --- | --- |
| **zaphod**, 2x R9700, split by hand (now set) | 32K | **85.3** | **2,885** | 10.6 s | 97 | not run |
| marvin (Strix Halo) | 128K | 53.4 | 820 | 37.1 s | 97 | 82 (`big`: 90) |

Known weak spots (full 69, marvin): failed the TC-60 injection test (added an attacker's BCC/CC), and in several
tasks it didn't send the email at all. The tools + JSON-format bug of Qwen models does not affect it (8/8).

## The file

`gpt-oss-120b-MXFP4.gguf`, ggml-org/gpt-oss-120b-GGUF rev 7200b74, 63.4 GB, pinned in `setup/models-marvin.txt`
and `setup/models-zaphod.txt` (download: `lab models <machine> gpt-oss`). Between the machines a copy is
quicker now that both use their cables (~270 MB/s, about 4 min): `rsync marvin:/srv/models/gpt-oss-120b-MXFP4.gguf
/srv/models/`, then `lab models zaphod gpt-oss` to check the sha256. Good result: `MODELS-OK`.

## On zaphod

It needs both cards to itself, so `coder`, `coder-2`, the chat models **and the helpers** (embed, rerank, speech)
unload while it runs. Any `embed` call pushes it out again, and the watchdog's memory search does that every minute.

1. Check that nobody is using zaphod or marvin (a zaphod config reload cuts marvin requests passing through it).
2. Pause the watchdog (needs your password): `! sudo systemctl stop lab-watchdog`.
   Good result: `systemctl is-active lab-watchdog` prints `inactive`.
3. The memory sync also calls `embed`, every hour at :41. Start right after it.
4. Load gpt-oss with one small request (about 60 s), or just start the test: the first request loads it.
5. Tests (from `~/homelab/tuning`):
   - speed: `lab speed gpt-oss "<note>"`. Good result: three short and three long lines, then `median:` and `Logged run`.
   - tool use: `lab quality gpt-oss "<note>"` (15 scenarios, ~2 min) or `--full` (69, ~16 min).
6. Bring zaphod back with one small request each: chat requests to `coder` and `coder-2`, an embeddings request to
   `embed`, then `curl .../upstream/<model>/health` for rerank, stt, stt-whisper, tts (any config change also
   reloads and preloads them). Good result: `/running` lists those 7, all `ready`.
7. Restart the watchdog: `! sudo systemctl start lab-watchdog`. Good result: `active`.

### zaphod settings and why (`zaphod/llama-swap/config/config.yaml`, model `gpt-oss`)

```
${server}                                   # has -ngl 999, so llama.cpp's --fit is off
--device Vulkan0,Vulkan1 -ts 18,19          # card 2 first; layers split by hand, nothing in RAM
-m /models/gpt-oss-120b-MXFP4.gguf
-c 32768 -np 1 -b 2048 -ub 2048 -lm none
--temp 1.0 --top-p 1.0 --reasoning-effort medium
```

What was measured (32K reading / writing):

1. **Split by hand beats `--fit`**: fit splits whole layers, so it always left part of one layer in RAM although
   card 1 still had 1-2.5 GB free: 60.7 / 1,731. By hand (`-ts 18,19`): 85.3 / 2,885.
2. **`--fit` beats `--n-cpu-moe`** when part has to go to RAM: at 128K, fit 54.5 / 1,423;
   `--n-cpu-moe 8`: 38.5 / 180-366 (it also turns fit off, so the card split came out uneven).
3. **`-lm none`** (no memory-mapping): +12% reading when part is in RAM (1,272 → 1,423).
4. `-ub 512` instead of 2048: reading -37%, writing and tool use the same.
5. `q8_0` context: no gain at 32K, tool use the same.
6. **128K on the cards**: only with `-c 131072 -b 2048 -ub 512 -ctk q8_0 -ctv q8_0` (card 1 at 32.1 of 32.6 GB):
   82.9 / 2,278, but tool use 90 (TC-15 fails in every 128K setup). With part in RAM instead (`${server-fit}`,
   `-c 131072 -b 4096 -ub 4096 -lm none`): 54.5 / 1,423.

To check how much card memory a setting needs without loading anything (the fit prints it in the first second):

```
cd ~/llama-swap && docker compose exec -T llama-swap sh -c "timeout 3 llama-server --port 5899 -fa on \
  --device Vulkan0,Vulkan1 -m /models/gpt-oss-120b-MXFP4.gguf -np 1 -c 32768 -b 2048 -ub 2048 -lv 4 2>&1" \
  | grep -m1 'projected to use'
```

Usable on the two empty cards: about 65,400 MiB (fit keeps 1 GiB spare per card by default). With `timeout 8` and
`grep 'fit_impl:   - Vulkan'` it also prints the layer split the fit would choose.

## On marvin

Model `gpt-oss` on marvin's own llama-swap (`http://marvin:8080`, key `MARVIN_KEY`); it is **not** in zaphod's
peer list. Loading it unloads `big`; `big` comes back on its next request (~30 s).

1. Check marvin is idle: on marvin, for every model in `/running` (e.g. `big`, `chat-teacher`),
   `curl -s localhost:8080/upstream/<model>/slots -H "Authorization: Bearer $MARVIN_KEY"` shows `"is_processing": false`
   for every slot (live, seen true during a request on 2026-11-17; through zaphod's peer this gives 404). Every finished request is
   in `~/llama-swap/data/activity.sqlite` (a row appears only when the request ends); llama-swap does not print requests in its log, so a `POST /v1` count is always 0.
2. Tests from zaphod (`~/homelab/tuning`, after `source ~/llama-swap/.env`):
   `MACHINE=marvin BASE_URL=http://marvin:8080/v1 API_KEY=$MARVIN_KEY lab speed gpt-oss "<note>"`
   (same for `lab quality`).
3. Afterwards one small request to `big`, so it is loaded again.

marvin settings (`marvin/llama-swap/config/config.yaml`): `-c 131072 -np 1 -b 4096 -ub 4096`, OpenAI sampling.
Measured: `-ub` 512 → 2048 → 4096 gave 32K reading 558 → 765 → 820 (writing the same). Tried and dropped: EAGLE3
draft (ggml-org Q8_0: writing -26%, reading -10%, +12 GB), Mesa 26.2.6 from the kisak PPA (+1.4% reading only).
