# 0002. Clef-flash: hosted (OpenRouter) for the demo, self-hosted for latency

**Status:** Accepted · 2026-10-05

## Context

The original spec used TypeSafe Jev, a closed hosted API. Cloudflare's Clef family (released 2026-10-01) is
Jev-API compatible, and its weights are open (Apache 2.0). Clef-flash is the 9B, latency-optimised member;
Cloudflare reports 38.8 ms median model latency on its own benchmark.

A hosted call from a laptop adds the public internet, a routing layer and a provider queue. Measured through
OpenRouter: p50 ~243 ms, p95 ~729 ms (`--bench 20`).

## Decision

- Default to `cloudflare/clef-flash` via OpenRouter's Decisions API: one key, no infrastructure, easy to fork.
- Make the endpoint a single variable (`CLEF_URL`, `CLEF_MODEL`, `CLEF_API_TOKEN`) so a self-hosted Clef or
  a Jev endpoint is a config change, not a code change.
- Ship `--bench` so anyone can measure a self-hosted deployment with the same harness and publish the numbers.

## Consequences

- The demo's latency is deliberately the slow path; the README says so and labels the self-hosted figure as a
  projection until measured.
- Self-hosting is not packaged here. Cloudflare describes its reference setup (`transformers` + PyTorch on H200)
  as a research setup; a production serving stack is future work.
- Open weights remove vendor lock-in for the core decision step.
