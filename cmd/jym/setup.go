package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/syumai/jevyoumean/internal/config"
	"github.com/syumai/jevyoumean/internal/creds"
	"github.com/syumai/jevyoumean/internal/jev"
)

// firstRunSetup asks for a TypeSafe API key once. A valid key is saved
// to the credentials file; an empty answer is recorded as "declined" so
// the prompt is never shown again automatically. Either way the wrapped
// command still runs afterwards — setup can never break execution.
func firstRunSetup(cfg *config.Config) string {
	fmt.Fprint(os.Stderr, `
jym: TypeSafe API key is not configured.
     Get one at https://console.typesafe.ai/ and paste it here (input hidden).
     Press Enter to skip. You can run 'jym --setup' later.
`)
	key := promptKey(cfg)
	if key == "" {
		if err := creds.SaveState(&creds.State{SetupDeclined: true}); err == nil {
			fmt.Fprintln(os.Stderr, "jym: skipping setup.")
		}
		return ""
	}
	if err := creds.Save(key); err != nil {
		fmt.Fprintf(os.Stderr, "jym: could not save key: %v\n", err)
		return key // still usable for this run
	}
	path, _ := creds.Path()
	fmt.Fprintf(os.Stderr, "jym: key saved to %s\n", path)
	return key
}

// runSetup implements `jym --setup`: prompt, verify, store.
func runSetup() int {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "jym: --setup requires a terminal")
		return 1
	}
	cfg, _ := config.Load()
	cfg.ApplyEnv()
	key := promptKey(cfg)
	if key == "" {
		fmt.Fprintln(os.Stderr, "jym: no key entered")
		return 1
	}
	if err := creds.Save(key); err != nil {
		fmt.Fprintf(os.Stderr, "jym: %v\n", err)
		return 1
	}
	path, _ := creds.Path()
	fmt.Fprintf(os.Stderr, "jym: key saved to %s\n", path)
	// Setup explicitly requested clears any previous decline.
	_ = creds.SaveState(&creds.State{})
	return 0
}

// promptKey reads a key with echo disabled and verifies it against the
// API, allowing one retry on 401.
func promptKey(cfg *config.Config) string {
	for attempt := 0; attempt < 2; attempt++ {
		fmt.Fprint(os.Stderr, "API key: ")
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "jym: %v\n", err)
			return ""
		}
		key := strings.TrimSpace(string(keyBytes))
		if key == "" {
			return ""
		}
		err = newClient(cfg, key).ValidateKey(context.Background())
		switch {
		case errors.Is(err, jev.ErrUnauthorized):
			fmt.Fprintln(os.Stderr, "jym: key rejected (401); try again or press Enter to skip.")
			continue
		case err != nil:
			fmt.Fprintf(os.Stderr, "jym: could not verify key (%v); saving anyway.\n", err)
		default:
			fmt.Fprintln(os.Stderr, "jym: key verified.")
		}
		return key
	}
	return ""
}
