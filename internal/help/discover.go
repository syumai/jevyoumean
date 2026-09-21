package help

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HelpTimeout bounds a single `--help` invocation. Help is local, so a
// slow response means something is wrong and we should give up rather
// than stall the wrapped command.
const HelpTimeout = 5 * time.Second

// CacheStore is the cache interface used by Discoverer. It is satisfied
// by the internal/cache package.
type CacheStore interface {
	Get(exe, level string) ([]Command, bool)
	Put(exe, level string, cmds []Command)
}

// Discoverer resolves the documented subcommands at a command path,
// using the cache when available and `--help` otherwise.
type Discoverer struct {
	Store CacheStore
}

// Commands returns the documented subcommands of `exe path...`.
// An empty result means "cannot judge": either the help output had no
// recognizable command section or the invocation failed.
func (d *Discoverer) Commands(ctx context.Context, exe string, path []string) ([]Command, error) {
	level := strings.Join(path, "\x00")
	if d.Store != nil {
		if cmds, ok := d.Store.Get(exe, level); ok {
			return cmds, nil
		}
	}
	cmds, err := Fetch(ctx, exe, path)
	if err != nil {
		return nil, err
	}
	if d.Store != nil {
		d.Store.Put(exe, level, cmds)
	}
	return cmds, nil
}

// Fetch runs `exe path... --help` and parses the subcommand list.
// Paginators are disabled and stdin is detached so discovery can never
// block on interaction.
func Fetch(ctx context.Context, exe string, path []string) ([]Command, error) {
	args := append(append([]string{}, path...), "--help")
	out, err := runHelp(ctx, exe, args)
	if len(out) == 0 {
		// Some tools only support a compact -h usage.
		args[len(args)-1] = "-h"
		out, err = runHelp(ctx, exe, args)
	}
	if err != nil && len(out) == 0 {
		return nil, err
	}
	return Parse(string(out)), nil
}

func runHelp(ctx context.Context, exe string, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, HelpTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(),
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
