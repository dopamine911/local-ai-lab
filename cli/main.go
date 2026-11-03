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
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
	"github.com/dopamine911/local-ai-lab/cli/internal/enso"
	"github.com/dopamine911/local-ai-lab/cli/internal/hub"
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
	root.AddCommand(versionCmd(), manifestCmd(), mcpCmd(), ensoCmd(), webuiCmd(), memoryCmd(), imageModeCmd())
	root.AddCommand(benchCmds()...)
	root.AddCommand(machineCmds()...)
	root.AddCommand(monitorsCmd())
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
	m := &cobra.Command{Use: "mcp", Short: "The MCP tool gateway (MCPJungle): status, setup"}
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
	var dry bool
	setup := &cobra.Command{
		Use:   "setup",
		Short: "Register tool servers and agent logins from hub/mcp (port of hub/mcp/setup.sh); --dry-run",
		Long: "Install the pinned mcpjungle CLI (checksum checked), create the admin once, register every tool server in\n" +
			"hub/mcp/servers/*.json, keep only the tools in <name>.tools, and recreate every agent login in\n" +
			"hub/mcp/clients/*.json (tokens from hub/.env, never printed). Safe to run again; agents lose their login for\n" +
			"a few seconds while it is recreated.\n" +
			"Good result: a line per tool server and per client, no FAILED; then `lab mcp status` says \"Matches the repo\".",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s := &mcp.Setup{Bin: config.Expand(cfg.MCPJungle.Bin), Registry: cfg.MCPJungle.Registry,
				Version: cfg.MCPJungle.Version, Repo: cfg.Repo, DryRun: dry}
			return s.Run()
		},
	}
	setup.Flags().BoolVar(&dry, "dry-run", false, "show, change nothing")
	m.AddCommand(setup)
	st.Flags().BoolVar(&asJSON, "json", false, "one JSON object (for agents)")
	st.Flags().BoolVar(&tools, "tools", false, "list every tool, on and off")
	m.AddCommand(st)
	return m
}

func ensoCmd() *cobra.Command {
	e := &cobra.Command{Use: "enso", Short: "The Enso agent container: build, setup, up, check, tui, shell, down"}
	var dry bool
	ops := func() (*enso.Ops, error) {
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		o := &enso.Ops{Repo: cfg.Repo, Image: cfg.Enso.Image, Container: cfg.Enso.Container, DryRun: dry}
		if o.Container == "" {
			o.Container = "enso"
		}
		if o.Image == "" {
			o.Image = "enso-agent:537559a"
		}
		return o, nil
	}
	sub := func(use, short, long string, f func(o *enso.Ops, args []string) error) {
		c := &cobra.Command{Use: use, Short: short, Long: long,
			RunE: func(cmd *cobra.Command, args []string) error {
				o, err := ops()
				if err != nil {
					return err
				}
				return f(o, args)
			}}
		c.Flags().BoolVar(&dry, "dry-run", false, "show, change nothing")
		e.AddCommand(c)
	}
	sub("build", "Build the Enso image (network at build time only)", "Good result: the image tag from cli/lab.yaml exists (docker images).",
		func(o *enso.Ops, _ []string) error { return o.Build() })
	sub("setup", "Make ~/agents/enso-home (keys, pi config) and ~/agents/work (enso-harness clone); safe to rerun",
		"Keys are copied (mode 600, never printed); configs are overwritten from the repo.\nGood result: \"setup done: ... (keys: mcp-token zaphod-key ), ... at <commit>\".",
		func(o *enso.Ops, _ []string) error { return o.Setup() })
	sub("up", "Start Enso and the relay; web page on http://localhost:5178 (zaphod only)", "Good result: \"Enso: http://localhost:5178\".",
		func(o *enso.Ops, _ []string) error { return o.Up() })
	sub("down", "Stop Enso and the relay", "Good result: both containers gone (docker ps).",
		func(o *enso.Ops, _ []string) error { return o.Down() })
	e.AddCommand(&cobra.Command{Use: "tui [pi options]", Short: "Enso's terminal on the workspace, with the lab's MCP tools (e.g. --model lab/coder)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := ops()
			if err != nil {
				return err
			}
			return o.TUI(args)
		}})
	e.AddCommand(&cobra.Command{Use: "shell", Short: "A shell inside the Enso container",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := ops()
			if err != nil {
				return err
			}
			return o.Shell()
		}})
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
	w := &cobra.Command{Use: "webui", Short: "Open WebUI: settings drift (and --sync), per-model settings"}
	var apply bool
	mc := &cobra.Command{
		Use:   "models",
		Short: "Per-model settings in Open WebUI vs hub/openwebui/models.yaml (port of models.sh); --apply writes them",
		Long: "Names, native function calling, abilities (vision only where listed) and hidden models, from\n" +
			"hub/openwebui/models.yaml and the llama-swap configs. Read-only unless --apply. Needs OPENWEBUI_API_KEY in hub/.env.\n" +
			"Good result: \"0 model(s) differ from models.yaml\".",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			key, err := hub.EnvValue(filepath.Join(cfg.Repo, "hub", ".env"), "OPENWEBUI_API_KEY")
			if err != nil {
				return fmt.Errorf("add OPENWEBUI_API_KEY to hub/.env: %w", err)
			}
			return webui.Models(cfg.Repo, key, apply)
		},
	}
	mc.Flags().BoolVar(&apply, "apply", false, "write the settings (Open WebUI models/import)")
	w.AddCommand(mc)
	var syncDrift, dryDrift bool
	dc := &cobra.Command{
		Use:   "drift",
		Short: "Settings changed by hand in the web page vs hub/compose.yaml (read-only; port of drift.sh)",
		Long: "Compare the running Open WebUI's settings with what hub/compose.yaml would give a fresh install\n" +
			"(a throwaway Open WebUI, no network, about 30-60 s). Read-only. Secrets show as <set>.\n" +
			"\"in compose\" = managed by compose; \"web page only\" = changed by hand: add it to compose.yaml or undo it.\n" +
			"--sync makes the running one match for the settings compose manages: stops Open WebUI for a few seconds,\n" +
			"saves a copy of its database first (data/open-webui/webui.db.before-sync-<date>). --dry-run shows the steps.\n" +
			"Good result: \"0 setting(s) differ\".",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return webui.Drift(cfg.Repo, syncDrift, dryDrift)
		},
	}
	dc.Flags().BoolVar(&syncDrift, "sync", false, "make the running Open WebUI match (managed settings only)")
	dc.Flags().BoolVar(&dryDrift, "dry-run", false, "show, change nothing")
	w.AddCommand(dc)
	w.AddCommand(&cobra.Command{
		Use:   "comfyui-env [zimage|qwen]",
		Short: "Write hub/openwebui/comfyui.env from a ComfyUI workflow (port of make-comfyui-env.sh)",
		Long: "Write hub/openwebui/comfyui.env (loaded by compose for open-webui) from a ComfyUI workflow file, so the\n" +
			"workflow file stays the only copy. Run after changing the workflow or switching models; then lab webui drift --sync.\n" +
			"  lab webui comfyui-env           Z-Image Turbo (default: 5 s per 1024px image, steps adjustable)\n" +
			"  lab webui comfyui-env qwen      Qwen-Image 2.1 Turbo (6 s; best at text inside images; steps fixed)\n" +
			"Good result: \"openwebui/comfyui.env written from ... (N bytes)\".",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"zimage", "qwen"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			which := "zimage"
			if len(args) == 1 {
				which = args[0]
			}
			msg, err := webui.ComfyEnv(filepath.Join(cfg.Repo, "hub"), which)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		},
	})
	return w
}

