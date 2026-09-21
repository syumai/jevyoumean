package jev

import (
	"context"

	"github.com/syumai/jevyoumean/internal/help"
)

// NoneOption is the escape hatch added to every Choice so Jev is never
// forced to pick a candidate that does not match the user's intent.
const NoneOption = "__none__"

const instructions = "Which documented subcommand best matches what the user appears to intend?"

// State is the context handed to Jev for a subcommand suggestion.
type State struct {
	Command         string   `json:"command"`
	InputSubcommand string   `json:"input_subcommand"`
	Arguments       []string `json:"arguments"`
}

// Suggestion is a recommendation that cleared the configured policy.
type Suggestion struct {
	Command     string
	Probability float64
	Confidence  float64
}

// Criteria builds the Choice criteria: each candidate maps to its help
// description (or nil), plus the mandatory __none__ option.
func Criteria(commands []help.Command) map[string]any {
	criteria := make(map[string]any, len(commands)+1)
	for _, c := range commands {
		if c.Description == "" {
			criteria[c.Name] = nil
		} else {
			criteria[c.Name] = c.Description
		}
	}
	criteria[NoneOption] = "None of these commands match the user's intent"
	return criteria
}

// Suggest asks Jev for the best matching subcommand and applies the
// recommendation policy. The raw answer is also returned for debugging.
// A nil Suggestion with a nil error means "no recommendation".
func (c *Client) Suggest(ctx context.Context, state State, commands []help.Command, minProb, minConf float64) (*Suggestion, *ChoiceAnswer, error) {
	ans, err := c.Choice(ctx, state, instructions, Criteria(commands))
	if err != nil {
		return nil, nil, err
	}
	return applyPolicy(ans, minProb, minConf), ans, nil
}

// applyPolicy implements the recommendation rules:
//   - __none__ is never suggested;
//   - the choice's probability must be >= minProb;
//   - the answer confidence must be >= minConf;
//   - if __none__ outscores the choice, the result is inconsistent and
//     nothing is suggested.
func applyPolicy(ans *ChoiceAnswer, minProb, minConf float64) *Suggestion {
	if ans.Choice == "" || ans.Choice == NoneOption {
		return nil
	}
	p := ans.Probabilities[ans.Choice]
	if p < minProb || ans.Confidence < minConf {
		return nil
	}
	if ans.Probabilities[NoneOption] > p {
		return nil
	}
	return &Suggestion{
		Command:     ans.Choice,
		Probability: p,
		Confidence:  ans.Confidence,
	}
}
