// Package runner executes the wrapped command. On Unix the process is
// replaced via syscall.Exec so signals, the TTY, the exit code and job
// control behave natively. A child-process path is kept for cases that
// must observe the exit code (learning) and for non-Unix platforms.
package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/syumai/jevyoumean/internal/resolve"
)

// childEnv returns the process environment with JYM_DEPTH incremented,
// so a wrapped command that somehow invokes jym again cannot loop.
func childEnv() []string {
	env := os.Environ()
	depth := strconv.Itoa(resolve.Depth() + 1)
	for i, e := range env {
		if len(e) > len("JYM_DEPTH=") && e[:len("JYM_DEPTH=")] == "JYM_DEPTH=" {
			env[i] = "JYM_DEPTH=" + depth
			return env
		}
	}
	return append(env, "JYM_DEPTH="+depth)
}

// Run executes path as a child process with stdio passthrough and
// returns its exit code. Unlike Replace it can observe the exit status,
// which the "run as typed" learning path needs.
func Run(name, path string, args []string) int {
	cmd := exec.Command(path, args...)
	cmd.Args[0] = name
	cmd.Env = childEnv()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code >= 0 {
				return code
			}
			// Terminated by a signal; mapping to 128+signal needs
			// platform-specific WaitStatus handling.
			return 1
		}
		fmt.Fprintf(os.Stderr, "jym: failed to execute %s: %v\n", path, err)
		return 1
	}
	return 0
}
