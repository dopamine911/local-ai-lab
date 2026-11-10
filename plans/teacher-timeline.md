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

