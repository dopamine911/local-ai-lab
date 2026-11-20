# coder context: 64K 16-bit → 128K 8-bit (#65 step 2)

Daniel 2026-11-20: try `coder` at 128K with an 8-bit (q8_0) context first; move nothing else.

## Why
- `coder` (Qwen3.8-27B UD-Q4_K_XL, card 1) has 64K today; omp and enso cap it at 64K.
- Only 16 of its 64 layers keep a per-token cache (file: `full_attention_interval` 4, 4 KV heads × 256):
  about 64 KB per token at 16-bit, 34 KB at 8-bit. 64K 16-bit ≈ 4.0 GB, 128K 8-bit ≈ 4.3 GB, 128K 16-bit ≈ 8.0 GB.
- Card 1, 2026-11-20 10:30 (driver per-process counters): `coder` 21.5 GB, `embed` 5.6, `rerank` 1.9, free 2.8 GB.
  128K 16-bit does not fit beside `embed` + `rerank`; 128K 8-bit should.
- Quality: q8_0 context on this file measured the same as 16-bit (KL 0.0088 both, 2026-10-28, tuning/results.md).
  Open: speed (a quantized cache can be slower on Vulkan) and tool use.

## The one change
`coder` line only: `-c 65536` → `-c 131072 -ctk q8_0 -ctv q8_0`.

## Steps
1. zaphod and marvin idle (a reload cuts marvin requests through the peer).
2. Before: `lab speed coder`, `tuning/depth.py coder 32 56`.
3. Change, commit, reload; card 1 memory with `embed` + `rerank` loaded.
4. After: `lab speed coder`, `tuning/depth.py coder 32 56 120`.
5. Only if speed passes: `lab quality coder --full --temperature 0.6` seeds 1-3 (before 2026-11-16: 96 / 93 / 92).
6. Then Daniel: keep or roll back; if kept, `coder-2` the same way, then the omp and enso limits.

Good result: writing within ±3%, 32K reading within ±5%, tool use within 92-96, card 1 at least 1 GB free.
Roll back: the old line (`-c 65536`).
If 8-bit is slower: shrink `embed`/`rerank` (`-c`/`-ub` 8192 → 2048) to make room for 128K 16-bit (check their input sizes first).

## Timeline
| When | What | Result |
| --- | --- | --- |
| 10:36-10:40 | Step 2, before (64K 16-bit): `lab speed coder`, `tuning/depth.py coder 32 56`; zaphod and marvin idle | **54.1 writing / 937.4 reading 32K** (shorts 53.9-54.3, longs 936-942); depth 30K: 939.8 / 48.5, 54K: 869.3 / 46.7. runs/2026-11-20-1036-speed-coder.json (MLflow 792e2f8f), runs/2026-11-20-1040-depth-coder.json |
| 10:41 | Step 3: `coder` `-c 65536` → `-c 131072 -ctk q8_0 -ctv q8_0`, saved (auto-reload) | **Reload cut a running `coder-2` streaming request** (someone's, since about 10:15; my idle check ran a few seconds earlier and showed `coder-2` free; `coder` showed busy from my own depth test's last request). `coder` failed to preload ("couldn't bind HTTP server socket", the old process still held port 5805); loaded on the next request ("17 x 23" → 391). All 7 models back |
| 10:42 | Card 1 memory (driver counters) | `coder` 21.52 → **22.15 GB** (+0.63), `embed` 5.55, `rerank` 1.94; card 1 30.4 of 32.6 GB, **2.2 GB free** (bar: 1). n_ctx 131072. Card 2 unchanged |
| 10:41-10:48 | Step 4, after (128K 8-bit): `lab speed coder`, `tuning/depth.py coder 32 56 120` | Speed test: **53.6 writing (-0.9%) / 904.8 reading 32K (-3.5%)** (longs 908 / 845 / 905: the middle one dipped). Depth: 30K 910.0 / **45.2** (reading -3.2%, writing **-6.8%**), 54K 830.5 / **43.7** (-4.5%, **-6.4%**), 120K (new, 64K could not) 662.5 / 36.1, first token 174 s. The speed test passes; writing with a long context (agent use) is about 6.5% slower, outside ±3% (the 64K depth numbers were 49.0 / 48.0 on 11-17 and 48.5 / 46.7 today: spread about 1 tok/s; one request per depth). runs/2026-11-20-1041-speed-coder.json (MLflow 8b745a13), runs/2026-11-20-1048-depth-coder.json. **Stopped before tool use: Daniel decides.** 128K 8-bit is live meanwhile |
| 14:49 | Daniel: 128K is worth it if tool use holds; step 5 started: `lab quality coder --full --temperature 0.6` seeds 1-3, series `coder-128k-q8-v1` | started |
| 14:49-16:02 | Step 5 done | **93 / 93 / 93** (129 of 138 each; MLflow 7fca3d53, a30718c0, bf6c5d10), mean 93.0 vs 64K 16-bit 96 / 93 / 92 (mean 93.7, 2026-11-16): inside the 92-96 bar. Behaviour: attacker echo 0, made-up query 0, looping 0-1, Q4 claim 1 (the same CFO-email scenario as before). Live writing 54.0 tok/s (short prompts), draft acceptance 68-82%. **Passes.** Next (Daniel): `coder-2` the same change, then omp, enso and `pi` to 131072 for both |
| 16:02 | Daniel: yes (step 6: `coder-2` the same, then client limits and docs) | — |
| 16:03 | Before: `lab speed coder-2` (64K 16-bit) | **59.2 / 980.7**. runs/2026-11-20-1603-speed-coder-2.json, MLflow 24935470 |
| 16:04-16:06 | Waited for `coder`, `coder-2` and marvin `big` idle 5 checks in a row (~10 s; `big` was busy with an Open Design session, its requests pass through zaphod), then saved `coder-2` `-c 65536` → `-c 131072 -ctk q8_0 -ctv q8_0` at 16:06:35 | Reload clean: no request cut, all 7 models ready 16:07:05 (stt/tts preload 404: known, harmless). n_ctx 131072 |
| 16:07 | Card 2 memory | `coder-2` 21.52 → **22.15 GB**, speech 3.7; card 2 26.5 of 32.6 GB, **6.0 GB free**. Card 1 unchanged (2.2 free) |
| 16:07-16:10 | After: `lab speed coder-2` | **58.5 (-1.2%) / 946.6 (-3.5%)**, same pattern as `coder`. runs/2026-11-20-1607-speed-coder-2.json, MLflow d1948e81 (its config tag says b28f06e: logged before this commit). Tool use not re-run: same model, file and setting as `coder` (93 / 93 / 93) |
| 16:10 | Clients: omp `agents/omp/models.yml` and enso `config/models.json` (enso has only `coder`) → 131072, names 128K; enso deployed with `lab enso setup`; omp needs a copy to `~/.omp/agent/models.yml` where it runs. CLAUDE.md and tuning/results.md updated | Daniel's `pi` file: `coder` and `coder-2` 131072. **Done** |
