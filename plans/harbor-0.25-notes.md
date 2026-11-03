# harbor 0.20.0 → 0.25.0: what changes for our Terminal-Bench runs (issue #2, step 1)

Read 2026-11-02: the GitHub release notes of v0.21.0 (08-10), v0.22.0 (08-22), v0.23.0 (09-12), v0.24.0 (10-05) and
v0.25.0 (10-25), 585 lines. Our setup: Terminus-2 agent through LiteLLM to an OpenAI-compatible endpoint, local
Docker, Core-19 tasks with their own tests, pass@2 (a second attempt as its own job).

## Nothing found that changes how we are graded
- No change listed to the Terminus-2 agent itself (prompt, loop, turn limits), to task timeouts, or to how Core-19's
  tests decide pass/fail. The only Terminus line is CI-only (0.25 #3564).
- Most of the five releases is RewardKit (LLM-judged grading; Core-19 doesn't use it), the Hub, cloud sandboxes
  (Modal, E2B, Daytona, ...) and other agents (Codex, Goose, Hermes, pi, ...).

## Fixes on the path we use (could change a few outcomes; size unknown without a run)
- 0.23 #2985 docker: stream stdout without a line-length limit (a command printing a very long line).
- 0.25 #3692 remove stale reward files before verification (a leftover result file can't be read as this run's).
- 0.25 #3551, #3552, #3720 LiteLLM model connections reworked; explicit API keys honoured (how Terminus-2 reaches
  llama-swap; most likely to need a smoke test).
- 0.25 #3580 docker: keep host credentials when a task changes HOME; #3677 absolute paths for compose cp.
- 0.22 #2580 docker: egress-control probe.

## Upstream (kyuz0/terminal-bench-mini, the suite we pin)
Still `HARBOR_VERSION = "0.20.0"` at its latest commit 60739af (2026-10-27): held through all five harbor releases
since July. No issue or PR about a harbor upgrade (its one open issue is about podman-compose). It publishes
versioned Core-19 suites, i.e. it freezes the runner on purpose.

## Test: smoke before and after (Daniel, 2026-11-02)

One task (`git-leak-recovery`, pass@2) on marvin `big` (TARGET=marvin, same model as the 2026-11-01 baseline), first
on harbor 0.20.0, then 0.25.0 (only `HARBOR_VERSION` in tuning/terminal-bench/terminal_bench.py changes). Each run has
its own profile (`mtp3-128k-np2-h020-smoke`, `...-h025-smoke`) so nothing merges with the full-run results. One task
checks that the chain works on 0.25 (LiteLLM rework) and compares time and tokens; it can't show a score change.

| Time | Action | Result | Commit |
| --- | --- | --- | --- |
| 17:59 | doctor on 0.20.0 (marvin big); smoke 0.20.0 started | doctor passed: Harbor 0.20.0, 131,072-token context | — |
| 18:02 | smoke on 0.20.0 | **1/1**, first attempt, 3.3 min (17:59:14-18:02:32), 14,660 input / 1,794 output tokens; MLflow 893f6157 | — |
| 18:02 | `HARBOR_VERSION` 0.20.0 → 0.25.0 (only change; test); doctor | doctor passed: Harbor 0.25.0 (uvx), 131,072-token context | — |
| 18:02 | smoke on 0.25.0 | **failed after 9 s, before any inference** (exit 1): the suite's own prompt hook (`compat/harbor_prompt/sitecustomize.py`, upstream code) reads `Terminus2._parser_name`, which 0.25.0 no longer has: `AttributeError: 'Terminus2' object has no attribute '_parser_name'`. Nothing logged | — |
| 18:03 | `HARBOR_VERSION` back to 0.20.0 (terminal_bench.py identical to upstream again). Compared Terminus-2's code in both versions (uv cache) | see "What the code shows" | this commit |

## What the code shows (Terminus-2, harbor 0.20.0 vs 0.25.0)
Correction to "No change listed to the Terminus-2 agent" above: the release notes don't list it, but its code changed
(448 lines in 2 files).
- **Same behaviour as far as the code shows:** prompt template files byte for byte the same; same defaults (temperature
  not set = the server's, parser json, summarize on at 8,000 free tokens, 160x40 terminal, no turn limit, no interleaved
  thinking).
- **Reorganised:** the settings moved into one `options` object (`self._parser_name` → `self.options.parser_name`):
  this is what breaks the suite's prompt hook. Model connection (base URL, API key) goes through new LiteLLM helpers.
- **tmux:** more fallbacks when installing tmux/asciinema inside a task container; not how the agent uses tmux.
- So the suite needs a code change to run on 0.25 at all; upstream can't bump harbor without changing its hook, and our
  results on a patched hook would no longer be the published suite's.

## Decision (Daniel, 2026-11-02): wait for upstream
Keep harbor 0.20.0 until Terminal-Bench-Local moves to a newer harbor (it must change its prompt hook to do so).
Then: `lab tbench setup` on the new pin, smoke, fresh 19-task baseline, results.md notes which harbor each row used.
Check at each version audit: `HARBOR_VERSION` in kyuz0/terminal-bench-mini's terminal_bench.py (setup/VERSIONS.md).
