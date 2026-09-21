// Package cache stores parsed --help output keyed by executable path and
// modification time. A corrupt or missing cache is never an error: the
// cache is strictly an optimization over re-running `--help`.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/syumai/jevyoumean/internal/help"
)

type entry struct {
	Executable string                    `json:"executable"`
	ModTime    int64                     `json:"mtime"`
	Levels     map[string][]help.Command `json:"levels"`
}

// Cache is a fail-open store for parsed help output.
type Cache struct {
	dir    string
	loaded map[string]*entry
	dirty  map[string]bool
}

// Dir returns the cache directory, preferring XDG_CACHE_HOME.
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "jevyoumean"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "jevyoumean"), nil
}

// Open returns a Cache rooted at the default cache directory.
// When the directory cannot be determined the cache silently no-ops.
func Open() *Cache {
	dir, err := Dir()
	if err != nil {
		dir = ""
	}
	return &Cache{
		dir:    dir,
		loaded: make(map[string]*entry),
		dirty:  make(map[string]bool),
	}
}

// file returns the cache file name for an executable path.
func (c *Cache) file(exe string) string {
	sum := sha256.Sum256([]byte(exe))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".json")
}

// load reads (or initializes) the cache entry for exe. Entries whose
// executable mtime differs from the current one are discarded.
func (c *Cache) load(exe string) *entry {
	if e, ok := c.loaded[exe]; ok {
		return e
	}
	e := &entry{Executable: exe, Levels: map[string][]help.Command{}}
	st, err := os.Stat(exe)
	if err == nil {
		e.ModTime = st.ModTime().Unix()
		if data, err := os.ReadFile(c.file(exe)); err == nil {
			var disk entry
			if json.Unmarshal(data, &disk) == nil &&
				disk.Executable == exe && disk.ModTime == e.ModTime {
				e = &disk
			}
		}
	}
	if e.Levels == nil {
		e.Levels = map[string][]help.Command{}
	}
	c.loaded[exe] = e
	return e
}

// Get returns cached commands for (exe, level), where level identifies the
// command path ("" for the top level, "pr" for `exe pr`, and so on).
func (c *Cache) Get(exe, level string) ([]help.Command, bool) {
	if c.dir == "" {
		return nil, false
	}
	cmds, ok := c.load(exe).Levels[level]
	return cmds, ok
}

// Put records commands for (exe, level). The data is persisted by Flush.
func (c *Cache) Put(exe, level string, cmds []help.Command) {
	if c.dir == "" {
		return
	}
	c.load(exe).Levels[level] = cmds
	c.dirty[exe] = true
}

// Flush writes modified entries to disk. Errors are ignored on purpose.
func (c *Cache) Flush() {
	if c.dir == "" {
		return
	}
	for exe := range c.dirty {
		data, err := json.MarshalIndent(c.loaded[exe], "", "  ")
		if err != nil {
			continue
		}
		if err := os.MkdirAll(c.dir, 0o755); err != nil {
			return
		}
		_ = os.WriteFile(c.file(exe), data, 0o644)
	}
	c.dirty = map[string]bool{}
}
