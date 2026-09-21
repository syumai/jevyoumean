// Package helptext discovers documented subcommands by running a CLI's
// help output and parsing command listing sections heuristically.
//
// There is no standard help format, so the parser is deliberately
// heuristic: it looks for section headers like "Commands:",
// "Available Commands:", "CORE COMMANDS" or "Subcommands:", then reads
// "name   description" and "name:   description" lines below them.
// When no header is found it falls back to scanning for two-column
// lines in the whole output.
package helptext

import (
	"regexp"
	"strings"
)

// Command is a single documented subcommand. Aliases are alternate
// spellings declared on the same help line ("co, checkout", "rm|del").
type Command struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases,omitempty"`
}

// Matches reports whether token names this command or one of its aliases.
func (c Command) Matches(token string) bool {
	if c.Name == token {
		return true
	}
	for _, a := range c.Aliases {
		if a == token {
			return true
		}
	}
	return false
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Parse extracts subcommands from help output. An empty result means
// "cannot judge": the output had no recognizable command listing.
func Parse(output string) []Command {
	var cmds []Command
	seen := map[string]bool{}
	inSection := false
	sawHeader := false
	entryIndent := -1
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if isSectionHeader(line) {
			inSection = true
			sawHeader = true
			entryIndent = -1
			continue
		}
		if !inSection {
			continue
		}
		if strings.TrimSpace(line) == "" {
			inSection = false
			continue
		}
		names, desc, indent, ok := parseEntry(line)
		if !ok {
			// A deeper-indented line is probably a wrapped description.
			if lineIndent := indentation(line); entryIndent >= 0 && lineIndent > entryIndent && len(cmds) > 0 {
				last := &cmds[len(cmds)-1]
				last.Description = strings.TrimSpace(last.Description + " " + strings.TrimSpace(line))
				continue
			}
			inSection = false
			continue
		}
		if entryIndent < 0 {
			entryIndent = indent
		}
		cmds = appendCommand(cmds, seen, names, desc)
	}
	if len(cmds) > 0 || sawHeader {
		return cmds
	}
	// Fallback: no section header found, so scan every line for
	// two-column "name   description" entries.
	return parseLoose(output)
}

// appendCommand registers a parsed entry, deduplicating by name and
// spreading aliases across every name on the line.
func appendCommand(cmds []Command, seen map[string]bool, names []string, desc string) []Command {
	if len(names) == 0 {
		return cmds
	}
	aliases := names[1:]
	if !seen[names[0]] {
		seen[names[0]] = true
		cmds = append(cmds, Command{Name: names[0], Description: desc, Aliases: aliases})
	}
	for _, a := range aliases {
		if !seen[a] {
			seen[a] = true
		}
	}
	return cmds
}

// parseLoose picks up two-column lines anywhere in the output, used when
// the help has no recognizable section header. Flag-like names and
// placeholders are excluded.
func parseLoose(output string) []Command {
	var cmds []Command
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		names, desc, _, ok := parseEntry(line)
		if !ok || desc == "" {
			continue
		}
		cmds = appendCommand(cmds, seen, names, desc)
	}
	return cmds
}

func indentation(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// isSectionHeader reports whether line introduces a command listing
// section. Headers are never indented; command entries always are.
func isSectionHeader(line string) bool {
	if indentation(line) != 0 {
		return false
	}
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}
	hasColon := strings.HasSuffix(s, ":")
	lower := strings.ToLower(strings.TrimSuffix(s, ":"))
	if i := strings.Index(lower, "("); i >= 0 {
		lower = strings.TrimSpace(lower[:i])
	}
	if !hasColon {
		return lower == "command" || lower == "commands" || lower == "subcommands" ||
			strings.HasSuffix(lower, " command") || strings.HasSuffix(lower, " commands") ||
			strings.HasSuffix(lower, " subcommands")
	}
	// A colon-terminated header mentioning commands, e.g. git's
	// "The most commonly used git commands are:".
	return strings.Contains(lower, "command")
}

// parseEntry extracts a "names <sep> description" entry line and returns
// every name on it (canonical name first, then aliases).
// Two separator forms are recognized: a colon directly after the name
// ("auth:  Authenticate ...") and a whitespace run of at least two
// characters ("create   Create a resource"). A single space is treated
// as prose, not an entry, to reduce false positives.
// Alias lists are expanded: "add, a" and "rm|del" yield ["add","a"] and
// ["rm","del"] respectively.
func parseEntry(line string) (names []string, desc string, indent int, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	indent = len(line) - len(trimmed)
	if indent == 0 || trimmed == "" {
		return nil, "", 0, false
	}

	// Colon form: "name: description" (aliases may follow: "a, b: ...").
	if i := strings.IndexByte(trimmed, ':'); i > 0 &&
		(i+1 == len(trimmed) || trimmed[i+1] == ' ' || trimmed[i+1] == '\t') {
		if ns := splitNames(trimmed[:i]); len(ns) > 0 {
			return ns, strings.TrimSpace(trimmed[i+1:]), indent, true
		}
	}

	// Whitespace form.
	rest := trimmed
	for {
		i := strings.IndexAny(rest, " \t")
		var token string
		if i < 0 {
			token, rest = rest, ""
		} else {
			token, rest = rest[:i], rest[i:]
		}
		hadComma := strings.HasSuffix(token, ",")
		for _, n := range strings.Split(strings.TrimSuffix(token, ","), "|") {
			if !namePattern.MatchString(n) {
				return nil, "", 0, false
			}
			names = append(names, n)
		}
		if rest == "" {
			return names, "", indent, true
		}
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		if j >= 2 {
			return names, strings.TrimSpace(rest[j:]), indent, true
		}
		if !hadComma {
			// "name word ..." with single spaces is prose, not an entry.
			return nil, "", 0, false
		}
		rest = rest[j:]
	}
}

// splitNames parses the left-hand side of a colon-form entry, accepting
// comma and pipe separated alias lists ("a, b" / "a|b").
func splitNames(s string) []string {
	var names []string
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '|' || r == ' ' || r == '\t'
	}) {
		if !namePattern.MatchString(tok) {
			return nil
		}
		names = append(names, tok)
	}
	return names
}
