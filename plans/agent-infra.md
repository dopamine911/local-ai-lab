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

### 1. A coding agent on the local models: Enso  (decided 2026-10-29)
The coding agent is Daniel's own **Enso** (github.com/dopamine911/enso-harness): a debugging and control layer around
a coding agent, built on **pi** (TypeScript/Bun). It already has guards (bash policy, writes confined to the workspace,
read gating, read-before-write), guarded `web_fetch`/`web_search`, and a daily JSONL log ("receipts"). Not a sandbox,
by its own README. No other agent will be evaluated unless Enso can't do something needed.

How it meets the lab (checked in pi's docs, not tried yet):
1. **Models**: pi reads custom OpenAI-compatible providers from `models.json` (`"api": "openai-completions"`,
   `baseUrl`, `apiKey` from an environment variable such as `${ZAPHOD_KEY}`, per model `"reasoning": true` and
   `"input": ["text", "image"]`). One provider entry for zaphod's llama-swap (`http://127.0.0.1:8080/v1`) covers
   `coder`, `ornith`, `chat` and `marvin/big` (marvin through zaphod). The key never goes in the file.
2. **Web search**: Enso's `web_search` uses the model's own search API, else Tavily (paid, cloud). Local models have
   no search API. Option (Daniel's code, his call): a SearXNG backend for Enso's `web_search`, keeping search local.
3. **Tools**: pi's docs don't mention MCP (it extends through its own extensions; check before relying on it), so the
   MCP gateway (step 3) may reach Enso only through an extension; Enso's guarded tools stay the primary path.
4. **Evidence**: Enso's logs (`~/.enso/logs/enso-*.jsonl`) can feed MLflow later (step 6, profiling).

Steps, one at a time, each with a good result:
1. pi `models.json` with the llama-swap provider (no key in the file). Good: `/model` lists coder, ornith, chat, marvin/big.
2. One small real task with `coder` (fast, zaphod) and the same with `marvin/big`. Good: finished, receipts in the log.
3. Note what breaks or is missing for local models (tool-call format, reasoning output, context limits, speed).
4. Then the own-task test (step 2) runs through Enso.

### 2. A test made of Daniel's own tasks
10-20 tasks he would normally give a frontier model (code changes, debugging, writing, research), each with a short
"what a good answer looks like". Run on each local model through Enso; Daniel scores them (pass / partly / fail).
Results in MLflow (`benchmark = own-tasks`), with the same manifest, series and real-use speed as other runs.

Good result: one number per model, "local handled N of 20 of my tasks", repeated after every model change.

### 3. One place for tools: an MCP gateway  — STARTED 2026-10-29
Chosen: **MCPJungle 0.4.6** (free, MPL-2.0; "enterprise mode" is just its locked-down mode): no Docker socket, one
token + allowed-server list per agent, SQLite in hub/data (backed up). Each tool in its own container. First tool:
`search` (mcp-searxng 2.6.0, token-protected, blocks private URLs). Rejected: Docker MCP Gateway (needs Docker
control), MetaMCP (Postgres, maintenance delays), ContextForge (too heavy). Playwright added 2026-10-29: own container on an internal network (no route out), Squid egress proxy refusing
private/home/Tailscale/Docker/zaphod addresses, profile in memory, read-only. Tested: public pages load; router,
marvin, SearXNG, Open WebUI refused (403). Chromium itself also refused a public->private redirect. Not yet seen:
the proxy catching a followed redirect (needs our own public test server). Chromium contacts Google in the background.

Tools (search, files, memory, browser) are defined once; every agent connects to the gateway: Open WebUI, the step-1
agent, Claude Code. Candidates *(check)*: Docker MCP Gateway, MetaMCP.
Rules: runs on zaphod, 127.0.0.1 or Tailscale only, each tool in its own container with only the access it needs,
keys in Proton Pass.

Good result: adding a tool once makes it available in every agent.

To do (Daniel 2026-10-30): `hub/mcp/status.sh`, read-only, one screen: registered tool servers, their tools, and
which agent (client) may use which. MCPJungle's web dashboard (docs.mcpjungle.com/guides/dashboard) is Beta and
"only supported when running mcpjungle in development mode" (no per-agent tokens or allowed lists, no login
described); our 0.4.6 in enterprise mode answers 404 there. Re-check at each MCPJungle update: use it only once it
works in enterprise mode with a login, behind Caddy on Tailscale, and is at least 2 weeks old.
Good result: one command shows every tool and who may use it, with no secrets printed.

### 4. Shared memory: LocalRecall (or similar)
github.com/mudler/LocalRecall (MIT, ~1k stars, no releases yet: pin an image digest). Uses zaphod's `embed` model.
Indexes `~/homelab` and `~/full-homelab` from Git automatically, served to agents through the gateway.
Set `API_KEYS` and a database password (Proton Pass); never the example defaults. Listens on 127.0.0.1 only:
the private repo's contents are inside it.

Good result: an agent asked "how did we fix the slow card?" finds `problems/2026-10-28-slow-gpu-card.md`.

### 4b. Memory from conversations: mem0 (candidate, Daniel 2026-10-29)
github.com/mem0ai/mem0 (Apache 2.0). Different job from LocalRecall: an LLM reads conversations and keeps facts and
decisions ("Daniel prefers...", "we decided..."), so every agent knows them next session. LocalRecall searches documents.
Checked 2026-10-29, conditions before use:
1. Telemetry is ON by default (PostHog, us.i.posthog.com: OS version, processor, model and store names; start/reset
   events always, others sampled 10%). Set `MEM0_TELEMETRY=false` AND run it on an internal Docker network with no
   route out (like plans/strix-strata.md), so a missed setting fails closed.
2. Default LLM and embedder are OpenAI (`gpt-5-mini`, `text-embedding-3-small`): point both at llama-swap (`embed` +
   a chat model); the sealed network stops a fallback to OpenAI.
3. No official MCP server in its README; community ones (e.g. elvismdev/mem0-mcp-selfhosted) need a code review first.
   No GitHub releases shown: pin a commit or a PyPI version at least 2 weeks old.
4. Every memory write is a model call: measure load on zaphod and how well the local model picks what to keep.
Not this: docs.mem0.ai/platform/mem0-mcp is the hosted Platform's MCP (`https://mcp.mem0.ai/mcp`, memories in a Mem0
account, not on our machines). Useful only as the tool list a self-hosted memory server should offer: add_memory,
search_memories, get_memories, get_memory, update_memory, delete_memory, delete_all_memories, list_entities,
delete_entities, list_events, get_event_status.
Not this either: OpenMemory (mem0.ai/openmemory). The self-hosted OpenMemory MCP server + UI was removed from the
mem0 repo on 2026-07-29 (commit ea2ee07586, "remove OpenMemory from the monorepo"); the name `mem0ai/openmemory` is now
a Beta CLI that moves coding sessions between Claude Code/Codex/OpenCode (install by curl | bash). The page itself does
not say where memories are stored; it has pricing and a plugin, so treat it as the hosted product unless shown otherwise.
So: mem0 library or its self-hosted `server/`, sealed, on our models, plus a reviewed or own small MCP server.
Good result: tell one agent a rule; a different agent, new session, follows it without being told.

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

1. Step 1 = Enso on local models (decided). First change: pi `models.json` provider for llama-swap.
2. Which 10-20 tasks go into the own-task test (step 2)?
3. Who else will ever be on the tailnet? Decides how urgent step 9.1 is.
