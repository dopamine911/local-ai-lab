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
	"github.com/dopamine911/local-ai-lab/cli/internal/enso"
	"github.com/dopamine911/local-ai-lab/cli/internal/mcp"
	"github.com/dopamine911/local-ai-lab/cli/internal/webui"
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
	root.AddCommand(versionCmd(), manifestCmd(), mcpCmd(), ensoCmd(), webuiCmd())
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

func mcpCmd() *cobra.Command {
	m := &cobra.Command{Use: "mcp", Short: "The MCP tool gateway (MCPJungle): status"}
	var asJSON, tools bool
	st := &cobra.Command{
		Use:   "status",
		Short: "Tool servers, their tools, which agent may use which; drift from the repo",
		Long: "One read-only screen of the MCP gateway: registered tool servers (enabled / total tools), which agent may\n" +
			"use which server, and any difference from the repo (hub/mcp/clients/*.json, hub/mcp/servers/*.tools).\n" +
			"No secrets are read or printed. Exit 1 when the gateway differs from the repo.\n" +
			"Good result: \"Matches the repo\" and every server with at least one tool enabled.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s, err := mcp.Get(config.Expand(cfg.MCPJungle.Bin), cfg.MCPJungle.Registry, cfg.Repo)
			if err != nil {
				return err
			}
			if asJSON {
				if err := json.NewEncoder(os.Stdout).Encode(s); err != nil {
					return err
				}
			} else {
				mcp.Print(s, tools)
			}
			if len(s.Drift) > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	st.Flags().BoolVar(&asJSON, "json", false, "one JSON object (for agents)")
	st.Flags().BoolVar(&tools, "tools", false, "list every tool, on and off")
	m.AddCommand(st)
	return m
}

func ensoCmd() *cobra.Command {
	e := &cobra.Command{Use: "enso", Short: "The Enso agent container: check"}
	e.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Enso reaches the models and its tools, and not the internet (port of agents/enso/enso.sh check)",
		Long: "Same three checks as agents/enso/enso.sh check, run inside the enso container.\n" +
			"Good result: ok: models (llama-swap) / ok: no internet / ok: tools (mcpjungle): connected, N tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			c := cfg.Enso.Container
			if c == "" {
				c = "enso"
			}
			if !enso.Check(c) {
				os.Exit(1)
			}
			return nil
		},
	})
	return e
}

func webuiCmd() *cobra.Command {
	w := &cobra.Command{Use: "webui", Short: "Open WebUI: settings drift"}
	w.AddCommand(&cobra.Command{
		Use:   "drift",
		Short: "Settings changed by hand in the web page vs hub/compose.yaml (read-only; port of drift.sh)",
		Long: "Compare the running Open WebUI's settings with what hub/compose.yaml would give a fresh install\n" +
			"(a throwaway Open WebUI, no network, about 30-60 s). Read-only. Secrets show as <set>.\n" +
			"\"in compose\" = managed by compose; \"web page only\" = changed by hand: add it to compose.yaml or undo it.\n" +
			"To make the running one match: hub/openwebui/drift.sh --sync (moves into lab in step 3).\n" +
			"Good result: \"0 setting(s) differ\".",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return webui.Drift(cfg.Repo)
		},
	})
	return w
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
