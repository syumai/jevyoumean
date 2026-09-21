// Command jym wraps arbitrary CLI commands. When the user types a
// subcommand the target CLI does not document, jym asks Jev (TypeSafe's
// System One model) for a semantic "Did you mean?" suggestion using the
// names and descriptions found in the target's help output.
//
//	jym [jym-flags] <command> [args...]
//
// jym has no subcommands of its own; management operations are flags so
// wrapped commands can use any name.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/syumai/jevyoumean/internal/cache"
	"github.com/syumai/jevyoumean/internal/config"
	"github.com/syumai/jevyoumean/internal/creds"
	"github.com/syumai/jevyoumean/internal/decide"
	"github.com/syumai/jevyoumean/internal/fallback"
	"github.com/syumai/jevyoumean/internal/helptext"
	"github.com/syumai/jevyoumean/internal/jev"
	"github.com/syumai/jevyoumean/internal/resolve"
	"github.com/syumai/jevyoumean/internal/runner"
	"github.com/syumai/jevyoumean/internal/ui"
)

// version is set via -ldflags "-X main.version=..." at release time.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

type invocation struct {
	debug   bool
	explain bool
	refresh bool
	sub     string   // jym's own operation: "help", "version", "setup", "doctor", "cache-clear", "print-mise", "completion"
	subArgs []string // arguments to sub
	wrapped []string // target command argv
}

// parseArgs separates jym's flags from the wrapped command line.
// Everything after "--" is the target command; otherwise the first
// non-flag argument begins it.
func parseArgs(args []string) (invocation, error) {
	var inv invocation
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			inv.wrapped = args[i+1:]
			return inv, nil
		}
		if !strings.HasPrefix(a, "-") {
			inv.wrapped = args[i:]
			return inv, nil
		}
		switch a {
		case "--debug":
			inv.debug = true
		case "--explain":
			inv.explain = true
		case "--refresh":
			inv.refresh = true
		case "--setup":
			inv.sub = "setup"
		case "--doctor":
			inv.sub = "doctor"
		case "--cache-clear":
			inv.sub = "cache-clear"
		case "--print-mise":
			inv.sub, inv.subArgs = "print-mise", args[i+1:]
			return inv, nil
		case "--completion":
			if i+1 >= len(args) {
				return inv, errors.New("--completion requires a shell: bash, zsh or fish")
			}
			inv.sub, inv.subArgs = "completion", args[i+1:i+2]
			return inv, nil
		case "-h", "--help":
			inv.sub = "help"
			return inv, nil
		case "--version":
			inv.sub = "version"
			return inv, nil
		default:
			return inv, fmt.Errorf("unknown flag: %s", a)
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
	case "setup":
		return runSetup()
	case "doctor":
		return runDoctor()
	case "cache-clear":
		cache.Open().ClearAll()
		fmt.Println("cache cleared")
		return 0
	case "print-mise":
		return printMise(inv.subArgs)
	case "completion":
		return printCompletion(inv.subArgs[0])
	}
	if len(inv.wrapped) == 0 {
		usage()
		return 2
	}
	return wrap(inv)
}

func usage() {
	fmt.Fprint(os.Stderr, `jym - semantic "Did you mean?" for any CLI, powered by Jev

Usage:
  jym [flags] -- <command> [args...]
  jym [flags] <command> [args...]

Flags:
  --setup                  Configure the TypeSafe API key interactively
  --explain                Show extraction, request, response and decision without executing
  --refresh                Discard the target command's help cache and refetch
  --cache-clear            Remove the whole help cache
  --print-mise <cmd>...    Print a [shell_alias] snippet for mise.toml
  --completion <shell>     Print a completion script (bash, zsh, fish)
  --doctor                 Diagnose key, API reachability, cache and TTY
  --debug                  Print debug output to stderr (also JYM_DEBUG=1)
  --help                   Show this help
  --version                Show version

Environment:
  TYPESAFE_API_KEY  TypeSafe API key (overrides the credentials file)
  JYM_CONFIG        Alternate config file path
  JYM_MODE          Override mode (prompt, hint, auto)
  JYM_DEBUG         Enable debug output
  JYM_API_ENDPOINT  Override the TypeSafe API endpoint (testing)
`)
}

