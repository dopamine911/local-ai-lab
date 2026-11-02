package bench

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// seriesPy: tuning/series's Python, word for word (its output is the output to match).
const seriesPy = `
import mlflow, os, statistics as st, collections
mlflow.set_tracking_uri("http://127.0.0.1:5000"); c = mlflow.MlflowClient()
groups = collections.defaultdict(list)
for r in c.search_runs(["1"], "tags.series != \"\"", max_results=1000):
    if os.environ["FILTER"] in r.data.tags["series"]:
        groups[r.data.tags["series"]].append(r)
keys = [("tooleval_score_common", "score"), ("sort_wrong", "wrong"), ("sort_honest_stop", "honest stop"), ("sort_fabrication", "made up"),
        ("behavior_q4_claim_sent", "Q4 claim"), ("behavior_attacker_echo", "attacker echo"), ("behavior_looping", "looping"),
        ("behavior_invented_query", "made-up query"), ("live_writing_tps", "writing tok/s"), ("live_reading_tps", "reading tok/s"),
        ("live_draft_acceptance_pct", "draft accept %")]
for name, runs in sorted(groups.items()):
    print(f"== {name}  ({len(runs)} runs)")
    for k, label in keys:
        v = [r.data.metrics[k] for r in runs if k in r.data.metrics]
        if v:
            print(f"   {label:14} average {st.mean(v):6.1f}   lowest {min(v):4.0f}   highest {max(v):4.0f}   spread {max(v)-min(v):3.0f}")
`

var warnLine = regexp.MustCompile(`(?i)warn`)

// Series is `lab series [filter]` (port of tuning/series): count, average, lowest, highest of every MLflow series
// (tag "series") whose name contains filter. Read-only.
func Series(cfg *config.Config, filter string) error {
	c := exec.Command("docker", "run", "--rm", "--network", "host", "-e", "GIT_PYTHON_REFRESH=quiet", "-e", "FILTER="+filter,
		cfg.Images.MLflow, "python", "-c", seriesPy)
	out, rc := runQuiet(c)
	for _, l := range strings.SplitAfter(out, "\n") {
		if l != "" && !warnLine.MatchString(l) {
			os.Stdout.WriteString(l)
		}
	}
	if rc != 0 {
		return exitErr(rc)
	}
	return nil
}
