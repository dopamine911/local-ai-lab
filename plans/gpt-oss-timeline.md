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
| 13:31 | Flags checked in marvin's image (b12020) | `--reasoning-effort`, `--chat-template-kwargs`, `--jinja` (on by default), `--spec-type draft-eagle3` all present | `llama-server --help` |
| 13:36 | Config entry `gpt-oss` written: `-c 131072 -np 1`, temp 1.0, top-p 1.0, reasoning effort medium, through `${server}` (keeps the cache-leak fix); added to the training profile pins | not deployed yet | this commit |