// wrap resolves the target command, checks its arguments against the
// documented subcommand tree, optionally asks Jev for a suggestion, and
// executes a command line — always a real command, corrected or not.
func wrap(inv invocation) int {
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		cfg = config.Default()
	}
	cfg.ApplyEnv()
	if inv.debug {
		cfg.Debug = true
	}
	logf := func(format string, a ...any) {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "[jym] "+format+"\n", a...)
		}
	}
	if cfgErr != nil {
		logf("config error: %v", cfgErr)
	}

	exe, err := resolve.LookPath(inv.wrapped[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "jym: %v\n", err)
		return 127
	}
	logf("executable: %s", exe)

	passthrough := func() int {
		return runner.Replace(inv.wrapped[0], exe, inv.wrapped[1:])
	}

	// Recursion guard: a jym-inside-jym invocation never intervenes.
	if resolve.Depth() >= resolve.MaxDepth {
		return passthrough()
	}

	// Never intervene in non-interactive contexts (scripts, CI, pipes):
	// if stderr is not a terminal there is no one to show a suggestion to.
	stderrTTY := term.IsTerminal(int(os.Stderr.Fd()))
	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	if !stderrTTY && !inv.explain {
		return passthrough()
	}
	logf("tty: stdin=%v stderr=%v", stdinTTY, stderrTTY)

	store := cache.Open()
	if inv.refresh {
		// Invalidate by the resolved executable's basename — never the
		// raw user-supplied name — so a path like ../x cannot reach
		// outside the cache directory.
		store.Invalidate(exe)
		logf("cache invalidated for %s", exe)
	}

	apiKey, keySource := creds.Resolve()
	if apiKey == "" && stdinTTY && !inv.explain {
		if st, err := creds.LoadState(); err == nil && !st.SetupDeclined {
			apiKey = firstRunSetup(cfg)
		}
	}
	if apiKey == "" {
		logf("no API key configured; offline matching only")
	} else {
		logf("API key source: %s", keySource)
	}

	det := detect(context.Background(), cfg, store, exe, inv.wrapped[0], inv.wrapped[1:], logf)
	if det.unknown == "" {
		if inv.explain {
			printExplain(det)
			return 0
		}
		return passthrough()
	}

	if inv.explain {
		return runExplain(cfg, apiKey, inv.wrapped[0], det)
	}

	// A suggestion path: Jev when a key exists, edit distance otherwise.
	var (
		action  decide.Action
		cands   []decide.Candidate
		offline bool
		jevOK   bool
	)
	if apiKey != "" {
		client := newClient(cfg, apiKey)
		state := jev.State{
			Command:        inv.wrapped[0],
			SubcommandPath: det.path,
			Typed:          det.unknown,
			Arguments:      jev.FilterArgs(det.trailing, cfg.ContextArgs),
		}
		start := time.Now()
		ans, err := client.Suggest(context.Background(), state, det.candidates)
		logf("jev latency: %dms", time.Since(start).Milliseconds())
		if err != nil {
			logf("jev request failed: %v", err)
		} else if ans != nil {
			jevOK = true
			logAnswer(ans, logf)
			action, cands = decide.Decide(decide.Config{
				Mode:             cfg.Mode,
				SuggestThreshold: cfg.SuggestThreshold,
				AutoRunThreshold: cfg.AutoRunThreshold,
				MinConfidence:    cfg.MinConfidence,
				Interactive:      stdinTTY,
				Denylist:         denylist(cfg),
			}, ans)
		}
	}
	if !jevOK {
		// No key, or Jev failed: try offline edit-distance matching.
		// When Jev answered (even __none__) its verdict stands. Offline
		// guesses are weaker than Jev's, so mode "auto" still prompts —
		// only a validated Jev answer may auto-run.
		if m := fallback.Suggest(det.unknown, det.candidates, decide.MaxCandidates); len(m) > 0 {
			offline = true
			for _, match := range m {
				cands = append(cands, decide.Candidate{Name: match.Name})
			}
			switch {
			case !stdinTTY:
				action = decide.Hint
			case cfg.Mode == "hint":
				action = decide.Hint
			default:
				action = decide.Prompt
			}
		}
	}

	switch action {
	case decide.PassThrough:
		return passthrough()
	case decide.AutoRun:
		corrected := correctedArgs(inv.wrapped[1:], det, cands[0].Name)
		ui.ShowAutoRun(os.Stderr, ui.JoinCmd(append([]string{inv.wrapped[0]}, corrected...)...), cands[0].P)
		return runner.Replace(inv.wrapped[0], exe, corrected)
	case decide.Hint:
		showSuggestion(det, inv.wrapped, cands, offline, false)
		if offline {
			ui.ShowOfflineNote(os.Stderr)
		}
		return passthrough()
	default: // decide.Prompt
		showSuggestion(det, inv.wrapped, cands, offline, true)
		choice, err := ui.Prompt(os.Stdin, os.Stderr, len(cands))
		if err != nil {
			logf("prompt error: %v", err)
			return passthrough()
		}
		switch {
		case choice == int(ui.Cancel):
			return 127
		case choice == int(ui.RunAsTyped):
			// Child execution so the exit code can be observed for learning.
			code := runner.Run(inv.wrapped[0], exe, inv.wrapped[1:])
			if code == 0 && det.entry != nil {
				store.AddLearned(det.entry, det.unknown, helpArgsOf(cfg, inv.wrapped[0]))
			}
			return code
		default:
			corrected := correctedArgs(inv.wrapped[1:], det, cands[choice-1].Name)
			return runner.Replace(inv.wrapped[0], exe, corrected)
		}
	}
}

