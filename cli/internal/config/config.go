// Package config reads cli/lab.yaml from the lab repo.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Machine struct {
	SSH                string `yaml:"ssh"`
	LlamaSwapPort      int    `yaml:"llama_swap_port"`
	LlamaSwapContainer string `yaml:"llama_swap_container"`
	LlamaSwapEnv       string `yaml:"llama_swap_env"`
	ImageCardPCI       string `yaml:"image_card_pci"`
}

type Config struct {
	Repo      string             `yaml:"-"`
	Machines  map[string]Machine `yaml:"machines"`
	MCPJungle struct {
		Version  string `yaml:"version"`
		Bin      string `yaml:"bin"`
		Registry string `yaml:"registry"`
	} `yaml:"mcpjungle"`
	Enso struct {
		Container string `yaml:"container"`
		Image     string `yaml:"image"`
	} `yaml:"enso"`
}

// Expand turns a leading ~/ into the home folder.
func Expand(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, rest)
	}
	return p
}

// Repo is $LAB_REPO, else ~/homelab.
func Repo() string {
	if r := os.Getenv("LAB_REPO"); r != "" {
		return r
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "homelab")
}

func Load() (*Config, error) {
	repo := Repo()
	p := filepath.Join(repo, "cli", "lab.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read %s (set LAB_REPO if the repo is elsewhere): %w", p, err)
	}
	c := &Config{Repo: repo}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	return c, nil
}

// Container is the llama-swap container name for this machine (default "llama-swap").
func (c *Config) Container(host string) string {
	if m, ok := c.Machines[host]; ok && m.LlamaSwapContainer != "" {
		return m.LlamaSwapContainer
	}
	return "llama-swap"
}
