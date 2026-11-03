# llama.cpp conversation leak (#27148): --cache-ram 0 --no-cache-idle-slots (issue #10) timeline

America/Chicago time. Upstream issue still open on 2026-11-02 (last activity 10-12), related PR #28058 not merged.
The fix needs both flags together (either alone still leaked upstream). Test: `tuning/cache-switch.py` (conversation
A, then B, then A again, ~12K-token prompts with a codeword each; time to first token and a leak check).

## zaphod, `coder` (card 1, -np 1)

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 18:57 | Trial: switch test, 1 round, current config | works: every answer its own codeword, 0 leaks. Second A: 12,116 of 12,159 prompt tokens restored from the RAM cache, 43 read, first token after 2.8 s (first A: 13.5 s for 12,120 tokens) | — |
| 18:57-19:02 | **Before** (current config): `lab speed coder`; switch test 3 rounds | speed **53.8** tok/s writing, **770.9** reading 32K (MLflow d1bee8db). Switch: first A 15.1 s (median; 12,119 tokens read), B 16.8 s, **A again 2.74 s** (42 tokens read, the rest from the RAM cache); own codeword every time; **0 leaks** (the leak is intermittent upstream: 0 here proves nothing) | — |
| 19:02 | Both flags added to zaphod's `${server}` macro (every llama-server model: coder, coder-2, coder-tb, ornith, chat, north, flint, embed, rerank, coder-long; not the speech servers) | llama-swap reloaded (the known stt/tts preload 404s only); running llama-server processes (coder, embed, rerank) show both flags | 131ffeb |
| 19:03-19:08 | **After**: `lab speed coder`; switch test 3 rounds | speed **54.2** writing (+0.7%, noise), **821.2** reading 32K (**+6.5%**: today's 4 earlier runs were 765-771, so outside the noise; likely cause: with the RAM cache on, each finished prompt was copied to RAM, ~2.6 GB per request per the log "making room for prompt cache entry", and that cost time; not measured separately). MLflow 3b7a0edc. Switch: first A **13.6 s** (was 15.1), B 15.5 (was 16.8), **A again 13.9 s** (was 2.74; 12,157 tokens read again): the cost is ~11 s per return to a 12K-token conversation after another took the slot. Own codeword every time; 0 leaks | — |

## marvin, `big` (-np 2 -kvu)

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 19:08-19:18 | **Before** (current config): `lab speed marvin/big`; switch test 3 rounds | speed **44.2** writing, **320.0** reading 32K (MLflow 2122c874). Switch: first A 34.9 s (12,120 tokens read), B 39.6 s, **A again 1.02 s** (43 read: kept in its own slot of 2 or restored from RAM); 0 leaks | — |
| 19:18 | Both flags added to marvin's `${server}` macro; marvin pulled, llama-swap reloaded and preloaded `big` | the running llama-server has both flags | d13ca9b |
| 19:19-19:29 | **After**: `lab speed marvin/big`; switch test 3 rounds | writing **44.3** (same). Reading 32K **298.2** (**-6.8%**, outside ±3%; today's 3 earlier: 318.4-320.0). Per run: **319.0, 298.2, 297.5** (before: 321.1, 320.0, 313.9): the first long prompt after the reload is normal, every later one ~7% slower. Switch: first A 35.9 s (was 34.9), B 40.7 (was 39.6), A again 0.81 s (A kept its own slot of 2: no switch cost here); 0 leaks. MLflow 75df3001 | — |

Hypothesis (not tested): `big` runs 2 slots with one shared (unified) KV cache. Before, `--cache-idle-slots` moved an
idle slot's tokens to RAM and cleared the slot; with `--no-cache-idle-slots` the previous 32K prompt stays in the shared
cache, and every new long prompt is processed alongside it (attention over more cells). That fits: first run normal,
later runs slower by the same amount. zaphod's `coder` has 1 slot, so nothing stays behind there (+6.5% instead).
Decision for Daniel: keep (privacy first, ~7% slower long prompts on marvin), test a variant (e.g. slots not unified, or
1 slot), or roll marvin back.

## Decision and records

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 19:31 | Daniel: keep the flags on marvin as they are (privacy first, ~7% slower long prompts there) | tuning/results.md section, CLAUDE.md Known issues "mitigated" (and "not only under concurrent load"), README known issues | this commit |
| 19:32 | Daniel: a new issue as the reminder: **#34** (remove the flags, measured, once llama.cpp fixes #27148). #10 left open for Daniel to close | README "What's next", FEATURES plans row and CLAUDE.md point to #34 | this commit |