func memoryCmd() *cobra.Command {
	m := &cobra.Command{Use: "memory", Short: "Shared memory (LocalRecall): sync the lab's notes"}
	var install, dry bool
	s := &cobra.Command{
		Use:   "sync",
		Short: "Upload changed Markdown to LocalRecall now (port of hub/memory/sync.sh); --install: hourly cron",
		Long: "Upload the lab's changed Markdown notes to LocalRecall (the `memory` tool).\n" +
			"--install puts the hourly line (at :41) in your crontab, replacing an older one (run it yourself).\n" +
			"--dry-run shows what it would do.\n" +
			"Good result ends with \"sync: N uploaded, M removed, K unchanged\".",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if install {
				return hub.MemorySyncInstall(cfg.Repo, dry)
			}
			return hub.MemorySync(cfg.Repo, dry)
		},
	}
	s.Flags().BoolVar(&install, "install", false, "put the hourly sync in your crontab")
	s.Flags().BoolVar(&dry, "dry-run", false, "show, change nothing")
	m.AddCommand(s)
	return m
}

func imageModeCmd() *cobra.Command {
	var dry bool
	c := &cobra.Command{
		Use:       "image-mode on|off|status",
		Short:     "Lend zaphod's card 2 to ComfyUI, or give it back (port of hub/image-mode)",
		ValidArgs: []string{"on", "off", "status"},
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		Long: "on: card 2's chat models off (coder-2 requests go to coder), ComfyUI started.\n" +
			"off: ComfyUI stopped (frees all its memory), card 2 back to llama-swap.\n" +
			"status (default): what is on card 2. Card 1 and the speech helpers keep running.\n" +
			"A llama-swap config reload switches the profile off: run `lab image-mode on` again then.\n" +
			"Good result: on -> \"ComfyUI ready\"; off -> \"profile off\"; status shows profile, ComfyUI, card 2 memory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			z := cfg.Machines["zaphod"]
			env, pci := z.LlamaSwapEnv, z.ImageCardPCI
			if env == "" {
				env = "~/llama-swap/.env"
			}
			if pci == "" {
				pci = "0000:0b:00.0"
			}
			m, err := hub.NewImageMode(cfg.Repo, config.Expand(env), pci, dry)
			if err != nil {
				return err
			}
			what := "status"
			if len(args) == 1 {
				what = args[0]
			}
			switch what {
			case "on":
				return m.On()
			case "off":
				return m.Off()
			}
			return m.Status()
		},
	}
	c.Flags().BoolVar(&dry, "dry-run", false, "show, change nothing (on/off)")
	return c
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func monitorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "monitors",
		Short: "Add zaphod's network monitors to Beszel (port of hub/beszel-monitors.sh)",
		Long: "Add zaphod's network monitors (llama-swap, Open WebUI, MLflow, SearXNG, ..., chat over HTTPS, 1.1.1.1) to\n" +
			"Beszel. Asks for your Beszel login; the password is only sent to the local hub. Safe to run again:\n" +
			"existing targets are skipped. Good result: one line per monitor, \"added\" or \"already there\".",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, pass, err := hub.AskLogin()
			if err != nil {
				return err
			}
			if err := hub.AddMonitors(email, pass); err == hub.ErrLoginFailed {
				os.Exit(1)
			} else if err != nil {
				return err
			}
			return nil
		},
	}
}
