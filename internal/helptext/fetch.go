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
	attempts := DefaultHelpArgs
	if len(helpArgs) > 0 {
		attempts = [][]string{helpArgs}
	}
	var lastErr error
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
			return cmds, nil
		}
	}
	return nil, lastErr
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