// detection is the outcome of walking argv against the command tree.
type detection struct {
	unknown    string             // first undocumented token ("" = all good)
	path       []string           // matched subcommand path above it
	trailing   []string           // args following the unknown token
	candidates []helptext.Command // documented subcommands at that level
	entry      *cache.Entry       // cache entry at the unknown level
	levels     []levelInfo        // per-level detail for --explain
}

type levelInfo struct {
	command string // "gh pr"
	source  string // "cache" or "fetched"
	checked string // token inspected at this level
	found   bool
	count   int
}

func helpArgsOf(cfg *config.Config, name string) []string {
	return cfg.Command(name).HelpArgs
}

// detect walks the argument vector level by level. Per the design, any
// flag before the first positional token ends detection (the token may
// be a flag value), and so do "help", a leaf level, a learned token, an
// extra_subcommands entry and a plugin executable named <cmd>-<token>.
// When in doubt jym keeps quiet: false negatives beat false positives.
func detect(ctx context.Context, cfg *config.Config, store *cache.Cache, exe, cmdName string, args []string, logf func(string, ...any)) detection {
	var det detection
	cmdCfg := cfg.Command(cmdName)
	maxDepth := cfg.MaxDepth
	if cmdCfg.MaxDepth > 0 {
		maxDepth = cmdCfg.MaxDepth
	}
	remaining := args
	for depth := 0; depth < maxDepth; depth++ {
		if len(remaining) == 0 || strings.HasPrefix(remaining[0], "-") || remaining[0] == "help" {
			return det
		}
		token := remaining[0]
		cmds, entry, source, err := commandsAt(ctx, store, exe, det.path, cmdCfg.HelpArgs)
		lvl := levelInfo{
			command: strings.Join(append([]string{cmdName}, det.path...), " "),
			source:  source,
			checked: token,
			count:   len(cmds),
		}
		if err != nil {
			logf("help discovery failed for %q: %v", lvl.command, err)
			det.levels = append(det.levels, lvl)
			return det
		}
		if len(cmds) == 0 {
			// Leaf or unparseable help: cannot judge this level.
			det.levels = append(det.levels, lvl)
			return det
		}
		lvl.found = matchCommand(cmds, token)
		det.levels = append(det.levels, lvl)
		if lvl.found || entry.HasLearned(token) || contains(cmdCfg.ExtraSubcommands, token) {
			det.path = append(det.path, token)
			remaining = remaining[1:]
			logf("command path: %s", strings.Join(append([]string{cmdName}, det.path...), " -> "))
			continue
		}
		if len(det.path) == 0 {
			// Plugin convention: an executable named <cmd>-<token>.
			if _, err := resolve.LookPath(cmdName + "-" + token); err == nil {
				logf("%q resolved via plugin executable %s-%s", token, cmdName, token)
				return det
			}
		}
		det.unknown, det.trailing, det.candidates, det.entry = token, remaining[1:], cmds, entry
		return det
	}
	return det
}

