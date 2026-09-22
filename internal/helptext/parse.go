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

// stripOverstrike removes nroff-style overstriking used by man pages and
// groff-formatted help: X\bX renders bold, _\bX underlined. Each
// backspace erases the previous output byte.
func stripOverstrike(line string) string {
	if !strings.ContainsRune(line, '\b') {
		return line
	}
	out := make([]byte, 0, len(line))
	for i := 0; i < len(line); i++ {
		if line[i] == '\b' {
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, line[i])
	}
	return string(out)
}

// looksLikeName filters out tokens that match the name charset but are
// almost certainly not subcommand names. Subcommands are lowercase by
// convention, so a multi-character token starting with a capital letter
// is a prose sentence starter ("The", "Install"), and a token without
// any lowercase letter is an environment variable (GIT_CONFIG_GLOBAL),
// a placeholder (UNIT) or a list marker ("1."). Names end in a letter
// or digit, never a full stop — prose fragments like "services." are
// rejected too. Single characters are kept either way.
func looksLikeName(n string) bool {
	if !namePattern.MatchString(n) || strings.HasSuffix(n, ".") {
		return false
	}
	if len(n) > 1 && n[0] >= 'A' && n[0] <= 'Z' {
		return false
	}
	for i := 0; i < len(n); i++ {
		if n[i] >= 'a' && n[i] <= 'z' {
			return true
		}
	}
	return len(n) <= 1
}

// headerSkipWords mark non-indented lines that introduce sections which
// never list runnable subcommands. Checked against tentative headers and
// in the loose fallback scan.
var headerSkipWords = []string{
	"option", "flag", "interface", "environment", "example",
	"parameter", "argument", "suffix", "alias",
}

// headerHead extracts the label portion of a candidate header: the text
// before a colon or parenthesis, so "Usage: tool <command> [options]" is
// judged on "usage" and does not trip on "[options]".
func headerHead(lower string) string {
	if i := strings.IndexAny(lower, ":("); i >= 0 {
		lower = lower[:i]
	}
	return strings.TrimSpace(lower)
}

func isSkippableHeader(lower string) bool {
	head := headerHead(lower)
	for _, w := range headerSkipWords {
		if strings.Contains(head, w) {
			return true
		}
	}
	return false
}

// bareKey reports whether line is an indented "name:" with nothing after
// the colon — the opening key of a YAML or config example block.
func bareKey(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasSuffix(t, ":") || len(t) < 2 {
		return false
	}
	return namePattern.MatchString(strings.TrimSuffix(t, ":"))
}

