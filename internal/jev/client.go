// Package jev is a minimal client for the TypeSafe API's Jev model.
// It speaks net/http directly rather than depending on an SDK.
// The request/response shape is confined to this package so API changes
// only touch one place while Jev is in early access.
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultEndpoint is the TypeSafe System One endpoint.
	DefaultEndpoint = "https://api.typesafe.ai/v1/systemone"
	// DefaultModel selects the current stable Jev model.
	DefaultModel = "jev-latest"
)

// ErrUnauthorized marks a rejected API key (HTTP 401).
var ErrUnauthorized = errors.New("TypeSafe API: unauthorized (check the API key)")

// Client calls the TypeSafe API.
type Client struct {
	APIKey     string
	Endpoint   string // DefaultEndpoint when empty
	Model      string // DefaultModel when empty
	HTTPClient *http.Client
}

// NewClient returns a Client with the given timeout.
func NewClient(apiKey string, timeout time.Duration) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Question is one typed question for Jev.
type Question struct {
	Type         string         `json:"type"`
	Instructions any            `json:"instructions"`
	Criteria     map[string]any `json:"criteria"`
}

type request struct {
	Model     string              `json:"model"`
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// ChoiceAnswer is the typed answer Jev returns for a choice question.
type ChoiceAnswer struct {
	Choice        string
	Confidence    float64
	Probabilities map[string]float64
}

type answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type response struct {
	Model   string            `json:"model"`
	Answers map[string]answer `json:"answers"`
}

// MarshalRequest renders the request body (used by --explain).
func MarshalRequest(model string, state any, questions map[string]Question) ([]byte, error) {
	return json.MarshalIndent(request{Model: model, State: state, Questions: questions}, "", "  ")
}

// Ask sends state plus a set of questions in a single round trip and
// returns the answers keyed by question id. HTTP 429 and 529 are
// retried once after a short backoff; every other failure is returned
// immediately — the user is waiting, so we do not press our luck.
func (c *Client) Ask(ctx context.Context, state any, questions map[string]Question) (map[string]ChoiceAnswer, error) {
	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	body, err := json.Marshal(request{Model: model, State: state, Questions: questions})
	if err != nil {
		return nil, err
	}

	var respBody []byte
	for attempt := 0; ; attempt++ {
		respBody, err = c.post(ctx, body)
		if err == nil || attempt > 0 {
			break
		}
		var httpErr *httpError
		if !errors.As(err, &httpErr) || (httpErr.status != http.StatusTooManyRequests && httpErr.status != 529) {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	if err != nil {
		return nil, err
	}

	var decoded response
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("TypeSafe API: malformed response: %w", err)
	}
	out := make(map[string]ChoiceAnswer, len(decoded.Answers))
	for id, a := range decoded.Answers {
		out[id] = ChoiceAnswer{
			Choice:        a.Choice,
			Confidence:    a.Confidence,
			Probabilities: a.Probabilities,
		}
	}
	return out, nil
}

type httpError struct {
	status int
	body   string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("TypeSafe API: HTTP %d: %s", e.status, e.body)
}

func (c *Client) post(ctx context.Context, body []byte) ([]byte, error) {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &httpError{status: resp.StatusCode, body: truncate(string(respBody), 200)}
	}
	return respBody, nil
}

// ValidateKey sends a minimal Noul question to check that the API key
// works. It returns ErrUnauthorized on HTTP 401.
func (c *Client) ValidateKey(ctx context.Context) error {
	_, err := c.Ask(ctx, "connectivity check", map[string]Question{
		"ok": {Type: "noul", Instructions: "Is this a connectivity check?"},
	})
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
