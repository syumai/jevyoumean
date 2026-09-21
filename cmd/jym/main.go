// Command jym wraps arbitrary CLI commands. When the user enters a
// subcommand that the target CLI does not document, jym asks Jev for a
// semantic "Did you mean?" suggestion using the names and descriptions
// found in the target's --help output.
//
//	jym [--debug] -- <command> [args...]
//	jym auth <status|set|remove>
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/syumai/jevyoumean/internal/cache"
	"github.com/syumai/jevyoumean/internal/config"
	"github.com/syumai/jevyoumean/internal/help"
	"github.com/syumai/jevyoumean/internal/jev"
	"github.com/syumai/jevyoumean/internal/runner"
)

// version is set via -ldflags "-X main.version=..." at release time.
var version = "dev"

// maxDepth bounds nested subcommand detection (gh -> pr -> view).
const maxDepth = 3

func main() {
	os.Exit(run(os.Args[1:]))
}

type invocation struct {
	debug   bool
	sub     string   // jym's own subcommand: "auth", "help", "version"
	subArgs []string // arguments to sub
	wrapped []string // target command argv (nil when sub is set)
}

// parseArgs splits jym's own arguments from the wrapped command line.
// Everything after "--" is the target command; without "--", the first
// non-flag argument either names a jym subcommand (auth, help, version)
// or begins the target command line.
func parseArgs(args []string) (invocation, error) {
	var inv invocation
	for i, a := range args {
		if a == "--" {
			inv.wrapped = args[i+1:]
			return inv, nil
		}
		if !strings.HasPrefix(a, "-") {
			switch a {
			case "auth", "help", "version":
				inv.sub = a
				inv.subArgs = args[i+1:]
			default:
				inv.wrapped = args[i:]
			}
			return inv, nil
		}
		switch a {
		case "--debug":
			inv.debug = true
		case "-h", "--help":
			inv.sub = "help"
			return inv, nil
		case "--version":
			inv.sub = "version"
			return inv, nil
		default:
			return inv, fmt.Errorf("unknown option: %s", a)
		}
	}
	return inv, nil
}

func run(args []string) int {
	inv, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jym: %v\n\n", err)
		usage()
		return 2
	}
	switch inv.sub {
	case "help":
		usage()
		return 0
	case "version":
		fmt.Println("jym " + version)
		return 0
	case "auth":
		return runAuth(inv.subArgs)
	}
	if inv.sub != "" || len(inv.wrapped) == 0 {
		usage()
		return 2
	}
	return wrap(inv)
}

func usage() {
	fmt.Fprint(os.Stderr, `jym - semantic "Did you mean?" for any CLI, powered by Jev

Usage:
  jym [--debug] -- <command> [args...]
  jym [--debug] <command> [args...]
  jym auth <status|set|remove>
  jym help
  jym version

Options:
  --debug   Print debug output to stderr (also JYM_DEBUG=1)

Environment:
  TYPESAFE_API_KEY  TypeSafe API key (overrides the config file)
  JYM_DEBUG         Enable debug output
  JYM_API_ENDPOINT  Override the TypeSafe API endpoint (testing)
  XDG_CONFIG_HOME   Config directory override
  XDG_CACHE_HOME    Cache directory override
`)
}

func debugEnabled(flag bool, cfg *config.Config) bool {
	if flag {
		return true
	}
	if v := os.Getenv("JYM_DEBUG"); v != "" && v != "0" {
		return true
	}
	return cfg != nil && cfg.Debug
}

// wrap resolves the target command, checks its argument vector against
// the documented subcommand tree, optionally asks Jev for a suggestion,
// and finally executes the original command unchanged.
func wrap(inv invocation) int {
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		cfg = config.Default()
	}
	debug := debugEnabled(inv.debug, cfg)
	logf := func(format string, a ...any) {
		if debug {
			fmt.Fprintf(os.Stderr, "[jym] "+format+"\n", a...)
		}
	}
	if cfgErr != nil {
		logf("config error: %v", cfgErr)
	}

	apiKey := os.Getenv("TYPESAFE_API_KEY")
	if apiKey == "" {
		apiKey = cfg.APIKey
	}
	if apiKey == "" && term.IsTerminal(int(os.Stdin.Fd())) {
		apiKey = firstRunSetup()
	}
	if apiKey == "" {
		logf("no API key configured; suggestions disabled")
	}

	exe, err := runner.LookPath(inv.wrapped[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "jym: executable not found: %s\n", inv.wrapped[0])
		return 127
	}
	logf("executable: %s", exe)

	store := cache.Open()
	defer store.Flush()
	disc := &help.Discoverer{Store: store}

	ctx := context.Background()
	unknown, path, candidates, trailing := detect(ctx, disc, exe, inv.wrapped[1:], logf)
	if unknown != "" {
		logf("unknown subcommand: %s", unknown)
		fmt.Fprintf(os.Stderr, "\nUnknown subcommand %q.\n", unknown)
		if apiKey != "" {
			suggest(ctx, cfg, apiKey, inv.wrapped[0], unknown, path, candidates, trailing, logf)
		}
	}
	return runner.Run(exe, inv.wrapped[1:])
}

