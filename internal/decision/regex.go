package decision

import (
	"context"
	"regexp"
	"time"
)

var (
	bullRe = regexp.MustCompile(`(?i)\b(list(s|ing|ed)?|approv(e|es|ed|al)|launch(es|ed)?|partnership|adopt(s|ed|ion)?|surge[sd]?|rall(y|ies))\b`)
	bearRe = regexp.MustCompile(`(?i)\b(hack(ed|s)?|exploit(ed)?|delist(s|ing|ed)?|ban(s|ned)?|lawsuit|sues?|crash(es|ed)?|insolven(t|cy))\b`)
)

// Regex is the naive keyword baseline the demo compares against.
type Regex struct{}

// Name implements Classifier.
func (Regex) Name() string { return "regex" }

// Classify returns the first keyword family that matches; bearish words win ties.
func (Regex) Classify(_ context.Context, ev Event) (Verdict, error) {
	start := time.Now()
	d := None
	switch {
	case bearRe.MatchString(ev.Headline):
		d = Bearish
	case bullRe.MatchString(ev.Headline):
		d = Bullish
	}
	return Verdict{Direction: d, PDirection: 1, Confirmed: 1, Latency: time.Since(start)}, nil
}
