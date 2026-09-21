package jev

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/syumai/jevyoumean/internal/helptext"
)

var candidates = []helptext.Command{
	{Name: "list", Description: "List pull requests"},
	{Name: "status", Description: "Show status of relevant pull requests"},
	{Name: "view", Description: "View a pull request"},
}

// serve stubs the TypeSafe endpoint. answerFn receives the request
// questions so tests can answer per question id (used by sharding).
func serve(t *testing.T, answerFn func(map[string]Question) map[string]answer, check func(*testing.T, request)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer token")
		}
		body, _ := io.ReadAll(r.Body)
		var req request
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("malformed request: %v", err)
		}
		if check != nil {
			check(t, req)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"model":   "jev-1.13.0",
			"answers": answerFn(req.Questions),
		})
	}))
	t.Cleanup(srv.Close)
	return &Client{
		APIKey:     "test-key",
		Endpoint:   srv.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func choiceAns(choice string, confidence float64, probs map[string]float64) answer {
	return answer{Type: "choice", Choice: choice, Confidence: confidence, Probabilities: probs}
}

func state() State {
	return State{Command: "gh pr", SubcommandPath: []string{"pr"}, Typed: "show"}
}

func TestSuggestRequestShape(t *testing.T) {
	client := serve(t, func(qs map[string]Question) map[string]answer {
		return map[string]answer{"intended": choiceAns("view", 0.86, map[string]float64{
			"view": 0.91, "status": 0.06, "list": 0.02, NoneOption: 0.01,
		})}
	}, func(t *testing.T, req request) {
		if req.Model != DefaultModel {
			t.Errorf("unexpected model: %q", req.Model)
		}
		q, ok := req.Questions["intended"]
		if !ok || q.Type != "choice" {
			t.Fatalf("missing choice question: %+v", req.Questions)
		}
		if _, ok := q.Criteria[NoneOption]; !ok {
			t.Errorf("criteria missing %q", NoneOption)
		}
		if q.Criteria["view"] != "View a pull request" {
			t.Errorf("unexpected criteria for view: %v", q.Criteria["view"])
		}
	})
	ans, err := client.Suggest(context.Background(), state(), candidates)
	if err != nil {
		t.Fatal(err)
	}
	if ans == nil || ans.Choice != "view" || ans.Probabilities["view"] != 0.91 {
		t.Fatalf("unexpected answer: %+v", ans)
	}
}

func TestSuggestSharding(t *testing.T) {
	// 300 commands exceed the 254-option limit and force two phases.
	var many []helptext.Command
	for i := 0; i < 300; i++ {
		many = append(many, helptext.Command{Name: fmt.Sprintf("cmd%03d", i)})
	}
	many = append(many, helptext.Command{Name: "view", Description: "View a thing"})

	var phase int
	client := serve(t, func(qs map[string]Question) map[string]answer {
		phase++
		out := map[string]answer{}
		for id, q := range qs {
			var pick string
			for name := range q.Criteria {
				if name == NoneOption {
					continue
				}
				if pick == "" || name == "view" {
					pick = name
				}
			}
			probs := map[string]float64{pick: 0.9, NoneOption: 0.1}
			if id == "intended" {
				// Phase 2: final choice over shard winners.
				probs[pick] = 0.9
				probs[NoneOption] = 0.02
				for name := range q.Criteria {
					if name != pick && name != NoneOption {
						probs[name] = 0.08
						break
					}
				}
			}
			out[id] = choiceAns(pick, 0.9, probs)
		}
		return out
	}, nil)
	ans, err := client.Suggest(context.Background(), state(), many)
	if err != nil {
		t.Fatal(err)
	}
	if ans == nil || ans.Choice != "view" {
		t.Fatalf("unexpected answer: %+v", ans)
	}
	if phase != 2 {
		t.Fatalf("expected two request phases, got %d", phase)
	}
}

func TestSuggestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	client := &Client{APIKey: "k", Endpoint: srv.URL, HTTPClient: &http.Client{Timeout: time.Second}}
	if _, err := client.Suggest(context.Background(), state(), candidates); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateKeyUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	client := &Client{APIKey: "bad", Endpoint: srv.URL, HTTPClient: &http.Client{Timeout: time.Second}}
	if err := client.ValidateKey(context.Background()); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRetryOn429(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"model": "jev-1.13.0",
			"answers": map[string]any{
				"intended": map[string]any{"type": "choice", "choice": "view",
					"confidence": 0.9, "probabilities": map[string]float64{"view": 0.9}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	client := &Client{APIKey: "k", Endpoint: srv.URL, HTTPClient: &http.Client{Timeout: 5 * time.Second}}
	ans, err := client.Suggest(context.Background(), state(), candidates)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if ans.Choice != "view" {
		t.Fatalf("unexpected choice: %v", ans.Choice)
	}
}

func TestFilterArgs(t *testing.T) {
	args := []string{"123", "--json", "title,body", "--repo=owner/repo"}
	if got := FilterArgs(args, "none"); got != nil {
		t.Fatalf("none: %v", got)
	}
	if got := FilterArgs(args, "flags"); len(got) != 2 || got[0] != "--json" || got[1] != "--repo" {
		t.Fatalf("flags: %v", got)
	}
	if got := FilterArgs(args, "all"); len(got) != 4 {
		t.Fatalf("all: %v", got)
	}
}

// TestFilterArgsPrivacy pins the "flag names only, never values"
// promise: attached values must never reach the API, whether written
// --flag=value, -pvalue or -p value.
func TestFilterArgsPrivacy(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"--token=secret"}, []string{"--token"}},
		{[]string{"--token", "secret"}, []string{"--token"}}, // value is a positional arg, dropped
		{[]string{"-psecret"}, nil},                          // ambiguous attached value, dropped
		{[]string{"-p", "secret"}, []string{"-p"}},
		{[]string{"-v"}, []string{"-v"}},
		{[]string{"-abc"}, nil}, // combined shorts are ambiguous too
		{[]string{"pos", "--json", "-n", "5"}, []string{"--json", "-n"}},
	}
	for _, c := range cases {
		got := FilterArgs(c.in, "flags")
		if len(got) != len(c.want) {
			t.Fatalf("flags %v: got %v, want %v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("flags %v: got %v, want %v", c.in, got, c.want)
			}
		}
	}
}
