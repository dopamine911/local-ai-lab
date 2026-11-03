# llama.cpp conversation leak (#27148): --cache-ram 0 --no-cache-idle-slots (issue #10) timeline

America/Chicago time. Upstream issue still open on 2026-11-02 (last activity 10-12), related PR #28058 not merged.
The fix needs both flags together (either alone still leaked upstream). Test: `tuning/cache-switch.py` (conversation
A, then B, then A again, ~12K-token prompts with a codeword each; time to first token and a leak check).

## zaphod, `coder` (card 1, -np 1)

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 18:57 | Trial: switch test, 1 round, current config | works: every answer its own codeword, 0 leaks. Second A: 12,116 of 12,159 prompt tokens restored from the RAM cache, 43 read, first token after 2.8 s (first A: 13.5 s for 12,120 tokens) | — |
| 18:57-19:02 | **Before** (current config): `lab speed coder`; switch test 3 rounds | speed **53.8** tok/s writing, **770.9** reading 32K (MLflow d1bee8db). Switch: first A 15.1 s (median; 12,119 tokens read), B 16.8 s, **A again 2.74 s** (42 tokens read, the rest from the RAM cache); own codeword every time; **0 leaks** (the leak is intermittent upstream: 0 here proves nothing) | — |
