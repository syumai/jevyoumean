// Package help discovers documented subcommands by invoking a CLI's
// `--help` output and parsing command listing sections heuristically.
//
// There is no standard help format, so the parser is deliberately
// heuristic: it recognizes section headers like "Commands:",
// "Available Commands:", "CORE COMMANDS" and "Subcommands:", then
// reads "name   description" and "name:   description" lines below them.
package help

import (
	"regexp"
	"strings"
)

// Command is a single documented subcommand.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Parse extracts subcommands from help output. When no command sections
// are recognized it returns nil; callers must treat that as "unknown"
// rather than "no subcommands exist".
func Parse(output string) []Command {
	var cmds []Command
	seen := map[string]bool{}
	inSection := false
	entryIndent := -1
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if isSectionHeader(line) {
			inSection = true
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
		name, desc, indent, ok := parseEntry(line)
		if !ok {
			// A deeper-indented line is probably a wrapped description.
			lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if entryIndent >= 0 && lineIndent > entryIndent && len(cmds) > 0 {
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
		if !seen[name] {
			seen[name] = true
			cmds = append(cmds, Command{Name: name, Description: desc})
		}
	}
	return cmds
}

// isSectionHeader reports whether line introduces a command listing section.
// It matches bare headers ("CORE COMMANDS", "Available Commands:") and
// colon-terminated headers that mention commands, such as git's
// "The most commonly used git commands are:".
func isSectionHeader(line string) bool {
	// Section headers start at column 0; command entries are indented.
	if line != strings.TrimLeft(line, " \t") {
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
	return strings.Contains(lower, "command")
}

// parseEntry extracts a "name <sep> description" entry line.
// Two separator forms are recognized: a colon directly after the name
// ("auth:  Authenticate ...") and a whitespace run of at least two
// characters ("create   Create a resource"). A single space is treated
// as prose, not an entry, to reduce false positives.
func parseEntry(line string) (name, desc string, indent int, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	indent = len(line) - len(trimmed)
	if indent == 0 || trimmed == "" {
		return "", "", 0, false
	}

	// Colon form: "name: description"
	if i := strings.IndexByte(trimmed, ':'); i > 0 &&
		(i+1 == len(trimmed) || trimmed[i+1] == ' ' || trimmed[i+1] == '\t') {
		if candidate := strings.TrimSuffix(trimmed[:i], ","); namePattern.MatchString(candidate) {
			return candidate, strings.TrimSpace(trimmed[i+1:]), indent, true
		}
	}

	// Whitespace form, with alias lists ("add, a   description") collapsed
	// to the first name.
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
		token = strings.TrimSuffix(token, ",")
		if !namePattern.MatchString(token) {
			return "", "", 0, false
		}
		if name == "" {
			name = token
		}
		if rest == "" {
			return name, "", indent, true
		}
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		if j >= 2 {
			return name, strings.TrimSpace(rest[j:]), indent, true
		}
		if !hadComma {
			// "name word ..." with single spaces is prose, not an entry.
			return "", "", 0, false
		}
		rest = rest[j:]
	}
}
