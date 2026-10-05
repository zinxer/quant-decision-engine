package bench

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zinxer/quant-decision-engine/internal/decision"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestSummarizeNearestRank(t *testing.T) {
	lat := make([]time.Duration, 0, 100)
	for i := 100; i >= 1; i-- { // unsorted input
		lat = append(lat, ms(i))
	}
	r := Summarize(lat, 0)
	if r.N != 100 || r.Min != ms(1) || r.Max != ms(100) {
		t.Fatalf("bad bounds: %+v", r)
	}
	if r.P50 != ms(50) || r.P95 != ms(95) || r.P99 != ms(99) {
		t.Fatalf("percentiles: p50=%v p95=%v p99=%v", r.P50, r.P95, r.P99)
	}
	if r.Mean != 50500*time.Microsecond {
		t.Fatalf("mean = %v", r.Mean)
	}
}

func TestSummarizeDoesNotMutateInput(t *testing.T) {
	in := []time.Duration{ms(3), ms(1), ms(2)}
	Summarize(in, 0)
	if in[0] != ms(3) {
		t.Fatal("input was sorted in place")
	}
}

func TestSummarizeEmpty(t *testing.T) {
	if r := Summarize(nil, 2); r.N != 0 || r.Errors != 2 || r.P50 != 0 {
		t.Fatalf("%+v", r)
	}
}

type stub struct {
	calls int
	fail  map[int]bool
}

func (s *stub) Name() string { return "stub" }
func (s *stub) Classify(_ context.Context, _ decision.Event) (decision.Verdict, error) {
	s.calls++
	if s.fail[s.calls] {
		return decision.Verdict{}, errors.New("boom")
	}
	return decision.Verdict{Latency: ms(s.calls)}, nil
}

func TestRunExcludesWarmupAndCountsErrors(t *testing.T) {
	s := &stub{fail: map[int]bool{5: true}}
	evs := []decision.Event{{Headline: "a"}, {Headline: "b"}}
	r, err := Run(context.Background(), s, evs, 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if s.calls != 8 { // 2 warmup + 6 measured
		t.Fatalf("calls = %d", s.calls)
	}
	// warmup calls 1-2 excluded; measured calls 3..8, call 5 failed
	if r.N != 5 || r.Errors != 1 || r.Min != ms(3) || r.Max != ms(8) {
		t.Fatalf("%+v", r)
	}
}

func TestRunAllFailuresIsAnError(t *testing.T) {
	s := &stub{fail: map[int]bool{1: true, 2: true, 3: true}}
	if _, err := Run(context.Background(), s, []decision.Event{{Headline: "a"}}, 3, 0); err == nil {
		t.Fatal("expected error when every call fails")
	}
}

func TestRunValidatesInput(t *testing.T) {
	if _, err := Run(context.Background(), &stub{}, nil, 5, 0); err == nil {
		t.Fatal("expected error for no events")
	}
	if _, err := Run(context.Background(), &stub{}, []decision.Event{{}}, 0, 0); err == nil {
		t.Fatal("expected error for n=0")
	}
}
