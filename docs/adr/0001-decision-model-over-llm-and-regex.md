# 0001. Use a typed decision model, not an LLM or keyword rules

**Status:** Accepted · 2026-10-05

## Context

Headline-driven signals fail on meaning, not on speed: denials, questions, rumors, retractions and stale news
contain the same keywords as the real event. Two common approaches:

- **Keyword / regex rules:** microseconds, deterministic, but blind to negation, modality, tense and attribution.
  Patching them ("ignore `denies`") breaks other cases ("SEC denies ETF application" is a real negative event).
- **General-purpose LLMs:** read meaning, but return free text that must be parsed and repaired, give no
  probability to threshold on, and take seconds.

Decision models (TypeSafe Jev, Cloudflare Clef) take *state* plus *typed questions* (`choice`, `noul`, `score`)
and return bounded answers with probabilities in a single pass.

## Decision

Classify each headline with a decision model, asking two questions in one request:
`direction` (choice: bullish / bearish / none) and `confirmed` (noul). Keep the regex baseline in the same
run as a visible control.

## Consequences

- The gate thresholds on probabilities and requires *both* direction and confirmation, which gives a second
  line of defence when the direction call is wrong (seen in the recorded "false alarm" headline).
- Behaviour changes by editing plain-language criteria, verified with `make record`, instead of growing a rule list.
- Adds a network dependency, per-token cost and model-version drift. Mitigated by replay recordings (ADR 0004)
  and a path to self-hosting (ADR 0002).
- Typed outputs constrain the *format* of answers, not their truth; prompt injection via headlines remains a risk
  bounded by the gate and kill switch (ADR 0003).
