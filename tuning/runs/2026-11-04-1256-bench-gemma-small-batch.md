# llama-bench, Gemma 4 31B QAT Q4_0, zaphod card 2 (Vulkan0), card empty, 2026-11-04 12:55:56-12:56:12
# image llama-swap-zaphod:269-media (llama.cpp b12020); -ngl 999 -fa on -p 1,2,4,8,16,70 -n 22 -r 3
| test | t/s |
| ---: | ---: |
| pp1 | 28.57 ± 0.05 |
| pp2 | 50.12 ± 0.13 |
| pp4 | 71.09 ± 0.08 |
| pp8 | 87.40 ± 0.20 |
| pp16 | 188.34 ± 16.79 |
| pp70 | 664.65 ± 64.85 |
| tg22 | 30.55 ± 0.01 |

# Server tests, test llama-server on port 8099 inside the llama-swap container, card 2 alone (helpers unloaded), 12:57-13:15
# Client: synthetic teacher-like calls, shared ~1,020-token start + ~110 new tokens, exactly 22 written, temp 0.7
# (script kept in the session scratchpad; same shape as the trace script, not the builder's prompt).
| test | settings (all: Gemma 31B QAT Q4_0, -fa on, Vulkan0) | 1 worker | 4 workers | 8 workers | card 2 memory |
| --- | --- | ---: | ---: | ---: | --- |
| A | chat-teacher as configured: -c 32768 -np 8, f16 | 26.1/min | 27.1 | 34.6 | 29.2 GB, no spill |
| B | A + -kvu | 26.3 | 32.5 | 17.8 | 26.5 GB |
| C | -c 4096 -np 1 | 26.3 | - | - | - |
| D | C without --cache-ram 0 --no-cache-idle-slots | 26.4 | - | - | - |
| E | C without --reasoning off | 26.5 | - | - | - |
| F | C, cache_prompt false vs true | fresh 1,133 tokens read in 1.9-2.1 s; reused start + ~120 new: 2.1 s (no saving) | | | |
| G | C + --swa-full | **68.4** (reuse read 110 tokens in 135 ms; fresh 1,133 in 968 ms) | - | - | - |
| H | A + --swa-full (32K) | 36.0 | 50.3 | 49.2 | 31.9 GB + 15.1 GB spilled to system memory |
| I | -c 16384 -np 4 --swa-full -ctk q8_0 -ctv q8_0 | 66.7 | **145.1** | 143.0 | 24.6 GB, no spill |
