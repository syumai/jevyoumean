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

const (
	ansiReset       = "\x1b[0m"
	ansiBoldMagenta = "\x1b[1;35m"
	ansiBoldGreen   = "\x1b[1;32m"
	ansiYellow      = "\x1b[33m"
	ansiCyan        = "\x1b[36m"
	ansiDim         = "\x1b[2m"
)

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("JYM_COLOR") == "never" {
		return false
	}
	if os.Getenv("JYM_COLOR") == "always" {
		return true
	}
	f, ok := w.(*os.File)
	return ok && os.Getenv("TERM") != "dumb" && term.IsTerminal(int(f.Fd()))
}

func styled(enabled bool, code, s string) string {
	if !enabled {
		return s
	}
	return code + s + ansiReset
}

// ShowPrompt prints the "Did you mean?" block with numbered choices.
// render maps a candidate name to the full corrected command line.
func ShowPrompt(w io.Writer, typed, context string, names []string, probs []float64, render func(string) string, offline bool) {
	color := colorEnabled(w)
	fmt.Fprintf(w, "\n%s: %s is not a %s subcommand. %s\n",
		styled(color, ansiBoldMagenta, "jym"),
		styled(color, ansiYellow, fmt.Sprintf("%q", typed)), context,
		styled(color, ansiBoldMagenta, "Did you mean?"))
	for i, name := range names {
		if offline {
			fmt.Fprintf(w, "  %s %s\n",
				styled(color, ansiCyan, fmt.Sprintf("%d)", i+1)),
				styled(color, ansiBoldGreen, render(name)))
		} else {
			fmt.Fprintf(w, "  %s %s   %s\n",
				styled(color, ansiCyan, fmt.Sprintf("%d)", i+1)),
				styled(color, ansiBoldGreen, render(name)),
				styled(color, ansiDim, fmt.Sprintf("%.2f", probs[i])))
		}
	}
	if offline {
		fmt.Fprintln(w, "  "+styled(color, ansiDim, "(offline matching)"))
	}
	fmt.Fprintf(w, "  %s   %s   %s ",
		styled(color, ansiCyan, "[Enter] run 1"),
		styled(color, ansiCyan, "[o] run as typed"),
		styled(color, ansiCyan, "[n] cancel"))
}

// ShowHint prints a non-interactive suggestion list.
func ShowHint(w io.Writer, typed, context string, names []string, probs []float64, render func(string) string, offline bool) {
	color := colorEnabled(w)
	fmt.Fprintf(w, "\n%s: %s is not a %s subcommand. %s\n",
		styled(color, ansiBoldMagenta, "jym"),
		styled(color, ansiYellow, fmt.Sprintf("%q", typed)), context,
		styled(color, ansiBoldMagenta, "Did you mean?"))
	for i, name := range names {
		if offline {
			fmt.Fprintf(w, "  %s\n", styled(color, ansiBoldGreen, render(name)))
		} else {
			fmt.Fprintf(w, "  %s   %s\n",
				styled(color, ansiBoldGreen, render(name)),
				styled(color, ansiDim, fmt.Sprintf("%.2f", probs[i])))
		}
	}
	if offline {
		fmt.Fprintln(w, "  "+styled(color, ansiDim, "(offline matching)"))
	}
}

// ShowAutoRun announces an auto-corrected execution.
func ShowAutoRun(w io.Writer, corrected string, p float64) {
	color := colorEnabled(w)
	fmt.Fprintf(w, "%s: running %s (%s)\n",
		styled(color, ansiBoldMagenta, "jym"),
		styled(color, ansiBoldGreen, "'"+corrected+"'"),
		styled(color, ansiDim, fmt.Sprintf("%.2f", p)))
}

// ShowOfflineNote reminds the user that Jev is unavailable.
func ShowOfflineNote(w io.Writer) {
	color := colorEnabled(w)
	fmt.Fprintf(w, "%s: %s\n", styled(color, ansiBoldMagenta, "jym"),
		styled(color, ansiDim, "using offline matching. Run 'jym --setup' to enable Jev."))
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
			endRawLine(w)
			return 0, err
		}
		switch c := buf[0]; {
		case c == '\r' || c == '\n':
			endRawLine(w)
			return 1, nil
		case c == 'o':
			endRawLine(w)
			return int(RunAsTyped), nil
		case c == 'n' || c == 0x1b || c == 0x03:
			endRawLine(w)
			return int(Cancel), nil
		case c >= '1' && c <= '9':
			if n := int(c - '0'); n <= numCandidates {
				endRawLine(w)
				return n, nil
			}
		}
	}
}

// Raw mode disables the terminal's usual NL-to-CRLF translation. Emit both
// bytes so output from the command that runs next starts in column zero.
func endRawLine(w io.Writer) {
	fmt.Fprint(w, "\r\n")
}

// JoinCmd renders a command line for display.
func JoinCmd(parts ...string) string {
	return strings.Join(parts, " ")
}