// Parse extracts subcommands from help output. An empty result means
// "cannot judge": the output had no recognizable command listing.
//
// Two section styles are recognized. Known headers (a column-0 line
// containing "command") commit however many entries follow. Any other
// non-indented line is a tentative header (e.g. git's "Interacting with
// Others") and commits only when at least two entry lines follow, which
// keeps single stray columns from becoming commands. Headers mentioning
// "interface" are skipped: git lists file-format/protocol entries there
// that are not runnable commands.
func Parse(output string) []Command {
	var cmds, pending []Command
	seen, sectionSeen := map[string]bool{}, map[string]bool{}
	inSection := false
	tentative := false
	flatMode := false
	sawKnownHeader := false
	entryIndent := -1
	yamlDepth := -1
	prefixSeen := map[string]map[string]string{}
	var prefixOrder []string

	startSection := func(tent bool) {
		pending = nil
		sectionSeen = map[string]bool{}
		inSection, tentative = true, tent
		flatMode = false
		if !tent {
			sawKnownHeader = true
		}
		entryIndent = -1
		yamlDepth = -1
	}
	flush := func() {
		min := 1
		if tentative {
			min = 2
		}
		if len(pending) >= min {
			for _, c := range pending {
				cmds = appendCommand(cmds, seen,
					append([]string{c.Name}, c.Aliases...), c.Description)
			}
		}
		pending = nil
		inSection = false
	}

	for _, line := range strings.Split(output, "\n") {
		line = stripOverstrike(strings.TrimRight(line, "\r"))
		// "tool sub ARGS..." — CLIs like brew and yarn repeat their own
		// name before every subcommand in usage-style listings. Lines
		// are collected wherever they appear; a prefix group only
		// counts when the same leading token yields two or more
		// distinct names, so prose cannot inject bogus names.
		if strings.TrimSpace(line) != "" {
			if f0, f1, fd, ok := prefixedEntry(line); ok {
				if prefixSeen[f0] == nil {
					prefixSeen[f0] = map[string]string{}
					prefixOrder = append(prefixOrder, f0)
				}
				if _, ok := prefixSeen[f0][f1]; !ok || fd != "" {
					prefixSeen[f0][f1] = fd
				}
			}
		}
		if isSectionHeader(line) {
			flush()
			startSection(false)
			continue
		}
		// Blank lines occur between a section header and its entries,
		// and between entries themselves in man-page-style listings.
		// Sections end at the next non-indented line instead.
		if strings.TrimSpace(line) == "" && inSection {
			continue
		}
		if indentation(line) == 0 {
			// Unindented "name (alias) args" entries — listings like
			// `tmux list-commands` put entries at column zero where
			// they would otherwise read as headers. A tentative flat
			// section still needs two entries to commit.
			if names, ok := flatEntry(line); ok {
				if !flatMode {
					flush()
					startSection(true)
					flatMode = true
				}
				pending = appendCommand(pending, sectionSeen, names, "")
				continue
			}
			flatMode = false
			// Every other non-indented line is a tentative header.
			flush()
			if s := strings.ToLower(strings.TrimSpace(line)); s != "" &&
				!isSkippableHeader(s) {
				startSection(true)
			}
			continue
		}
		if !inSection {
			continue
		}
		// Indented "key:" blocks (YAML and similar config examples
		// embedded in prose, like helm's Chart.yaml samples) keep all
		// deeper content from being read as entries until the block
		// dedents back or the section ends at column zero.
		if lineIndent := indentation(line); yamlDepth >= 0 {
			if lineIndent >= yamlDepth {
				continue
			}
			yamlDepth = -1
		}
		if bareKey(line) && yamlDepth < 0 {
			yamlDepth = indentation(line)
			continue
		}
		names, desc, indent, ok := parseEntry(line)
		if !ok {
			if len(pending) == 0 {
				// Prose between a section header and its first
				// entry (common in man pages) does not end the
				// section.
				continue
			}
			// A deeper-indented line is probably a wrapped description.
			if lineIndent := indentation(line); entryIndent >= 0 && lineIndent > entryIndent {
				last := &pending[len(pending)-1]
				last.Description = strings.TrimSpace(last.Description + " " + strings.TrimSpace(line))
				continue
			}
			flush()
			continue
		}
		if entryIndent < 0 {
			entryIndent = indent
		}
		// npm-style help prints a comma-separated command inventory with
		// no descriptions. Those names are independent commands, not aliases.
		if desc == "" && len(names) > 1 {
			for _, name := range names {
				pending = appendCommand(pending, sectionSeen, []string{name}, "")
			}
			continue
		}
		pending = appendCommand(pending, sectionSeen, names, desc)
	}
	flush()
	// Prefix-form listings (brew, yarn) count as a result when the same
	// tool-name prefix introduces two or more distinct names. This also
	// rescues outputs whose only listing sits inside a skipped or empty
	// section. Two-token prefixes ("npm cache", "bun pm") are used only
	// when no one-token group qualified — "yarn config get X" inside a
	// root listing must not fabricate a top-level "get" command.
	if len(cmds) == 0 {
		oneToken, twoToken := false, false
		for _, f0 := range prefixOrder {
			if len(prefixSeen[f0]) < 2 {
				continue
			}
			if len(strings.Fields(f0)) == 1 {
				oneToken = true
			} else {
				twoToken = true
			}
		}
		depth := 1
		if !oneToken && twoToken {
			depth = 2
		}
		for _, f0 := range prefixOrder {
			if len(prefixSeen[f0]) < 2 || len(strings.Fields(f0)) != depth {
				continue
			}
			for name, d := range prefixSeen[f0] {
				cmds = appendCommand(cmds, seen, []string{name}, d)
			}
		}
	}
	if len(cmds) > 0 {
		return cmds
	}
	// Fallback: when no recognized command section existed at all, scan
	// every line for two-column "name   description" entries. A known
	// header that yielded nothing means "cannot judge", so loose scan
	// is skipped. Like tentative sections, a single stray line is not
	// enough.
	if sawKnownHeader {
		return nil
	}
	if loose := parseLoose(output); len(loose) >= 2 {
		return loose
	}
	return nil
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
	skipSection := false
	for _, line := range strings.Split(output, "\n") {
		line = stripOverstrike(strings.TrimRight(line, "\r"))
		if indentation(line) == 0 {
			// Non-indented lines are section boundaries here too, so
			// value tables under e.g. "Suffixes accepted by ..." do not
			// leak in as commands.
			// Blank lines keep the current skip state.
			if s := strings.ToLower(strings.TrimSpace(line)); s != "" {
				skipSection = isSkippableHeader(s)
			}
			continue
		}
		if skipSection {
			continue
		}
		names, desc, _, ok := parseEntry(line)
		if !ok || desc == "" {
			continue
		}
		cmds = appendCommand(cmds, seen, names, desc)
	}
	return cmds
}

