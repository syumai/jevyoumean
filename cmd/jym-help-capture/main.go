// jym-help-capture records real CLI help output into the help corpus in
// testdata/help and prints manifest.tsv stub rows for what it captured.
//
// Usage:
//
//	go run ./cmd/jym-help-capture docker "docker compose" gh "gh pr"
//	go run ./cmd/jym-help-capture -help-args "help -a" git "git remote"
//
// Each argument is a command spec: the executable name optionally
// followed by a subcommand path. The fixture lands at
// <root>/<command>/<path>.txt (root.txt for the top level). Review the
// output's license before committing captures — see
// testdata/help/NOTICE.md; GPL/LGPL/Artistic CLIs should use
// source=local (fixture not committed) or hand-written synthetic
// fixtures instead.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/syumai/jevyoumean/internal/helptext"
)

func main() {
	root := flag.String("root", "testdata/help", "corpus root directory")
	helpArgs := flag.String("help-args", "", "help invocation args for all specs (e.g. \"help -a\"), replacing the default tries")
	flag.Parse()
	specs := flag.Args()
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: jym-help-capture [-root dir] [-help-args \"args\"] \"cmd [sub...]\"...")
		os.Exit(2)
	}
	var ha []string
	if *helpArgs != "" {
		ha = strings.Fields(*helpArgs)
	}
	ctx := context.Background()
	failed := false
	for _, spec := range specs {
		parts := strings.Fields(spec)
		if len(parts) == 0 {
			continue
		}
		cmd, path := parts[0], parts[1:]
		if err := capture(ctx, *root, cmd, path, ha); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", spec, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func capture(ctx context.Context, root string, cmd string, path, helpArgs []string) error {
	exe, err := exec.LookPath(cmd)
	if err != nil {
		return fmt.Errorf("not on PATH: %w", err)
	}
	raw, cmds, err := helptext.FetchRaw(ctx, exe, path, helpArgs)
	if len(raw) == 0 {
		return fmt.Errorf("no help output (err=%v)", err)
	}

	entry := helptext.ManifestEntry{Command: cmd, Path: path}
	dest := entry.FixturePath(root)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# captured %s\n", time.Now().UTC().Format("2006-01-02"))
	fmt.Fprintf(&b, "# exe: %s\n", exe)
	fmt.Fprintf(&b, "# args: %s\n", strings.Join(helpArgs, " "))
	b.Write(raw)
	if err := os.WriteFile(dest, []byte(b.String()), 0o644); err != nil {
		return err
	}

	status := helptext.StatusEmpty
	var include []string
	if len(cmds) > 0 {
		status = helptext.StatusOK
		for _, c := range cmds {
			include = append(include, c.Name)
		}
	}
	fmt.Printf("%s\t%s\tsource\t%s\t%s\t-\tnote\t# %d parsed -> %s\n",
		cmd, pathLabel(path), status, strings.Join(include, ","), len(cmds), dest)
	return nil
}

func pathLabel(path []string) string {
	if len(path) == 0 {
		return "-"
	}
	return strings.Join(path, " ")
}
