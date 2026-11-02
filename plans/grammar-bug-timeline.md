# llama.cpp grammar bug (tools + required JSON format) timeline

America/Chicago time. Bug: Qwen3.8 (`coder`, `big`) and Ornith requests that send tools **and** a required JSON
format (`response_format` json_schema or json_object) get HTTP 400 "Failed to initialize samplers: failed to parse
grammar". Costs 4 tool-eval scenarios per run (TC-65, 66, 67, 69). Flint is unaffected (other chat template).

## Cause (found 2026-11-02, llama.cpp b12020 = 0ec4027, unchanged on master)

`common/parsers/qwen3-coder.cpp` line 194 (the handler for Qwen3-Coder-style templates: Qwen3.5/3.8, Ornith):

    data.grammar_lazy = has_tools && inputs.tool_choice == COMMON_CHAT_TOOL_CHOICE_AUTO;

With a response format the parser holds only the JSON-format rule (tools are left out by design). But the grammar
is still made "lazy" because tools exist, and a lazy grammar is built only from tool-call trigger rules: there are
none, so the grammar is `root ::= ` (empty) and llama.cpp refuses it. The general handler
(`chat-auto-parser-generator.cpp`) already uses `!has_response_format && ...`. Fix: the same check here.

Upstream: issue ggml-org/llama.cpp#31342 (open, no replies). PR #31395 has this exact fix (plus the same bug in
ministral3.cpp and deepseek.cpp, with tests); its author closed it 2026-11-01 without a review or a reason.

Side effect of the fix (same as the other handlers): with a response format set, that request can't call a tool;
the model always answers with the JSON.

## Steps

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 16:46 | `tuning/grammar-repro.py coder`: 7 small requests | schema alone OK; tools alone OK; tools + json_schema 400; tools + json_object 400; turn 2 (after tool result) + tools + schema 400; turn 2 schema without tools OK; tools + schema + `tool_choice: none` OK | — |
| 16:47 | Server log | grammar printed as `root ::= ` (empty), then `space ::= ...`: the empty root is the parse error | — |
| 16:47 | Read b12020 source | cause above | — |
| 16:47 | Repro variant 8: tools + schema + `tool_choice: required` | OK: confirms the lazy-grammar cause | — |
| 16:48 | Upstream search | issue #31342 open; fix PR #31395 closed by its author, unreviewed; master unchanged | 22b60f2 |
| 16:50 | Checked the 2026-11-02 reference to PR #31395: commit 5889d80 is a cherry-pick in another user's fork, not in any open upstream PR | no fix in flight upstream; next step (comment on #31342 and/or reopen the fix as a PR) is Daniel's call | this commit |
| 16:51 | Daniel: nothing upstream for now | keep the client workaround (tool_choice required/none with a format); re-test with `tuning/grammar-repro.py` after each llama.cpp update | this commit |

Workaround for clients now: send `tool_choice: "required"` or `"none"` together with a response format, or drop
`tools` from the request that asks for the JSON answer.