// flatEntry recognizes unindented listing lines of the form
// "name (alias) [flags] args" used by `tmux list-commands` and similar
// usage dumps: a leading name, an optional parenthesized alias list,
// then argument placeholders. Bare lowercase positionals ("command",
// "match-string") count only after a structured arg appeared, so prose
// lines cannot pass. Returns names (name first, then aliases).
func flatEntry(line string) (names []string, ok bool) {
	trimmed := strings.TrimSpace(line)
	i := strings.IndexAny(trimmed, " \t")
	if i < 0 {
		return nil, false
	}
	if !looksLikeName(trimmed[:i]) {
		return nil, false
	}
	names = append(names, trimmed[:i])
	rest := strings.TrimSpace(trimmed[i:])
	structured := false
	if strings.HasPrefix(rest, "(") {
		e := strings.Index(rest, ")")
		if e < 0 {
			return nil, false
		}
		for _, a := range strings.Fields(rest[1:e]) {
			if !looksLikeName(a) {
				return nil, false
			}
			names = append(names, a)
		}
		rest = strings.TrimSpace(rest[e+1:])
		structured = true
	}
	for rest != "" {
		var token string
		if j := strings.IndexAny(rest, " \t"); j < 0 {
			token, rest = rest, ""
		} else {
			token, rest = rest[:j], strings.TrimSpace(rest[j:])
		}
		if token == "" {
			continue
		}
		switch {
		case token[0] == '[' || token[0] == '<':
			close := "]"
			if token[0] == '<' {
				close = ">"
			}
			for !strings.Contains(token, close) {
				k := strings.IndexAny(rest, " \t")
				if k < 0 {
					if !strings.Contains(rest, close) {
						return nil, false
					}
					token += rest
					rest = ""
					break
				}
				token += rest[:k+1]
				rest = strings.TrimSpace(rest[k+1:])
			}
			structured = true
		case token == "..." || token[0] == '-' || isPlaceholder(token):
			structured = true
		case structured && isLowerWord(token):
		default:
			return nil, false
		}
	}
	if !structured {
		return nil, false
	}
	return names, true
}

