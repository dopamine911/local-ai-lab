# llama-swap 267 → 269 (issue #1) timeline

America/Chicago time. zaphod's switch is recorded in tuning/results.md (2026-11-02, before 13:15).

## 2026-11-02, marvin (step 2)

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 14:07-15:10 | Before on 267: `tuning/speed marvin/big`, then `quality --full --temperature 0.6` seeds 1-3 (series `big-ls267`) | 44.6 / 319.6 tok/s; tool use 89 / 95 / 92; 4 grammar failures per run | — |
| 15:21 | Image `llama-swap-marvin:267` → `:269` (Mesa pinned 26.0.8-1ubuntu0.3), built and started on marvin | `big` loads | c28e833 |
| 15:23-16:39 | After on 269: same speed test and 3 seeds (series `big-ls269`) | 44.3 / 318.4 tok/s (-0.7% / -0.4%); tool use 94 / 91 / 88; still 4 grammar failures per run | — |
| 16:40 | Results recorded (results.md, VERSIONS.md, CLAUDE.md, marvin/SETUP.md row 20); issue #1 comment | Speed within ±3%; seed 3 is 1 point under the before range, mean 92.0 → 91.0, seed swings 4-6: kept on 269 | ac96a3a |
| 17:52 | Issue #1 closed (Daniel): all five steps done on both machines; title corrected to 267 → 269; summary comment with both machines' numbers and the grammar-bug cause | — | — |
