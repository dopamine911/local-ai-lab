<!-- Experiment result (autoresearch #5 or by hand). Title: "experiment · <what> · <from> → <to> · <verdict>".
     Label: experiment. The "Experiment check" workflow fails if a section below is missing or still says TODO.
     This PR adds the result (tuning/runs/*.json and a tuning/results.md entry); it never changes production configs. -->
## Proposal
Closes #TODO

## Change
<!-- The one setting, before → after, and the test instance it ran on (machine, unit, port). -->
TODO

## Environment
<!-- From tuning/manifest.sh: machine, BIOS power mode, GPU/VBIOS, kernel, Mesa/ROCm, image digest, llama.cpp or engine
     commit, model file + sha256. Same for baseline and change, or say what differs. -->
TODO

## Results
<!-- At least 3 runs per side. Report every run, the mean and the spread (min-max). -->
| Metric | Baseline (runs) | Mean | Change (runs) | Mean | Difference |
| --- | --- | ---: | --- | ---: | ---: |
| TODO | | | | | |

## Quality guard
<!-- Tool-use score (tuning/quality subset) and/or KL (tuning/kl), baseline vs change. A speed gain only counts when
     these stay within their noise (tool use about ±4 between seeds). -->
TODO

## Power and temperature
<!-- GPU watts while busy (average, min-max) and peak temperature, both sides. Busy under 100 W on marvin = power mode
     problem: the run is invalid. -->
TODO

## MLflow
<!-- Run ids for every run, both sides. -->
TODO

## Verdict
<!-- One of: better / no change / worse / unclear, and one sentence why. "unclear" when the difference is inside the spread. -->
TODO

## Reverted
<!-- Confirm the test instance is back to the baseline and the reservation was released. -->
TODO

## Memory
<!-- The `recall` memory saved for this result (id), so the idea isn't tested again. -->
TODO
