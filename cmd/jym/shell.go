package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var shellCommandName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+-]*$`)

// printShellIntegration emits sourceable bash or zsh code. Explicit mode
// deliberately replaces aliases for the named commands. --all is more
// conservative: it only wraps names that the shell currently resolves to an
// external file, preserving builtins, aliases and functions.
func printShellIntegration(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "jym: --shell-integration requires a shell: bash or zsh")
		return 2
	}
	shell := args[0]
	all := false
	for _, arg := range args[1:] {
		if arg == "--all" {
			all = true
		}
	}
	if all && (len(args) != 2 || args[1] != "--all") {
		fmt.Fprintln(os.Stderr, "jym: --all cannot be combined with command names")
		return 2
	}
	if len(args) == 1 {
		fmt.Fprintln(os.Stderr, "jym: --shell-integration needs at least one command name or --all")
		return 2
	}

	commands := args[1:]
	if all {
		commands = externalCommands(os.Getenv("PATH"))
	}
	script, err := renderShellIntegration(shell, commands, all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jym: %v\n", err)
		return 2
	}
	fmt.Print(script)
	return 0
}

func renderShellIntegration(shell string, commands []string, all bool) (string, error) {
	if shell != "bash" && shell != "zsh" {
		return "", fmt.Errorf("unsupported shell %q (want bash or zsh)", shell)
	}
	if len(commands) == 0 {
		return "", fmt.Errorf("no external commands found on PATH")
	}

	seen := make(map[string]bool, len(commands))
	var b strings.Builder
	fmt.Fprintf(&b, "# jym shell integration for %s\n", shell)
	if all {
		b.WriteString("# WARNING: --all wraps every external command currently found on PATH.\n")
		b.WriteString("# Use `command <name> ...` to bypass a wrapper.\n")
	}
	for _, name := range commands {
		if !shellCommandName.MatchString(name) {
			if all {
				continue
			}
			return "", fmt.Errorf("unsafe command name %q", name)
		}
		if name == "jym" || seen[name] {
			continue
		}
		seen[name] = true

		if all {
			switch shell {
			case "bash":
				fmt.Fprintf(&b, "if [[ $(type -t -- %s) == file ]]; then\n", name)
			case "zsh":
				fmt.Fprintf(&b, "if [[ $(whence -w -- %s) == '%s: command' ]]; then\n", name, name)
			}
			fmt.Fprintf(&b, "  function %s { command jym -- %s \"$@\"; }\n", name, name)
			b.WriteString("fi\n")
			continue
		}

		fmt.Fprintf(&b, "unalias %s 2>/dev/null || :\n", name)
		fmt.Fprintf(&b, "function %s { command jym -- %s \"$@\"; }\n", name, name)
	}
	return b.String(), nil
}

// externalCommands returns safe executable basenames from PATH. Results are
// sorted for stable generated output. Symlinks to the running jym executable
// are omitted along with the literal name "jym".
func externalCommands(pathEnv string) []string {
	seen := map[string]bool{}
	var commands []string
	self, _ := os.Executable()
	selfInfo, _ := os.Stat(self)

	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if seen[name] || name == "jym" || !shellCommandName.MatchString(name) {
				continue
			}
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
				continue
			}
			if selfInfo != nil && os.SameFile(info, selfInfo) {
				continue
			}
			seen[name] = true
			commands = append(commands, name)
		}
	}
	sort.Strings(commands)
	return commands
}
