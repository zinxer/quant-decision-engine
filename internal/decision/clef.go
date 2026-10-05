package decision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultModel is the OpenRouter slug of the latency-optimised Clef model.
	DefaultModel = "cloudflare/clef-flash"
	// OpenRouterURL is OpenRouter's (alpha) Decisions API.
	OpenRouterURL = "https://openrouter.ai/api/alpha/decisions"
)

// ClefConfig points the client at OpenRouter, a self-hosted Clef, or Jev.
// Any server speaking the Decisions/Jev-style API works: only URL, model and token change.
type ClefConfig struct {
	// URL is the full POST endpoint, e.g. OpenRouterURL.
	URL string
	// Model is sent in the body's "model" field when non-empty.
	Model string
	Token string
	// Timeout bounds one call. Defaults to 2s.
	Timeout time.Duration
}

// Clef is a Classifier backed by a Clef/Jev-compatible HTTP API.
type Clef struct {
	cfg  ClefConfig
	http *http.Client
}

// NewClef returns a client with a keep-alive transport and a per-call timeout (default 2s).
func NewClef(cfg ClefConfig) *Clef {
	if cfg.Timeout == 0 {
		cfg.Timeout = 2 * time.Second
	}
	return &Clef{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
			// Keep-alive connection reuse avoids a TLS handshake on every headline.
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        16,
				MaxIdleConnsPerHost: 16,
				IdleConnTimeout:     90 * time.Second,
				ForceAttemptHTTP2:   true,
			},
		},
	}
}

// Name implements Classifier.
func (c *Clef) Name() string { return "clef" }

type question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

type request struct {
	Model     string              `json:"model,omitempty"`
	State     map[string]any      `json:"state"`
	Questions map[string]question `json:"questions"`
}

type answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
	Noul          float64            `json:"noul"`
}

type answers struct {
	Answers map[string]answer `json:"answers"`
}

// response is the Decisions API body ({"answers":...}) or its error shape
// ({"error":{"code":...,"message":...}}). A {"result":{"answers":...}} envelope
// (Cloudflare Workers AI direct) is also accepted.
type response struct {
	answers
	Result *answers `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Success *bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// BuildRequest is exported so the replay recorder and tests share the exact wire shape.
func BuildRequest(model string, ev Event) any {
	return request{
		Model: model,
		State: map[string]any{
			"headline":          ev.Headline,
			"source":            ev.Source,
			"candidate_symbols": ev.CandidateSymbols,
		},
		Questions: map[string]question{
			"direction": {
				Type:         "choice",
				Instructions: "Given the `headline`, what is the likely short-term price direction for the candidate symbols? Denials, rumors, questions, speculation, and old news are 'none'.",
				Criteria: map[string]string{
					"bullish": "Confirmed positive news that would likely push the price up",
					"bearish": "Confirmed negative news that would likely push the price down",
					"none":    "Denial, rumor, question, speculation, stale news, or irrelevant",
				},
			},
			"confirmed": {
				Type:         "noul",
				Instructions: "Does the `headline` report an official, confirmed event?",
				Criteria: map[string]string{
					"true":  "An official announcement or verified fact that has actually happened",
					"false": "A rumor, denial, question, speculation, or report that is disputed or stale",
				},
			},
		},
	}
}

// Classify sends one request with both questions and returns the parsed verdict.
// Latency covers the HTTP round trip and body read.
func (c *Clef) Classify(ctx context.Context, ev Event) (Verdict, error) {
	body, err := json.Marshal(BuildRequest(c.cfg.Model, ev))
	if err != nil {
		return Verdict{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return Verdict{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return Verdict{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	latency := time.Since(start)
	if err != nil {
		return Verdict{}, err
	}
	v, err := parseVerdict(resp.StatusCode, raw)
	v.Latency = latency
	return v, err
}

func parseVerdict(status int, raw []byte) (Verdict, error) {
	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return Verdict{}, fmt.Errorf("clef: status %d, bad JSON: %w", status, err)
	}
	if status/100 != 2 || r.Error != nil || (r.Success != nil && !*r.Success) {
		msgs := make([]string, 0, len(r.Errors)+1)
		if r.Error != nil {
			msgs = append(msgs, r.Error.Message)
		}
		for _, e := range r.Errors {
			msgs = append(msgs, e.Message)
		}
		return Verdict{}, fmt.Errorf("clef: status %d: %s", status, strings.Join(msgs, "; "))
	}
	a := r.answers
	if r.Result != nil {
		a = *r.Result
	}
	dir, ok := a.Answers["direction"]
	if !ok {
		return Verdict{}, errors.New("clef: response missing answer \"direction\"")
	}
	conf, ok := a.Answers["confirmed"]
	if !ok {
		return Verdict{}, errors.New("clef: response missing answer \"confirmed\"")
	}
	d := Direction(dir.Choice)
	switch d {
	case Bullish, Bearish, None:
	default:
		return Verdict{}, fmt.Errorf("clef: unexpected direction %q", dir.Choice)
	}
	return Verdict{Direction: d, PDirection: dir.Probabilities[dir.Choice], Confirmed: conf.Noul}, nil
}
