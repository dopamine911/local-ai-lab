package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dopamine911/local-ai-lab/cli/internal/bench"
	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// benchCmds: the benchmark commands (issue #6 step 4). Each keeps its script's MLflow fields, so results compare.
func benchCmds() []*cobra.Command {
	log := &cobra.Command{
		Use:   "log <machine> <model> <note> [name=value ...] [files ...]",
		Short: "Log one result to MLflow and notify the phone (port of tuning/log)",
		Long: "Log one result to MLflow (tuning/log_run.py in the MLflow image) from the current folder, then send the\n" +
			"phone notification with Rerun/Skip/Open buttons. Same MLflow fields as tuning/log.\n" +
			"Settings read from the environment: SERIES, SKILL_NAME, SKILL_SHA, RUN_START (adds live_* speed from\n" +
			"llama-swap's request log), NOTIFY=0 (no notification), MLFLOW_URL (tests only: another MLflow).\n" +
			"Good result: \"Logged run <id>\" and a notification \"Done: <model> (<machine>)\".",
		DisableFlagParsing: true, // name=value and file arguments pass through untouched, like the script's "$@"
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 || args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			rc, err := bench.Log(cfg, args[0], args[1], args[2], args[3:])
			if err != nil {
				return err
			}
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
	series := &cobra.Command{
		Use:   "series [filter]",
		Short: "Average, lowest, highest of repeated runs (MLflow tag series; port of tuning/series)",
		Long: "Summarize repeated runs of one setup (MLflow tag \"series\"): count, average, lowest, highest, spread.\n" +
			"  lab series            every series\n" +
			"  lab series marvin     only series whose name contains \"marvin\"\n" +
			"Start a series: SERIES=<name> lab quality <model> \"<note>\" --full --seed 1 (same name for every repeat).\n" +
			"Read-only. Good result: one block per series; if the gap between two settings is smaller than the spread\n" +
			"of their own repeats, treat them as the same.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			f := ""
			if len(args) == 1 {
				f = args[0]
			}
			return exitWith(bench.Series(cfg, f))
		},
	}
	return []*cobra.Command{log, series}
}

// exitWith exits with a benchmark's own exit code (bench.ExitError); other errors go to cobra.
func exitWith(err error) error {
	if e, ok := err.(bench.ExitError); ok {
		os.Exit(e.Code)
	}
	return err
}
