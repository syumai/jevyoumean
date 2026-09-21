package decide

import (
	"testing"

	"github.com/syumai/jevyoumean/internal/jev"
)

func cfg(mode string) Config {
	return Config{
		Mode:             mode,
		SuggestThreshold: 0.30,
		AutoRunThreshold: 0.95,
		MinConfidence:    0.50,
		Interactive:      true,
		Denylist:         DefaultDenylist,
	}
}

func ans(choice string, confidence float64, probs map[string]float64) *jev.ChoiceAnswer {
	return &jev.ChoiceAnswer{Choice: choice, Confidence: confidence, Probabilities: probs}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		ans        *jev.ChoiceAnswer
		wantAction Action
		wantTop    string
		wantN      int
	}{
		{
			name: "clear winner prompts",
			cfg:  cfg("prompt"),
			ans: ans("view", 0.86, map[string]float64{
				"view": 0.91, "status": 0.06, "list": 0.02, jev.NoneOption: 0.01,
			}),
			wantAction: Prompt, wantTop: "view", wantN: 1,
		},
		{
			name: "none wins",
			cfg:  cfg("prompt"),
			ans: ans(jev.NoneOption, 0.9, map[string]float64{
				"view": 0.05, jev.NoneOption: 0.9,
			}),
			wantAction: PassThrough,
		},
		{
			name: "below threshold",
			cfg:  cfg("prompt"),
			ans: ans("view", 0.8, map[string]float64{
				"view": 0.20, "list": 0.15, jev.NoneOption: 0.6,
			}),
			wantAction: PassThrough,
		},
		{
			name: "low confidence suppresses",
			cfg:  cfg("prompt"),
			ans: ans("view", 0.40, map[string]float64{
				"view": 0.44, "list": 0.37, "status": 0.15, jev.NoneOption: 0.04,
			}),
			wantAction: PassThrough,
		},
		{
			name: "ambiguous but confident lists top 3",
			cfg:  cfg("prompt"),
			ans: ans("view", 0.8, map[string]float64{
				"view": 0.44, "list": 0.37, "status": 0.32, "other": 0.1, jev.NoneOption: 0.01,
			}),
			wantAction: Prompt, wantTop: "view", wantN: 3,
		},
		{
			name: "hint mode never prompts",
			cfg:  cfg("hint"),
			ans: ans("view", 0.9, map[string]float64{
				"view": 0.98, jev.NoneOption: 0.02,
			}),
			wantAction: Hint, wantTop: "view", wantN: 1,
		},
		{
			name: "auto runs clear winner",
			cfg:  cfg("auto"),
			ans: ans("view", 0.9, map[string]float64{
				"view": 0.97, jev.NoneOption: 0.03,
			}),
			wantAction: AutoRun, wantTop: "view", wantN: 1,
		},
		{
			name: "auto downgraded below threshold",
			cfg:  cfg("auto"),
			ans: ans("view", 0.9, map[string]float64{
				"view": 0.80, jev.NoneOption: 0.20,
			}),
			wantAction: Prompt, wantTop: "view", wantN: 1,
		},
		{
			name: "auto downgraded for denylisted command",
			cfg:  cfg("auto"),
			ans: ans("delete", 0.99, map[string]float64{
				"delete": 0.99, jev.NoneOption: 0.01,
			}),
			wantAction: Prompt, wantTop: "delete", wantN: 1,
		},
		{
			name: "prompt degrades to hint without tty",
			cfg:  func() Config { c := cfg("prompt"); c.Interactive = false; return c }(),
			ans: ans("view", 0.9, map[string]float64{
				"view": 0.91, jev.NoneOption: 0.09,
			}),
			wantAction: Hint, wantTop: "view", wantN: 1,
		},
		{
			name: "auto degrades to hint without tty",
			cfg:  func() Config { c := cfg("auto"); c.Interactive = false; return c }(),
			ans: ans("view", 0.9, map[string]float64{
				"view": 0.99, jev.NoneOption: 0.01,
			}),
			wantAction: Hint, wantTop: "view", wantN: 1,
		},
		{
			name:       "nil answer passes through",
			cfg:        cfg("prompt"),
			ans:        nil,
			wantAction: PassThrough,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, cands := Decide(tt.cfg, tt.ans)
			if action != tt.wantAction {
				t.Fatalf("action = %v, want %v", action, tt.wantAction)
			}
			if len(cands) != tt.wantN {
				t.Fatalf("candidates = %v, want %d", cands, tt.wantN)
			}
			if tt.wantTop != "" && cands[0].Name != tt.wantTop {
				t.Fatalf("top = %q, want %q", cands[0].Name, tt.wantTop)
			}
		})
	}
}
