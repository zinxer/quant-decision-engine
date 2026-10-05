// Package exec places (or just prints) orders.
package exec

import (
	"context"
	"fmt"
	"strconv"

	"github.com/adshao/go-binance/v2"

	"github.com/zinxer/quant-decision-engine/internal/signal"
)

// DemoURL is Binance Spot Demo Mode (virtual funds). Never point this at production.
const DemoURL = "https://demo-api.binance.com"

// Executor turns an intent into a result string (order id or description).
type Executor interface {
	Execute(ctx context.Context, in signal.Intent) (string, error)
}

// DryRun prints what would be sent and touches no network.
type DryRun struct{}

// Execute describes the order without sending it.
func (DryRun) Execute(_ context.Context, in signal.Intent) (string, error) {
	return fmt.Sprintf("dry-run %s %s MARKET quote=%.2f USDT", in.Action, in.Symbol, in.QuoteUSDT), nil
}

// Binance sends MARKET orders to Spot Demo Mode.
type Binance struct{ c *binance.Client }

// NewBinance builds a client pinned to baseURL (DemoURL unless a test overrides it).
func NewBinance(apiKey, secret, baseURL string) *Binance {
	return &Binance{c: binance.NewClient(apiKey, secret).SetApiEndpoint(baseURL)}
}

// Execute places a MARKET order sized in quote currency (USDT).
func (b *Binance) Execute(ctx context.Context, in signal.Intent) (string, error) {
	side := binance.SideTypeBuy
	if in.Action == signal.Sell {
		side = binance.SideTypeSell
	}
	o, err := b.c.NewCreateOrderService().
		Symbol(in.Symbol).
		Side(side).
		Type(binance.OrderTypeMarket).
		QuoteOrderQty(strconv.FormatFloat(in.QuoteUSDT, 'f', 2, 64)).
		Do(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("demo order #%d %s %s", o.OrderID, o.Status, in.Symbol), nil
}
