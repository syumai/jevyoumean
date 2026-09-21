// Package runner resolves and executes the wrapped command.
package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// LookPath resolves name to an executable path using the user's PATH.
// Shell aliases are intentionally not expanded.
func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// Run executes the command at path with stdin/stdout/stderr passthrough
// and returns the command's exit code.
func Run(path string, args []string) int {
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code >= 0 {
				return code
			}
			// The process was terminated by a signal. Mapping to
			// 128+signal requires platform-specific WaitStatus
			// handling; that refinement is out of MVP scope.
			return 1
		}
		fmt.Fprintf(os.Stderr, "jym: failed to execute %s: %v\n", path, err)
		return 1
	}
	return 0
}