// isLowerWord reports whether s is a bare lowercase word used for
// positional argument names ("command", "match-string").
func isLowerWord(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

// prefixedEntry recognizes usage-style lines of the form
// "tool subcommand ARG...": the leading tokens repeat the tool's own
// invocation path ("brew install X", "npm cache add X") and everything
// after the name is argument placeholders, flags or brackets. Returns
// the prefix and the subcommand name; the caller decides whether the
// prefix is trusted. One- and two-token prefixes are tried.
func prefixedEntry(line string) (prefix, name, desc string, ok bool) {
	fields := strings.Fields(line)
	trimmed := strings.TrimSpace(line)
	for plen := 1; plen <= 2; plen++ {
		if len(fields) <= plen {
			continue
		}
		// A one-token prefix needs at least one arg token after the
		// name; a two-token prefix already carries the path so a bare
		// name ("npm cache verify") counts too.
		if plen == 1 && len(fields) < 3 {
			continue
		}
		idx := 0
		valid := true
		for i := 0; i <= plen; i++ {
			if !looksLikeName(fields[i]) {
				valid = false
				break
			}
			j := strings.Index(trimmed[idx:], fields[i])
			if j < 0 {
				valid = false
				break
			}
			idx += j + len(fields[i])
		}
		if !valid {
			continue
		}
		rest := trimmed[idx:]
		// "tool sub   description" — a whitespace run of two or more
		// after the name opens the description column ("bun pm scan
		//                 scan all packages ...").
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		if j >= 2 {
			d := strings.TrimSpace(rest[j:])
			if d != "" {
				return strings.Join(fields[:plen], " "), fields[plen], d, true
			}
			continue
		}
		if !argsOnly(strings.TrimSpace(rest)) {
			continue
		}
		return strings.Join(fields[:plen], " "), fields[plen], "", true
	}
	return "", "", "", false
}

// argsOnly reports whether s consists only of argument-like tokens:
// "[...]" or "<...>" groups (which may contain spaces), "..." ellipses,
// "-" prefixed flags and uppercase placeholders like FORMULA or FILE.
func argsOnly(s string) bool {
	for s != "" {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			return true
		}
		if s[0] == '[' || s[0] == '<' {
			close := "]"
			if s[0] == '<' {
				close = ">"
			}
			e := strings.Index(s, close)
			if e < 0 {
				return false
			}
			s = s[e+1:]
			continue
		}
		var token string
		if i := strings.IndexAny(s, " \t"); i < 0 {
			token, s = s, ""
		} else {
			token, s = s[:i], s[i:]
		}
		if token == "..." || token[0] == '-' || isPlaceholder(token) {
			continue
		}
		return false
	}
	return true
}

func indentation(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// isSectionHeader reports whether line introduces a command listing
// section. Headers are never indented; command entries always are.
// Both bare ("CORE COMMANDS") and colon-terminated ("Available
// Commands:", "The most commonly used git commands are:") forms match,
// as long as the line contains the plural "commands" — the singular is
// avoided so placeholders like "<command>" in usage lines do not count;
// a real singular header still wins through the tentative rule.
// Sections about "interfaces" are excluded: they document concepts,
// not runnable subcommands.
func isSectionHeader(line string) bool {
	if indentation(line) != 0 {
		return false
	}
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}
	lower := strings.ToLower(strings.TrimSuffix(s, ":"))
	if i := strings.Index(lower, "("); i >= 0 {
		lower = strings.TrimSpace(lower[:i])
	}
	// The plural word is required so placeholders like "<command>" in
	// usage lines do not count. Singular headers still win through the
	// tentative-section rule when two or more entries follow.
	return strings.Contains(lower, "commands") && !strings.Contains(lower, "interface")
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
	if names, desc, ok := parseEntryLine(trimmed); ok {
		return names, desc, indent, true
	}
	// nroff renders bulleted entries as "o name  description". Retry
	// without the bullet; a leading "o" never parses on its own because
	// the single space after it already fails the entry check.
	if rest := strings.TrimPrefix(trimmed, "o "); rest != trimmed {
		if names, desc, ok := parseEntryLine(rest); ok {
			return names, desc, indent, true
		}
	}
	return nil, "", 0, false
}

