# Morning guide, 2026-10-29

What was built while you slept, and what is left for you. Do the steps in order; each says what a good result looks like.

## 1. Overnight results

| Done | Result | Where |
| --- | --- | --- |
| Qwen-Image-2.1 Turbo in ComfyUI | 1024x1024 in 6 s; writes text in images exactly ("ZAPHOD LAB") | `hub/comfyui/workflows/qwen-image-2.1-turbo-api.json` |
| Strata test (Qwen3.8-Flash-Next) | Tool-use 86 / 92 / 86 on one card (Strata picked one card by itself); 90 on two cards, 85 tok/s writing, 12 min per suite (vs 33). Good, but needs both cards | `plans/strata.md`, `tuning/results.md`, MLflow |
| Homepage | ComfyUI link added, zaphod model list updated | `hub/homepage/services.yaml` |
| audio.cpp v0.9.3 | Checked: nothing for our speech models; not updating | `plans/more-models.md` |
| `./quality` | Can now test any server: `BASE_URL=... API_KEY=...` | `tuning/quality` |

Not done (needs you): the models panel file (that tool was off in this session), the Open WebUI settings,
the Terminal-Bench smoke runs.

## 2. Check zaphod is back to normal

1. `docker ps --format '{{.Names}}  {{.Status}}' | sort`
   Good: llama-swap, open-webui, caddy and the other hub services `Up`; no `strata` and no `comfyui` container.
2. `~/homelab/hub/image-mode status`
   Good: `profile: none`, `ComfyUI: stopped`.
3. `cd ~/llama-swap && docker compose logs --tail 40 | grep 'Health check passed'`
   Good: `coder`, `coder-2`, `embed`, `rerank`, `stt`, `stt-whisper`, `tts` (the 404 "failed to preload" lines for
   stt/tts are harmless).

## 3. Push the commits

1. `cd ~/homelab && git status` — good: "nothing to commit".
2. `git log --oneline origin/main..HEAD | wc -l` — the number of commits waiting (about 40).
3. `git push`

## 4. Open WebUI settings (Admin Panel → Settings), about 10 minutes

All values are also in `README.md`, section "Open WebUI tools". Key everywhere: your ZAPHOD_KEY
(`grep ZAPHOD_KEY ~/llama-swap/.env`). URL everywhere: `http://127.0.0.1:8080/v1` unless stated.

| Page | Setting | Value | Good result |
| --- | --- | --- | --- |
| Documents | Embedding engine / model | OpenAI / `embed` | Upload a file to a chat, ask about it: the answer quotes it |
| Documents | Hybrid search, reranking engine / URL / model | on / External / `http://127.0.0.1:8080/v1/rerank` / `rerank` | (same test) |
| Audio | Speech-to-text engine / model | OpenAI / `stt` | Microphone: your words appear |
| Audio | Text-to-speech engine / model / voice | OpenAI / `tts` / `af_heart` | Speaker icon reads the answer |
| Images (optional) | Engine / URL / workflow | ComfyUI / `http://127.0.0.1:8188` / a file from `hub/comfyui/workflows/` | Only works while image mode is on |

After the Documents change: "Reindex Knowledge Base Vectors" if you already had knowledge bases.

## 5. Try the new things

1. **Images:** `~/homelab/hub/image-mode on`, open https://comfy.ai.dopamine911.com, make an image,
   then `~/homelab/hub/image-mode off`. Good: `on` ends with "ComfyUI ready"; `off` says card 2 is back.
   While it is on, `coder-2` requests are answered by `coder`, and `ornith`/`chat` are unavailable.
2. **Vision:** in Open WebUI pick `ornith` or `chat` (Gemma 4), attach a screenshot, ask what it shows.
3. **Voice:** after step 4, talk to it with the microphone, and press the speaker icon on an answer.

## 6. Terminal-Bench smoke runs (yours; the safety check blocks me from running that code)

In tmux, one at a time (10-30 minutes each):
1. `cd ~/homelab/tuning && MODEL=ornith ./tbench smoke "ornith 1.5"`
2. Good: ends with the task's pass/fail and "Logged run".

## 7. Decisions waiting for you

- **Strata (Qwen3.8-Flash-Next):** on two cards it scored 92 on the same basis as Ornith (93) (corrected later that day: the first numbers were out of 138, not 130), writes at 85 tok/s, and was the
  only model besides coder + the-craft (and north + j5-craft) to refuse the CFO "Q4 will improve" claim (on two cards; on one card it sent it). But it needs BOTH cards, so llama-swap
  (coder, coder-2, speech, document search) has to stop while it runs. My view: not worth that as a daily model; Ornith
  gives the same quality on half of one card. Worth more tests only if you want it for a specific job. Next tests if yes:
  `plans/strata.md`, "Possible next tests".
- **Qwen-Image vs Z-Image as the Open WebUI default:** Qwen-Image is better at text in pictures; Z-Image is a bit faster
  (5 s vs 6 s). Neither follows "watercolor" well; both lean photographic.
- **models panel:** copy `models-panel/index.html` from the claude.ai project into `hub/models-panel/`
  (the panel at https://models.ai.dopamine911.com shows nothing until then).

## 8. Memory map now (for reference)

| Card | PCI | llama.cpp name | Always loaded | Swaps in |
| --- | --- | --- | --- | --- |
| 1 (OCuLink, x4) | 06:00.0 | Vulkan1 | `coder` 22 GB, `embed` + `rerank` ~7.6 GB | `coder-tb` |
| 2 (x16 slot) | 0b:00.0 | Vulkan0 | `stt`, `stt-whisper`, `tts` ~4 GB | `coder-2`, `ornith`, `chat`, `north`, `tts-qwen`, ComfyUI (image mode) |
