// Package resolve locates the wrapped executable while guarding against
// jym wrapping itself.
package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	if strings.ContainsRune(name, os.PathSeparator) ||
		(runtime.GOOS == "windows" && strings.ContainsRune(name, '/')) {
		if isExecutable(name) && !sameFile(name, self) {
			return name, nil
		}
		return "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			dir = "."
		}
		for _, cand := range candidates(dir, name) {
			if !isExecutable(cand) {
				continue
			}
			if sameFile(cand, self) {
				continue
			}
			return cand, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrNotFound, name)
}

// candidates lists the file paths to try for name in dir. On Windows a
// bare name needs a PATHEXT extension (fakecli → fakecli.exe).
func candidates(dir, name string) []string {
	if runtime.GOOS != "windows" || filepath.Ext(name) != "" {
		return []string{filepath.Join(dir, name)}
	}
	out := make([]string, 0, len(pathexts())+1)
	for _, ext := range pathexts() {
		out = append(out, filepath.Join(dir, name+ext))
	}
	return out
}

// pathexts returns the executable extensions on Windows, or nil
// elsewhere. PATHEXT wins; the cmd defaults cover the common case.
func pathexts() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	if v := os.Getenv("PATHEXT"); v != "" {
		return strings.Split(v, string(os.PathListSeparator))
	}
	return []string{".COM", ".EXE", ".BAT", ".CMD"}
}

// ExecutableName reports whether fileName can be executed directly —
// on Windows, whether it carries a PATHEXT extension — and returns the
// command name to invoke it by (extension stripped).
func ExecutableName(fileName string) (string, bool) {
	if runtime.GOOS != "windows" {
		return fileName, true
	}
	ext := filepath.Ext(fileName)
	for _, pe := range pathexts() {
		if strings.EqualFold(ext, pe) {
			return fileName[:len(fileName)-len(ext)], true
		}
	}
	return "", false
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

// isExecutable reports whether path is a runnable file. Windows has no
// execute bit — candidates() already filtered by PATHEXT, so a regular
// file is enough.
func isExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
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
