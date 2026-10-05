package signal

import "github.com/zinxer/quant-decision-engine/internal/decision"

// Action is what the executor should do with an intent.
type Action string

// Possible actions.
const (
	Buy      Action = "BUY"
	Sell     Action = "SELL"
	NoAction Action = "NO_ACTION"
)

// Gate holds the decision thresholds.
type Gate struct {
	MinDirection float64 // min P(direction)
	MinConfirmed float64 // min P(confirmed)
	QuoteUSDT    float64 // fixed notional per order
}

// DefaultGate returns the thresholds used in the demo: P(direction) ≥ 0.80, P(confirmed) ≥ 0.70, 20 USDT.
func DefaultGate() Gate { return Gate{MinDirection: 0.80, MinConfirmed: 0.70, QuoteUSDT: 20} }

// Intent is what the executor should do.
type Intent struct {
	Action    Action
	Symbol    string
	QuoteUSDT float64
	Reason    string
}

// Decide turns a verdict into an intent. Spot has no shorting, so a bearish
// call only produces SELL for a symbol we hold; otherwise it is NO_ACTION.
func (g Gate) Decide(v decision.Verdict, symbols []string, held map[string]bool) Intent {
	no := func(reason string) Intent { return Intent{Action: NoAction, Reason: reason} }
	if v.Direction == decision.None {
		return no("no directional signal")
	}
	if len(symbols) != 1 {
		return no("need exactly one candidate symbol")
	}
	sym := symbols[0]
	if v.PDirection < g.MinDirection {
		return no("direction probability below threshold")
	}
	if v.Confirmed < g.MinConfirmed {
		return no("not confirmed")
	}
	switch v.Direction {
	case decision.Bullish:
		return Intent{Action: Buy, Symbol: sym, QuoteUSDT: g.QuoteUSDT, Reason: "confirmed bullish"}
	case decision.Bearish:
		if !held[sym] {
			return no("bearish but no position (spot cannot short)")
		}
		return Intent{Action: Sell, Symbol: sym, QuoteUSDT: g.QuoteUSDT, Reason: "confirmed bearish"}
	}
	return no("unhandled direction")
}
