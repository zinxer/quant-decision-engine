package signal

import (
	"reflect"
	"testing"

	"github.com/zinxer/quant-decision-engine/internal/decision"
)

func TestSymbols(t *testing.T) {
	cases := map[string][]string{
		"SEC approves spot Ethereum ETF": {"ETHUSDT"},
		"$BTC and SOL rally":             {"BTCUSDT", "SOLUSDT"},
		"Bitcoin, BTC again":             {"BTCUSDT"},
		"Binance lists $XYZ":             {},
		"Method ethics panel":            {}, // no substring matches
	}
	for in, want := range cases {
		if got := Symbols(in); !reflect.DeepEqual(got, want) {
			t.Errorf("%q = %v, want %v", in, got, want)
		}
	}
}

func TestGate(t *testing.T) {
	g := DefaultGate()
	v := func(d decision.Direction, p, c float64) decision.Verdict {
		return decision.Verdict{Direction: d, PDirection: p, Confirmed: c}
	}
	eth := []string{"ETHUSDT"}
	cases := []struct {
		name string
		v    decision.Verdict
		syms []string
		held map[string]bool
		want Action
	}{
		{"confirmed bull buys", v(decision.Bullish, 0.93, 0.97), eth, nil, Buy},
		{"low direction prob", v(decision.Bullish, 0.60, 0.97), eth, nil, NoAction},
		{"unconfirmed", v(decision.Bullish, 0.93, 0.20), eth, nil, NoAction},
		{"none direction", v(decision.None, 0.99, 0.99), eth, nil, NoAction},
		{"bearish without position", v(decision.Bearish, 0.95, 0.95), eth, nil, NoAction},
		{"bearish with position", v(decision.Bearish, 0.95, 0.95), eth, map[string]bool{"ETHUSDT": true}, Sell},
		{"no symbol", v(decision.Bullish, 0.99, 0.99), nil, nil, NoAction},
		{"ambiguous symbols", v(decision.Bullish, 0.99, 0.99), []string{"BTCUSDT", "ETHUSDT"}, nil, NoAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := g.Decide(tc.v, tc.syms, tc.held).Action; got != tc.want {
				t.Errorf("got %s want %s", got, tc.want)
			}
		})
	}
}
