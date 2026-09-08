package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFinnhubValidateUSStock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" || r.URL.Query().Get("exchange") != "US" || r.Header.Get("X-Finnhub-Token") != "secret" {
			t.Fatalf("unexpected request: %s headers=%v", r.URL.String(), r.Header)
		}
		_, _ = w.Write([]byte(`{"count":1,"result":[{"description":"Apple Inc.","displaySymbol":"AAPL","symbol":"AAPL"}]}`))
	}))
	defer server.Close()
	client := NewFinnhubClient(server.Client(), server.URL, "secret")
	stock, err := client.ValidateUSStock(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("ValidateUSStock() error = %v", err)
	}
	if stock.Symbol != "AAPL" || stock.Currency != "USD" {
		t.Fatalf("stock = %#v", stock)
	}
}

func TestFinnhubQuoteMapsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"c":190.5,"d":1.5,"dp":0.79,"h":192,"l":188,"o":189,"pc":189,"t":1700000000}`))
	}))
	defer server.Close()
	quote, err := NewFinnhubClient(server.Client(), server.URL, "secret").Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if quote.Price != 190.5 || quote.ChangePercent != 0.79 || quote.UpdatedAt.Unix() != 1700000000 {
		t.Fatalf("quote = %#v", quote)
	}
}
