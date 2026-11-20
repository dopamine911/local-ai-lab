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
