// Package decision turns a headline into a typed Verdict.
package decision

import (
	"context"
	"time"
)

// Event is one incoming headline.
type Event struct {
	Headline         string   `json:"headline"`
	Source           string   `json:"source,omitempty"`
	CandidateSymbols []string `json:"candidate_symbols,omitempty"`
}

// Direction is the model's (or baseline's) read of the headline.
type Direction string

// Directions a classifier can return.
const (
	Bullish Direction = "bullish"
	Bearish Direction = "bearish"
	None    Direction = "none"
)

// Verdict is the classifier output the trading gate consumes.
type Verdict struct {
	Direction Direction
	// PDirection is the probability assigned to Direction (1.0 for the regex baseline).
	PDirection float64
	// Confirmed is P("this is an official, confirmed event"). 1.0 for the baseline.
	Confirmed float64
	Latency   time.Duration
}

// Classifier classifies a headline.
type Classifier interface {
	Name() string
	Classify(ctx context.Context, ev Event) (Verdict, error)
}
