#!/usr/bin/env bash
# omp: keep subagents off marvin, one at a time (run on the machine where you use omp; needs models.yml from this folder).
#
# Why (2026-11-04): omp's defaults allow 32 subagents at once, and subagents use the same model as the main
# conversation unless roles are set. 3-4 conversations then shared marvin's 2 slots: in one hour marvin read
# 450K prompt tokens and wrote only 29K, and writing fell to 0.4-5 tok/s (plans/marvin-writing-speed.md, theory 7).
#
# After this:
#   - main conversation (role default) and reviewer (role slow): marvin/big
#   - task, scout, sonic subagents (roles task and smol): coder-2 on zaphod card 2 (64K, 1 slot); since 2026-11-17
#     (issue #51): card 2 writes 9% faster (58.9 vs 54.0 tok/s) and reads 6% faster than card 1 (PCIe 3.0 x4). Loading
#     chat, ornith or a teacher on card 2 unloads it until the next subagent request reloads it; in image mode llama-swap
#     answers coder-2 requests with coder (card 1). Back to card 1: replace lab/coder-2 with lab/coder below.
#   - at most 1 subagent at a time (coder-2 has 1 slot; 2 would swap it and re-read), and subagents can't start their own
# Tested 2026-11-04 in an empty omp folder: @task and @smol both answered from coder (zaphod activity log, status 200).
# Note: coder and coder-2 reject thinking "off"/"minimal" (Qwen template: low, medium, xhigh only); omp's agents use medium/auto.
#
# Check: omp config list --scope global      Undo: omp config reset <key>
set -euo pipefail
omp config set task.maxConcurrency 1
omp config set task.maxRecursionDepth 1
omp config set modelRoles '{"default":"lab/marvin/big","slow":"lab/marvin/big","task":"lab/coder-2","smol":"lab/coder-2"}'
omp config list --scope global
