// Package config loads jym's configuration file.
//
// The file lives at $XDG_CONFIG_HOME/jym/config.toml (or
// ~/.config/jym/config.toml). JYM_CONFIG points at an alternate file,
// which lets mise switch settings per project via [env]. Settings are
// never read implicitly from repository files.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Default values for the decision policy and the Jev client.
const (
	DefaultMode             = "prompt"
	DefaultSuggestThreshold = 0.30
	DefaultAutoRunThreshold = 0.95
	DefaultMinConfidence    = 0.50
	DefaultTimeoutMS        = 1500
	DefaultMaxDepth         = 2
	DefaultContextArgs      = "none"
	DefaultModel            = "jev-latest"
)

// CommandConfig holds per-command overrides keyed by command name.
type CommandConfig struct {
	HelpArgs         []string `toml:"help_args"`
	ExtraSubcommands []string `toml:"extra_subcommands"`
	MaxDepth         int      `toml:"max_depth"` // 0 means inherit the global value
}

// Config holds jym's configuration.
type Config struct {
	Mode             string   `toml:"mode"`
	SuggestThreshold float64  `toml:"suggest_threshold"`
	AutoRunThreshold float64  `toml:"auto_run_threshold"`
	MinConfidence    float64  `toml:"min_confidence"`
	TimeoutMS        int      `toml:"timeout_ms"`
	MaxDepth         int      `toml:"max_depth"`
	ContextArgs      string   `toml:"context_args"`
	Model            string   `toml:"model"`
	Debug            bool     `toml:"debug"`
	Denylist         []string `toml:"denylist"` // added to the built-in denylist

	Commands map[string]CommandConfig `toml:"commands"`
}

// Default returns a Config with the documented defaults.
func Default() *Config {
	return &Config{
		Mode:             DefaultMode,
		SuggestThreshold: DefaultSuggestThreshold,
		AutoRunThreshold: DefaultAutoRunThreshold,
		MinConfidence:    DefaultMinConfidence,
		TimeoutMS:        DefaultTimeoutMS,
		MaxDepth:         DefaultMaxDepth,
		ContextArgs:      DefaultContextArgs,
		Model:            DefaultModel,
		Commands:         map[string]CommandConfig{},
	}
}

// Timeout returns the configured client-side API timeout.
func (c *Config) Timeout() time.Duration {
	return time.Duration(c.TimeoutMS) * time.Millisecond
}

// BuiltinCommands holds shipped per-command overrides. git's --help
// lists only the most common subcommands, so it uses `help -a` instead.
var BuiltinCommands = map[string]CommandConfig{
	"git": {HelpArgs: []string{"help", "-a"}},
}

// Command returns the per-command overrides for name: user config first,
// then the shipped builtins.
func (c *Config) Command(name string) CommandConfig {
	if cc, ok := c.Commands[name]; ok {
		return cc
	}
	return BuiltinCommands[name]
}

// Dir returns the configuration directory, preferring XDG_CONFIG_HOME.
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "jym"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "jym"), nil
}

// Path returns the configuration file path. JYM_CONFIG wins, then
// XDG_CONFIG_HOME, then ~/.config.
func Path() (string, error) {
	if p := os.Getenv("JYM_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the configuration file. A missing file yields the defaults.
func Load() (*Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

// ApplyEnv overlays the JYM_* environment overrides onto cfg.
func (c *Config) ApplyEnv() {
	if v := os.Getenv("JYM_MODE"); v != "" {
		c.Mode = v
	}
	if v := os.Getenv("JYM_DEBUG"); v != "" && v != "0" {
		c.Debug = true
	}
}
