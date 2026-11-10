# Timeline: improving the pocket-terrarium trace teacher (2026-11-09)

Daniel: look at the original project (github.com/mediacutlet/pocket-tank) and see how to make the teacher faster
and better. Allowed: a Gemma 4 26B-A4B candidate and chat-teacher's server settings. Lab side only: the project code
(`~/pt-train`, a copy of the builder's project, not a git repo) is not changed; things it fixes (temperature 0.7,
48 tokens, thinking off) are findings. A better teacher means new labels and retraining: the builder's call.
Pilot result files (they hold the project's prompt) stay in `~/pt-train/lab-pilot/`, not in this public repo.
Times are Chicago.

## From the original project (docs/stats.md, retrain-v3/v4.md, pipeline.md)

- Teacher: gemma4:26b (Q4_K_M) via Ollama on a Mac Mini; 36-51 decisions/min with 2-3 workers (1.2-1.7 s each).
- Teacher agrees with itself (same 40 states twice): 82%, the "practical ceiling"; student vs teacher 72-75%.
- Biggest quality factor: the prompt ("prompt is DNA": one sentence made 45% of labels follow_friend);
  `prompt_check.py` (diagnostic panel, ~7 min) before any long run.
- Our lab teacher `chat-teacher` is Gemma 4 31B dense (not the 26B), 4 slots of 4K, thinking off, --swa-full.

| Time | Step | Result | Record |
| --- | --- | --- | --- |
| 18:20 | Orientation: `~/pt-train/probe-img/model` is the newer copy (gen_traces.py 11-05 13:04: an extra "solitary lizard" rule in the prompt); pilot.py, pilot_l1.jsonl, eval.py identical in both copies | | `cmp`, `diff` |
| 18:29 | Baseline: `pilot.py --model chat-teacher` (37 L1 edge states x k=3, 4 workers), twice | agreement **99.1% / 97.3%**; 0 failures; **103 / 156 decisions/min** (run 1 includes the model load); latency median 1.50 / 1.53 s. Self-agreement: goal **96.4% / 93.7%**, goal+urgency **75.7% / 82.9%**; prompt cache 92-95%. The pilot is near its ceiling: it can show a worse teacher, hardly a better one | `~/pt-train/lab-pilot/baseline-chat-teacher-{1,2}.json` |
| 18:31 | Gemma 4 26B-A4B QAT Q4_0 pinned (google rev d1c082b, 14.4 GB, no mmproj) and downloading; entry `chat-teacher-26b` prepared = chat-teacher with only the model changed; hidden in Open WebUI | | this commit |
| 18:33 | Baseline 2: `prompt_check.py --schema L1 --model chat-teacher --workers 4` (the original's ~7-min prompt/teacher panel; 464 calls) | 3 min 9 s (~147/min), 0 failures. **All acceptance checks OK except one: "max single goal, content" = explore 0.46 (want < 0.45)**, the attractor the original also fought. Report only: P(hunt, hunger 0-2, prey near) 0.58. Per-case mixes kept in the log | `~/pt-train/lab-pilot/promptcheck-chat-teacher.{log,json}` |
| 18:38 | Gemma 26B downloaded (`MODELS-OK`); `chat-teacher-26b` applied (reload), loaded | load 16 s; answers without thinking; card 2 19.6 GB used (chat-teacher: 28.4), no spill | 79a2e64 |
| 18:40 | pilot.py on chat-teacher-26b, twice | agreement **80.2% / 81.1%** (31B: 97-99%); 0 failures; **243 / 245 decisions/min**, latency 0.93-0.99 s (31B: ~150/min, 1.5 s). Self-agreement goal 91.0% (31B 94-96%), goal+urgency 58-64% (31B 76-83%) | `~/pt-train/lab-pilot/cand-26b-{1,2}.json` |
| 18:41 | prompt_check on chat-teacher-26b (464 calls, 2 min) | **5 BAD** (31B: 1): A7 sleep → rest/hide 0.62 (want ≥ 0.85); A11 bold 9 vs bold 0 +0.00 (want ≥ 0.15); r8 hatchling vs adult hide +0.00; r3 shed soon vs no +0.00; content attractor explore 0.87 (want < 0.45). The personality differences the student should learn are mostly gone. **Verdict: 1.6x faster, clearly worse; not a replacement** with this prompt (tuned on the 31B; a 26B-tuned prompt is project work) | `~/pt-train/lab-pilot/promptcheck-26b.{log,json}` |
| 18:43 | chat-teacher `-np 4 → 8` (same 16K total: 8 slots of 2K; n_slots 8 / n_ctx_slot 2048 confirmed), card 2 28.3 GB, no spill | | d1d03be |
| 18:44 | pilot k=6 (222 calls) at 4 and at 8 workers | 4 workers **154 / min**, latency 1.47 s, 99.1%; 8 workers **163 / min (+6%)**, latency 2.77 s, 100%. The card is the limit (31B dense). **Reverted** (+6% not worth losing the 4K room per request) | `~/pt-train/lab-pilot/np8-w{4,8}.json` |
| 18:47 | Gemma 4 31B MTP drafter ("-assistant"): only community GGUFs (no google / ggml-org / unsloth), llama.cpp support for its drafter type not confirmed: not tried | | HF search |

