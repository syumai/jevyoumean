package help

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
		if c.Name == name {
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

// No command section means "cannot judge", not "no subcommands".
func TestParseNoSection(t *testing.T) {
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

// Alias lists collapse to the first name.
func TestParseAliasList(t *testing.T) {
	out := `Commands:
  add, a    Add something
`
	cmds := Parse(out)
	if len(cmds) != 1 || cmds[0].Name != "add" {
		t.Fatalf("unexpected commands: %+v", cmds)
	}
	if cmds[0].Description != "Add something" {
		t.Fatalf("unexpected description: %q", cmds[0].Description)
	}
}
