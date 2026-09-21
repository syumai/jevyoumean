// Package decide turns a Jev answer plus the configured policy into an
// Action. It is a pure function so the threshold behavior can be pinned
// down by table tests.
package decide

import (
	"sort"

	"github.com/syumai/jevyoumean/internal/jev"
)

// Action is what jym does with a suggestion result.
type Action int

const (
	// PassThrough shows nothing and runs the command as typed.
	PassThrough Action = iota
	// Hint prints candidates to stderr, then runs the command as typed.
	Hint
	// Prompt prints candidates and waits for a one-key choice.
	Prompt
	// AutoRun prints a notice and runs the corrected command.
	AutoRun
)

func (a Action) String() string {
	switch a {
	case PassThrough:
		return "pass-through"
	case Hint:
		return "hint"
	case Prompt:
		return "prompt"
	case AutoRun:
		return "auto-run"
	}
	return "unknown"
}

// Candidate is a displayable suggestion.
type Candidate struct {
	Name string
	P    float64
}

// Config carries the decision inputs. Interactive reports whether stdin
// is a terminal; Prompt and AutoRun degrade to Hint without it.
type Config struct {
	Mode             string // "prompt" | "hint" | "auto"
	SuggestThreshold float64
	AutoRunThreshold float64
	MinConfidence    float64
	Interactive      bool
	Denylist         map[string]bool
}

// DefaultDenylist names subcommands that must never be auto-run; an
// auto decision for one of these is downgraded to a prompt.
var DefaultDenylist = map[string]bool{
	"rm": true, "remove": true, "delete": true, "destroy": true,
	"clean": true, "reset": true, "prune": true, "purge": true,
	"drop": true, "push": true, "publish": true, "apply": true,
	"deploy": true,
}

// MaxCandidates caps how many suggestions are shown.
const MaxCandidates = 3

// Decide applies the recommendation policy:
//   - __none__ or every candidate below SuggestThreshold → PassThrough;
//   - answer confidence below MinConfidence → PassThrough (ambiguous);
//   - mode "auto" upgrades to AutoRun only above AutoRunThreshold and
//     off the denylist, otherwise falls back to Prompt;
//   - without an interactive stdin, Prompt and AutoRun degrade to Hint.
func Decide(cfg Config, ans *jev.ChoiceAnswer) (Action, []Candidate) {
	if ans == nil || ans.Choice == "" || ans.Choice == jev.NoneOption {
		return PassThrough, nil
	}
	if ans.Confidence < cfg.MinConfidence {
		return PassThrough, nil
	}
	cands := candidates(ans.Probabilities, cfg.SuggestThreshold)
	if len(cands) == 0 {
		return PassThrough, nil
	}

	var action Action
	switch cfg.Mode {
	case "hint":
		action = Hint
	case "auto":
		if cands[0].P >= cfg.AutoRunThreshold && !cfg.Denylist[cands[0].Name] {
			action = AutoRun
		} else {
			action = Prompt
		}
	default: // "prompt"
		action = Prompt
	}
	if !cfg.Interactive && (action == Prompt || action == AutoRun) {
		action = Hint
	}
	return action, cands
}

func candidates(probs map[string]float64, minP float64) []Candidate {
	var out []Candidate
	for name, p := range probs {
		if name != jev.NoneOption && p >= minP {
			out = append(out, Candidate{Name: name, P: p})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].P != out[j].P {
			return out[i].P > out[j].P
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > MaxCandidates {
		out = out[:MaxCandidates]
	}
	return out
}
