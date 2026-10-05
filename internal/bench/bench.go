// Package bench measures classifier latency so hosted and self-hosted Clef can be compared.
package bench

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/zinxer/quant-decision-engine/internal/decision"
)

// Result summarises measured (non-warmup) calls.
type Result struct {
	N, Errors               int
	Min, P50, P95, P99, Max time.Duration
	Mean                    time.Duration
}

// Summarize computes nearest-rank percentiles. It does not modify lat.
func Summarize(lat []time.Duration, errs int) Result {
	r := Result{N: len(lat), Errors: errs}
	if len(lat) == 0 {
		return r
	}
	s := append([]time.Duration(nil), lat...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	var sum time.Duration
	for _, d := range s {
		sum += d
	}
	rank := func(p float64) time.Duration {
		i := int(math.Ceil(p*float64(len(s)))) - 1
		return s[max(0, min(i, len(s)-1))]
	}
	r.Min, r.Max, r.Mean = s[0], s[len(s)-1], sum/time.Duration(len(s))
	r.P50, r.P95, r.P99 = rank(0.50), rank(0.95), rank(0.99)
	return r
}

// Run makes warmup+n sequential calls, cycling through events. Warmup calls
// (connection setup, caches) are discarded. Latency is the classifier's own
// measurement, so loop overhead is excluded. Failed calls are counted, not fatal,
// unless every measured call fails.
func Run(ctx context.Context, c decision.Classifier, events []decision.Event, n, warmup int) (Result, error) {
	if len(events) == 0 {
		return Result{}, errors.New("bench: no events")
	}
	if n <= 0 {
		return Result{}, errors.New("bench: n must be > 0")
	}
	lat := make([]time.Duration, 0, n)
	errs := 0
	var lastErr error
	for i := 0; i < warmup+n; i++ {
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		v, err := c.Classify(cctx, events[i%len(events)])
		cancel()
		if i < warmup {
			continue
		}
		if err != nil {
			errs++
			lastErr = err
			continue
		}
		lat = append(lat, v.Latency)
	}
	if len(lat) == 0 {
		return Summarize(nil, errs), fmt.Errorf("bench: all %d calls failed, last error: %w", n, lastErr)
	}
	return Summarize(lat, errs), nil
}

func fmtMS(d time.Duration) string { return fmt.Sprintf("%.1f", float64(d.Microseconds())/1000) }

// Report renders a human summary plus a markdown row to paste into a PR.
func (r Result) Report(label string, warmup int) string {
	return fmt.Sprintf(`%s: %d measured calls (%d warmup discarded), %d errors
  min %s ms   p50 %s ms   p95 %s ms   p99 %s ms   max %s ms   mean %s ms

| setup | n | p50 | p95 | p99 | max |
|---|---|---|---|---|---|
| %s | %d | %s ms | %s ms | %s ms | %s ms |
`, label, r.N, warmup, r.Errors, fmtMS(r.Min), fmtMS(r.P50), fmtMS(r.P95), fmtMS(r.P99), fmtMS(r.Max), fmtMS(r.Mean),
		label, r.N, fmtMS(r.P50), fmtMS(r.P95), fmtMS(r.P99), fmtMS(r.Max))
}
