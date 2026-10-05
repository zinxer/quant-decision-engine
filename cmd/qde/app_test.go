package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClef answers like the Decisions API. Synthetic: it tests plumbing, not model quality.
func fakeClef(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			State map[string]any `json:"state"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		dir, conf := "bullish", 0.95
		if strings.Contains(req.State["headline"].(string), "denies") {
			dir, conf = "none", 0.1
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"answers": map[string]any{
			"direction": map[string]any{"type": "choice", "choice": dir, "probabilities": map[string]float64{dir: 0.9}},
			"confirmed": map[string]any{"type": "noul", "noul": conf},
		}})
	}))
}

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func baseCfg(t *testing.T, e map[string]string) config {
	return config{Mode: "auto", Headlines: "testdata/headlines.json", ReplayFile: filepath.Join(t.TempDir(), "replay.json"),
		Format: "table", Color: "never", MaxOrders: 3, Env: env(e)}
}

func TestRecordThenReplayOffline(t *testing.T) {
	srv := fakeClef(t)
	defer srv.Close()

	cfg := baseCfg(t, map[string]string{"CLEF_URL": srv.URL})
	cfg.Record = true
	var live bytes.Buffer
	if err := run(context.Background(), cfg, &live); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"regex 1/2", "clef-flash 2/2", "dry-run BUY ETHUSDT", "recorded 2 responses"} {
		if !strings.Contains(live.String(), want) {
			t.Errorf("live output missing %q:\n%s", want, live.String())
		}
	}

	srv.Close() // replay must not touch the network
	off := baseCfg(t, nil)
	off.ReplayFile = cfg.ReplayFile
	var replay bytes.Buffer
	if err := run(context.Background(), off, &replay); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(replay.String(), "clef-flash 2/2") {
		t.Errorf("replay output:\n%s", replay.String())
	}
}

func TestJSONLFormat(t *testing.T) {
	srv := fakeClef(t)
	defer srv.Close()
	cfg := baseCfg(t, map[string]string{"CLEF_URL": srv.URL})
	cfg.Format = "jsonl"
	var out bytes.Buffer
	if err := run(context.Background(), cfg, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 JSONL lines, got %d:\n%s", len(lines), out.String())
	}
	for _, l := range lines {
		if !json.Valid([]byte(l)) {
			t.Errorf("invalid JSON line: %s", l)
		}
	}
}

func TestKillSwitchCapsOrders(t *testing.T) {
	srv := fakeClef(t)
	defer srv.Close()
	cfg := baseCfg(t, map[string]string{"CLEF_URL": srv.URL})
	cfg.MaxOrders = 0
	var out bytes.Buffer
	if err := run(context.Background(), cfg, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "KILL SWITCH") || strings.Contains(out.String(), "dry-run BUY") {
		t.Errorf("breaker did not stop the order:\n%s", out.String())
	}
}

func TestCancelledContextStops(t *testing.T) {
	srv := fakeClef(t)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, baseCfg(t, map[string]string{"CLEF_URL": srv.URL}), &bytes.Buffer{}); err == nil {
		t.Fatal("expected context error")
	}
}

func TestConfigErrors(t *testing.T) {
	cases := map[string]func(*config){
		"live without creds":   func(c *config) { c.Mode = "live" },
		"replay file missing":  func(c *config) { c.Mode = "replay" },
		"unknown mode":         func(c *config) { c.Mode = "yolo"; c.Env = env(map[string]string{"CLEF_URL": "http://x"}) },
		"unknown format":       func(c *config) { c.Format = "xml"; c.Env = env(map[string]string{"CLEF_URL": "http://x"}) },
		"execute without keys": func(c *config) { c.Execute = true; c.Env = env(map[string]string{"CLEF_URL": "http://x"}) },
		"bad headlines file":   func(c *config) { c.Headlines = "testdata/missing.json" },
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := baseCfg(t, nil)
			mut(&cfg)
			if err := run(context.Background(), cfg, &bytes.Buffer{}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBenchAgainstFakeServer(t *testing.T) {
	srv := fakeClef(t)
	defer srv.Close()
	cfg := baseCfg(t, map[string]string{"CLEF_URL": srv.URL})
	cfg.BenchN, cfg.BenchWarmup, cfg.Label = 10, 2, "fake"
	var out bytes.Buffer
	if err := runBench(context.Background(), cfg, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "| fake | 10 |") {
		t.Errorf("bench output:\n%s", out.String())
	}
}
