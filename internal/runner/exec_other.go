//go:build !unix

package runner

// Replace falls back to child-process execution on platforms without
// syscall.Exec (e.g. Windows).
func Replace(name, path string, args []string) int {
	return Run(name, path, args)
}
