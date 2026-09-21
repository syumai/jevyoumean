// Package jev is a minimal client for the TypeSafe API's Jev model.
// It speaks net/http directly rather than depending on an SDK.
package jev

import (
	"bytes"
	"context"
	"encoding/json"
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

type question struct {
	Type         string         `json:"type"`
	Instructions string         `json:"instructions"`
	Criteria     map[string]any `json:"criteria"`
}

type request struct {
	Model     string              `json:"model"`
	State     any                 `json:"state"`
	Questions map[string]question `json:"questions"`
}

// ChoiceAnswer is the typed answer Jev returns for a choice question.
type ChoiceAnswer struct {
	Choice        string
	Confidence    float64
	Probabilities map[string]float64
}

type response struct {
	Model   string `json:"model"`
	Answers map[string]struct {
		Type          string             `json:"type"`
		Choice        string             `json:"choice"`
		Confidence    float64            `json:"confidence"`
		Probabilities map[string]float64 `json:"probabilities"`
	} `json:"answers"`
}

// Choice asks Jev to pick one of the criteria options against state and
// returns the typed answer under the fixed question id "subcommand".
func (c *Client) Choice(ctx context.Context, state any, instructions string, criteria map[string]any) (*ChoiceAnswer, error) {
	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	body, err := json.Marshal(request{
		Model: model,
		State: state,
		Questions: map[string]question{
			"subcommand": {
				Type:         "choice",
				Instructions: instructions,
				Criteria:     criteria,
			},
		},
	})
	if err != nil {
		return nil, err
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TypeSafe API: %s: %s", resp.Status, truncate(string(respBody), 200))
	}

	var decoded response
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("TypeSafe API: malformed response: %w", err)
	}
	ans, ok := decoded.Answers["subcommand"]
	if !ok || ans.Type != "choice" {
		return nil, fmt.Errorf("TypeSafe API: missing choice answer")
	}
	return &ChoiceAnswer{
		Choice:        ans.Choice,
		Confidence:    ans.Confidence,
		Probabilities: ans.Probabilities,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
