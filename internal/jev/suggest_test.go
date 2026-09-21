package jev

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/syumai/jevyoumean/internal/help"
)

var candidates = []help.Command{
	{Name: "list", Description: "List pull requests"},
	{Name: "status", Description: "Show status of relevant pull requests"},
	{Name: "view", Description: "View a pull request"},
}

func serve(t *testing.T, answers map[string]any, check func(*testing.T, request)) *Client {
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
			"answers": answers,
		})
	}))
	t.Cleanup(srv.Close)
	return &Client{
		APIKey:     "test-key",
		Endpoint:   srv.URL,
		HTTPClient: &http.Client{Timeout: time.Second},
	}
}

func choiceAnswer(choice string, confidence float64, probs map[string]float64) map[string]any {
	return map[string]any{
		"subcommand": map[string]any{
			"type":          "choice",
			"choice":        choice,
			"confidence":    confidence,
			"probabilities": probs,
		},
	}
}

func state() State {
	return State{Command: "gh pr", InputSubcommand: "show", Arguments: []string{"123"}}
}

func TestSuggestMatch(t *testing.T) {
	client := serve(t, choiceAnswer("view", 0.86, map[string]float64{
		"view": 0.91, "status": 0.06, "list": 0.02, NoneOption: 0.01,
	}), func(t *testing.T, req request) {
		if req.Model != DefaultModel {
			t.Errorf("unexpected model: %q", req.Model)
		}
		q := req.Questions["subcommand"]
		if q.Type != "choice" {
			t.Errorf("unexpected question type: %q", q.Type)
		}
		if _, ok := q.Criteria[NoneOption]; !ok {
			t.Errorf("criteria missing %q", NoneOption)
		}
		if q.Criteria["view"] != "View a pull request" {
			t.Errorf("unexpected criteria for view: %v", q.Criteria["view"])
		}
	})
	sug, _, err := client.Suggest(context.Background(), state(), candidates, 0.60, 0.50)
	if err != nil {
		t.Fatal(err)
	}
	if sug == nil || sug.Command != "view" {
		t.Fatalf("expected suggestion view, got %+v", sug)
	}
}

func TestSuggestNoneWins(t *testing.T) {
	client := serve(t, choiceAnswer(NoneOption, 0.9, map[string]float64{
		"view": 0.05, NoneOption: 0.9,
	}), nil)
	sug, _, err := client.Suggest(context.Background(), state(), candidates, 0.60, 0.50)
	if err != nil {
		t.Fatal(err)
	}
	if sug != nil {
		t.Fatalf("expected no suggestion, got %+v", sug)
	}
}

func TestSuggestBelowThreshold(t *testing.T) {
	client := serve(t, choiceAnswer("view", 0.40, map[string]float64{
		"view": 0.44, "list": 0.37, "status": 0.15, NoneOption: 0.04,
	}), nil)
	sug, _, err := client.Suggest(context.Background(), state(), candidates, 0.60, 0.50)
	if err != nil {
		t.Fatal(err)
	}
	if sug != nil {
		t.Fatalf("expected no suggestion, got %+v", sug)
	}
}

func TestSuggestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	client := &Client{APIKey: "k", Endpoint: srv.URL, HTTPClient: &http.Client{Timeout: time.Second}}
	_, _, err := client.Suggest(context.Background(), state(), candidates, 0.60, 0.50)
	if err == nil {
		t.Fatal("expected error")
	}
}
