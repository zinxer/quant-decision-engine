# 0003. Safe-by-default execution: dry-run, Demo Mode, kill switch

**Status:** Accepted · 2026-10-05

## Context

The repo is public and meant to be forked and run by people who have not read the code. A trading tool that
places real orders on `go run` is a liability.

## Decision

- **Dry-run by default.** Orders are printed, not sent.
- **`--execute` sends only to Binance Spot Demo Mode** (`demo-api.binance.com`, virtual funds). There is no
  production endpoint in the code.
- **Fixed, small notional** (20 USDT) per order; no leverage, no shorting (spot only; bearish sells only a held position).
- **Kill switch** (`internal/risk`): halts after a max order count or consecutive failures, and stays halted.
  Concurrency-safe via `sync/atomic`, tested with 200 concurrent callers.
- **Symbols mapped in Go** from a static list of already-listed majors; the model never chooses the ticker.

## Consequences

- Going live requires deliberate code changes, which is the point.
- The fixed notional ignores the model's probabilities; probability-based sizing is on the roadmap behind a calibration check.
