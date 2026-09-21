// Package ui renders suggestions on stderr and reads the one-key
// prompt answer. stdout is never touched: it belongs to the wrapped
// command.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ShowPrompt prints the "Did you mean?" block with numbered choices.
// render maps a candidate name to the full corrected command line.
func ShowPrompt(w io.Writer, typed, context string, names []string, probs []float64, render func(string) string, offline bool) {
	fmt.Fprintf(w, "\njym: %q is not a %s subcommand. Did you mean?\n", typed, context)
	for i, name := range names {
		if offline {
			fmt.Fprintf(w, "  %d) %s\n", i+1, render(name))
		} else {
			fmt.Fprintf(w, "  %d) %s   %.2f\n", i+1, render(name), probs[i])
		}
	}
	if offline {
		fmt.Fprintln(w, "  (offline matching)")
	}
	fmt.Fprint(w, "  [Enter] run 1   [o] run as typed   [n] cancel ")
}

// ShowHint prints a non-interactive suggestion list.
func ShowHint(w io.Writer, typed, context string, names []string, probs []float64, render func(string) string, offline bool) {
	fmt.Fprintf(w, "\njym: %q is not a %s subcommand. Did you mean?\n", typed, context)
	for i, name := range names {
		if offline {
			fmt.Fprintf(w, "  %s\n", render(name))
		} else {
			fmt.Fprintf(w, "  %s   %.2f\n", render(name), probs[i])
		}
	}
	if offline {
		fmt.Fprintln(w, "  (offline matching)")
	}
}

// ShowAutoRun announces an auto-corrected execution.
func ShowAutoRun(w io.Writer, corrected string, p float64) {
	fmt.Fprintf(w, "jym: running '%s' (%.2f)\n", corrected, p)
}

// ShowOfflineNote reminds the user that Jev is unavailable.
func ShowOfflineNote(w io.Writer) {
	fmt.Fprintln(w, "jym: using offline matching. Run 'jym --setup' to enable Jev.")
}

// Answer is the outcome of the one-key prompt.
type Answer int

const (
	// Cancel aborts; jym exits 127 without running anything.
	Cancel Answer = -1
	// RunAsTyped executes the original command unchanged.
	RunAsTyped Answer = 0
)

// Prompt reads a single key from a terminal stdin in raw mode:
// Enter accepts candidate 1, digits 1-9 pick a candidate, 'o' runs the
// command as typed, 'n'/Esc/Ctrl-C cancel. It returns RunAsTyped, Cancel,
// or 1..numCandidates to accept that candidate.
func Prompt(f *os.File, w io.Writer, numCandidates int) (int, error) {
	fd := int(f.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer term.Restore(fd, old)

	var buf [1]byte
	for {
		if _, err := f.Read(buf[:]); err != nil {
			fmt.Fprintln(w)
			return 0, err
		}
		switch c := buf[0]; {
		case c == '\r' || c == '\n':
			fmt.Fprintln(w)
			return 1, nil
		case c == 'o':
			fmt.Fprintln(w)
			return int(RunAsTyped), nil
		case c == 'n' || c == 0x1b || c == 0x03:
			fmt.Fprintln(w)
			return int(Cancel), nil
		case c >= '1' && c <= '9':
			if n := int(c - '0'); n <= numCandidates {
				fmt.Fprintln(w)
				return n, nil
			}
		}
	}
}

// JoinCmd renders a command line for display.
func JoinCmd(parts ...string) string {
	return strings.Join(parts, " ")
}
