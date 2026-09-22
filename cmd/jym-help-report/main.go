// jym-help-report generates docs/supported-commands.md — the checklist
// of commands whose help jym can parse — from
// testdata/help/manifest.tsv plus the committed fixtures. With -check
// it exits 1 when the committed doc is stale instead of writing it.
//
//	go run ./cmd/jym-help-report          # regenerate the doc
//	go run ./cmd/jym-help-report -check   # CI: verify it is fresh
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/syumai/jevyoumean/internal/helptext"
)

func main() {
	manifest := flag.String("manifest", "testdata/help/manifest.tsv", "manifest.tsv path")
	root := flag.String("root", "testdata/help", "corpus root directory")
	out := flag.String("out", "docs/supported-commands.md", "generated document path")
	check := flag.Bool("check", false, "exit 1 if the generated doc differs from -out")
	flag.Parse()

	doc, err := render(*manifest, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *check {
		cur, err := os.ReadFile(*out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v (run `go run ./cmd/jym-help-report`)\n", *out, err)
			os.Exit(1)
		}
		if string(cur) != doc {
			fmt.Fprintf(os.Stderr, "%s is stale; run `go run ./cmd/jym-help-report`\n", *out)
			os.Exit(1)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(doc), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func render(manifestPath, root string) (string, error) {
	entries, err := helptext.LoadManifest(manifestPath)
	if err != nil {
		return "", err
	}

	// Group nested entries under their top-level command, preserving
	// first-seen order.
	groups := map[string][]helptext.ManifestEntry{}
	var order []string
	for _, e := range entries {
		if _, ok := groups[e.Command]; !ok {
			order = append(order, e.Command)
		}
		groups[e.Command] = append(groups[e.Command], e)
	}
	sort.Strings(order)

	var b strings.Builder
	b.WriteString(`# Supported commands

Which CLI help outputs jym can extract subcommands from. Generated from
` + "`testdata/help/manifest.tsv`" + ` and the fixtures under ` + "`testdata/help/`" + ` by
` + "`go run ./cmd/jym-help-report`" + ` — do not edit by hand; change the manifest and
regenerate instead. Checked rows are asserted by ` + "`go test ./internal/helptext`" + `.

Status: **ok** parses fully, **partial** parses a known subset, **leaf**
has no documented subcommands (empty is correct), **empty** currently
yields nothing, **bogus** extracts words that are not subcommands,
**todo** is tracked for support, **out-of-scope** will not be supported.

`)

	summary := map[helptext.Status]int{}
	for _, cmd := range order {
		group := groups[cmd]
		first := group[0]
		for _, e := range group {
			summary[e.Status]++
		}
		if len(group) == 1 && len(first.Path) == 0 {
			b.WriteString(entryLine("- ", first, root))
			continue
		}
		b.WriteString(fmt.Sprintf("- **%s**\n", cmd))
		for _, e := range group {
			b.WriteString(entryLine("  - ", e, root))
		}
	}

	b.WriteString("\n## Totals\n\n")
	for _, s := range []helptext.Status{
		helptext.StatusOK, helptext.StatusPartial, helptext.StatusLeaf,
		helptext.StatusEmpty, helptext.StatusBogus, helptext.StatusTodo,
		helptext.StatusOutOfScope,
	} {
		if n := summary[s]; n > 0 {
			fmt.Fprintf(&b, "- %s: %d\n", s, n)
		}
	}
	return b.String(), nil
}

func entryLine(prefix string, e helptext.ManifestEntry, root string) string {
	box := "[ ]"
	if e.Status == helptext.StatusOK || e.Status == helptext.StatusPartial || e.Status == helptext.StatusLeaf {
		box = "[x]"
	}
	label := "`" + e.Title() + "`"
	status := string(e.Status)
	var extra []string
	if data, err := os.ReadFile(e.FixturePath(root)); err == nil {
		if cmds := helptext.Parse(string(data)); len(cmds) > 0 {
			extra = append(extra, fmt.Sprintf("%d parsed", len(cmds)))
		}
	}
	if e.Source == helptext.SourceLocal {
		extra = append(extra, "fixture not committed")
	}
	if e.Note != "" && e.Note != "-" {
		extra = append(extra, e.Note)
	}
	line := fmt.Sprintf("%s%s %s — %s", prefix, box, label, status)
	if len(extra) > 0 {
		line += " (" + strings.Join(extra, "; ") + ")"
	}
	return line + "\n"
}
