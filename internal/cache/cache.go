// Package cache stores parsed help output under
// $XDG_CACHE_HOME/jym/<command>/<key>.json, keyed by the resolved
// executable's path, mtime and size, the subcommand path, and the help
// invocation. A rebuilt binary therefore invalidates automatically; a
// 7-day TTL covers changes that leave the binary untouched.
// All operations are fail-open: a broken cache must never break the
// wrapped command.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/syumai/jevyoumean/internal/helptext"
)

// TTL bounds how long an entry is trusted even when the binary is
// unchanged (e.g. plugins that add subcommands out of band).
const TTL = 7 * 24 * time.Hour

// Entry is one cached extraction result for a command path.
type Entry struct {
	Executable  string             `json:"executable"`
	ModTime     int64              `json:"mtime"`
	Size        int64              `json:"size"`
	Path        []string           `json:"path"`
	FetchedAt   int64              `json:"fetched_at"`
	IsLeaf      bool               `json:"is_leaf"`
	Subcommands []helptext.Command `json:"subcommands"`
	// Learned holds tokens the user confirmed valid by choosing
	// "run as typed" on a command that then exited 0 (aliases,
	// plugins and other subcommands absent from --help).
	Learned []string `json:"learned,omitempty"`
}

// HasLearned reports whether token was previously confirmed valid.
func (e *Entry) HasLearned(token string) bool {
	for _, l := range e.Learned {
		if l == token {
			return true
		}
	}
	return false
}

// Cache is a fail-open store for parsed help output.
type Cache struct {
	dir string
}

// Dir returns the cache root, preferring XDG_CACHE_HOME.
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "jym"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "jym"), nil
}

// Open returns a Cache rooted at the default directory. When the
// directory cannot be determined the cache silently no-ops.
func Open() *Cache {
	dir, err := Dir()
	if err != nil {
		dir = ""
	}
	return &Cache{dir: dir}
}

// key derives the file name from executable identity, subcommand path
// and help invocation.
func key(exe string, st os.FileInfo, level string, helpArgs []string) string {
	h := sha256.New()
	h.Write([]byte(exe))
	h.Write([]byte{0})
	h.Write([]byte(st.ModTime().String()))
	h.Write([]byte{0})
	h.Write([]byte(level))
	h.Write([]byte{0})
	h.Write([]byte(strings.Join(helpArgs, " ")))
	return hex.EncodeToString(h.Sum(nil))
}

// Load returns the cached entry for (exe, path, helpArgs), or false on
// any miss: absent, expired, unparseable, or for a stale binary.
func (c *Cache) Load(exe string, path []string, helpArgs []string) (*Entry, bool) {
	if c.dir == "" {
		return nil, false
	}
	st, err := os.Stat(exe)
	if err != nil {
		return nil, false
	}
	level := strings.Join(path, "\x00")
	data, err := os.ReadFile(c.file(exe, key(exe, st, level, helpArgs)))
	if err != nil {
		return nil, false
	}
	var e Entry
	if json.Unmarshal(data, &e) != nil ||
		e.Executable != exe || e.ModTime != st.ModTime().Unix() || e.Size != st.Size() ||
		time.Since(time.Unix(e.FetchedAt, 0)) > TTL {
		return nil, false
	}
	return &e, true
}

// Save persists an extraction result and prunes expired siblings.
func (c *Cache) Save(e *Entry, helpArgs []string) {
	if c.dir == "" {
		return
	}
	st, err := os.Stat(e.Executable)
	if err != nil {
		return
	}
	e.ModTime = st.ModTime().Unix()
	e.Size = st.Size()
	e.FetchedAt = time.Now().Unix()
	level := strings.Join(e.Path, "\x00")
	dir := c.cmdDir(e.Executable)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.file(e.Executable, key(e.Executable, st, level, helpArgs)), data, 0o644)
	c.prune(dir)
}

// AddLearned records token as a confirmed-valid subcommand at e's level.
func (c *Cache) AddLearned(e *Entry, token string, helpArgs []string) {
	if e.HasLearned(token) {
		return
	}
	e.Learned = append(e.Learned, token)
	c.Save(e, helpArgs)
}

// Invalidate removes every cached level for the named command
// (--refresh). name is the command's base name.
func (c *Cache) Invalidate(name string) {
	if c.dir == "" {
		return
	}
	_ = os.RemoveAll(filepath.Join(c.dir, name))
}

// ClearAll removes the whole cache (--cache-clear).
func (c *Cache) ClearAll() {
	if c.dir == "" {
		return
	}
	_ = os.RemoveAll(c.dir)
}

// Entries lists the cached command base names (for --doctor).
func (c *Cache) Entries() []string {
	des, err := os.ReadDir(c.dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, de := range des {
		if de.IsDir() {
			out = append(out, de.Name())
		}
	}
	return out
}

func (c *Cache) cmdDir(exe string) string {
	return filepath.Join(c.dir, filepath.Base(exe))
}

func (c *Cache) file(exe, key string) string {
	return filepath.Join(c.cmdDir(exe), key+".json")
}

// prune removes sibling files older than the TTL (stale binary keys).
func (c *Cache) prune(dir string) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		if st, err := de.Info(); err == nil && time.Since(st.ModTime()) > TTL {
			_ = os.Remove(filepath.Join(dir, de.Name()))
		}
	}
}
