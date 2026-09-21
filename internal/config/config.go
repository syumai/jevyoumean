// Package config loads and saves jym's local configuration file.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Default values for the recommendation policy and the Jev client.
const (
	DefaultMinProbability = 0.60
	DefaultMinConfidence  = 0.50
	DefaultTimeout        = time.Second
)

// Duration is a time.Duration that marshals to a TOML string such as "1s".
type Duration struct {
	time.Duration
}

// UnmarshalText parses a duration string like "1s" or "500ms".
func (d *Duration) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = v
	return nil
}

// MarshalText renders the duration using time.Duration.String.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// Config holds jym's local configuration.
type Config struct {
	APIKey         string   `toml:"api_key"`
	MinProbability float64  `toml:"min_probability"`
	MinConfidence  float64  `toml:"min_confidence"`
	Timeout        Duration `toml:"timeout"`
	Debug          bool     `toml:"debug"`
}

// Default returns a Config with the documented default policy.
func Default() *Config {
	return &Config{
		MinProbability: DefaultMinProbability,
		MinConfidence:  DefaultMinConfidence,
		Timeout:        Duration{DefaultTimeout},
	}
}

// Path returns the configuration file path, preferring XDG_CONFIG_HOME.
func Path() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "jevyoumean", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "jevyoumean", "config.toml"), nil
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

// Save writes the configuration file with mode 0600.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return err
	}
	// Enforce permissions even when the file already existed.
	return os.Chmod(path, 0o600)
}

// SetAPIKey stores the API key while preserving other settings.
func SetAPIKey(key string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.APIKey = key
	return Save(cfg)
}

// RemoveAPIKey clears the stored API key while preserving other settings.
func RemoveAPIKey() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.APIKey = ""
	return Save(cfg)
}
