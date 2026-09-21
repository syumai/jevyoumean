package jev

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/syumai/jevyoumean/internal/helptext"
)

// NoneOption is the escape hatch added to every Choice so Jev is never
// forced to pick a candidate that does not match the user's intent.
const NoneOption = "__none__"

// maxOptions is the Choice limit (255) minus the __none__ slot.
const maxOptions = 254

// State is the context handed to Jev for a subcommand suggestion.
// Arguments is populated according to the context_args setting and may
// be empty.
type State struct {
	Command        string   `json:"command"`
	SubcommandPath []string `json:"subcommand_path"`
	Typed          string   `json:"typed"`
	Arguments      []string `json:"arguments,omitempty"`
}

// FilterArgs applies the context_args privacy policy: "none" sends no
// arguments, "flags" sends flag names only (never values), "all" sends
// the argument vector verbatim.
func FilterArgs(args []string, mode string) []string {
	switch mode {
	case "all":
		return append([]string{}, args...)
	case "flags":
		var out []string
		for _, a := range args {
			if strings.HasPrefix(a, "-") {
				if i := strings.IndexByte(a, '='); i >= 0 {
					a = a[:i]
				}
				out = append(out, a)
			}
		}
		return out
	default: // "none"
		return nil
	}
}

var instructions = map[string]any{
	"question": "A user ran `command` with `typed` as a subcommand, but `typed` is not a valid subcommand. Which valid subcommand did the user most likely intend?",
	"note":     "Judge by meaning, not only by spelling. For example 'remove' means 'rm' and 'list' may mean 'ls' or 'ps'.",
}

// criteria builds the Choice criteria: each candidate maps to its help
// description (or nil), plus the mandatory __none__ option.
func criteria(commands []helptext.Command) map[string]any {
	c := make(map[string]any, len(commands)+1)
	for _, cmd := range commands {
		if cmd.Description == "" {
			c[cmd.Name] = nil
		} else {
			c[cmd.Name] = cmd.Description
		}
	}
	c[NoneOption] = "None of the subcommands match, or `typed` is probably not a mistyped subcommand at all (a file path, a plugin, a user-defined alias)."
	return c
}

// BuildQuestions constructs the question set for a candidate list,
// sharding when the list exceeds the Choice option limit. Exported for
// --explain and tests.
func BuildQuestions(commands []helptext.Command) map[string]Question {
	if len(commands) <= maxOptions {
		return map[string]Question{
			"intended": {Type: "choice", Instructions: instructions, Criteria: criteria(commands)},
		}
	}
	qs := map[string]Question{}
	for i, shard := range shard(commands, maxOptions) {
		qs[fmt.Sprintf("shard%d", i)] = Question{
			Type:         "choice",
			Instructions: instructions,
			Criteria:     criteria(shard),
		}
	}
	return qs
}

func shard(commands []helptext.Command, size int) [][]helptext.Command {
	var out [][]helptext.Command
	for i := 0; i < len(commands); i += size {
		out = append(out, commands[i:min(i+size, len(commands))])
	}
	return out
}

// Suggest asks Jev which documented subcommand the user intended and
// returns the winning answer. With more than maxOptions candidates it
// runs a two-phase choice: one request scoring every shard in parallel,
// then a final Choice over the shard winners. Probabilities are not
// comparable across shards, so phase-1 values only select the pool.
func (c *Client) Suggest(ctx context.Context, state State, commands []helptext.Command) (*ChoiceAnswer, error) {
	if len(commands) <= maxOptions {
		answers, err := c.Ask(ctx, state, BuildQuestions(commands))
		if err != nil {
			return nil, err
		}
		return answerOrNil(answers, "intended"), nil
	}

	// Phase 1: score each shard in a single request.
	answers, err := c.Ask(ctx, state, BuildQuestions(commands))
	if err != nil {
		return nil, err
	}
	byName := make(map[string]helptext.Command, len(commands))
	for _, cmd := range commands {
		byName[cmd.Name] = cmd
	}
	var pool []helptext.Command
	for id, ans := range answers {
		if !strings.HasPrefix(id, "shard") || ans.Probabilities == nil {
			continue
		}
		for _, name := range top(ans.Probabilities, 0.1, 3) {
			if cmd, ok := byName[name]; ok {
				pool = append(pool, cmd)
			}
		}
	}
	if len(pool) == 0 {
		return nil, nil
	}
	if len(pool) > maxOptions {
		pool = pool[:maxOptions]
	}

	// Phase 2: final choice over the pooled winners.
	final, err := c.Ask(ctx, state, map[string]Question{
		"intended": {Type: "choice", Instructions: instructions, Criteria: criteria(pool)},
	})
	if err != nil {
		return nil, err
	}
	return answerOrNil(final, "intended"), nil
}

func answerOrNil(answers map[string]ChoiceAnswer, id string) *ChoiceAnswer {
	if a, ok := answers[id]; ok {
		return &a
	}
	return nil
}

// top returns the names with probability >= minP, highest first,
// capped at n.
func top(probs map[string]float64, minP float64, n int) []string {
	type kv struct {
		k string
		v float64
	}
	var items []kv
	for k, v := range probs {
		if k != NoneOption && v >= minP {
			items = append(items, kv{k, v})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].v > items[j].v })
	out := make([]string, 0, min(n, len(items)))
	for _, it := range items {
		if len(out) == n {
			break
		}
		out = append(out, it.k)
	}
	return out
}
