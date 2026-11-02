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
	speed := &cobra.Command{
		Use:   "speed <model> <note>",
		Short: "Writing and 32K reading speed, medians of 3, logged (port of tuning/speed)",
		Long: "Speed test any OpenAI-compatible server the same way (llama.cpp, Strata, ...): fixed prompts, medians of 3.\n" +
			"  short: code prompt (~40 tokens in), 400 out, temp 0: writing tok/s (best case for MTP)\n" +
			"  long:  ~32K-token prompt (Python standard library source), 32 out: reading tok/s = prompt / time to first token\n" +
			"  lab speed coder \"baseline\"                                    zaphod's llama-swap (ZAPHOD_KEY)\n" +
			"  MACHINE=marvin BASE_URL=http://marvin:8099/v1 API_KEY=... lab speed <model> \"<note>\"   (MACHINE: who served it)\n" +
			"Writes tuning/runs/<date>-speed-<model>.json and logs speed_* to MLflow (NOLOG=1 skips logging).\n" +
			"Good result: three short and three long lines, then \"median: writing N tok/s; 32K reading N tok/s\" and \"Logged run\".",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return exitWith(bench.Speed(cfg, args[0], args[1]))
		},
	}
	quality := &cobra.Command{
		Use:   "quality <model> <note> [--full] [tool-eval-bench options]",
		Short: "Tool-use quality test (tool-eval-bench), logged (port of tuning/quality)",
		Long: "Tool-use quality test (tool-eval-bench 3.0.0) through llama-swap, logged to MLflow with live_* speed.\n" +
			"  lab quality coder \"baseline\"                  15 core scenarios, about 2 minutes\n" +
			"  lab quality coder \"MTP depth 3\" --full        all 69 scenarios, about 10-25 minutes\n" +
			"  lab quality coder \"temp 0.6\" --temperature 0.6\n" +
			"  SKILL=instructions/the-craft@3c329d0.md lab quality coder \"with the-craft\" --full\n" +
			"    (SKILL adds an instruction file after the built-in system prompt; frontmatter is stripped)\n" +
			"  SERIES=ornith-temp0.6 lab quality ornith \"seed 2\" --full --temperature 0.6 --seed 2\n" +
			"    (SERIES groups repeats of one setup; lab series shows average, lowest, highest)\n" +
			"  BASE_URL=http://127.0.0.1:8099/v1 API_KEY=... lab quality strata \"...\" --full   (another server)\n" +
			"Defaults: temperature 0 and seed 42. Paths (SKILL) are relative to tuning/.\n" +
			"Good result: \"tool-eval score = N\" and \"Logged run\"; the report shows every conversation.",
		DisableFlagParsing: true, // options after the note go to tool-eval-bench untouched
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			if len(args) < 2 { // like the script: usage and exit 1
				cmd.Help()
				return exitWith(bench.ExitError{Code: 1})
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return exitWith(bench.Quality(cfg, args[0], args[1], args[2:]))
		},
	}
	benchC := &cobra.Command{
		Use:   "bench [--dry-run] <card 1|2> <note> [llama-bench options]",
		Short: "llama-bench on one zaphod card, llama-swap stopped meanwhile, logged (port of tuning/bench)",
		Long: "Benchmark one card with llama-bench (3 repeats), save the output and log it to MLflow. Stops llama-swap so\n" +
			"the card is free, and always starts it again (also on Ctrl-C or kill).\n" +
			"  lab bench 1 \"baseline\"\n" +
			"  lab bench 2 \"ubatch sweep\" -ub 512,1024,2048\n" +
			"  MODEL=/models/flint-UD-Q4_K_XL.gguf lab bench 1 \"flint first look\"\n" +
			"  lab bench --dry-run 1 \"x\"     print the commands, change nothing\n" +
			"Good result: \"== results\", \"Logged run\", then \"== starting llama-swap\".",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dry, args := cutDryRun(args)
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			if len(args) < 2 {
				cmd.Help()
				return exitWith(bench.ExitError{Code: 1})
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return exitWith(bench.Bench(cfg, dry, args[0], args[1], args[2:]))
		},
	}
	kl := &cobra.Command{
		Use:   "kl [--dry-run] base | test <note> [llama-perplexity options]",
		Short: "KL-divergence of a model file or setting vs the Q8_0 reference, logged (port of tuning/kl)",
		Long: "KL-divergence: how far a model file or setting drifts from the full-precision reference (Q8_0).\n" +
			"Lower = closer to the original. Stops llama-swap while it runs and always starts it again.\n" +
			"One-time setup (needs the 29 GB Q8_0 file and about 4 GB for the reference):\n" +
			"  lab kl base\n" +
			"Then, for each thing to measure:\n" +
			"  lab kl test \"Q4_K_XL as served\"\n" +
			"  lab kl test \"Q4_K_XL, q8_0 context\" -ctk q8_0 -ctv q8_0\n" +
			"  MODEL=/models/Qwen3.8-27B-UD-Q5_K_XL.gguf lab kl test \"Q5_K_XL\"\n" +
			"  lab kl --dry-run test \"x\"     print the commands, change nothing\n" +
			"Good result: \"mean KLD\" in the logged summary, then \"== starting llama-swap\".",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dry, args := cutDryRun(args)
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			switch {
			case len(args) >= 1 && args[0] == "base":
				return exitWith(bench.KLBase(cfg, dry))
			case len(args) >= 2 && args[0] == "test":
				return exitWith(bench.KLTest(cfg, dry, args[1], args[2:]))
			}
			cmd.Help()
			return exitWith(bench.ExitError{Code: 1})
		},
	}
	tb := &cobra.Command{
		Use:   "tbench setup|doctor|smoke <note>|full <note>|log <note>",
		Short: "Terminal-Bench-Local (19 terminal tasks, pass@2), logged (port of tuning/tbench)",
		Long: bench.TBenchHelp + "\nWith a skill: SKILL=instructions/the-craft@3c329d0.md lab tbench smoke \"the-craft\"\n" +
			"(the profile gets \"+<skill>\" appended, so these runs never mix with plain ones).\n" +
			"Good result: the suite's pass/fail table, then \"== logging ...summary.json\" and \"Logged run\".",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return exitWith(bench.TBench(cfg, args))
		},
	}
	return []*cobra.Command{log, series, speed, quality, benchC, kl, tb}
}

// cutDryRun takes a leading --dry-run off args (the rest pass through to the benchmark tool untouched).
func cutDryRun(args []string) (bool, []string) {
	if len(args) > 0 && args[0] == "--dry-run" {
		return true, args[1:]
	}
	return false, args
}

// exitWith exits with a benchmark's own exit code (bench.ExitError); other errors go to cobra.
func exitWith(err error) error {
	if e, ok := err.(bench.ExitError); ok {
		os.Exit(e.Code)
	}
	return err
}
