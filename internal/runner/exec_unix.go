//go:build unix

package runner

import (
	"syscall"
)

// Replace execs the target command, replacing the jym process so
// signals, TTY handling, exit codes and job control are fully native.
// On failure it falls back to a child process.
func Replace(name, path string, args []string) int {
	argv := append([]string{name}, args...)
	if err := syscall.Exec(path, argv, childEnv()); err != nil {
		return Run(name, path, args)
	}
	return 0 // unreachable: syscall.Exec does not return on success
}
