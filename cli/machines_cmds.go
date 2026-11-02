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
	return []*cobra.Command{models}
}
