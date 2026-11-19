# Results page: timeline

Benchmark results from MLflow on one page, next to the models panel. Times are Chicago time, from `date`.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 2026-11-18 17:40 | Daniel: an improved results page, using MLflow's data. Surveyed MLflow: 175 runs in `llama-tuning`: speed 82 (13 models), tool use 76 (12), Terminal-Bench 11 (7), KL 2, card history 4; every run has a note and its config commit | — | — |
| 17:42 | Daniel chose: all four jobs (current numbers, history per model, before / after, compare models); one test at a time plus a side-by-side comparison; next to the panel (/results) | — | — |
| 17:45 | Facts for the page: tool-use suite sizes differ (138 now, 130 older, 30 quick: only 128+ compared); seeds grouped by the `series` tag; 8 speed runs are old llama-bench card runs (no writing number); tool-eval category names A-O from a report | — | — |
| 17:50 | Shared look moved to `hub/models-panel/common.css` (fonts, colours, base text); both pages link each other. Caddy: `try_files {path} {path}.html` so /results serves results.html; validated, reloaded | /results, /common.css, fonts: 200 | — |
| 17:55 | `results.html` written: tabs; latest per model with bars; history chart and run list with notes; tick two for side by side (change in green/red by which way is better) | browser test: no errors, no sideways scroll at 390 px; the panel unchanged after the CSS move | — |
| 17:58 | First test found: Terminal-Bench counted 1-task smoke runs (`big` showed "1 of 19" from Nov 2, hiding its 18/19 full run): now full 19-task runs only; KL showed nothing (its "model" is a file, not a config entry): KL skips the "in use now" filter | Terminal-Bench: big 18/19, coder-radiance 17/19, big-halogen 17/19; KL 1 row; before / after on coder speed: reading +22.0%, first token -18.0%; coder vs coder-radiance tool use: 93.7 (92-96) vs 94.3 (93-96) | — |
| 18:02 | The history chart misled: its scale ran 53.8-55.3, so coder's 1% noise looked like big jumps, and same-day points overlapped. Now the scale starts at 0 (KL keeps a tight one) and points are evenly spaced. Added a choice of number (speed: writing, reading, first token; tool use: score, median turn; Terminal-Bench: tasks, time, real-use writing; KL: mean, same top token), used by the bars and the chart | coder writing: flat ~54 since Nov 2; coder reading: two steps (cache flags Nov 2, llama.cpp 1.0 Nov 16) to ~937; 0 errors | this commit |
