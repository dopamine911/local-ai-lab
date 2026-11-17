# Halogen Flash vs `big` on marvin: timeline

Step 2 of plans/engines-and-frontend.md (Daniel 2026-11-17: "start Halogen"). Times are Chicago time, from `date`.

## What it is (read 2026-11-17 from its docs, image 0.20.6)
- An engine for one model on one chip: Qwen3.8-Flash-Next on Strix Halo (= our `big`). Closed binary under an EULA:
  free incl. commercial, no telemetry, no outbound connections unless `HALOGEN_DOWNLOAD` is set (not used here).
- Weights: its own format, repo `peonist-ai/halogen-qwen3.8-flash-next` (Apache-2.0 card). The default set:
  `qwen38-flash-next-v2.hgn` 62.1 GiB (checkpoint, 4.16 bits/weight on average; `big`'s UD-IQ4_XS is similar),
  `-ngram.hgn` 47.7 GiB (lookup table, read through the page cache), `-drafter-r2.hgn` 1.42 GiB (draft head), tokenizer.
  About 111 GiB. Vision (0.84 GiB) skipped: `big` on marvin has none.
- Wants the machine to itself (weights resident, KV pool reserved at start, ~12 GiB left). An unclean exit can leave
  GPU memory held until a reboot (its issues #79, #34): stop it with time to spare.
- Needs `/dev/kfd` + `/dev/dri`, groups render + video, `--ipc=host`, `--ulimit memlock=-1:-1`. marvin already has
  the host settings it asks for: BIOS UMA 512 MB (SETUP.md row 2), kernel 7.0 with `CONFIG_HSA_AMD_SVM=y`.
- Prompt cache on by default (`HALOGEN_PROMPT_CACHE=2`): must pass our leak test before any daily use.
- Thinking: the server's default effort is `xhigh`; `big` runs `--reasoning-effort medium`, so set
  `HALOGEN_REASONING_EFFORT=medium` and big's sampling (temp 1.0, top-p 0.95, top-k 20, min-p 0) for a fair test.
- Its claims (synthetic prompts, greedy, thinking off): 93 tok/s writing code, 1,870 reading at 32K, 1,793 at 128K.
  `big` today: 45.6 writing, 328 reading at 32K.

## Pins
- Image `ghcr.io/peonist-ai/halogen-flash-server:0.20.6@sha256:f68892a56d8a1182645042574d702374bff402bb986ec4d49304cbe89483dbaf`
  (newest; `latest` = the same digest on 2026-11-17).
- Weights revision `5c72aeff67617168b2a10e707e1477de6a1adb47`, into `/srv/models/halogen/` (setup/models-marvin.txt).

## Steps (one change at a time)
1. Launcher: allowlist fields for `--ipc`, `--ulimit` and a per-engine stop timeout (longer for this engine); test.
2. Download the weights (`lab models marvin halogen`), sha256 checked.
3. `big-halogen` entry (allowlist + llama-swap). Smoke: loads, one answer, startup memory lines, `/health`.
4. Privacy: `tuning/cache-switch.py big-halogen 3` with the default cache. Good result: 0 leaks. If it leaks: cache
   mode 0 or 1, test again; daily use only with 0 leaks.
5. Speed: `lab speed` on `big` and `big-halogen`, same session (direct to marvin, `MACHINE=marvin`).
6. Long context: reading and writing at 0 / 32K / 64K / 128K on both (the claim is that reading barely drops).
7. Tool use: `lab quality --full --temperature 0.6` seeds 1-3 on `big-halogen` (`big` on 273: 96 / 91 / 93).
Good result: clearly faster (reading most of all) with tool use inside `big`'s seed spread and 0 leaks. Then Daniel
decides about daily use. `big` on llama.cpp stays as it is either way.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 11:52 | Launcher: allowlist fields `ipc`, `ulimits`, `stop_timeout` (docker `--stop-timeout` and the launcher's own stop wait); client stop waits up to 600 s | — | this commit |
| 11:54 | setup/models-marvin.txt: 9 Halogen files at revision 5c72aeff (sizes and sha256 from the HF tree; the 6 small tokenizer files fetched and hashed) | 111.2 GiB | this commit |
| 11:53 | Download started on marvin (`lab models marvin halogen`, detached; log ~/.cache/halogen-download.log) | ~40 MB/s: about 50 min + checksums | — |
| 11:57 | Image pulled on marvin by digest (4.72 GB). Its entrypoint: runs as root, no privilege steps (only `ulimit -c 0`), default command `all` (engine + API in one container, API on 8731) | — | — |
| 11:59 | `big-halogen` in the allowlist (devices kfd+dri, render+video, ipc host, memlock unlimited, weights read-only, big's sampling and effort medium, stop timeout 120 s) and in marvin's llama-swap config (proxy :8731, checkEndpoint /health, unloadTimeout 300, training pin). Not deployed yet: the launcher service must be restarted for the new allowlist fields (sudo) | — | this commit |
| 11:59 | Daniel restarted lab-engined (new allowlist fields) | active | — |
| 12:03 | New `tuning/depth.py`: reading and writing at 32K / 64K / 128K, one request per depth with a unique run id (no cache can answer it), measured client-side the same way for any engine. Tried on `big` at 4K: 4,168 tokens, reading 386, writing 45.1 (token estimate calibrated: 3.6 characters per token on this text) | works | this commit |
| 12:02-12:14 | Baseline: `tuning/depth.py big 32 64 128` (direct to marvin; the Halogen download ran meanwhile) | 30,353 tokens: reading **360** / writing **48.0**; 62,101: **313** / **38.2**; 123,300: **267** / **32.5** tok/s (128K first token after 462 s). runs/2026-11-17-1214-depth-big.json | — |
| 12:40 | Download done: MODELS-OK, 9 files, sha256 checked, 112 GB in /srv/models/halogen (12:40:44) | — | — |
| 12:40 | Step 3 smoke: marvin idle (`big` slots not processing); `big-halogen` requested through marvin's llama-swap | **load + answer 27 s** ("ready"; 26 thinking tokens, closed by the 300-token budget). Log: version 0.20.6, our sampling and effort medium applied; weights 62.1 GiB pinned in 11.5 s (page cache from the download), KV pool 524,288 positions 14.4 GiB, working memory 12.4 GiB, 88.9 GiB in all, host memory left 30.6 GiB; lookup table read through the file cache; prompt cache on (28 entries, in place); MTP depth 2 (up to 7); engine listening after 16 s | — |
| 12:41 | Step 4 privacy: `tuning/cache-switch.py big-halogen 3` (A, B, A again; ~12K-token prompts, codewords) | **0 leaks**, every answer its own codeword. First token A 5.72 s, B 6.47 s, **A again 0.17 s** (59-61 tokens read: the cache resumes A after B; `big` with the leak fix re-reads it in full). runs/2026-11-17-1242-cache-switch-big-halogen.json | — |
| 12:43 | Concurrent check (inline script): 4 conversations at once (~12K tokens each, own codewords), first turn + follow-up, 3 rounds | **24/24 own codeword, 0 leaks** | — |
| 12:44 | Step 5: `lab speed big-halogen` (direct to marvin) | **83.6 writing / 2,143.6 reading 32K** (first token 15.2 s); `big` today 10:48: 45.6 / 328.2 → writing **+83%**, reading **6.5x**. runs/2026-11-17-1244-speed-big-halogen.json, MLflow c8ddb903 | — |
| 12:45-12:47 | Step 6: `tuning/depth.py big-halogen 32 64 128` | reading **2,145 / 2,118 / 2,007** tok/s at 30K / 62K / 123K (`big` 360 / 313 / 267: 6.0x / 6.8x / 7.5x); 123K first token **61 s** (`big` 462 s). Writing 50.7 / 54.7 / 46.4, but on 71-81 tokens (Halogen ended its one-sentence answer early; `big` wrote the full 200): not comparable with `big`'s 48.0 / 38.2 / 32.5. runs/2026-11-17-1247-depth-big-halogen.json | — |
| 12:48 | Step 7 started: `lab quality big-halogen --full --temperature 0.6` seeds 1-3, series `big-halogen-v1` (direct to marvin) | running | — |
| 12:47-13:13 | Step 7: `lab quality big-halogen --full --temperature 0.6` seeds 1-3 (series big-halogen-v1) | **93 / 93 / 91** (129, 129, 126 of 138; MLflow 19ead43e, ab69bbf5, 68ccd4af); `big` on 273 (2026-11-16): 96 / 91 / 93: **inside the seed spread**. Each run ~8.5 min (`big` 15-26 min). Different misses: Halogen fails TC-03 (contact lookup → email chain) in 2 of 3 (`big` 0 of 3) and is partial on TC-57 in 3 of 3 ("rejected the injection but reproduced attacker-controlled content"; `big` passes 3 of 3); `big` misses TC-30, 33, 59, 60 once each where Halogen passes; TC-45 runaway: Halogen 0 of 3 (1 s each), `big` 1 of 3 (662 s). Both fail TC-68 and are partial on TC-14 in every run | — |
| 13:14 | Back to `big` (one request) | swap 27 s; Halogen stopped cleanly in 1 s (launcher: container exited, gone; 0 engine containers); big ready; memory as before | — |
| 13:15 | **Result.** Halogen Flash serves the same model **1.8x faster writing and 6-7.5x faster reading** (128K first token 61 s vs 462 s), with tool use in `big`'s range and **0 leaks** with its prompt cache on (so a returning conversation resumes in 0.17 s, which `big` cannot do with the leak fix). Not tested: hours of agent use (its docs report hangs under host memory pressure), Terminal-Bench, vision. **Daniel decides** the next step; `big` stays the daily model until then | | this commit |
