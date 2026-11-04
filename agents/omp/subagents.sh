#!/usr/bin/env bash
# omp: keep subagents off marvin, one at a time (run on the machine where you use omp; needs models.yml from this folder).
#
# Why (2026-11-04): omp's defaults allow 32 subagents at once, and subagents use the same model as the main
# conversation unless roles are set. 3-4 conversations then shared marvin's 2 slots: in one hour marvin read
# 450K prompt tokens and wrote only 29K, and writing fell to 0.4-5 tok/s (plans/marvin-writing-speed.md, theory 7).
#
# After this:
#   - main conversation (role default) and reviewer (role slow): marvin/big
#   - task, scout, sonic subagents (roles task and smol): coder on zaphod card 1 (64K, 1 slot)
#   - at most 1 subagent at a time (coder has 1 slot; 2 would swap it and re-read), and subagents can't start their own
# Tested 2026-11-04 in an empty omp folder: @task and @smol both answered from coder (zaphod activity log, status 200).
# Note: coder rejects thinking "off"/"minimal" (Qwen template: low, medium, xhigh only); omp's agents use medium/auto.
#
# Check: omp config list --scope global      Undo: omp config reset <key>
set -euo pipefail
omp config set task.maxConcurrency 1
omp config set task.maxRecursionDepth 1
omp config set modelRoles '{"default":"lab/marvin/big","slow":"lab/marvin/big","task":"lab/coder","smol":"lab/coder"}'
omp config list --scope global
