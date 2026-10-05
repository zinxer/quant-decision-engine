// Package signal maps text to tradable symbols and verdicts to order intents.
package signal

import (
	"regexp"
	"sort"
	"strings"
)

// Only majors that already trade on Binance spot. Listing-announcement tokens
// are deliberately absent: the pair usually doesn't exist yet.
var aliases = map[string]string{
	"BTC": "BTCUSDT", "BITCOIN": "BTCUSDT",
	"ETH": "ETHUSDT", "ETHEREUM": "ETHUSDT", "ETHER": "ETHUSDT",
	"SOL": "SOLUSDT", "SOLANA": "SOLUSDT",
	"BNB": "BNBUSDT",
	"XRP": "XRPUSDT", "RIPPLE": "XRPUSDT",
	"DOGE": "DOGEUSDT", "DOGECOIN": "DOGEUSDT",
}

var wordRe = regexp.MustCompile(`[A-Za-z]+`)

// Symbols returns the distinct Binance pairs mentioned in text, sorted.
func Symbols(text string) []string {
	seen := map[string]bool{}
	for _, w := range wordRe.FindAllString(text, -1) {
		if s, ok := aliases[strings.ToUpper(w)]; ok {
			seen[s] = true
		}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
