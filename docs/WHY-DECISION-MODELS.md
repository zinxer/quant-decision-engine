# Why a decision model is a step up from regex

> Not financial advice. The evidence here supports a claim about **interpretation**,
> not about trading edge. Read the [limits](#7-where-it-is-not-better-and-what-can-go-wrong) before quoting numbers.

## 1. Short version

A regex answers *"which words appear?"*. A headline trade needs the answer to *"what happened, and is it confirmed?"*.
Those are different questions, and the gap between them is exactly where news-driven bots lose money:
denials, questions, rumors, corrections, stale news and negation all contain the same keywords as the real event.

A decision model answers the second question directly, in one pass, as typed output you can threshold:
a chosen option, a probability per option, and a yes/no probability. It does not make the bot smarter about
markets. It removes one specific, well-known class of failure (misreading text) at a cost of tens to hundreds of milliseconds.

## 2. What a regex can and cannot see

A regex matches **surface form**. Headlines carry meaning in features the surface form hides:

| Feature | Example | What a keyword rule sees |
|---|---|---|
| Denial | "Binance **denies** rumors of **listing** $XYZ" | `listing` → BUY |
| Question | "Will Bitcoin be **banned** next week?" | `banned` → SELL |
| Speculation / hearsay | "**Unconfirmed**: Dogecoin could be **listed**… insiders say" | `listed` → BUY |
| Stale news | "ETF **approval** was rejected last year, analysts recall" | `approval` → BUY |
| Retraction | "**Hack** report was a false alarm, no funds lost" | `hack` → SELL |
| Disputed report | "Coinbase to **delist** Solana? Company calls the report false" | `delist` → SELL |
| Negated action | "SEC **denies** spot Ethereum ETF application" | nothing matches → no signal |

Each row needs *negation*, *modality* ("could", "unconfirmed"), *tense* ("last year"), or *attribution*
("insiders say") to be read correctly. None of that is a word list.

## 3. Evidence from this repo

### Recorded run (8 headlines, `data/replay/clef-flash.json`)

| Headline | Expected | Regex says (matched) | Clef-flash says |
|---|---|---|---|
| SEC approves spot Ethereum ETF, trading begins Thursday | bullish | bullish (`approves`) | bullish 0.97 / confirmed 0.92 |
| Binance denies rumors of listing $XYZ token | none | bullish (`listing`) | none 0.95 / confirmed 0.29 |
| Anonymous account asks: will Bitcoin be banned next week? | none | bearish (`banned`) | none 0.94 / confirmed 0.02 |
| Solana bridge exploit confirmed: $40M drained, validators halt network | bearish | bearish (`exploit`) | bearish 0.97 / confirmed 0.87 |
| Unconfirmed: Dogecoin could be listed by major retailer, insiders say | none | bullish (`listed`) | none 0.61 / confirmed 0.03 |
| Bitcoin ETF approval was rejected last year, analysts recall | none | bullish (`approval`) | none 0.63 / confirmed 0.85 |
| BNB Chain hack report was a false alarm, team confirms no funds lost | none | bearish (`hack`) | **bullish 0.83** / confirmed 0.29 |
| Coinbase to delist Solana? Company calls the report false | none | bearish (`delist`) | none 0.87 / confirmed 0.02 |

Regex 2/8, Clef-flash 7/8. **Six of these eight were written as traps for keyword logic**, so this
is an adversarial set by construction. It demonstrates the failure mode; it does not estimate how often
real headlines fall into it.

### Extra run (4 headlines, one-off, not recorded or added to the eval set)

Run live on 2026-10-05 to test whether patching the regex would close the gap:

| Headline | Regex | Clef-flash | Note |
|---|---|---|---|
| SEC denies spot Ethereum ETF application | none | bearish 0.94 | Regex has no keyword; model reads the negative outcome |
| Binance will not delist Solana, spokesperson confirms | bearish (`delist`) | bullish 0.90 | Model reads relief as positive; the gate **would place a dry-run BUY** |
| Hackers fail to breach Coinbase, no funds lost | none (by luck) | bullish 0.86 | Same relief pattern; no trade only because no single symbol was mapped |
| Bitcoin ETF approved by SEC | bullish | bullish 0.97 | Both correct |

The two "relief" rows are judgement calls (my label was `none`), but they are a real finding: **the model tends to
read "bad thing didn't happen" as bullish**, and the current gate acts on it. See the limits below.

## 4. Why patching the regex doesn't converge

The tempting fix for "Binance **denies** rumors of listing" is a rule: *if the headline contains `denies`, ignore it.*
That fixes row 2 and breaks the next headline, "SEC **denies** spot Ethereum ETF application", which is a
confirmed, negative event. Every patch adds exceptions: `denies`, `false`, `rejects`, `not`, `no`, `fail`, `?`,
`unconfirmed`, `could`, `reportedly`, "last year"... and their combinations ("will **not** delist", "no funds lost").
The rule set grows, interacts with itself, and still has no notion of *who* is denying *what*.

A decision model moves that maintenance from a growing rule list to a short, readable statement of what you
mean by each option, in `BuildRequest` (`internal/decision/clef.go`):

- `bullish`: confirmed positive news that would likely push the price up
- `none`: denial, rumor, question, speculation, stale news, or irrelevant

Changing behaviour means editing those sentences, and `make record` shows the effect on the eval set.

## 5. What the model adds besides accuracy

| Capability | Regex | Decision model |
|---|---|---|
| Reads meaning (negation, modality, tense) | no | yes |
| Probability on the answer | none (match / no match) | per option, so the gate can abstain below a threshold |
| Separates *direction* from *did it happen* | one blob | two typed questions: `direction` (choice) and `confirmed` (noul) |
| Output shape | boolean | typed and bounded: only the options you defined can come back |
| Extending to a new case | add rules, re-test interactions | edit the question text, re-run the eval |
| Auditability | pattern that matched | the chosen option and its probabilities, logged per decision |

The `confirmed` question is the clearest win. In the recorded run, the model called the BNB "false alarm"
headline `bullish` (wrong) but scored it 0.29 confirmed, so the gate still declined to trade. The two-question
structure gave a second line of defence that a single keyword match cannot.

## 6. The honest case for regex

Regex is better on several axes, and a good system keeps it:

- **Speed:** microseconds versus tens-to-hundreds of milliseconds per headline.
- **Cost:** free, versus a per-token price (small, but non-zero at scale).
- **Determinism and debuggability:** the same input always gives the same output, and you can point at the exact pattern.
- **No dependency:** no network, no model version to drift.

The sensible architecture is a **hybrid**: a cheap regex or symbol-mapper prefilter drops irrelevant headlines
(no tradable symbol, wrong language, obvious spam) before they reach the model. This repo already does the symbol
step in Go; a keyword prefilter is a natural next addition.

## 7. Where it is not better, and what can go wrong

- **Selection bias in the evidence.** The eval set was written to trip up keyword logic. On ordinary headlines
  ("Fed holds rates steady") regex does fine. n = 8, labels are the author's.
- **Relief headlines.** In the extra run, "will not delist" and "fail to breach" came back bullish and one produced a BUY.
  The model is reading sentiment as direction. A `materiality` or "price-moving" question would likely help; this is untested.
- **No world knowledge.** The model only sees the `state` you pass. It cannot know a move is already priced in,
  and it recognises "stale" only when the text says so ("last year").
- **Prompt injection.** The headline is untrusted text placed in front of a model. Typed outputs constrain the
  *format* of an answer, not its *truth*, so a crafted headline could still push a verdict. The fixed notional,
  both thresholds and the kill switch limit the damage; they do not remove the risk.
- **Probabilities are not verified as calibrated.** "0.9" is a number to threshold, not a proven 90% hit rate. Check on held-out data before sizing on it.
- **Hosted latency.** Through OpenRouter the measured p50 is a few hundred milliseconds. Self-hosting is how to
  get toward sub-50 ms; see [ARCHITECTURE.md](ARCHITECTURE.md) and the `--bench` flag.
- **Model drift.** Hosted models can change underneath you. Pin the model version and re-run the eval when it changes.

## 8. Bottom line

For turning unstructured text into a decision a program can act on, a decision model is a genuine step up from
regex: it reads what the sentence *means*, returns probabilities, separates "which way" from "did it happen", and
replaces an ever-growing rule list with plain-language criteria. Being open-weight and self-hostable is what makes
it viable for latency-sensitive use. That is the claim this repo supports.

It is not evidence of profitable trading. That needs a larger labelled set, a backtest with fees and slippage,
and a calibration check, all listed in the roadmap.
