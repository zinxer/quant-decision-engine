package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sample() []Row {
	return []Row{
		{Headline: "SEC approves spot Ethereum ETF", Expected: "bullish", Regex: "bullish", Clef: "bullish", PClef: 0.97, Confirmed: 0.92, Latency: 40 * time.Millisecond, Order: "dry-run BUY"},
		{Headline: "Binance denies rumors of listing $XYZ token", Expected: "none", Regex: "bullish", Clef: "none", PClef: 0.95, Confirmed: 0.29, Latency: 60 * time.Millisecond, Order: "no directional signal"},
	}
}

func TestTableScoreboardAndNoColor(t *testing.T) {
	var b bytes.Buffer
	tb := NewTable(&b, false)
	tb.Header()
	for _, r := range sample() {
		tb.Add(r)
	}
	tb.Summary()
	out := b.String()
	if strings.Contains(out, "\033[") {
		t.Error("ANSI codes emitted with color=false")
	}
	for _, want := range []string{"regex 1/2", "clef-flash 2/2", "✅", "❌", "60ms"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTableAlignmentIgnoresColor(t *testing.T) {
	var plain, color bytes.Buffer
	NewTable(&plain, false).Add(sample()[1])
	NewTable(&color, true).Add(sample()[1])
	strip := strings.NewReplacer("\033[0m", "", "\033[2m", "", "\033[31m", "", "\033[32m", "")
	if strip.Replace(color.String()) != plain.String() {
		t.Errorf("colored row differs from plain once codes are stripped:\n%q\n%q", color.String(), plain.String())
	}
}

func TestJSONL(t *testing.T) {
	var b bytes.Buffer
	j := NewJSONL(&b)
	for _, r := range sample() {
		j.Add(r)
	}
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &m); err != nil {
		t.Fatal(err)
	}
	if m["clef"] != "none" || m["latency_ms"] != 60.0 || m["p_confirmed"] != 0.29 {
		t.Errorf("row = %v", m)
	}
}
