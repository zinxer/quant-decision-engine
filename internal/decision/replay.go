package decision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Key is the stable lookup key for a headline in a replay file.
func Key(headline string) string {
	h := sha256.Sum256([]byte(headline))
	return hex.EncodeToString(h[:8])
}

// Recording is one captured real response.
type Recording struct {
	Headline  string        `json:"headline"`
	Direction Direction     `json:"direction"`
	PDir      float64       `json:"p_direction"`
	Confirmed float64       `json:"confirmed"`
	LatencyNS time.Duration `json:"latency_ns"`
}

// RecordingFile is the on-disk replay format written by `qde --record`.
type RecordingFile struct {
	Model      string               `json:"model"`
	RecordedAt time.Time            `json:"recorded_at"`
	Entries    map[string]Recording `json:"entries"`
}

// Replay serves previously recorded verdicts, including their original latency.
type Replay struct{ f RecordingFile }

// LoadReplay reads a recording written by SaveRecording.
func LoadReplay(path string) (*Replay, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f RecordingFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("replay %s: %w", path, err)
	}
	return &Replay{f: f}, nil
}

// Name implements Classifier. It reports "clef" because it replays Clef answers.
func (r *Replay) Name() string { return "clef" }

// Classify returns the recorded verdict for the headline, or an error if none was recorded.
func (r *Replay) Classify(_ context.Context, ev Event) (Verdict, error) {
	rec, ok := r.f.Entries[Key(ev.Headline)]
	if !ok {
		return Verdict{}, fmt.Errorf("replay: no recording for %q (run `make record`)", ev.Headline)
	}
	return Verdict{Direction: rec.Direction, PDirection: rec.PDir, Confirmed: rec.Confirmed, Latency: rec.LatencyNS}, nil
}

// SaveRecording writes a replay file.
func SaveRecording(path string, f RecordingFile) error {
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	// The recording is committed to the repo and holds no secrets, so world-readable is intended.
	return os.WriteFile(path, append(raw, '\n'), 0o644) //nolint:gosec // G306: see above
}
