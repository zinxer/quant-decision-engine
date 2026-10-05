package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/zinxer/quant-decision-engine/internal/bench"
	"github.com/zinxer/quant-decision-engine/internal/decision"
	"github.com/zinxer/quant-decision-engine/internal/exec"
	"github.com/zinxer/quant-decision-engine/internal/risk"
	"github.com/zinxer/quant-decision-engine/internal/signal"
	"github.com/zinxer/quant-decision-engine/internal/ui"
)

type config struct {
	Mode, Headlines, ReplayFile, Format, Color, Label string
	Execute, Record                                   bool
	Delay                                             time.Duration
	MaxOrders, BenchN, BenchWarmup                    int
	// Env looks up configuration; os.Getenv in production, a map in tests.
	Env func(string) string
}

type headline struct {
	Headline string `json:"headline"`
	Source   string `json:"source"`
	Expected string `json:"expected"`
}

func loadHeadlines(path string) ([]headline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var hs []headline
	if err := json.Unmarshal(raw, &hs); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(hs) == 0 {
		return nil, fmt.Errorf("%s: no headlines", path)
	}
	return hs, nil
}

func toEvent(h headline) decision.Event {
	return decision.Event{Headline: h.Headline, Source: h.Source, CandidateSymbols: signal.Symbols(h.Headline)}
}

// clefFromEnv builds the live client: OpenRouter by default, or CLEF_URL for a self-hosted server.
func clefFromEnv(env func(string) string) (*decision.Clef, bool) {
	token := env("OPENROUTER_API_KEY")
	if v := env("CLEF_API_TOKEN"); v != "" {
		token = v
	}
	url, model := env("CLEF_URL"), env("CLEF_MODEL")
	if url == "" {
		if token == "" {
			return nil, false
		}
		url = decision.OpenRouterURL
		if model == "" {
			model = decision.DefaultModel
		}
	}
	return decision.NewClef(decision.ClefConfig{URL: url, Model: model, Token: token}), true
}

func pickClassifier(cfg config) (decision.Classifier, error) {
	live, haveLive := clefFromEnv(cfg.Env)
	switch {
	case cfg.Record || cfg.Mode == "live":
		if !haveLive {
			return nil, errors.New("live mode needs OPENROUTER_API_KEY (or CLEF_URL for a self-hosted server); see .env.example")
		}
		return live, nil
	case cfg.Mode == "replay" || (cfg.Mode == "auto" && !haveLive):
		r, err := decision.LoadReplay(cfg.ReplayFile)
		if err != nil {
			return nil, fmt.Errorf("no replay recording at %s (%w)\nset OPENROUTER_API_KEY in .env, then run `make record` once", cfg.ReplayFile, err)
		}
		return r, nil
	case cfg.Mode == "auto":
		return live, nil
	}
	return nil, fmt.Errorf("unknown mode %q (want auto | live | replay)", cfg.Mode)
}

func pickSink(cfg config, w io.Writer) (ui.Sink, error) {
	switch cfg.Format {
	case "jsonl":
		return ui.NewJSONL(w), nil
	case "table", "":
	default:
		return nil, fmt.Errorf("unknown format %q (want table | jsonl)", cfg.Format)
	}
	color := false
	switch cfg.Color {
	case "always":
		color = true
	case "never":
	case "auto", "":
		color = isTTY(w) && cfg.Env("NO_COLOR") == ""
	default:
		return nil, fmt.Errorf("unknown color %q (want auto | always | never)", cfg.Color)
	}
	return ui.NewTable(w, color), nil
}

func run(ctx context.Context, cfg config, stdout io.Writer) error {
	hs, err := loadHeadlines(cfg.Headlines)
	if err != nil {
		return err
	}
	clef, err := pickClassifier(cfg)
	if err != nil {
		return err
	}
	sink, err := pickSink(cfg, stdout)
	if err != nil {
		return err
	}
	var ex exec.Executor = exec.DryRun{}
	if cfg.Execute {
		k, s := cfg.Env("BINANCE_DEMO_API_KEY"), cfg.Env("BINANCE_DEMO_API_SECRET")
		if k == "" || s == "" {
			return errors.New("--execute needs BINANCE_DEMO_API_KEY and BINANCE_DEMO_API_SECRET (Spot Demo Mode keys)")
		}
		ex = exec.NewBinance(k, s, exec.DemoURL)
	}

	gate, breaker, baseline := signal.DefaultGate(), risk.NewBreaker(cfg.MaxOrders, 2), decision.Regex{}
	rec := decision.RecordingFile{Model: decision.DefaultModel, RecordedAt: time.Now().UTC(), Entries: map[string]decision.Recording{}}

	sink.Header()
	for _, h := range hs {
		if err := ctx.Err(); err != nil {
			return err
		}
		ev := toEvent(h)
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		rv, _ := baseline.Classify(cctx, ev)
		cv, err := clef.Classify(cctx, ev)
		cancel()
		if err != nil {
			return fmt.Errorf("%q: %w", h.Headline, err)
		}
		if cfg.Record {
			rec.Entries[decision.Key(h.Headline)] = decision.Recording{Headline: h.Headline, Direction: cv.Direction, PDir: cv.PDirection, Confirmed: cv.Confirmed, LatencyNS: cv.Latency}
		}
		sink.Add(ui.Row{Headline: h.Headline, Expected: h.Expected, Regex: string(rv.Direction), Clef: string(cv.Direction),
			PClef: cv.PDirection, Confirmed: cv.Confirmed, Latency: cv.Latency, Order: act(ctx, gate, breaker, ex, cv, ev)})
		if cfg.Delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.Delay):
			}
		}
	}
	sink.Summary()

	if cfg.Record {
		if err := decision.SaveRecording(cfg.ReplayFile, rec); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "recorded %d responses to %s\n", len(rec.Entries), cfg.ReplayFile)
	}
	return nil
}

// act runs gate → breaker → executor and returns a one-line outcome for the row.
func act(ctx context.Context, gate signal.Gate, breaker *risk.Breaker, ex exec.Executor, v decision.Verdict, ev decision.Event) string {
	in := gate.Decide(v, ev.CandidateSymbols, nil)
	if in.Action == signal.NoAction {
		return in.Reason
	}
	if !breaker.Allow() {
		return "KILL SWITCH: trading halted"
	}
	res, err := ex.Execute(ctx, in)
	breaker.Record(err == nil)
	if err != nil {
		return "order failed: " + err.Error()
	}
	return res
}

func runBench(ctx context.Context, cfg config, stdout io.Writer) error {
	clef, ok := clefFromEnv(cfg.Env)
	if !ok {
		return errors.New("--bench needs OPENROUTER_API_KEY, or CLEF_URL for a self-hosted server; see .env.example")
	}
	hs, err := loadHeadlines(cfg.Headlines)
	if err != nil {
		return err
	}
	evs := make([]decision.Event, len(hs))
	for i, h := range hs {
		evs[i] = toEvent(h)
	}
	label := cfg.Label
	if label == "" {
		label = "clef-flash"
		if cfg.Env("CLEF_URL") == "" {
			label += " via OpenRouter"
		}
	}
	fmt.Fprintf(stdout, "benchmarking %q: %d warmup + %d calls, sequential...\n", label, cfg.BenchWarmup, cfg.BenchN)
	res, err := bench.Run(ctx, clef, evs, cfg.BenchN, cfg.BenchWarmup)
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, res.Report(label, cfg.BenchWarmup))
	return nil
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
