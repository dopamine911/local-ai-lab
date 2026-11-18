# Models panel (#64)

One page for both machines: https://models.ai.dopamine911.com (tailnet only). `index.html` is the whole panel: plain
HTML, CSS and JS, no outside files. Caddy serves this folder and adds the API keys (hub/caddy/Caddyfile, `@models`).

What it shows, refreshed every 5 s:
- zaphod's two cards (memory, temperature, busy, watts) and marvin's shared memory (llama-swap `/api/performance`;
  marvin's GPU uses system memory, so its own 512 MB VRAM figure is not shown);
- the active profile (training, image mode), if any;
- every model, loaded first, with its card, a Load or Unload button and its latest `lab speed` result from MLflow;
- **details** per model: card, context, engine, what it can do, routing sets it is in, other entries with the same model
  file (how to get it on another card), the full command (llama-swap's own while loaded; otherwise the config's with the
  macros filled in), and its config block with comments. Refreshing pauses while a model's details are open.

Card: each entry runs on one card, fixed in its command (`--device`: llama.cpp calls card 1 Vulkan1, card 2 Vulkan0).
There is no "load on card N": load the entry for that card (`coder` / `coder-2`). Most models have one entry only.

Load = one GET to `/upstream/<model>/health`: llama-swap starts the model and unloads what its routing sets say cannot
run next to it. Unload = POST `/api/models/unload/<model>`; it first reads `/upstream/<model>/slots` and warns if the
model is answering a request (llama-server only). Both ask first.

Caddy passes only these calls, anything else under `/zaphod/`, `/marvin/`, `/mlflow/` is 404 (no chat through the
panel's keys): GET `running`, `v1/models`, `api/performance`, `api/profiles`, `upstream/*/health`, `upstream/*/slots`;
POST `api/models/unload/*`; POST `mlflow/runs/search` (Origin header dropped: MLflow refuses other sites' Origin);
GET `config/zaphod.yaml`, `config/marvin.yaml` (the llama-swap config files, mounted read-only into Caddy from the repo;
public, `${env.X}` references only). marvin's is zaphod's repo copy: marvin runs its own checkout, the same after a pull.
Anyone on the tailnet who opens the page can load and unload models (Daniel, 2026-11-18).

Design (2026-11-18): an instrument panel. Type: Atkinson Hyperlegible Next (text) and Mono (commands, config), by the
Braille Institute, chosen because its letters are easy to tell apart; served from `fonts/` (OFL 1.1, licences there;
@fontsource 5.3.0 woff2, latin, 35 KB), so the page loads nothing from outside. Colours: pale aluminium base, graphite
ink, one teal for the meters, lamp green / amber / red only for state; dark mode the same roles. The one bold element:
memory meters with one block per GB, so free memory can be counted. Unload is a quiet button that turns red on hover.

Good result: the page lists both machines with card/memory bars, and Unload then Load of `tts` takes a few seconds.
Record: plans/models-panel-timeline.md.