// commandsAt returns the documented subcommands at a command path, from
// cache or by fetching help. A freshly fetched level is cached including
// its is_leaf marker.
func commandsAt(ctx context.Context, store *cache.Cache, exe string, path, helpArgs []string) ([]helptext.Command, *cache.Entry, string, error) {
	if e, ok := store.Load(exe, path, helpArgs); ok {
		if e.IsLeaf {
			return nil, e, "cache", nil
		}
		return e.Subcommands, e, "cache", nil
	}
	cmds, err := helptext.Fetch(ctx, exe, path, helpArgs)
	if err != nil {
		return nil, nil, "", err
	}
	e := &cache.Entry{
		Executable:  exe,
		Path:        append([]string{}, path...),
		IsLeaf:      len(cmds) == 0,
		Subcommands: cmds,
	}
	store.Save(e, helpArgs)
	return cmds, e, "fetched", nil
}

func matchCommand(cmds []helptext.Command, token string) bool {
	for _, c := range cmds {
		if c.Matches(token) {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func denylist(cfg *config.Config) map[string]bool {
	set := make(map[string]bool, len(decide.DefaultDenylist)+len(cfg.Denylist))
	for k := range decide.DefaultDenylist {
		set[k] = true
	}
	for _, k := range cfg.Denylist {
		set[k] = true
	}
	return set
}

func newClient(cfg *config.Config, apiKey string) *jev.Client {
	c := jev.NewClient(apiKey, cfg.Timeout())
	c.Model = cfg.Model
	c.Endpoint = os.Getenv("JYM_API_ENDPOINT") // empty means default
	return c
}

// showSuggestion renders the candidate list; render maps a candidate
// name to the full corrected command line.
func showSuggestion(det detection, wrapped []string, cands []decide.Candidate, offline, prompt bool) {
	names := make([]string, len(cands))
	probs := make([]float64, len(cands))
	for i, c := range cands {
		names[i], probs[i] = c.Name, c.P
	}
	context := strings.Join(append([]string{wrapped[0]}, det.path...), " ")
	render := func(name string) string {
		return ui.JoinCmd(append([]string{wrapped[0]}, correctedArgs(wrapped[1:], det, name)...)...)
	}
	if prompt {
		ui.ShowPrompt(os.Stderr, det.unknown, context, names, probs, render, offline)
	} else {
		ui.ShowHint(os.Stderr, det.unknown, context, names, probs, render, offline)
	}
}

// correctedArgs replaces the unknown token with the chosen subcommand.
// The unknown token sits exactly at position len(det.path): detection
// bails on any flag before a candidate, so the matched args are exactly
// the path.
func correctedArgs(args []string, det detection, choice string) []string {
	out := append([]string{}, args...)
	out[len(det.path)] = choice
	return out
}

func logAnswer(ans *jev.ChoiceAnswer, logf func(string, ...any)) {
	logf("jev:")
	keys := make([]string, 0, len(ans.Probabilities))
	for k := range ans.Probabilities {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		logf("  %-14s %.2f", k, ans.Probabilities[k])
	}
	logf("confidence: %.2f", ans.Confidence)
}
