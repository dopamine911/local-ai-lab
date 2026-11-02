package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
	"github.com/dopamine911/local-ai-lab/cli/internal/machines"
)

// machineCmds: commands that set up or look after a machine (issue #6 step 5).
func machineCmds() []*cobra.Command {
	models := &cobra.Command{
		Use:   "models [machine] [filter]",
		Short: "Download this machine's models into /srv/models and check their sha256 (port of setup/models.sh)",
		Long: "Download every model listed in setup/models-<machine>.txt into /srv/models, then check each sha256.\n" +
			"  lab models                 this machine's list\n" +
			"  lab models zaphod gemma    only lines whose destination contains \"gemma\"\n" +
			"Safe to re-run: finished files are skipped (still checked), partial ones resume. Pinned revisions: same bytes\n" +
			"every time. Takes hours for a full machine: run it inside tmux.\n" +
			"Good result: \"have\"/\"checking\" per file, then MODELS-OK (exit 0). MODELS-FAILED exits 1.",
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			args = append(args, "", "")
			ok, err := machines.Models(cfg, args[0], args[1])
			if err != nil {
				return err
			}
			if !ok {
				os.Exit(1)
			}
			return nil
		},
	}
	var dry bool
	backup := &cobra.Command{
		Use:   "backup [run|list|install [--dry-run]]",
		Short: "Back up zaphod's data (not in git) to marvin; list snapshots; nightly cron line (port of setup/backup-to-marvin.sh)",
		Long: "Back up zaphod's data that is not in git to marvin:~/backups/zaphod/<date>/.\n" +
			"  lab backup            one snapshot now (same as `lab backup run`)\n" +
			"  lab backup list       the snapshots on marvin\n" +
			"  lab backup install    run it every night at 04:17 (crontab; replaces the script's line). --dry-run shows it\n" +
			"What: hub/data (Open WebUI chats/uploads, Beszel, MLflow, ComfyUI, ...), llama-swap activity, all .env files\n" +
			"(mode 600). Not: model files (lab models gets them back), caches, Caddy certificates. Databases are copied\n" +
			"with SQLite's online backup, so services keep running. Keeps the last 14 snapshots; unchanged files are\n" +
			"hard links. Temporary until the NAS is rebuilt (setup/REBUILD.md section 6).\n" +
			"Good result: \"copied N databases ... and N other files\", then \"== ... done: marvin:backups/zaphod/<date> (size)\".",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"run", "list", "install"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			what := "run"
			if len(args) == 1 {
				what = args[0]
			}
			switch what {
			case "run":
				return machines.BackupRun(cfg)
			case "list":
				return machines.BackupList()
			case "install":
				return machines.BackupInstall(dry)
			}
			cmd.Help()
			os.Exit(1)
			return nil
		},
	}
	backup.Flags().BoolVar(&dry, "dry-run", false, "install: show the crontab it would write, change nothing")
	return []*cobra.Command{models, backup}
}
