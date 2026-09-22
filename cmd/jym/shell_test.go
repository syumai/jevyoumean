package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseShellIntegration(t *testing.T) {
	inv, err := parseArgs([]string{"--shell-integration", "bash", "git", "gh"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.sub != "shell-integration" || strings.Join(inv.subArgs, " ") != "bash git gh" {
		t.Fatalf("unexpected invocation: %+v", inv)
	}
}

func TestPrintShellIntegrationRejectsMixedAll(t *testing.T) {
	if code := printShellIntegration([]string{"bash", "git", "--all"}); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRenderExplicitShellIntegration(t *testing.T) {
	script, err := renderShellIntegration("bash", []string{"git", "gh", "git"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"unalias git 2>/dev/null || :",
		`function git { command jym -- git "$@"; }`,
		`function gh { command jym -- gh "$@"; }`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q:\n%s", want, script)
		}
	}
	if strings.Count(script, "function git ") != 1 {
		t.Fatalf("duplicate wrapper generated:\n%s", script)
	}
	if _, err := renderShellIntegration("bash", []string{"bad;name"}, false); err == nil {
		t.Fatal("unsafe explicit command name was accepted")
	}
	if _, err := renderShellIntegration("fish", []string{"git"}, false); err == nil {
		t.Fatal("unsupported shell was accepted")
	}
}

func TestRenderAllShellIntegration(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		script, err := renderShellIntegration(shell, []string{"git", "jym"}, true)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(script, "WARNING: --all") || !strings.Contains(script, "function git ") {
			t.Fatalf("%s script lacks warning or wrapper:\n%s", shell, script)
		}
		if strings.Contains(script, "function jym ") {
			t.Fatalf("%s script wrapped jym itself:\n%s", shell, script)
		}
	}
}

func TestExternalCommands(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha", 0o755)
	write("not-executable", 0o644)
	write("bad;name", 0o755)
	write("jym", 0o755)
	if err := os.Symlink(os.Args[0], filepath.Join(dir, "jym-alias")); err != nil {
		t.Fatal(err)
	}

	got := externalCommands(dir)
	if strings.Join(got, ",") != "alpha" {
		t.Fatalf("externalCommands() = %v, want [alpha]", got)
	}
}