func parseEntryLine(trimmed string) (names []string, desc string, ok bool) {
	// Colon form: "name: description" (aliases may follow: "a, b: ...").
	// A colon preceded by a whitespace run of two or more sits inside an
	// already-separated description ("change   Record a change intent:
	// ..."), so the whitespace form takes precedence there. The colon
	// must be followed by actual text: a bare "key:" line is YAML-style
	// example data, not a command entry.
	if i := strings.IndexByte(trimmed, ':'); i > 0 &&
		i+1 < len(trimmed) && (trimmed[i+1] == ' ' || trimmed[i+1] == '\t') &&
		strings.TrimSpace(trimmed[i+1:]) != "" && !hasGap(trimmed[:i]) {
		if ns := splitNames(trimmed[:i]); len(ns) > 0 {
			// Space-separated words before a colon are a prose
			// heading ("Install packages from:"), not an alias list.
			// Real alias lists use commas or pipes ("a, b:" / "a|b:").
			if len(ns) > 1 && !strings.ContainsAny(trimmed[:i], ",|") {
				return nil, "", false
			}
			return ns, strings.TrimSpace(trimmed[i+1:]), true
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
			if !looksLikeName(n) {
				return nil, "", false
			}
			names = append(names, n)
		}
		if rest == "" {
			return names, "", true
		}
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		if j >= 2 {
			return names, strings.TrimSpace(rest[j:]), true
		}
		if !hadComma {
			// "name - description" is a common apt/help2man style.
			if d := strings.TrimPrefix(rest[j:], "- "); d != rest[j:] {
				return names, strings.TrimSpace(d), true
			}
			// Argument placeholders between the name and the
			// description ("start UNIT...", "status [PATTERN...]").
			if d, ok := skipArgs(rest); ok {
				return names, d, true
			}
			// "name word ..." with single spaces is prose, not an entry.
			return nil, "", false
		}
		rest = rest[j:]
	}
}

// hasGap reports whether s contains a run of at least two whitespace
// characters — the column separator used by whitespace-form entries.
func hasGap(s string) bool {
	for i := 1; i < len(s); i++ {
		if (s[i] == ' ' || s[i] == '\t') && (s[i-1] == ' ' || s[i-1] == '\t') {
			return true
		}
	}
	return false
}

// skipArgs consumes argument-placeholder tokens following a name —
// bracketed forms like "[PATTERN...]" or "<file>", and uppercase words
// like "UNIT..." or "PROPERTY=VALUE..." — and returns the description
// after a run of two or more spaces, or "" when the line ends.
func skipArgs(rest string) (desc string, ok bool) {
	for {
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		if j >= 2 {
			return strings.TrimSpace(rest[j:]), true
		}
		if j == 0 || j >= len(rest) {
			return "", false
		}
		rest = rest[j:]
		i := strings.IndexAny(rest, " \t")
		var token string
		if i < 0 {
			token, rest = rest, ""
		} else {
			token, rest = rest[:i], rest[i:]
		}
		// A bare colon is the actual separator between args and the
		// description ("config [Experimental] : Manage ...").
		if token == ":" {
			return strings.TrimSpace(rest), true
		}
		if token[0] == '[' || token[0] == '<' {
			// Bracket groups may contain spaces
			// ("[-c working-directory]"): consume to the close.
			close := "]"
			if token[0] == '<' {
				close = ">"
			}
			for !strings.Contains(token, close) {
				k := strings.IndexAny(rest, " \t")
				if k < 0 {
					if !strings.Contains(rest, close) {
						return "", false
					}
					token += rest
					rest = ""
					break
				}
				token += rest[:k+1]
				rest = rest[k+1:]
			}
		} else if !isPlaceholder(token) {
			return "", false
		}
		if rest == "" {
			return "", true
		}
	}
}

// isPlaceholder reports whether token looks like a CLI argument
// placeholder rather than a subcommand name: "[...]", "<...>", or an
// uppercase token like "UNIT", "PATTERN..." or "PROPERTY=VALUE".
func isPlaceholder(token string) bool {
	if token == "" {
		return false
	}
	if token[0] == '[' || token[0] == '<' {
		return true
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '=' || c == '.' || c == '|' || c == '+' || c == '/') {
			return false
		}
	}
	return token[0] >= 'A' && token[0] <= 'Z'
}

// splitNames parses the left-hand side of a colon-form entry, accepting
// comma and pipe separated alias lists ("a, b" / "a|b").
func splitNames(s string) []string {
	var names []string
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '|' || r == ' ' || r == '\t'
	}) {
		if !looksLikeName(tok) {
			return nil
		}
		names = append(names, tok)
	}
	return names
}
