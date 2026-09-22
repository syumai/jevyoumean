package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestShowPromptColorAlways(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("JYM_COLOR", "always")
	var out bytes.Buffer
	ShowPrompt(&out, "record", "git", []string{"commit"}, []float64{0.98},
		func(string) string { return `git commit -m "Initial commit"` }, false)

	got := out.String()
	if !strings.Contains(got, ansiBoldMagenta) || !strings.Contains(got, ansiBoldGreen) {
		t.Fatalf("colored prompt lacks ANSI styles: %q", got)
	}
	plain := ansiPattern.ReplaceAllString(got, "")
	for _, want := range []string{`jym: "record" is not a git subcommand. Did you mean?`, `git commit`, `[Enter] run 1`} {
		if !strings.Contains(plain, want) {
			t.Fatalf("plain prompt missing %q: %q", want, plain)
		}
	}
}

func TestNoColorWins(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("JYM_COLOR", "always")
	var out bytes.Buffer
	ShowHint(&out, "record", "git", []string{"commit"}, []float64{0.98},
		func(string) string { return "git commit" }, false)
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("NO_COLOR output contains ANSI escapes: %q", out.String())
	}
}
