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
