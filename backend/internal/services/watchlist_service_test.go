package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"stock_linebot/backend/internal/models"
)

type watchlistStoreStub struct {
	items []models.WatchlistItem
	count int
	added models.StockMetadata
}

func (s *watchlistStoreStub) List(context.Context, string) ([]models.WatchlistItem, error) {
	return s.items, nil
}
func (s *watchlistStoreStub) Count(context.Context, string) (int, error) { return s.count, nil }
func (s *watchlistStoreStub) Add(_ context.Context, _ string, stock models.StockMetadata) (models.WatchlistItem, error) {
	s.added = stock
	return models.WatchlistItem{ID: "entry-1", Symbol: stock.Symbol}, nil
}
func (s *watchlistStoreStub) Delete(context.Context, string, string) error { return nil }

type stockProviderStub struct{ validateCalls, quoteCalls atomic.Int32 }

func (p *stockProviderStub) ValidateUSStock(_ context.Context, symbol string) (models.StockMetadata, error) {
	p.validateCalls.Add(1)
	if symbol == "MISSING" {
		return models.StockMetadata{}, ErrStockNotFound
	}
	return models.StockMetadata{Symbol: symbol, Name: "Apple Inc.", Exchange: "US", Currency: "USD"}, nil
}
func (p *stockProviderStub) Quote(_ context.Context, symbol string) (models.StockQuote, error) {
	p.quoteCalls.Add(1)
	return models.StockQuote{Symbol: symbol, Price: 200, Currency: "USD", UpdatedAt: time.Now()}, nil
}

type quoteCacheStub struct {
	quote models.StockQuote
	found bool
}

func (c *quoteCacheStub) Get(context.Context, string) (models.StockQuote, bool, error) {
	return c.quote, c.found, nil
}
func (c *quoteCacheStub) Set(context.Context, models.StockQuote, time.Duration) error { return nil }

func TestWatchlistAddNormalizesSymbolAndIncludesQuote(t *testing.T) {
	store, provider, cache := &watchlistStoreStub{}, &stockProviderStub{}, &quoteCacheStub{}
	service := NewWatchlistService(store, provider, cache, 50, 15*time.Second)
	item, err := service.Add(context.Background(), "user-1", models.AddWatchlistRequest{Symbol: " aapl "})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if store.added.Symbol != "AAPL" {
		t.Fatalf("added symbol = %q", store.added.Symbol)
	}
	if item.Quote == nil || item.Quote.Price != 200 {
		t.Fatalf("quote = %#v", item.Quote)
	}
}

func TestWatchlistAddEnforcesLimit(t *testing.T) {
	service := NewWatchlistService(&watchlistStoreStub{count: 2}, &stockProviderStub{}, &quoteCacheStub{}, 2, time.Second)
	_, err := service.Add(context.Background(), "user-1", models.AddWatchlistRequest{Symbol: "AAPL"})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "WATCHLIST_LIMIT_REACHED" {
		t.Fatalf("error = %#v", err)
	}
}

func TestWatchlistListFetchesEachUniqueSymbolOnce(t *testing.T) {
	store := &watchlistStoreStub{items: []models.WatchlistItem{{Symbol: "AAPL"}, {Symbol: "AAPL"}, {Symbol: "MSFT"}}}
	provider := &stockProviderStub{}
	service := NewWatchlistService(store, provider, &quoteCacheStub{}, 50, time.Second)
	items, err := service.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if provider.quoteCalls.Load() != 2 {
		t.Fatalf("quote calls = %d, want 2", provider.quoteCalls.Load())
	}
	for _, item := range items {
		if item.Quote == nil {
			t.Fatalf("missing quote for %s", item.Symbol)
		}
	}
}
