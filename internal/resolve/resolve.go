// Package resolve locates the wrapped executable while guarding against
// jym wrapping itself.
package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrNotFound reports that no executable named name exists on PATH.
var ErrNotFound = errors.New("executable not found")

// MaxDepth is the JYM_DEPTH value at which jym unconditionally passes
// through, breaking any accidental exec loop.
const MaxDepth = 2

// Depth returns the current JYM_DEPTH value.
func Depth() int {
	n, _ := strconv.Atoi(os.Getenv("JYM_DEPTH"))
	return n
}

// LookPath resolves name to an executable path using PATH. Shell aliases
// are not expanded. Candidates that resolve to jym itself (e.g. a shim
// symlink) are skipped so a self-referencing PATH cannot loop.
func LookPath(name string) (string, error) {
	self, _ := selfInfo()

	if strings.ContainsRune(name, os.PathSeparator) {
		if isExecutable(name) && !sameFile(name, self) {
			return name, nil
		}
		return "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			dir = "."
		}
		cand := filepath.Join(dir, name)
		if !isExecutable(cand) {
			continue
		}
		if sameFile(cand, self) {
			continue
		}
		return cand, nil
	}
	return "", fmt.Errorf("%w: %s", ErrNotFound, name)
}

func selfInfo() (os.FileInfo, bool) {
	exe, err := os.Executable()
	if err != nil {
		return nil, false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	st, err := os.Stat(exe)
	return st, err == nil
}

func isExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Mode()&0o111 != 0
}

func sameFile(path string, self os.FileInfo) bool {
	if self == nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	st, err := os.Stat(resolved)
	if err != nil {
		return false
	}
	return os.SameFile(st, self)
}
