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
	return []*cobra.Command{log}
}
