# Plans: index

One line per file, with its status. Open work lives in [GitHub issues](https://github.com/dopamine911/local-ai-lab/issues)
(the one backlog); a plan here holds the reasoning, a timeline the record of what was done and when.
Status words: **active** (work going on), **open** (ideas or steps not started), **waiting** (on Daniel or upstream),
**on hold**, **done** (kept as the record), **reference** (how-to). Last checked 2026-11-17.

## Active
| File | What | Status |
| --- | --- | --- |
| [engines-and-frontend.md](engines-and-frontend.md) | Other inference engines through the launcher, one place to manage them | active: launcher done, Halogen tested, Radiance next; Gufo, DS4, NPU, ROCm (#16), panel later |
| [halogen-timeline.md](halogen-timeline.md) | Halogen Flash vs `big` on marvin | active: Terminal-Bench running; daily use is Daniel's decision |
| [radiance-timeline.md](radiance-timeline.md) | Radiance on zaphod's two R9700s | active: prepared, tests after Terminal-Bench |
| [audit-2026-11-17.md](audit-2026-11-17.md) | Project audit: gaps, visibility, performance | done; its items are issues #44-#52 (and #13, #30) |
| [renovate-timeline.md](renovate-timeline.md) | Every Renovate update deployed and checked | ongoing log |
| [agent-infra.md](agent-infra.md) | Infrastructure for every agent: gateway, memory, tracing, reliability, hardening | partly done (gateway, memory, recall); the rest open |

## Waiting or on hold
| File | What | Status |
| --- | --- | --- |
| [pocket-tank-pr-draft.md](pocket-tank-pr-draft.md) | PR text for the pocket-tank project, not posted | waiting on Daniel |
| [teacher-timeline.md](teacher-timeline.md) | Faster pocket-tank trace teacher | paused since 2026-11-10 |
| [harbor-0.25-notes.md](harbor-0.25-notes.md) | Terminal-Bench runner 0.25 fails in its prompt hook | waiting on upstream (#2) |
| [strix-strata.md](strix-strata.md) | Strata on marvin, then our UnbiasedStrata port to gfx1151 | on hold (fork built, 19/19 Terminal-Bench) |
| [strix-strata-port-instructions.md](strix-strata-port-instructions.md), [strix-strata-port-notes.md](strix-strata-port-notes.md), [strix-strata-timeline.md](strix-strata-timeline.md) | The port: instructions, code map, record | on hold with it |
| [maya.md](maya.md) | Bigger models (GLM-5.3-Flash) on marvin | on hold (after the port) |

## Open ideas
| File | What | Status |
| --- | --- | --- |
| [marvin-writing-speed.md](marvin-writing-speed.md) | Why `big`'s writing slows with long context: theories and a test each | open (none tested) |
| [strix-performance-review.md](strix-performance-review.md) | Ranked ideas for marvin speed | partly tested (gpt-oss `-ub`, Mesa 26.2: +1%, rolled back); `-ub` on `big`: #45 |

## Reference
| File | What |
| --- | --- |
| [gpt-oss-notes.md](gpt-oss-notes.md) | How to run gpt-oss on zaphod and marvin, best settings, what was measured |

## Done (the record)
| File | What |
| --- | --- |
| [engine-launcher-timeline.md](engine-launcher-timeline.md) | The engine launcher (option B), built and tested on both machines (2026-11-17) |
| [llama-cpp-v1-timeline.md](llama-cpp-v1-timeline.md) | llama.cpp 1.0 feature review and the 269 → 273 upgrade on both machines (2026-11-16) |
| [grammar-bug-timeline.md](grammar-bug-timeline.md) | Tools + JSON-format grammar bug: cause found, fixed upstream in 1.0 |
| [cache-leak-timeline.md](cache-leak-timeline.md) | Prompt-cache conversation leak workaround (`--cache-ram 0 --no-cache-idle-slots`); removal waits on #34 |
| [gpt-oss-timeline.md](gpt-oss-timeline.md) | gpt-oss-120b on marvin and zaphod (test model) |
| [llama-swap-269-timeline.md](llama-swap-269-timeline.md) | llama-swap 267 → 269 on both machines |
| [cli-timeline.md](cli-timeline.md) | The `lab` CLI (#6; step 6 still open there) |
| [notifications-timeline.md](notifications-timeline.md) | Phone notifications with buttons (#11) |
| [watchdog-timeline.md](watchdog-timeline.md) | The watchdog (#12; the peer check is left: #47) |
| [switch-move-timeline.md](switch-move-timeline.md) | Moving both machines to the network switch (#14 item 2) |
| [marvin-fan-timeline.md](marvin-fan-timeline.md) | marvin fan control |
| [strata.md](strata.md), [strata-test/](strata-test/) | Strata tests on zaphod (its model files deleted 2026-11-17, re-downloadable) |
| [more-models.md](more-models.md) | Vision, chat, document search, speech, images |
| [morning-guide-2026-10-29.md](morning-guide-2026-10-29.md) | One night's summary and steps for Daniel |
