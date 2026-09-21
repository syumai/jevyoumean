package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("defaults failed validation: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Config){
		"mode":           func(c *Config) { c.Mode = "scream" },
		"context_args":   func(c *Config) { c.ContextArgs = "everything" },
		"timeout zero":   func(c *Config) { c.TimeoutMS = 0 },
		"timeout neg":    func(c *Config) { c.TimeoutMS = -5 },
		"timeout huge":   func(c *Config) { c.TimeoutMS = maxTimeoutMS + 1 },
		"suggest neg":    func(c *Config) { c.SuggestThreshold = -0.1 },
		"suggest >1":     func(c *Config) { c.SuggestThreshold = 1.01 },
		"suggest NaN":    func(c *Config) { c.SuggestThreshold = math.NaN() },
		"autorun neg":    func(c *Config) { c.AutoRunThreshold = -1 },
		"confidence Inf": func(c *Config) { c.MinConfidence = math.Inf(1) },
		"max_depth neg":  func(c *Config) { c.MaxDepth = -1 },
		"max_depth huge": func(c *Config) { c.MaxDepth = maxMaxDepth + 1 },
		"per-cmd depth":  func(c *Config) { c.Commands["git"] = CommandConfig{MaxDepth: -2} },
	}
	for name, mutate := range cases {
		cfg := Default()
		mutate(cfg)
		if err := cfg.Validate(); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestLoadInvalidFallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("mode = \"nonsense\"\ntimeout_ms = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JYM_CONFIG", path)
	cfg, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if cfg.Mode != DefaultMode || cfg.TimeoutMS != DefaultTimeoutMS {
		t.Fatalf("expected defaults on invalid config, got mode=%q timeout=%d", cfg.Mode, cfg.TimeoutMS)
	}
}

func TestLoadMissingUsesDefaults(t *testing.T) {
	t.Setenv("JYM_CONFIG", filepath.Join(t.TempDir(), "absent.toml"))
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != DefaultMode {
		t.Fatalf("unexpected mode: %q", cfg.Mode)
	}
}

func TestApplyEnvMode(t *testing.T) {
	cfg := Default()
	t.Setenv("JYM_MODE", "auto")
	cfg.ApplyEnv()
	if cfg.Mode != "auto" {
		t.Fatalf("expected auto, got %q", cfg.Mode)
	}
	t.Setenv("JYM_MODE", "bogus")
	cfg.ApplyEnv()
	if cfg.Mode != "auto" {
		t.Fatalf("invalid JYM_MODE must be ignored, got %q", cfg.Mode)
	}
}
