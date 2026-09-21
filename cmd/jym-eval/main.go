// Command jym-eval measures how well semantic matching beats edit
// distance for CLI "Did you mean?" suggestions. It reads a TSV file of
// cases — "command <TAB> typed <TAB> expected" — and reports hit rate,
// false-suggestion rate and latency for Jev and for the offline
// edit-distance fallback.
//
//	jym-eval [-cache=false] evals.tsv
//
// An empty "expected" column means the input should NOT be suggested.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/syumai/jevyoumean/internal/cache"
	"github.com/syumai/jevyoumean/internal/config"
	"github.com/syumai/jevyoumean/internal/creds"
	"github.com/syumai/jevyoumean/internal/decide"
	"github.com/syumai/jevyoumean/internal/fallback"
	"github.com/syumai/jevyoumean/internal/helptext"
	"github.com/syumai/jevyoumean/internal/jev"
	"github.com/syumai/jevyoumean/internal/resolve"
)

type evalCase struct {
	command  string
	typed    string
	expected string
}

func main() {
	noCache := flag.Bool("no-cache", false, "ignore the help cache")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: jym-eval [-no-cache] <evals.tsv>")
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	os.Exit(run(flag.Arg(0), *noCache))
}

func run(path string, noCache bool) int {
	cases, err := readCases(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "jym-eval:", err)
		return 1
	}
	cfg, _ := config.Load()
	cfg.ApplyEnv()
	apiKey, _ := creds.Resolve()
	var client *jev.Client
	if apiKey != "" {
		client = jev.NewClient(apiKey, cfg.Timeout())
		client.Model = cfg.Model
		client.Endpoint = os.Getenv("JYM_API_ENDPOINT")
	} else {
		fmt.Fprintln(os.Stderr, "jym-eval: no API key; reporting edit-distance results only")
	}
	store := cache.Open()
	ctx := context.Background()

	var jevHit, jevFP, jevMiss, fbHit, fbFP, fbMiss int
	var latencies []time.Duration
	for _, c := range cases {
		exe, err := resolve.LookPath(c.command)
		if err != nil {
			fmt.Printf("%-10s %-12s SKIP: %v\n", c.command, c.typed, err)
			continue
		}
		cmds := commands(ctx, store, exe, cfg, c.command, noCache)

		jevPred := ""
		var lat time.Duration
		if client != nil && len(cmds) > 0 {
			start := time.Now()
			ans, err := client.Suggest(ctx, jev.State{
				Command: c.command,
				Typed:   c.typed,
			}, cmds)
			lat = time.Since(start)
			latencies = append(latencies, lat)
			if err != nil {
				fmt.Printf("%-10s %-12s JEV ERROR: %v\n", c.command, c.typed, err)
			} else if action, cands := decide.Decide(decide.Config{
				Mode:             "prompt",
				SuggestThreshold: cfg.SuggestThreshold,
				MinConfidence:    cfg.MinConfidence,
				Interactive:      true,
			}, ans); action != decide.PassThrough && len(cands) > 0 {
				jevPred = cands[0].Name
			}
		}
		fbPred := ""
		if m := fallback.Suggest(c.typed, cmds, 1); len(m) > 0 {
			fbPred = m[0].Name
		}

		jevHit, jevFP, jevMiss = tally(jevPred, c.expected, jevHit, jevFP, jevMiss)
		fbHit, fbFP, fbMiss = tally(fbPred, c.expected, fbHit, fbFP, fbMiss)
		fmt.Printf("%-10s typed=%-14s expected=%-12s jev=%-12s fallback=%-12s %s\n",
			c.command, c.typed, orDash(c.expected), orDash(jevPred), orDash(fbPred), lat.Round(time.Millisecond))
	}

	n := len(cases)
	fmt.Printf("\njev:      %d/%d correct, %d false suggestions, %d misses\n", jevHit, n, jevFP, jevMiss)
	fmt.Printf("fallback: %d/%d correct, %d false suggestions, %d misses\n", fbHit, n, fbFP, fbMiss)
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		fmt.Printf("jev latency: p50=%s p95=%s max=%s\n",
			latencies[len(latencies)/2].Round(time.Millisecond),
			latencies[len(latencies)*95/100].Round(time.Millisecond),
			latencies[len(latencies)-1].Round(time.Millisecond))
	}
	return 0
}

// tally counts a prediction: hit when it equals the expectation, false
// positive when it suggests where none was expected, miss otherwise.
func tally(pred, expected string, hit, fp, miss int) (int, int, int) {
	switch {
	case pred == expected:
		return hit + 1, fp, miss
	case expected == "":
		return hit, fp + 1, miss
	default:
		return hit, fp, miss + 1
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// commands resolves the top-level subcommand list for a command,
// honoring the per-command help_args override and the cache.
func commands(ctx context.Context, store *cache.Cache, exe string, cfg *config.Config, name string, noCache bool) []helptext.Command {
	helpArgs := cfg.Command(name).HelpArgs
	if !noCache {
		if e, ok := store.Load(exe, nil, helpArgs); ok {
			return e.Subcommands
		}
	}
	cmds, err := helptext.Fetch(ctx, exe, nil, helpArgs)
	if err != nil {
		return nil
	}
	if !noCache {
		store.Save(&cache.Entry{
			Executable:  exe,
			IsLeaf:      len(cmds) == 0,
			Subcommands: cmds,
		}, helpArgs)
	}
	return cmds
}

func readCases(path string) ([]evalCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []evalCase
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			return nil, fmt.Errorf("malformed line: %q", line)
		}
		c := evalCase{command: fields[0], typed: fields[1]}
		if len(fields) > 2 {
			c.expected = fields[2]
		}
		cases = append(cases, c)
	}
	return cases, sc.Err()
}
