# Models panel (#64)

One page for both machines: https://models.ai.dopamine911.com (tailnet only). `index.html` is the whole panel: plain
HTML, CSS and JS, no outside files. Caddy serves this folder and adds the API keys (hub/caddy/Caddyfile, `@models`).

What it shows, refreshed every 5 s:
- zaphod's two cards (memory, temperature, busy, watts) and marvin's shared memory (llama-swap `/api/performance`;
  marvin's GPU uses system memory, so its own 512 MB VRAM figure is not shown);
- the active profile (training, image mode), if any;
- every model, loaded first, with a Load or Unload button and its latest `lab speed` result from MLflow.

Load = one GET to `/upstream/<model>/health`: llama-swap starts the model and unloads what its routing sets say cannot
run next to it. Unload = POST `/api/models/unload/<model>`; it first reads `/upstream/<model>/slots` and warns if the
model is answering a request (llama-server only). Both ask first.

Caddy passes only these calls, anything else under `/zaphod/`, `/marvin/`, `/mlflow/` is 404 (no chat through the
panel's keys): GET `running`, `v1/models`, `api/performance`, `api/profiles`, `upstream/*/health`, `upstream/*/slots`;
POST `api/models/unload/*`; POST `mlflow/runs/search` (Origin header dropped: MLflow refuses other sites' Origin).
Anyone on the tailnet who opens the page can load and unload models (Daniel, 2026-11-18).

Good result: the page lists both machines with card/memory bars, and Unload then Load of `tts` takes a few seconds.
Record: plans/models-panel-timeline.md.
