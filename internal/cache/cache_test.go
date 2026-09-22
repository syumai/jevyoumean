package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/syumai/jevyoumean/internal/helptext"
)

// fakeExe creates a real file so os.Stat succeeds in Save/Load.
func fakeExe(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fakecli")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSaveLoadRoundTrip(t *testing.T) {
	c := &Cache{dir: t.TempDir()}
	exe := fakeExe(t)
	e := &Entry{
		Executable:  exe,
		Subcommands: []helptext.Command{{Name: "view", Description: "View things"}},
	}
	c.Save(e, nil)
	got, ok := c.Load(exe, nil, nil)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got.Subcommands) != 1 || got.Subcommands[0].Name != "view" {
		t.Fatalf("unexpected entry: %+v", got)
	}
}

func TestCachePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows has no Unix permission bits; the modes set in Save
		// are best-effort and os.Stat does not reflect them.
		t.Skip("permission bits are a Unix concept")
	}
	c := &Cache{dir: t.TempDir()}
	exe := fakeExe(t)
	c.Save(&Entry{Executable: exe, Subcommands: []helptext.Command{{Name: "x"}}}, nil)

	st, err := os.Stat(c.cmdDir(exe))
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Mode().Perm(); got != 0o700 {
		t.Fatalf("cache dir mode %o, want 0700", got)
	}
	des, err := os.ReadDir(c.cmdDir(exe))
	if err != nil || len(des) != 1 {
		t.Fatalf("expected one cache file: %v %d", err, len(des))
	}
	st, err = des[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Fatalf("cache file mode %o, want 0600", got)
	}
}

// TestInvalidateStaysInsideCache is the regression test for the
// --refresh path traversal: a user-controlled command name must never
// delete anything outside the cache root.
func TestInvalidateStaysInsideCache(t *testing.T) {
	root := t.TempDir()
	c := &Cache{dir: filepath.Join(root, "jym")}

	// A sibling directory outside the cache root that must survive.
	outside := filepath.Join(root, "precious")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(victim, []byte("do not delete"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A legit per-command cache dir that SHOULD be removed.
	inside := filepath.Join(c.dir, "mycli")
	if err := os.MkdirAll(inside, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"../precious",
		"../../precious",
		outside, // absolute path
		"mycli/../../precious",
		"./../precious",
		"..",
		".",
		"/",
		"",
	} {
		c.Invalidate(name)
		if _, err := os.Stat(victim); err != nil {
			t.Fatalf("Invalidate(%q) deleted outside the cache: %v", name, err)
		}
	}

	// Basename still resolves: Invalidate("mycli") removes its dir.
	c.Invalidate("mycli")
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("Invalidate(\"mycli\") did not remove the cache dir")
	}
	// And a resolved executable path removes only its own basename dir.
	if err := os.MkdirAll(inside, 0o700); err != nil {
		t.Fatal(err)
	}
	c.Invalidate("/usr/local/bin/mycli")
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("Invalidate on a resolved path did not remove the cache dir")
	}
}

// TestSaveAtomic ensures no partial JSON is ever observable: writing
// replaces the destination via rename, and no temp files linger.
func TestSaveAtomic(t *testing.T) {
	c := &Cache{dir: t.TempDir()}
	exe := fakeExe(t)
	e := &Entry{Executable: exe, Subcommands: []helptext.Command{{Name: "a"}}}
	c.Save(e, nil)
	e.Subcommands[0].Name = "b"
	c.Save(e, nil)

	des, err := os.ReadDir(c.cmdDir(exe))
	if err != nil {
		t.Fatal(err)
	}
	for _, de := range des {
		if filepath.Ext(de.Name()) != ".json" {
			t.Fatalf("leftover temp file: %s", de.Name())
		}
		data, err := os.ReadFile(filepath.Join(c.cmdDir(exe), de.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var decoded Entry
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("partial write left invalid JSON: %v", err)
		}
		if decoded.Subcommands[0].Name != "b" {
			t.Fatalf("expected newest entry, got %+v", decoded)
		}
	}
}

func TestLoadRejectsStaleAndExpired(t *testing.T) {
	c := &Cache{dir: t.TempDir()}
	exe := fakeExe(t)
	e := &Entry{Executable: exe, Subcommands: []helptext.Command{{Name: "a"}}}
	c.Save(e, nil)

	// Expire the entry by rewriting FetchedAt inside the file.
	st, _ := os.Stat(exe)
	dir := c.cmdDir(exe)
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("expected one file, got %d", len(des))
	}
	p := filepath.Join(dir, des[0].Name())
	data, _ := os.ReadFile(p)
	var stored Entry
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	stored.FetchedAt = time.Now().Add(-2 * TTL).Unix()
	out, _ := json.Marshal(stored)
	if err := os.WriteFile(p, out, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Load(exe, nil, nil); ok {
		t.Fatal("expected TTL-expired entry to miss")
	}

	// A changed binary (size/mtime) invalidates too.
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n# newer\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if nst, _ := os.Stat(exe); nst.Size() != st.Size() {
		if _, ok := c.Load(exe, nil, nil); ok {
			t.Fatal("expected stale-binary entry to miss")
		}
	}
}

func TestNoopCache(t *testing.T) {
	c := &Cache{dir: ""}
	c.Save(&Entry{Executable: "/x"}, nil)
	c.Invalidate("../anything")
	c.ClearAll()
	if _, ok := c.Load("/x", nil, nil); ok {
		t.Fatal("no-op cache returned a hit")
	}
}
