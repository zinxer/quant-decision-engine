package decision

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplayRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.json")
	h := "SEC approves spot Ethereum ETF"
	in := RecordingFile{Model: DefaultModel, Entries: map[string]Recording{
		Key(h): {Headline: h, Direction: Bullish, PDir: 0.97, Confirmed: 0.92, LatencyNS: 42 * time.Millisecond},
	}}
	if err := SaveRecording(path, in); err != nil {
		t.Fatal(err)
	}
	r, err := LoadReplay(path)
	if err != nil {
		t.Fatal(err)
	}
	v, err := r.Classify(context.Background(), Event{Headline: h})
	if err != nil {
		t.Fatal(err)
	}
	if v.Direction != Bullish || v.PDirection != 0.97 || v.Latency != 42*time.Millisecond {
		t.Errorf("verdict = %+v", v)
	}
	if _, err := r.Classify(context.Background(), Event{Headline: "unknown"}); err == nil {
		t.Error("expected error for unrecorded headline")
	}
}

func TestLoadReplayErrors(t *testing.T) {
	if _, err := LoadReplay(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("expected error for missing file")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(bad, []byte("{"), 0o644)
	if _, err := LoadReplay(bad); err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestKeyIsStable(t *testing.T) {
	a1, a2 := Key("a"), Key("a")
	if a1 != a2 || a1 == Key("b") || len(a1) != 16 {
		t.Fatal("key not stable or wrong length")
	}
}
