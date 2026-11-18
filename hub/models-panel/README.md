# Models panel (#64)

One page for both machines: https://models.ai.dopamine911.com (tailnet only). `index.html` is the whole panel: plain
HTML, CSS and JS, no outside files. Caddy serves this folder and adds the API keys (hub/caddy/Caddyfile, `@models`).

Layout (2026-11-18, Daniel: by card; less text, one thing at a time, easy to find), refreshed every 5 s:
- one machine after the other; zaphod has a box per card (card 1, card 2 side by side; on a phone stacked), then a box
  for models on both cards and one for tests; marvin has one box (its GPU shares the system's memory);
- each box: free memory (meter, one block per GB), the models loaded there (lamp, name, Unload), and "Can load here";
- click a model's name: its details open right under its box (Load / Unload with the warning in plain view, last
  speed, context, the same model on other cards, and "Show config": routing sets, full command, config block).
  Refreshing waits while details are open. The active profile (training, image mode) shows above the boxes.

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
Text 18 px (17 on a phone), names bold near-black; card temperature and power are left to the watchdog and Beszel.

Good result: the page lists both machines with card/memory bars, and Unload then Load of `tts` takes a few seconds.
Record: plans/models-panel-timeline.md.