// detect walks argv alongside the documented command tree and reports
// the first non-option argument that is not a documented subcommand.
// Every ambiguity resolves toward "no opinion": jym prefers missing a
// suggestion over misjudging a valid command.
func detect(ctx context.Context, d *help.Discoverer, exe string, args []string, logf func(string, ...any)) (unknown string, path []string, candidates []help.Command, trailing []string) {
	remaining := args
	for depth := 0; depth < maxDepth; depth++ {
		idx, arg, ambiguous := firstPositional(remaining)
		if idx < 0 {
			return "", nil, nil, nil
		}
		if arg == "help" {
			// `cmd help` is a meta-command nearly everywhere.
			return "", nil, nil, nil
		}
		cmds, err := d.Commands(ctx, exe, path)
		if err != nil {
			logf("help discovery failed for %q: %v", exe+" "+strings.Join(path, " "), err)
			return "", nil, nil, nil
		}
		if len(cmds) == 0 {
			// No documented subcommands at this level; remaining
			// arguments are positional data, not subcommands.
			return "", nil, nil, nil
		}
		found := false
		for _, c := range cmds {
			if c.Name == arg {
				found = true
				break
			}
		}
		if !found {
			if ambiguous {
				// The argument follows a bare flag, so it may be a
				// flag value rather than a subcommand.
				logf("argument %q may be a flag value; skipping detection", arg)
				return "", nil, nil, nil
			}
			return arg, path, cmds, remaining[idx+1:]
		}
		logf("command path: %s -> %s", exe, strings.Join(append(append([]string{}, path...), arg), " -> "))
		path = append(path, arg)
		remaining = remaining[idx+1:]
	}
	return "", nil, nil, nil
}

// firstPositional returns the index and value of the first argument that
// may be a subcommand. ambiguous reports whether the argument directly
// follows a bare flag, in which case it may be the flag's value.
func firstPositional(args []string) (idx int, arg string, ambiguous bool) {
	for i, a := range args {
		if a == "--" {
			// Everything after -- is positional data.
			return -1, "", false
		}
		if strings.HasPrefix(a, "-") {
			if strings.Contains(a, "=") {
				continue // --flag=value is self-contained
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				return i + 1, args[i+1], true
			}
			continue
		}
		return i, a, false
	}
	return -1, "", false
}

// suggest asks Jev for a recommendation and prints it. All failures are
// reported on stderr only in debug mode; a failed suggestion must never
// change the wrapped command's behavior.
func suggest(ctx context.Context, cfg *config.Config, apiKey, cmdName, unknown string, path []string, candidates []help.Command, trailing []string, logf func(string, ...any)) {
	logf("candidates:")
	for _, c := range candidates {
		logf("  %-12s %s", c.Name, c.Description)
	}

	client := jev.NewClient(apiKey, cfg.Timeout.Duration)
	client.Endpoint = os.Getenv("JYM_API_ENDPOINT") // empty means default
	state := jev.State{
		Command:         strings.Join(append([]string{cmdName}, path...), " "),
		InputSubcommand: unknown,
		Arguments:       trailing,
	}

	start := time.Now()
	sug, ans, err := client.Suggest(ctx, state, candidates, cfg.MinProbability, cfg.MinConfidence)
	latency := time.Since(start)
	if err != nil {
		logf("jev request failed: %v", err)
		return
	}
	if ans != nil {
		logf("jev:")
		keys := make([]string, 0, len(ans.Probabilities))
		for k := range ans.Probabilities {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			logf("  %-12s %.2f", k, ans.Probabilities[k])
		}
		logf("confidence: %.2f", ans.Confidence)
		logf("latency: %dms", latency.Milliseconds())
	}
	if sug != nil {
		fmt.Fprintf(os.Stderr, "\nDid you mean %q?\n", sug.Command)
	} else {
		logf("no suggestion met the thresholds (min_probability=%.2f, min_confidence=%.2f)", cfg.MinProbability, cfg.MinConfidence)
	}
}

// firstRunSetup interactively asks for a TypeSafe API key once and stores
// it in the local configuration. It returns the key for immediate use.
func firstRunSetup() string {
	fmt.Fprint(os.Stderr, "\njym needs a TypeSafe API key to use Jev.\n\nTypeSafe API key: ")
	keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jym: setup skipped: %v\n", err)
		return ""
	}
	key := strings.TrimSpace(string(keyBytes))
	if key == "" {
		fmt.Fprintln(os.Stderr, "jym: no API key provided; continuing without suggestions.")
		return ""
	}
	if err := config.SetAPIKey(key); err != nil {
		fmt.Fprintf(os.Stderr, "jym: could not save API key: %v\n", err)
		return key // still usable for this run
	}
	fmt.Fprintln(os.Stderr, "\n✓ API key saved.")
	return key
}

// runAuth implements `jym auth status|set|remove`.
func runAuth(args []string) int {
	if len(args) == 0 {
		authUsage()
		return 2
	}
	switch args[0] {
	case "status":
		configured := os.Getenv("TYPESAFE_API_KEY") != ""
		if !configured {
			if cfg, err := config.Load(); err == nil {
				configured = cfg.APIKey != ""
			}
		}
		if configured {
			fmt.Println("API key: configured")
		} else {
			fmt.Println("API key: not configured")
		}
		return 0
	case "set":
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprintln(os.Stderr, "jym: auth set requires a terminal")
			return 1
		}
		fmt.Fprint(os.Stderr, "TypeSafe API key: ")
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "jym: %v\n", err)
			return 1
		}
		key := strings.TrimSpace(string(keyBytes))
		if key == "" {
			fmt.Fprintln(os.Stderr, "jym: empty API key")
			return 1
		}
		if err := config.SetAPIKey(key); err != nil {
			fmt.Fprintf(os.Stderr, "jym: %v\n", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "API key saved.")
		return 0
	case "remove":
		if err := config.RemoveAPIKey(); err != nil {
			fmt.Fprintf(os.Stderr, "jym: %v\n", err)
			return 1
		}
		fmt.Println("API key removed")
		return 0
	default:
		authUsage()
		return 2
	}
}

func authUsage() {
	fmt.Fprint(os.Stderr, `Usage:
  jym auth status   Show whether an API key is configured
  jym auth set      Prompt for and store a TypeSafe API key
  jym auth remove   Remove the stored API key
`)
}
