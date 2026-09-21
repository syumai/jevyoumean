// Package creds resolves and stores the TypeSafe API key, and records
// whether the user has already answered (or declined) the first-run
// setup prompt so it is only ever shown once.
package creds

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/syumai/jevyoumean/internal/config"
)

type file struct {
	APIKey string `toml:"api_key"`
}

// Path returns the credentials file path ($XDG_CONFIG_HOME/jym/credentials.toml).
func Path() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.toml"), nil
}

// Resolve returns the configured API key. The TYPESAFE_API_KEY
// environment variable takes precedence over the credentials file.
func Resolve() (key, source string) {
	if k := os.Getenv("TYPESAFE_API_KEY"); k != "" {
		return k, "TYPESAFE_API_KEY"
	}
	if k, err := LoadFile(); err == nil && k != "" {
		return k, "credentials file"
	}
	return "", ""
}

// LoadFile reads the API key from the credentials file.
func LoadFile() (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	var f file
	if _, err := toml.DecodeFile(path, &f); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return f.APIKey, nil
}

// Save writes the API key to the credentials file with mode 0600.
func Save(key string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(file{APIKey: key}); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// Remove deletes the credentials file. A missing file is not an error.
func Remove() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Masked renders a key for display without revealing it.
func Masked(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// State is persisted in $XDG_STATE_HOME/jym/state.json.
type State struct {
	SetupDeclined bool `json:"setup_declined"`
}

func statePath() (string, error) {
	var base string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		base = xdg
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "jym", "state.json"), nil
}

// LoadState reads the persisted state. A missing file yields zero state.
func LoadState() (*State, error) {
	path, err := statePath()
	if err != nil {
		return &State{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return &State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return &State{}, nil // corrupt state is harmless; start fresh
	}
	return &s, nil
}

// SaveState persists the state file.
func SaveState(s *State) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
