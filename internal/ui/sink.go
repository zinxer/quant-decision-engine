package ui

import (
	"encoding/json"
	"io"
	"time"
)

// Sink receives decision rows. Table renders for humans; JSONL is an audit log for machines.
type Sink interface {
	Header()
	Add(Row)
	Summary()
}

// JSONL writes one JSON object per decision.
type JSONL struct{ enc *json.Encoder }

// NewJSONL writes decision records to w.
func NewJSONL(w io.Writer) *JSONL { return &JSONL{enc: json.NewEncoder(w)} }

type jsonRow struct {
	Time      time.Time `json:"ts"`
	Headline  string    `json:"headline"`
	Expected  string    `json:"expected,omitempty"`
	Regex     string    `json:"regex"`
	Clef      string    `json:"clef"`
	PClef     float64   `json:"p_clef"`
	Confirmed float64   `json:"p_confirmed"`
	LatencyMS float64   `json:"latency_ms"`
	Decision  string    `json:"decision"`
}

// Header is a no-op: JSONL has no header line.
func (j *JSONL) Header() {}

// Summary is a no-op: consumers aggregate the records themselves.
func (j *JSONL) Summary() {}

// Add writes one record.
func (j *JSONL) Add(r Row) {
	_ = j.enc.Encode(jsonRow{
		Time: time.Now().UTC(), Headline: r.Headline, Expected: r.Expected, Regex: r.Regex, Clef: r.Clef,
		PClef: r.PClef, Confirmed: r.Confirmed, LatencyMS: float64(r.Latency.Microseconds()) / 1000, Decision: r.Order,
	})
}
