# Plan: infrastructure that helps every agent

Status: draft, 2026-10-29. Nothing here is installed yet. Tool names marked *(check)* are candidates I know of but
have not researched for this setup: check current version, security and local-model support before choosing.

## Goal

Supplement frontier models, not replace them: "when they pull the rug, I want one hand out." Today's frontier plans
are subsidized and their limits are already tightening. So the local stack should take a growing share of everyday
work, and frontier models stay for the hard parts. How big that share is gets **measured on Daniel's own work**,
not assumed from public benchmarks.

What follows from that:
- Reliability beats speed tuning: a fallback has to work on the day it's needed.
- Everything goes through one daily workflow, so it gets used and tested every day, not only in benchmarks.
- Same rules as the rest of the lab: pinned versions, listening on 127.0.0.1 or Tailscale only, secrets in Proton Pass,
  added to `setup/` (rebuildable), one change at a time, measured before and after.

## What exists today

| Need | Have | Gap |
| --- | --- | --- |
| Models | llama-swap on zaphod (coder, Ornith, Gemma chat, speech, embed/rerank) and marvin (Flash-Next, via peer) | Covered |
| A daily coding agent on local models | None | **Biggest gap** (step 1) |
| Knowing how much local can cover | Benchmarks (tool-eval, Terminal-Bench) | Daniel's own tasks (step 2) |
| Tools shared by all agents | Per app (Open WebUI has SearXNG, Open Terminal) | One place for tools (step 3) |
| Shared memory / docs | Open WebUI Knowledge (chat only) | For every agent (step 4) |
| Reading web pages, browser | SearXNG search only | Page fetch/crawl, browser (step 5) |
| Seeing what agents did | MLflow (benchmarks, real-use speed) | Live sessions (step 6) |
| Using local and frontier together | Manual choice | Routing easy work to local (step 7) |
| Survives failures | Rebuild scripts, nightly backup | Failure tests, wired network (step 8) |

## Steps (in order; each one finished and measured before the next)

### 1. A coding agent that runs on the local models
Pick one open-source coding agent that speaks the OpenAI API and works well with llama-swap.
Candidates *(check)*: OpenCode, Aider, Qwen Code, Goose, Crush, Codex CLI with a local endpoint.
1. Research: which handle tool calling with Qwen3.8 / Ornith / Flash-Next well, which are maintained, how each
   handles context limits (marvin 128K shared by 2 slots, coder 64K, coder-tb 256K).
2. Try the top 2 on the same 3 real tasks from Daniel's repos; keep the one that finishes them.
3. Config in the repo (no keys: `ZAPHOD_KEY` from `~/llama-swap/.env`), install steps in `setup/`.

Good result: Daniel can open a terminal and do a real coding task with `coder`, `ornith` or `marvin/big`.

### 2. A test made of Daniel's own tasks
10-20 tasks he would normally give a frontier model (code changes, debugging, writing, research), each with a short
"what a good answer looks like". Run on each local model through the step-1 agent; Daniel scores them (pass / partly / fail).
Results in MLflow (`benchmark = own-tasks`), with the same manifest, series and real-use speed as other runs.

Good result: one number per model, "local handled N of 20 of my tasks", repeated after every model change.

### 3. One place for tools: an MCP gateway
Tools (search, files, memory, browser) are defined once; every agent connects to the gateway: Open WebUI, the step-1
agent, Claude Code. Candidates *(check)*: Docker MCP Gateway, MetaMCP.
Rules: runs on zaphod, 127.0.0.1 or Tailscale only, each tool in its own container with only the access it needs,
keys in Proton Pass.

Good result: adding a tool once makes it available in every agent.

### 4. Shared memory: LocalRecall (or similar)
github.com/mudler/LocalRecall (MIT, ~1k stars, no releases yet: pin an image digest). Uses zaphod's `embed` model.
Indexes `~/homelab` and `~/full-homelab` from Git automatically, served to agents through the gateway.
Set `API_KEYS` and a database password (Proton Pass); never the example defaults. Listens on 127.0.0.1 only:
the private repo's contents are inside it.

Good result: an agent asked "how did we fix the slow card?" finds `problems/2026-10-28-slow-gpu-card.md`.

### 5. Reading the web: page fetch and a browser
Clean page text for agents (candidate *(check)*: Crawl4AI) and a browser they can drive (Playwright MCP *(check)*),
both behind the gateway.

Good result: an agent can read a GitHub issue or a docs page it found with SearXNG.

### 6. Recording live agent sessions (tracing)
Every prompt, tool call and answer from real use, in MLflow (MLflow 3 tracing) or Langfuse *(check)*. Needs a small
proxy in front of llama-swap, because neither llama-swap nor most agents send traces themselves.
Feeds the planned model-profiling and prompt-transformer project.

Good result: pick any session from yesterday and see what each step sent and got back.

### 7. Local and frontier together
Send routine work to local models now, keep frontier for the hard parts: stretches today's frontier limits and builds
the habit before it's forced. Start manually (per task), automate later only if step 2 shows clear task types that
local handles reliably.

### 8. Reliability: survive failures (from the ChatGPT review, P3/P4)
1. Wire zaphod and marvin to the MikroTik switch (bought; `~/full-homelab/devices/switch`). Measure zaphod → marvin
   before (Wi-Fi: 1 GB in 97 s) and after.
2. Failure tests, each written down in `~/full-homelab/problems/` with what happened:
   marvin offline mid-request, a model fails to load, llama-swap restart during a session, disk full, a corrupt model
   file (`setup/models.sh` must catch it).
3. Two slots on marvin share one 128K context: test two agents at once with long contexts.

### 9. Hardening (from the ChatGPT review, P1)
1. Models panel: Caddy adds the API keys, so any tailnet device can use the models fully. Fine while the tailnet holds
   only Daniel's devices; add a login before anyone else joins. Same for MLflow (no login).
2. Beszel agent: test which extra permissions are really needed (`SYS_ADMIN`, `SYS_RAWIO`, Docker socket, AppArmor off),
   remove the rest one at a time, checking disk health and service status still show.

### Later
- Local Git server (Forgejo *(check)*) where agents push branches and run checks without touching GitHub.
- NAS rebuild: shared storage, model archive, backups (next project after marvin).

## Decisions for Daniel

1. Step 1 first? (Recommended: yes; everything else is measured through it.)
2. Which 10-20 tasks go into the own-task test (step 2)?
3. Who else will ever be on the tailnet? Decides how urgent step 9.1 is.
