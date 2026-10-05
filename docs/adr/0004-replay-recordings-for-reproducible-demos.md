# 0004. Commit real model recordings so the demo runs offline

**Status:** Accepted · 2026-10-05

## Context

A demo that needs an API key loses most people at step one, and live model output varies between runs and
model versions, which makes screenshots and claims hard to reproduce.

## Decision

`--record` captures real responses (direction, probabilities, confirmation and the measured latency) into
`data/replay/clef-flash.json`, which is committed. `--mode auto` replays it when no key is set.
Recordings are only ever written by the tool from real API calls, never by hand. Unit-test fixtures are
synthetic and labelled as such.

## Consequences

- `git clone && go run ./cmd/qde` works with no keys, and the README figures are reproducible.
- The recording goes stale as the model changes; re-run `make record` and review the diff.
- Replayed latency is historical, not live; the README and table say which mode is running.
