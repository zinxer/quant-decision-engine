package exec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zinxer/quant-decision-engine/internal/signal"
)

func TestDryRunTouchesNothing(t *testing.T) {
	out, err := DryRun{}.Execute(context.Background(), signal.Intent{Action: signal.Buy, Symbol: "ETHUSDT", QuoteUSDT: 20})
	if err != nil || !strings.Contains(out, "dry-run BUY ETHUSDT") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBinanceOrderParams(t *testing.T) {
	var q map[string][]string
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/order" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		key = r.Header.Get("X-MBX-APIKEY")
		_ = r.ParseForm()
		q = r.Form
		_, _ = w.Write([]byte(`{"symbol":"ETHUSDT","orderId":42,"status":"FILLED"}`))
	}))
	defer srv.Close()

	out, err := NewBinance("k", "s", srv.URL).Execute(context.Background(),
		signal.Intent{Action: signal.Buy, Symbol: "ETHUSDT", QuoteUSDT: 20})
	if err != nil {
		t.Fatal(err)
	}
	if key != "k" || q["symbol"][0] != "ETHUSDT" || q["side"][0] != "BUY" ||
		q["type"][0] != "MARKET" || q["quoteOrderQty"][0] != "20.00" || q["signature"] == nil {
		t.Errorf("bad request: key=%q form=%v", key, q)
	}
	if !strings.Contains(out, "#42") {
		t.Errorf("out=%q", out)
	}
}
