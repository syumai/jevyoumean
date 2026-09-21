package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/syumai/jevyoumean/internal/cache"
	"github.com/syumai/jevyoumean/internal/config"
	"github.com/syumai/jevyoumean/internal/creds"
	"github.com/syumai/jevyoumean/internal/jev"
)

// runDoctor implements --doctor: report key status, API reachability,
// cache location and TTY state.
func runDoctor() int {
	cfg, cfgErr := config.Load()
	cfgPath, _ := config.Path()
	fmt.Printf("jym %s\n", version)
	fmt.Printf("config:       %s\n", cfgPath)
	if cfgErr != nil {
		fmt.Printf("              error: %v\n", cfgErr)
		cfg = config.Default()
	}
	cfg.ApplyEnv()
	fmt.Printf("mode:         %s (suggest>=%.2f, auto>=%.2f, min_confidence=%.2f)\n",
		cfg.Mode, cfg.SuggestThreshold, cfg.AutoRunThreshold, cfg.MinConfidence)
	fmt.Printf("timeout:      %s\n", cfg.Timeout())

	key, source := creds.Resolve()
	if key == "" {
		fmt.Println("api key:      not configured (run 'jym --setup')")
	} else {
		fmt.Printf("api key:      %s via %s\n", creds.Masked(key), source)
		client := newClient(cfg, key)
		start := time.Now()
		err := client.ValidateKey(context.Background())
		switch {
		case err == nil:
			fmt.Printf("api:          ok (%dms)\n", time.Since(start).Milliseconds())
		case errors.Is(err, jev.ErrUnauthorized):
			fmt.Println("api:          key rejected (401)")
		default:
			fmt.Printf("api:          unreachable (%v)\n", err)
		}
	}

	cacheDir, _ := cache.Dir()
	fmt.Printf("cache dir:    %s\n", cacheDir)
	if entries := cache.Open().Entries(); len(entries) > 0 {
		fmt.Printf("cached:       %v\n", entries)
	}
	fmt.Printf("interactive:  stdin=%v stderr=%v\n",
		term.IsTerminal(int(os.Stdin.Fd())), term.IsTerminal(int(os.Stderr.Fd())))
	return 0
}

// printMise implements --print-mise: emit a [shell_alias] snippet.
func printMise(cmds []string) int {
	if len(cmds) == 0 {
		fmt.Fprintln(os.Stderr, "jym: --print-mise needs at least one command name")
		return 2
	}
	fmt.Println("[shell_alias]")
	for _, c := range cmds {
		fmt.Printf("%s = \"jym -- %s\"\n", c, c)
	}
	return 0
}

// printCompletion implements --completion: emit a shell completion
// script that delegates to the wrapped command's own completion.
func printCompletion(shell string) int {
	switch shell {
	case "zsh":
		fmt.Print(`#compdef jym
# jym wraps a command; strip "jym" (and its flags) and let zsh complete
# the remaining words as the wrapped command.
_jym() {
  local i=2
  while (( i <= $#words )) && [[ ${words[i]} == -* ]]; do
    (( i++ ))
  done
  [[ ${words[i-1]} == "--" ]] && (( i++ ))
  words=("${words[@]:$((i-1))}")
  (( CURRENT -= i - 2 ))
  _normal
}
compdef _jym jym
`)
	case "bash":
		fmt.Print(`# bash completion for jym: complete command names for the first
# argument, then delegate via bash-completion's _command_offset.
_jym() {
    local i=1
    while (( i < COMP_CWORD )) && [[ ${COMP_WORDS[i]} == -* ]]; do ((i++)); done
    [[ ${COMP_WORDS[i-1]} == "--" ]] && ((i++))
    if declare -F _command_offset >/dev/null; then
        _command_offset "$i"
    else
        COMPREPLY=($(compgen -c -- "${COMP_WORDS[COMP_CWORD]}"))
    fi
}
complete -F _jym jym
`)
	case "fish":
		fmt.Print(`# fish completion for jym: offer command names for the first
# argument only. Deeper delegation is not supported by fish.
complete -c jym -n 'test (count (commandline -opc)) -eq 1' -xa '(__fish_complete_command)'
`)
	default:
		fmt.Fprintf(os.Stderr, "jym: unsupported shell %q (want bash, zsh or fish)\n", shell)
		return 2
	}
	return 0
}
