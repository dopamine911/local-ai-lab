# Other inference engines and one place to manage them (plan, 2026-11-17)

Daniel asked how Lemonade would fit the lab, whether its server could run on zaphod and use marvin, whether it
could launch other projects (ComfyUI, Strata), and to look into Gufo, DS4 and Halogen, ideally with one frontend
to manage everything. Nothing is changed by this plan. Each step needs Daniel's go-ahead.
**Checked** = I looked it up or ran it today. **Claim** = the project's own numbers, not measured here.

## 1. The rule that decides where things run
- A model runs on the machine whose GPU or NPU it uses. Nothing on zaphod can run a model on marvin's GPU.
- The router uses almost no memory. Checked on marvin: llama-swap 85 MB; the `big` model process 29 GB of system
  RAM plus its GPU memory. Moving a router from marvin to zaphod frees nothing that matters.
- We already have "front door on zaphod, models on marvin": zaphod's llama-swap with marvin as a peer (`marvin/big`).
- Gufo, DS4 and Halogen Flash are all built for Strix Halo (gfx1151). They can only run on **marvin**.

## 2. What each project is (checked 2026-11-17)
| Project | What it is | Runs on | Licence | Fits how |
| --- | --- | --- | --- | --- |
| **Lemonade Server** (lemonade-sdk, v2026.46.2, 11-11) | A whole local AI server: its own router, model downloads, llama.cpp ROCm/Vulkan, speech, images, NPU (FastFlowLM), "container backends" | its own machine only | Apache 2.0 | Not as our router (section 4). Possibly as an NPU-only server on marvin |
| **llamacpp-rocm** (lemonade-sdk, nightly, b1379 on 11-17) | ROCm builds of `llama-server` for gfx1151 and gfx120X, runtime inside the archive | either machine | MIT | As a second binary inside our llama-swap images: issue #16 |
| **Halogen Flash** (peonist-ai, pushed 11-16) | An engine for exactly one model on one chip: **Qwen3.8-Flash-Next on Strix Halo**, i.e. our `big`. Closed binary in a container image, own weights (bring-your-own-GGUF also documented) | marvin | Proprietary EULA: free incl. commercial, **no telemetry, no outbound connections** (§5) | A llama-swap entry on marvin, compared with `big` |
| **Gufo** (gufo-org, pushed 11-17) | An engine for Strix Halo serving text, embedding, speech, image (video in progress), one chosen model per use, OpenAI-compatible | marvin | MIT | A llama-swap entry on marvin |
| **DS4 / DwarfStar 4** (antirez, pushed 11-08) | A C engine for DeepSeek V4 / V4.1 Flash (also Qwen3.8-Flash-Next and GLM 5.x per its site), Metal/CUDA/ROCm; Strix Halo container by kyuz0 | marvin | MIT | A llama-swap entry on marvin, for a model we don't run yet |
| **AI Toolbox Cockpit** (kyuz0) | A terminal app that installs and launches llama.cpp, DS4, Gufo, Halogen, Strata, ComfyUI, vLLM in Toolbx/Distrobox containers | the machine it runs on | **none stated** (all rights reserved) | Ideas only; not usable as-is (section 5) |

Halogen Flash's claims for Qwen3.8-Flash-Next on a Ryzen AI Max+ 395 (its README; synthetic prompts, greedy,
thinking off, its prompt cache on): writing 93 tok/s code / 67 prose; reading 1,870 tok/s at 32K and 1,793 at 128K;
an agent turn at 30K-96K context 2.6 s. Ours today for `big`: 45.4 writing, 329 reading at 32K. If even part of
that holds on our tests, it is the biggest gain on the list. Its kernel needs (Linux 6.18.4+ with
`CONFIG_HSA_AMD_SVM`): marvin has 7.0.0 with `CONFIG_HSA_AMD_SVM=y` (checked).

## 3. The shape: one router, engines as entries
```
clients (Open WebUI, omp, Enso, Claude Code, lab tests)
   │
zaphod llama-swap  ── zaphod engines: llama.cpp Vulkan (now), llama.cpp ROCm (#16), ComfyUI (later)
   │ peer
marvin llama-swap  ── marvin engines: llama.cpp Vulkan `big` (now), Halogen Flash, Gufo, DS4, llama.cpp ROCm,
                       Strata fork, NPU server (Lemonade/FastFlowLM)
```
- Each engine is one llama-swap model entry: a name, the command that starts it, how to stop it. Clients see every
  engine as one more model name; nothing changes for them.
- llama-swap's swap rules keep one big engine at a time on marvin (128 GB holds `big` or one other, not both).
- Every test we have works unchanged: `lab speed`, `lab quality`, the activity log, the watchdog, profiles.
- Test entries sit next to `big` (for example `big-halogen`, `big-gufo`, `big-rocm`); switching back = using
  `big`. One change at a time.

## 4. Why not Lemonade Server as the router
- It downloads from its own catalog; we pin every file by revision and sha256 (`setup/models-*.txt`).
- Its settings live in its JSON config, not in our `config.yaml`; its Ubuntu install is a PPA service started at boot.
- We would lose or rebuild: the `marvin/big` peer, the training and image-mode profiles, per-card routing, the
  `${server}` macro with the cache-leak flags, the activity log that `lab log` and agent-stats read.
