# marvin `big`: why writing speed falls (theories, 2026-11-04)

No changes made. Each theory has its evidence and a test. Test one at a time, only when Daniel says marvin is free.

## What we see (marvin's log, real use, `tuning/marvin-log-speeds.py`)

| Conversation size | 128K setting (11-02/03) | 256K setting (since 11-03 21:19) |
| --- | --- | --- |
| short (benchmark) | ~41-44 tok/s | not measured yet |
| 30-60K | 32.1 | 27.3-27.9 |
| 60-90K | 28.8 | 25.2-25.9 |
| 90K+ | 27.4 | 23.3-24.7 |
| both slots writing at once | (overflowed instead) | ~14.5 each |

MTP draft acceptance stayed 0.55-0.62 throughout. No swap use. Power while writing 89-125 W, 2,300-2,880 MHz.

## Model facts (GGUF header, qwen4exp)

48 layers, only 12 with attention memory (`full_attention_interval` 4); 2 KV heads, key/value length 256; plus an
indexer (4 heads, 128). Memory per token of conversation: about 24-27 KB in f16 (~3.5 GB per 128K, matching the
3.8 GB measured at load). Active weights per written token: very roughly 3 GB (10 of 512 experts).

## Theories

1. **Long conversations cost more per token (expected; ~80% sure it is the main cause).** Each written token reads
   the active weights (~3 GB) plus the conversation memory (~2.5 GB at 100K). marvin is limited by memory speed, so
   100K roughly doubles the bytes per token: from ~41 to the mid-20s, as seen. Test: `lab speed` with prompts of 1K,
   32K, 64K, 100K on an idle marvin; the speed should fall along that line.
2. **The shared cache (`-kvu`) makes each slot also read the other slot's tokens (~55%).** At the same own size, the
   256K setting is ~10% slower, although draft acceptance went up. Since the switch the idle slot almost always holds
   a 90K+ conversation (before, both together were capped at 128K). The log can't separate the two, because the other
   slot was nearly always full. Test: same 64K prompt with the other slot empty, then with it holding ~100K; then,
   if it matters, `-c 262144 -np 2` without `-kvu` (each slot its own fixed 128K, same memory).
3. **Two requests at once share the GPU (certain, by design).** ~14.5 each, ~29 together. omp running a sub-task
   next to the main conversation causes it. Only fewer parallel jobs (or `-np 1`) avoid it.
4. **Conversation memory in f16 instead of q8_0 (idea for a gain, ~60% that it helps at 60K+).** q8_0 halves the
   memory read per token (zaphod's 256K models already use it). Risk: small quality change; must be measured
   (`lab speed` at long prompts, plus `lab quality`/tool-use). Also frees ~1.7 GB per 128K.
5. **Not the cause (ruled out by evidence):** MTP acceptance falling (it stayed ~0.6); the 55 W power fault (89-125 W,
   power mode performance); swapping (0 used); the privacy switches from #10 (they affect reading, not writing).
6. **Open, unrelated to speed: marvin uses ~11 GB more since the 256K switch** (Beszel, 2026-11-04 08:30 check):
   - **GPU memory +5.4 GB** (72.8 → 77.6 GB at the switch, 78.2 after, flat since): conversation memory ~3.5 GB
     (GGUF math) plus, likely, larger working buffers for the longer context (~1.9 GB, not proven). Expected.
   - **The other ~5-7 GB is not GPU memory.** System use then 110.5-113.4 GB, moving ±1.5 GB (at 128K: flat
     101.5). llama-server's own memory was steady at 35.7 GB over 50 s with both slots busy, so it is not per-request
     allocation. No number from before the switch exists, so it is unknown whether that 35.7 GB grew.
   - Theories: (a) per-slot snapshots of the non-rewindable layers (up to 32 per slot, one every 8,192 tokens; ~118 MB
     each by GGUF math, so ~2.8 GB for 105K + 117K), but the log shows no snapshot lines (~35%); (b) the memory
     allocator keeping memory from the largest prompts (~30%); (c) kernel bookkeeping for GPU memory (~20%).
   - Test without a config change: at the next `big` restart, record llama-server's RssAnon and Beszel GPU memory right
     after load and after the first long conversation. Not urgent: flat for 10 h, ~9 GB free.

## Suggested order

Theory 1 test first (it also gives the 256K "after" baseline), then 2, then 4. One change at a time, `lab speed`
before and after, results in tuning/results.md and the timeline.
