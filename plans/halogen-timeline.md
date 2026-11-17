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
