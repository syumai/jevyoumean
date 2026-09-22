package helptext

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// HelpTimeout bounds a single help invocation. Help is local, so a slow
// response means something is wrong and we should give up rather than
// stall the wrapped command.
const HelpTimeout = 3 * time.Second

// DefaultHelpArgs is the tried order for help invocation: most CLIs
// understand --help, some only -h, some only a `help` subcommand.
var DefaultHelpArgs = [][]string{{"--help"}, {"-h"}, {"help"}}

// Fetch obtains the documented subcommands of `exe path...`. helpArgs,
// when non-empty, replaces the default invocation arguments (for
// commands like git whose --help output omits most subcommands).
// The first invocation yielding at least one extracted command wins;
// when all yield nothing, the (empty) result means "cannot judge".
// Paginators are disabled and stdin is detached so discovery can never
// block on interaction.
func Fetch(ctx context.Context, exe string, path []string, helpArgs []string) ([]Command, error) {
	_, cmds, err := FetchRaw(ctx, exe, path, helpArgs)
	return cmds, err
}

// FetchRaw is Fetch plus the raw help output it parsed. When no attempt
// yields a parseable command listing, raw holds the first non-empty
// output (if any) and cmds is nil — useful for capturing fixtures of
// help formats the parser cannot handle yet.
func FetchRaw(ctx context.Context, exe string, path []string, helpArgs []string) (raw []byte, cmds []Command, err error) {
	attempts := DefaultHelpArgs
	if len(helpArgs) > 0 {
		attempts = [][]string{helpArgs}
	}
	var lastErr error
	var fallback []byte
	for _, extra := range attempts {
		args := append(append([]string{}, path...), extra...)
		out, err := runHelp(ctx, exe, args)
		if len(out) == 0 {
			if err != nil {
				lastErr = err
			}
			continue
		}
		if cmds := Parse(string(out)); len(cmds) > 0 {
			return out, cmds, nil
		}
		if fallback == nil {
			fallback = out
		}
	}
	return fallback, nil, lastErr
}

func runHelp(ctx context.Context, exe string, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, HelpTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"TERM=dumb",
		"PAGER=cat",
		"GIT_PAGER=cat",
		"MANPAGER=cat",
		"GH_PAGER=cat",
		"LESS=FRX",
	)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, ctx.Err()
	}
	return out, err
}