- Its container backends are a fixed list it defines and pins (DS4, Halogen, Gufo, three llama.cpp forks), on its
  own machine only. ComfyUI and Strata are not on it; adding one means writing a backend into Lemonade.
- Its server on zaphod pointing at marvin ("cloud offload", experimental) would be a second front door, nothing more.

## 5. Decisions needed first
1. **How llama-swap starts container engines.** Halogen, Gufo, DS4, Strata and ComfyUI ship as container images.
   llama-swap runs in a container itself, so it cannot start a sibling container on its own.
   - A. Mount the Docker socket into llama-swap and use `docker run` / `docker stop` as the entry's commands.
     Simple, but gives llama-swap root-level control of that machine.
   - B. A small launcher on the host (like `hub/actions`), with a fixed list of allowed engines; llama-swap asks it
     to start or stop one. Safer, more work.
   - C. Run llama-swap on the host instead of in a container. Changes how both machines are built.
   - My lean: B, or A on marvin only if B is too slow to build. Unverified which is less work.
   - **Decided 2026-11-17 (Daniel): B.** Reason: the Docker socket stays with one small host service that only
     starts what the repo's allowlist names; llama-swap sends only an engine name. Build log:
     plans/engine-launcher-timeline.md.
2. **The frontend** (section 6).
3. **The order of engines** (section 7).

## 6. One place to manage it: options
| Option | What you get | Cost |
| --- | --- | --- |
| **llama-swap's own web UI** (in 273) | models per machine, load/unload, logs, activity; every engine shows once it is an entry | none; one UI per machine (zaphod's also shows peers) |
| **Our models panel** (`hub/models-panel`, built 2026-11-18, #64) | one page for both machines: what is loaded, start/stop an engine, card memory, last speed result from MLflow | build it (a small page behind Caddy, using llama-swap's API) |
| **`lab` menu** (terminal, exists) | the same actions from the terminal and for agents (`--json`) | add an "engines" screen |
| Lemonade App | manages only what Lemonade runs | doesn't see llama-swap |
| AI Toolbox Cockpit | a terminal app for exactly these engines | no licence stated; Toolbx/Podman; its own model folders |

My lean: engines as llama-swap entries first (step 1-3 below), so llama-swap's UI already shows them; then the
models panel as the one page for both machines, with `lab` doing the same from the terminal.

## 7. Steps (one change at a time; each needs Daniel's go-ahead)
1. **Decide 5.1** (launch method). Done 2026-11-17: B, built and tested on marvin (plans/engine-launcher-timeline.md:
   no leftovers in any failure test; a GPU engine as fast as native, 61.3 vs 61.7 calls/min).
2. **Halogen Flash vs `big` on marvin** (same model, the biggest possible gain). Tested 2026-11-17: see tuning/results.md
   and plans/halogen-timeline.md; daily use is Daniel's decision.
   - pin the image by digest and the weights by revision + sha256 (`setup/models-marvin.txt`);
   - check privacy first: its prompt cache and cache modes against our cache-leak rule (CLAUDE.md, #27148);
     two unrelated long prompts in a row must not leak;
   - `big-halogen` entry; `lab speed`, a depth test (0 / 32K / 64K / 128K), `lab quality --full` seeds 1-3.
   - Good result: reading and writing clearly faster, tool use within `big`'s seed spread (96 / 91 / 93), no leak.
2b. **Radiance on zaphod** (both R9700s; Daniel 2026-11-17): tested, see plans/radiance-timeline.md and tuning/results.md.
3. **llama.cpp ROCm (#16)**, updated for llama.cpp e574a17 (the matching llamacpp-rocm build: check its
   qualification file). Same tests.
4. **Gufo** (`big-gufo`, and maybe its embedding/speech models), same tests.
5. **DS4** with a model we don't run yet (DeepSeek V4 Flash: size and fit on marvin to check first).
6. **NPU on marvin** (Lemonade/FastFlowLM), a peer for small models or helpers. Driver loaded (checked:
   `amdxdna`, `/dev/accel/accel0`). Linux maturity, models, speed: unknown.
7. **ComfyUI and Strata as entries** (replacing `lab image-mode` and `fork.sh`), using the 5.1 method.
8. **Models panel**: one page for both machines (section 6). Built 2026-11-18 (#64, plans/models-panel-timeline.md); engines appear in it as llama-swap entries.

Each engine test records its numbers in tuning/results.md and MLflow like every other test, and its pins in
setup/VERSIONS.md.

## Risks
- Halogen Flash is a closed binary under an EULA (no telemetry, no outbound connections, per §5), from a small
  company: it could stop being published. Keep `big` on llama.cpp as the fallback whatever the result.
- Every engine has its own prompt cache; each one needs the privacy check before daily use.
- marvin's memory holds one big engine at a time; tests must unload `big` (marvin idle first).
- Disk: Halogen and DS4 bring their own weights (marvin has 1.6 TB free, checked).
- Vendor numbers are measured their way (greedy, thinking off, synthetic prompts); ours use real prompts.
