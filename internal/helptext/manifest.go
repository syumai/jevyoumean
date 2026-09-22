package helptext

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Status of a manifest entry: how well the parser handles this help
// output. The set drives corpus_test.go's expected-failure semantics.
type Status string

const (
	// StatusOK means the fixture parses and yields all must_include
	// names with none of must_not_include.
	StatusOK Status = "ok"
	// StatusPartial is StatusOK with a known incomplete listing (e.g.
	// only a subset of sections is recognized).
	StatusPartial Status = "partial"
	// StatusLeaf means the command has no documented subcommands, so
	// an empty parse is the correct result.
	StatusLeaf Status = "leaf"
	// StatusEmpty means the output has a command listing the parser
	// cannot read; it must currently parse to nothing.
	StatusEmpty Status = "empty"
	// StatusBogus means the parser extracts names that are not real
	// subcommands — the most dangerous state.
	StatusBogus Status = "bogus"
	// StatusTodo is a known-broken entry whose exact failure mode is
	// not pinned; it must not yet satisfy the ok criteria.
	StatusTodo Status = "todo"
	// StatusOutOfScope documents a command jym does not intend to
	// support; corpus_test skips it.
	StatusOutOfScope Status = "out-of-scope"
)

// Source of a manifest entry's fixture.
type Source string

const (
	// SourceCapture is a verbatim capture of real help output. Only
	// used for CLIs whose license permits redistribution; see
	// testdata/help/NOTICE.md.
	SourceCapture Source = "capture"
	// SourceSynthetic is a hand-written fixture mimicking a real help
	// format, used for CLIs whose license makes verbatim capture
	// undesirable (GPL/LGPL/Artistic).
	SourceSynthetic Source = "synthetic"
	// SourceLocal means the fixture is captured locally and not
	// committed; corpus_test skips the entry when the file is absent.
	SourceLocal Source = "local"
)

// ManifestEntry is one row of testdata/help/manifest.tsv.
type ManifestEntry struct {
	Command        string   // executable name, e.g. "git"
	Path           []string // subcommand path, empty for the top level
	Source         Source
	Status         Status
	MustInclude    []string // subcommand names a successful parse must contain
	MustNotInclude []string // names a successful parse must not contain
	Note           string
}

// Title renders the entry as "git remote" (or "git").
func (e ManifestEntry) Title() string {
	return strings.Join(append([]string{e.Command}, e.Path...), " ")
}

// FixturePath is the corpus file for this entry under root:
// <root>/<command>/<path...>.txt, or <root>/<command>/root.txt at the
// top level.
func (e ManifestEntry) FixturePath(root string) string {
	parts := append([]string{e.Command}, e.Path...)
	if len(parts) == 1 {
		parts = append(parts, "root")
	}
	return filepath.Join(append([]string{root}, parts...)...) + ".txt"
}

// LoadManifest reads a manifest.tsv: tab-separated rows
// "command, path, source, status, must_include, must_not_include, note"
// with #-comments and blank lines ignored. path is space-separated
// subcommands or "-"; the include lists are comma-separated or "-".
func LoadManifest(path string) ([]ManifestEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []ManifestEntry
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 6 || len(f) > 7 {
			return nil, fmt.Errorf("manifest line %d: want 6 or 7 tab-separated fields, got %d", i+1, len(f))
		}
		e := ManifestEntry{
			Command:        strings.TrimSpace(f[0]),
			Source:         Source(strings.TrimSpace(f[2])),
			Status:         Status(strings.TrimSpace(f[3])),
			MustInclude:    splitList(f[4]),
			MustNotInclude: splitList(f[5]),
		}
		if p := strings.TrimSpace(f[1]); p != "-" {
			e.Path = strings.Fields(p)
		}
		if len(f) == 7 {
			e.Note = strings.TrimSpace(f[6])
		}
		if e.Command == "" {
			return nil, fmt.Errorf("manifest line %d: empty command", i+1)
		}
		switch e.Status {
		case StatusOK, StatusPartial, StatusLeaf, StatusEmpty, StatusBogus, StatusTodo, StatusOutOfScope:
		default:
			return nil, fmt.Errorf("manifest line %d: unknown status %q", i+1, e.Status)
		}
		switch e.Source {
		case SourceCapture, SourceSynthetic, SourceLocal:
		default:
			return nil, fmt.Errorf("manifest line %d: unknown source %q", i+1, e.Source)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func splitList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return nil
	}
	var out []string
	for _, tok := range strings.Split(s, ",") {
		if tok = strings.TrimSpace(tok); tok != "" {
			out = append(out, tok)
		}
	}
	return out
}
