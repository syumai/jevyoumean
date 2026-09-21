package helptext

import (
	"testing"
)

func names(cmds []Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.Name
	}
	return out
}

func has(cmds []Command, name string) bool {
	for _, c := range cmds {
		if c.Matches(name) {
			return true
		}
	}
	return false
}

// gh-style output: uppercase headers without colons, "name: desc" entries.
func TestParseGHStyle(t *testing.T) {
	out := `Work seamlessly with GitHub from the command line.

USAGE
  gh <command> <subcommand> [flags]

CORE COMMANDS
  auth:        Authenticate gh and git with GitHub
  browse:      Open repositories in the browser
  issue:       Manage issues
  pr:          Manage pull requests
  repo:        Manage repositories

INHERITED FLAGS
  --help   Show help for command
`
	cmds := Parse(out)
	for _, want := range []string{"auth", "browse", "issue", "pr", "repo"} {
		if !has(cmds, want) {
			t.Fatalf("missing command %q in %v", want, names(cmds))
		}
	}
	for _, c := range cmds {
		if c.Name == "pr" && c.Description != "Manage pull requests" {
			t.Fatalf("unexpected description: %q", c.Description)
		}
	}
}

// Cobra-style output: "Available Commands:" with space-separated columns.
func TestParseCobraStyle(t *testing.T) {
	out := `A longer description.

Usage:
  tool [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  create      Create a new resource
  delete      Delete an existing resource
  help        Help about any command

Flags:
  -h, --help   help for tool

Use "tool [command] --help" for more information about a command.
`
	cmds := Parse(out)
	for _, want := range []string{"completion", "create", "delete", "help"} {
		if !has(cmds, want) {
			t.Fatalf("missing command %q in %v", want, names(cmds))
		}
	}
}

// git-style output: a colon-terminated prose header mentioning commands.
func TestParseGitStyle(t *testing.T) {
	out := `usage: git [-v | --version] [-h | --help] <command> [<args>]

The most commonly used git commands are:
   add        Add file contents to the index
   bisect     Find by binary search the change that introduced a bug
   switch     Switch branches

'git help -a' lists available subcommands.
`
	cmds := Parse(out)
	for _, want := range []string{"add", "bisect", "switch"} {
		if !has(cmds, want) {
			t.Fatalf("missing command %q in %v", want, names(cmds))
		}
	}
}

// git help -a output: multiple sections, some without the word
// "command" in the header and some listing non-command interfaces.
func TestParseGitHelpAll(t *testing.T) {
	out := `See 'git help <command>' to read about a specific subcommand

Main Porcelain Commands
   add                     Add file contents to the index
   commit                  Record changes to the repository
   status                  Show the working tree status

Ancillary Commands / Manipulators
   config                  Get and set repository or global options
   remote                  Manage set of tracked repositories
   reflog                  Manage reflog information

Interacting with Others
   archimport              Import a GNU Arch repository into Git
   svn                     Bidirectional operation between a Subversion repository and Git

User-facing repository, command and file interfaces
   attributes              Defining attributes per path
   hooks                   Hooks used by Git
   ignore                  Specifies intentionally untracked files to ignore

External commands
   lfs
   wt
`
	cmds := Parse(out)
	for _, want := range []string{"add", "status", "config", "remote", "reflog", "archimport", "svn", "lfs"} {
		if !has(cmds, want) {
			t.Fatalf("missing command %q in %v", want, names(cmds))
		}
	}
	// Interface sections document concepts, not runnable commands.
	for _, unwanted := range []string{"attributes", "hooks", "ignore"} {
		if has(cmds, unwanted) {
			t.Fatalf("interface doc %q leaked into commands %v", unwanted, names(cmds))
		}
	}
}

// A stray two-column block under an unrecognized header with only one
// entry is not a command section.
func TestParseTentativeSingleEntryRejected(t *testing.T) {
	out := `Some text.

See also
   config   an example line
`
	if cmds := Parse(out); has(cmds, "config") {
		t.Fatalf("single tentative entry should be rejected, got %v", names(cmds))
	}
}

// But two or more entries under an unrecognized header do form a section.
func TestParseTentativeSection(t *testing.T) {
	out := `Some text.

Plugins
   alpha    First plugin
   beta     Second plugin
`
	cmds := Parse(out)
	if !has(cmds, "alpha") || !has(cmds, "beta") {
		t.Fatalf("missing tentative commands in %v", names(cmds))
	}
}

// kubectl-style output: parenthesized section headers.
func TestParseKubectlStyle(t *testing.T) {
	out := `kubectl controls the Kubernetes cluster manager.

Basic Commands (Beginner):
  create          Create a resource from a file or from stdin
  expose          Take a replication controller and expose it

Basic Commands (Intermediate):
  explain         Get documentation for a resource
  get             Display one or many resources
`
	cmds := Parse(out)
	for _, want := range []string{"create", "expose", "explain", "get"} {
		if !has(cmds, want) {
			t.Fatalf("missing command %q in %v", want, names(cmds))
		}
	}
}

// No recognizable listing at all means "cannot judge".
func TestParseNoCommands(t *testing.T) {
	out := `Usage: tool [options]

Options:
  -v   verbose
`
	if cmds := Parse(out); len(cmds) != 0 {
		t.Fatalf("expected no commands, got %v", names(cmds))
	}
}

// A single-space gap between a token and prose must not produce entries.
func TestParseRejectsProse(t *testing.T) {
	out := `Commands:
  use the tool wisely
`
	if cmds := Parse(out); len(cmds) != 0 {
		t.Fatalf("expected no commands, got %v", names(cmds))
	}
}

// Comma alias lists expand so every spelling is valid.
func TestParseCommaAliases(t *testing.T) {
	out := `Commands:
  add, a    Add something
`
	cmds := Parse(out)
	if len(cmds) != 1 || cmds[0].Name != "add" {
		t.Fatalf("unexpected commands: %+v", cmds)
	}
	if !has(cmds, "a") || !has(cmds, "add") {
		t.Fatalf("alias not registered: %+v", cmds[0])
	}
	if cmds[0].Description != "Add something" {
		t.Fatalf("unexpected description: %q", cmds[0].Description)
	}
}

// Pipe alias lists ("rm|del") expand the same way.
func TestParsePipeAliases(t *testing.T) {
	out := `Commands:
  rm|del    Remove something
`
	cmds := Parse(out)
	if len(cmds) != 1 || cmds[0].Name != "rm" {
		t.Fatalf("unexpected commands: %+v", cmds)
	}
	if !has(cmds, "del") {
		t.Fatalf("alias not registered: %+v", cmds[0])
	}
}

// Without a section header the loose scan still picks up two-column
// entries while skipping flags and placeholders.
func TestParseLooseFallback(t *testing.T) {
	out := `Usage: mytool <command> [options]

  build     Build the project
  deploy    Deploy the project
  -v        verbose flag
  <arg>     placeholder-ish line
`
	cmds := Parse(out)
	if !has(cmds, "build") || !has(cmds, "deploy") {
		t.Fatalf("missing commands in %v", names(cmds))
	}
	if has(cmds, "-v") || has(cmds, "<arg>") {
		t.Fatalf("flag/placeholder leaked into %v", names(cmds))
	}
}
