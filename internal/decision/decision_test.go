package decision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Synthetic fixtures shaped like the documented Clef response. NOT real model output.
const wrapped = `{"result":{"model":"clef-flash","answers":{
 "direction":{"type":"choice","choice":"none","probabilities":{"bullish":0.03,"bearish":0.01,"none":0.96},"confidence":0.9},
 "confirmed":{"type":"noul","noul":0.05}}},"success":true,"errors":[],"messages":[]}`

const bare = `{"answers":{
 "direction":{"type":"choice","choice":"bullish","probabilities":{"bullish":0.93,"bearish":0.02,"none":0.05},"confidence":0.9},
 "confirmed":{"type":"noul","noul":0.97}}}`

func TestClefClassify(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		want       Direction
		wantP      float64
		wantErr    bool
	}{
		{"result envelope", wrapped, 200, None, 0.96, false},
		{"bare body", bare, 200, Bullish, 0.93, false},
		{"api error", `{"error":{"code":401,"message":"Missing Authentication header"}}`, 401, "", 0, true},
		{"cloudflare-style error", `{"success":false,"errors":[{"code":10000,"message":"auth"}],"result":null}`, 401, "", 0, true},
		{"missing answer", `{"answers":{}}`, 200, "", 0, true},
		{"unknown choice", `{"answers":{"direction":{"choice":"moon"},"confirmed":{"noul":1}}}`, 200, "", 0, true},
		{"bad json", `nope`, 200, "", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAuth string
			var gotBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c := NewClef(ClefConfig{URL: srv.URL, Token: "tok", Model: "clef-flash"})
			v, err := c.Classify(context.Background(), Event{Headline: "h", CandidateSymbols: []string{"ETHUSDT"}})
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if gotAuth != "Bearer tok" {
				t.Errorf("auth header = %q", gotAuth)
			}
			if gotBody["model"] != "clef-flash" {
				t.Errorf("model = %v", gotBody["model"])
			}
			qs, _ := gotBody["questions"].(map[string]any)
			if qs["direction"] == nil || qs["confirmed"] == nil {
				t.Errorf("questions missing: %v", qs)
			}
			if tc.wantErr {
				return
			}
			if v.Direction != tc.want || v.PDirection != tc.wantP {
				t.Errorf("verdict = %+v", v)
			}
			if v.Latency <= 0 {
				t.Error("latency not measured")
			}
		})
	}
}

func TestClefTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	c := NewClef(ClefConfig{URL: srv.URL, Timeout: 20 * time.Millisecond})
	if _, err := c.Classify(context.Background(), Event{Headline: "h"}); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRegexBaselineFallsForDenials(t *testing.T) {
	cases := map[string]Direction{
		"Binance will list $XYZ":                Bullish,
		"Binance denies rumors of listing $XYZ": Bullish, // the demo's point: the baseline is wrong here
		"Exchange hacked, $40M drained":         Bearish,
		"Fed holds rates steady":                None,
	}
	for h, want := range cases {
		v, _ := Regex{}.Classify(context.Background(), Event{Headline: h})
		if v.Direction != want {
			t.Errorf("%q = %s, want %s", h, v.Direction, want)
		}
	}
}
