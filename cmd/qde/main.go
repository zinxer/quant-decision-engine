// Command qde streams headlines through a regex baseline and a Clef decision
// model, gates the verdicts, and prints (or, opt-in, places) demo orders.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// version is set at build time: go build -ldflags "-X main.version=v0.1.0".
var version = "dev"

func main() {
	var cfg config
	flag.StringVar(&cfg.Mode, "mode", "auto", "auto | live | replay")
	flag.BoolVar(&cfg.Execute, "execute", false, "send orders to Binance Spot Demo Mode (default: dry-run)")
	flag.StringVar(&cfg.Headlines, "headlines", "data/headlines.json", "headlines file")
	flag.StringVar(&cfg.ReplayFile, "replay-file", "data/replay/clef-flash.json", "replay recording")
	flag.BoolVar(&cfg.Record, "record", false, "call the live API and write the replay file")
	flag.DurationVar(&cfg.Delay, "delay", 0, "pause between rows (for screen recordings)")
	flag.IntVar(&cfg.MaxOrders, "max-orders", 3, "kill switch: max orders per session")
	flag.StringVar(&cfg.Format, "format", "table", "table | jsonl (one decision record per line, for audit logs)")
	flag.StringVar(&cfg.Color, "color", "auto", "auto | always | never")
	flag.IntVar(&cfg.BenchN, "bench", 0, "latency benchmark: N live calls (needs OPENROUTER_API_KEY or CLEF_URL), no trading")
	flag.IntVar(&cfg.BenchWarmup, "bench-warmup", 5, "calls discarded before measuring (connection setup, caches)")
	flag.StringVar(&cfg.Label, "label", "", "name for the --bench report row, e.g. \"self-hosted H100\"")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("qde", version)
		return
	}
	loadDotEnv(".env")
	cfg.Env = os.Getenv

	// Ctrl-C cancels in-flight model calls and stops the loop cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	if cfg.BenchN > 0 {
		err = runBench(ctx, cfg, os.Stdout)
	} else {
		err = run(ctx, cfg, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
