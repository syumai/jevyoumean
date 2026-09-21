package jev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// serveRaw returns a Client whose transport is rerouted to a stub that
// writes body verbatim. Requests still target the real production URL.
func serveRaw(t *testing.T, body string) *Client {
	t.Helper()
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	return &Client{APIKey: "k", HTTPClient: srv.Client()}
}

// TestProductionEndpoint proves the client really sends requests to
// the TypeSafe URL — the in-memory transport reroutes the network, not
// the request URL, so host, method, path and TLS all stay testable.
func TestProductionEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST api.typesafe.ai/v1/systemone", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			t.Error("expected HTTPS request")
		}
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query: %q", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"model": "jev-1.13.0",
			"answers": map[string]any{
				"intended": map[string]any{"type": "choice", "choice": "view", "confidence": 0.9,
					"probabilities": map[string]float64{"view": 0.9, "__none__": 0.1}},
			},
		})
	})
	srv := httptest.NewTestServer(t, mux)
	client := &Client{APIKey: "k", HTTPClient: srv.Client()}
	ans, err := client.Suggest(context.Background(), state(), candidates)
	if err != nil {
		t.Fatal(err)
	}
	if ans == nil || ans.Choice != "view" {
		t.Fatalf("unexpected answer: %+v", ans)
	}
}

// validResponse answers "view" correctly for the candidates fixture.
const validResponse = `{
  "model": "jev-1.13.0",
  "answers": {"intended": {"type": "choice", "choice": "view", "confidence": 0.9,
    "probabilities": {"view": 0.9, "list": 0.05, "status": 0.03, "__none__": 0.02}}}
}`

func TestAskValidResponse(t *testing.T) {
	client := serveRaw(t, validResponse)
	ans, err := client.Suggest(context.Background(), state(), candidates)
	if err != nil {
		t.Fatal(err)
	}
	if ans == nil || ans.Choice != "view" {
		t.Fatalf("unexpected answer: %+v", ans)
	}
}

// TestMalformedResponses ensures anything the API could return that does
// not match the sent criteria is rejected at the boundary — before it
// can reach decision logic or auto execution.
func TestMalformedResponses(t *testing.T) {
	cases := map[string]string{
		"probability key outside criteria": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":0.9,
			"probabilities":{"view":0.9,"rm -rf /":0.1}}}}`,
		"choice outside criteria": `{"answers":{"intended":{"type":"choice","choice":"evilcmd","confidence":0.9,
			"probabilities":{"evilcmd":0.9,"view":0.1}}}}`,
		"choice not argmax": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":0.9,
			"probabilities":{"view":0.3,"list":0.7}}}}`,
		"choice missing from probabilities": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":0.9,
			"probabilities":{"list":0.7,"status":0.3}}}}`,
		"probability above 1": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":0.9,
			"probabilities":{"view":1.5,"list":0.1}}}}`,
		"probability negative": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":0.9,
			"probabilities":{"view":0.9,"list":-0.4}}}}`,
		"confidence above 1": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":1.4,
			"probabilities":{"view":0.9,"list":0.1}}}}`,
		"confidence negative": `{"answers":{"intended":{"type":"choice","choice":"view","confidence":-0.1,
			"probabilities":{"view":0.9,"list":0.1}}}}`,
		"empty choice": `{"answers":{"intended":{"type":"choice","choice":"","confidence":0.9,
			"probabilities":{"view":0.9,"list":0.1}}}}`,
	}
	for name, body := range cases {
		client := serveRaw(t, body)
		if _, err := client.Suggest(context.Background(), state(), candidates); err == nil {
			t.Errorf("%s: expected rejection, got success", name)
		}
	}
}

// TestNonFiniteResponse confirms a NaN/Inf serialized as a non-number
// token cannot slip through (encoding/json refuses them, which is also
// a rejection at the boundary).
func TestNonFiniteResponse(t *testing.T) {
	body := `{"answers":{"intended":{"type":"choice","choice":"view","confidence":"NaN",
		"probabilities":{"view":0.9}}}}`
	client := serveRaw(t, body)
	if _, err := client.Suggest(context.Background(), state(), candidates); err == nil {
		t.Fatal("expected rejection for non-numeric confidence")
	}
}

func TestMarshalRequestShape(t *testing.T) {
	b, err := MarshalRequest("jev-latest", "state", map[string]Question{
		"q": {Type: "choice", Criteria: map[string]any{"a": "desc", NoneOption: "none"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["model"] != "jev-latest" {
		t.Fatalf("unexpected model: %v", decoded["model"])
	}
}
