// lab: one command for the homelab (issue #6). Replaces the scattered scripts one group at a time; each port keeps
// the script's exact output (benchmarks must stay comparable). Rules carried over from the scripts:
//   - sudo commands are printed for Daniel, never run;
//   - nothing listens beyond 127.0.0.1 / Tailscale;
//   - every command says what a good result looks like;
//   - --dry-run where a command changes something; --json for agents.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
	"github.com/dopamine911/local-ai-lab/cli/internal/manifest"
	"github.com/dopamine911/local-ai-lab/cli/internal/menu"
)

// version is set at build time (cli/build.sh: git describe).
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "lab",
		Short: "The homelab's commands in one place (zaphod and marvin)",
		Long: "lab: the homelab's commands in one place (issue #6).\n" +
			"Run `lab` on a terminal for a menu, or `lab <command> --help`.\n" +
			"Config: $LAB_REPO/cli/lab.yaml (LAB_REPO defaults to ~/homelab). Secrets stay in the .env files.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// A menu only on a real terminal: agents, cron and `ssh marvin lab` get the help instead of a hang.
			if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
				return cmd.Help()
			}
			var items []menu.Item
			for _, c := range cmd.Commands() {
				if c.IsAvailableCommand() && c.Name() != "help" && c.Name() != "completion" {
					items = append(items, menu.Item{Name: c.Name(), Short: c.Short})
				}
			}
			chosen, err := menu.Run(items)
			if err != nil || chosen == "" {
				return err
			}
			cmd.SetArgs([]string{chosen})
			return cmd.Execute()
		},
	}
	root.AddCommand(versionCmd(), manifestCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the lab build (git commit) and where its config is",
		Long:  "Show the lab build and config path.\nGood result: a commit id (not \"dev\") and a config path that exists.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("lab %s\nconfig %s/cli/lab.yaml\n", version, config.Repo())
			return nil
		},
	}
}

func manifestCmd() *cobra.Command {
	var asJSON bool
	var host string
	c := &cobra.Command{
		Use:   "manifest",
		Short: "Machine and software facts for test results (port of tuning/manifest.sh)",
		Long: "Print the machine and software facts that belong with every test result, as key=value lines\n" +
			"(tuning/log tags each as env.<key> in MLflow). Same output as tuning/manifest.sh.\n" +
			"--host marvin runs it there over ssh (needs lab installed on marvin).\n" +
			"Good result: no \"unknown\" except where a part really is missing; llama_cpp_commit matches the image.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host != "" {
				h, _ := os.Hostname()
				if host != h {
					a := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", host, "~/.local/bin/lab", "manifest"}
					if asJSON {
						a = append(a, "--json")
					}
					x := exec.Command("ssh", a...)
					x.Stdout, x.Stderr = os.Stdout, os.Stderr
					return x.Run()
				}
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			h, _ := os.Hostname()
			fields := manifest.Collect(cfg.Container(h))
			if asJSON {
				m := map[string]string{}
				for _, f := range fields {
					m[f.Key] = f.Value
				}
				return json.NewEncoder(os.Stdout).Encode(m)
			}
			for _, f := range fields {
				fmt.Printf("%s=%s\n", f.Key, f.Value)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "one JSON object (for agents)")
	c.Flags().StringVar(&host, "host", "", "run on another machine over ssh (e.g. marvin)")
	return c
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
