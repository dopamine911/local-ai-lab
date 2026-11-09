# Timeline: gpt-oss-120b test on marvin (2026-11-09)

Daniel asked for a benchmark of gpt-oss-120b. Times are Chicago, from `date` / logs.

Why marvin: the file is 63.4 GB. zaphod's two cards give at most ~62-63 GB with every other model off
(free 13:20: card 2 28.9 GB, card 1 3.0 GB), so zaphod would need ~6-10 GB of the model in RAM and both cards to itself.
marvin fits it fully, but `big` is unloaded while gpt-oss runs (llama-swap there runs one model at a time).

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 13:20 | Hugging Face lookup | ggml-org/gpt-oss-120b-GGUF rev 7200b74: `gpt-oss-120b-MXFP4.gguf` 63,387,346,208 bytes; also EAGLE3 draft files (Q8_0 0.85 GB, BF16 1.6 GB), for a later separate test | HF API |
| 13:22 | marvin download speed sample | 500 MB in 15.0 s = 34.8 MB/s (Wi-Fi), so ~30 min for the file (one sample) | curl |
| 13:25 | Pinned the file in `setup/models-marvin.txt` | | this commit |
| 13:26 | Download started on marvin (`lab models marvin gpt-oss`, tmux `gptoss`, log `~/gpt-oss-download.log`) | ~46 MB/s (12.6 GB by 13:30) | |
| 13:30 | Flags checked in marvin's image (b12020) | `--reasoning-effort`, `--chat-template-kwargs`, `--jinja` (on by default), `--spec-type draft-eagle3` all present | `llama-server --help` |
| 13:31 | Config entry `gpt-oss` written: `-c 131072 -np 1`, temp 1.0, top-p 1.0, reasoning effort medium, through `${server}` (keeps the cache-leak fix); added to the training profile pins | not deployed yet | this commit |
| 13:50 | Download finished, sha256 checked | `MODELS-OK` (13:51:04) | `~/gpt-oss-download.log` |
| 13:51 | Config copied to marvin's checkout (scp; `git checkout --` before the next pull), reload | reload OK, `big` preloaded again; marvin idle (0 requests in 10 min) | llama-swap log |
| 13:52 | gpt-oss loaded by one small request | ready in ~25 s; 1 slot, n_ctx_slot 131072; `big` unloaded; 56 GB RAM available | log, `/running` |
| 13:52 | `lab speed gpt-oss` (3 short + 3 x 30.4K-token prompts) | **writing 53.2 tok/s** (53.2-53.3), **32K reading 558.2 tok/s** (557-559), first token 54.6 s. `big` for comparison (11-02, 128K): 44.3 / 298-318 | MLflow fe70683c, `tuning/runs/2026-11-09-1352-speed-gpt-oss.json` |
| 13:56 | `lab quality gpt-oss` core 15 (temp 0, seed 42) | **97 (29/30)**; only miss TC-14 partial (recovered after a tool error, didn't say so); behavior flags all 0. **Not comparable with `big`**: every `big` run is the full 69 (88-95) | MLflow a6cd16b2, `tuning/tool-eval/runs/2026-11-09-1356-gpt-oss.json` |

Next after the gpt-oss tests (Daniel, 2026-11-09): Mesa 26.0.8 → 26.2.6 (kisak PPA; Ubuntu has nothing newer) in
`marvin/llama-swap/Dockerfile`, one change, `lab speed` on `big` and `gpt-oss` before and after. Save the .debs at
build time (PPAs drop old versions). Expected gain small: reading 0-6%, writing ~0% (plans/strix-performance-review.md #3).
