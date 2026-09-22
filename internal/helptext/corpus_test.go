package helptext

import (
	"os"
	"testing"
)

const corpusRoot = "../../testdata/help"

// TestCorpus parses every committed help fixture in testdata/help and
// asserts the status recorded in manifest.tsv. Known-broken entries
// (empty, bogus, todo) are expected failures: the test fails when they
// start passing, so a fixed parser forces a manifest update — the
// manifest is the single source of truth for support status.
func TestCorpus(t *testing.T) {
	entries, err := LoadManifest(corpusRoot + "/manifest.tsv")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	for _, e := range entries {
		t.Run(e.Title(), func(t *testing.T) {
			if e.Status == StatusOutOfScope {
				t.Skip("out of scope")
			}
			data, err := os.ReadFile(e.FixturePath(corpusRoot))
			if err != nil {
				if e.Source == SourceLocal || e.Status == StatusTodo {
					t.Skipf("no fixture (%v)", err)
				}
				t.Fatalf("fixture missing: %v", err)
			}
			cmds := Parse(string(data))
			missing, spurious := checkExpectations(cmds, e)

			switch e.Status {
			case StatusOK, StatusPartial:
				if len(missing) > 0 {
					t.Errorf("missing expected subcommands %v (got %v)", missing, names(cmds))
				}
				if len(spurious) > 0 {
					t.Errorf("spurious subcommands %v (got %v)", spurious, names(cmds))
				}
			case StatusLeaf, StatusEmpty:
				if len(cmds) > 0 {
					t.Errorf("status is %s but now parses %v; verify the result and update the manifest", e.Status, names(cmds))
				}
			case StatusBogus:
				if len(cmds) == 0 {
					t.Errorf("no longer produces bogus output; update status to empty")
				} else if len(missing) == 0 && len(spurious) == 0 {
					t.Errorf("fixture now parses correctly; update status to ok")
				}
			case StatusTodo:
				if len(cmds) > 0 && len(missing) == 0 && len(spurious) == 0 {
					t.Errorf("fixture now parses correctly; update status to ok")
				}
			}
		})
	}
}

// checkExpectations returns the must_include names absent from cmds and
// the must_not_include names present in it (matching names and aliases).
func checkExpectations(cmds []Command, e ManifestEntry) (missing, spurious []string) {
	present := func(name string) bool {
		for _, c := range cmds {
			if c.Matches(name) {
				return true
			}
		}
		return false
	}
	for _, want := range e.MustInclude {
		if !present(want) {
			missing = append(missing, want)
		}
	}
	for _, ban := range e.MustNotInclude {
		if present(ban) {
			spurious = append(spurious, ban)
		}
	}
	return missing, spurious
}
