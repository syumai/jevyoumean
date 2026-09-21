package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/syumai/jevyoumean/internal/config"
	"github.com/syumai/jevyoumean/internal/decide"
	"github.com/syumai/jevyoumean/internal/fallback"
	"github.com/syumai/jevyoumean/internal/jev"
)

// runExplain implements --explain: it runs the full detection pipeline,
// shows the Jev request and response, and reports the decision without
// executing anything.
func runExplain(cfg *config.Config, apiKey, cmdName string, det detection) int {
	printExplain(det)

	if apiKey == "" {
		fmt.Println("api key: not configured — offline fallback would be used")
		for _, m := range fallback.Suggest(det.unknown, det.candidates, decide.MaxCandidates) {
			fmt.Printf("  fallback: %s (distance %d)\n", m.Name, m.Distance)
		}
		return 0
	}

	state := jev.State{
		Command:        cmdName,
		SubcommandPath: det.path,
		Typed:          det.unknown,
		Arguments:      jev.FilterArgs(det.trailing, cfg.ContextArgs),
	}
	questions := jev.BuildQuestions(det.candidates)
	if reqBody, err := jev.MarshalRequest(cfg.Model, state, questions); err == nil {
		fmt.Printf("\nrequest:\n%s\n", reqBody)
	}

	start := time.Now()
	ans, err := newClient(cfg, apiKey).Suggest(context.Background(), state, det.candidates)
	fmt.Printf("\nlatency: %dms\n", time.Since(start).Milliseconds())
	if err != nil {
		fmt.Printf("response error: %v\n", err)
		return 1
	}
	if ans == nil {
		fmt.Println("response: <no answer>")
		return 0
	}
	fmt.Println("response:")
	keys := make([]string, 0, len(ans.Probabilities))
	for k := range ans.Probabilities {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-16s %.3f\n", k, ans.Probabilities[k])
	}
	fmt.Printf("  confidence       %.3f\n", ans.Confidence)

	action, cands := decide.Decide(decide.Config{
		Mode:             cfg.Mode,
		SuggestThreshold: cfg.SuggestThreshold,
		AutoRunThreshold: cfg.AutoRunThreshold,
		MinConfidence:    cfg.MinConfidence,
		Interactive:      true,
		Denylist:         denylist(cfg),
	}, ans)
	fmt.Printf("\ndecision: %s\n", action)
	for _, c := range cands {
		fmt.Printf("  %-16s %.2f\n", c.Name, c.P)
	}
	return 0
}

// printExplain dumps the per-level detection detail.
func printExplain(det detection) {
	for _, lvl := range det.levels {
		status := "leaf/unparseable"
		if lvl.count > 0 {
			mark := "not found"
			if lvl.found {
				mark = "ok"
			}
			status = fmt.Sprintf("%d subcommands, %q %s", lvl.count, lvl.checked, mark)
		}
		fmt.Printf("%s: %s (%s)\n", lvl.command, status, lvl.source)
	}
	if det.unknown != "" {
		fmt.Printf("unknown subcommand: %q under %q\n", det.unknown, strings.Join(det.path, " "))
	} else {
		fmt.Println("no unknown subcommand detected; the command would run unchanged")
	}
}
