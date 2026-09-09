package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTwelveDataClientCandles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/time_series" || r.URL.Query().Get("symbol") != "NVDA" || r.URL.Query().Get("interval") != "5min" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"currency":"USD"},"values":[{"datetime":"2026-09-09 13:30:00","open":"230.10","high":"231.20","low":"229.80","close":"230.90","volume":"1200"}]}`))
	}))
	defer server.Close()
	client := NewTwelveDataClient(&http.Client{Timeout: time.Second}, server.URL, "test-key")
	series, err := client.Candles(context.Background(), "NVDA", "5min", 78)
	if err != nil {
		t.Fatalf("Candles returned error: %v", err)
	}
	if len(series.Candles) != 1 || series.Candles[0].Close != 230.9 || series.Currency != "USD" {
		t.Fatalf("unexpected series: %+v", series)
	}
}

func TestTwelveDataClientMapsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }))
	defer server.Close()
	client := NewTwelveDataClient(server.Client(), server.URL, "test-key")
	_, err := client.Candles(context.Background(), "NVDA", "5min", 78)
	if err != ErrCandleProviderRateLimit {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}
