package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

var stockSymbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,19}$`)

type WatchlistStore interface {
	List(context.Context, string) ([]models.WatchlistItem, error)
	Count(context.Context, string) (int, error)
	Add(context.Context, string, models.StockMetadata) (models.WatchlistItem, error)
	Delete(context.Context, string, string) error
}

type WatchlistService struct {
	store    WatchlistStore
	provider StockProvider
	cache    QuoteCache
	limit    int
	cacheTTL time.Duration
}

func NewWatchlistService(store WatchlistStore, provider StockProvider, cache QuoteCache, limit int, cacheTTL time.Duration) *WatchlistService {
	return &WatchlistService{store: store, provider: provider, cache: cache, limit: limit, cacheTTL: cacheTTL}
}

func (s *WatchlistService) List(ctx context.Context, userID string) ([]models.WatchlistItem, error) {
	items, err := s.store.List(ctx, userID)
	if err != nil {
		return nil, newError("INTERNAL_SERVER_ERROR", "Could not load the watchlist.", err)
	}
	indices := make(map[string][]int)
	for i := range items {
		indices[items[i].Symbol] = append(indices[items[i].Symbol], i)
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)
	for symbol, positions := range indices {
		symbol, positions := symbol, positions
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			quote, quoteErr := s.quote(ctx, symbol)
			if quoteErr != nil {
				return
			}
			mu.Lock()
			for _, i := range positions {
				copyQuote := quote
				items[i].Quote = &copyQuote
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return items, nil
}

func (s *WatchlistService) Add(ctx context.Context, userID string, request models.AddWatchlistRequest) (models.WatchlistItem, error) {
	symbol := strings.ToUpper(strings.TrimSpace(request.Symbol))
	if !stockSymbolPattern.MatchString(symbol) {
		return models.WatchlistItem{}, newError("VALIDATION_ERROR", "A valid US stock symbol is required.", nil)
	}
	count, err := s.store.Count(ctx, userID)
	if err != nil {
		return models.WatchlistItem{}, newError("INTERNAL_SERVER_ERROR", "Could not check the watchlist limit.", err)
	}
	if count >= s.limit {
		return models.WatchlistItem{}, newError("WATCHLIST_LIMIT_REACHED", "The watchlist limit has been reached.", nil)
	}
	stock, err := s.provider.ValidateUSStock(ctx, symbol)
	if err != nil {
		return models.WatchlistItem{}, mapStockProviderError(err)
	}
	item, err := s.store.Add(ctx, userID, stock)
	if errors.Is(err, repositories.ErrStockAlreadyWatched) {
		return models.WatchlistItem{}, newError("STOCK_ALREADY_IN_WATCHLIST", "This stock is already in the watchlist.", err)
	}
	if err != nil {
		return models.WatchlistItem{}, newError("INTERNAL_SERVER_ERROR", "Could not add the stock to the watchlist.", err)
	}
	if quote, quoteErr := s.quote(ctx, stock.Symbol); quoteErr == nil {
		item.Quote = &quote
	}
	return item, nil
}

func (s *WatchlistService) Delete(ctx context.Context, userID, entryID string) error {
	if strings.TrimSpace(entryID) == "" {
		return newError("VALIDATION_ERROR", "A watchlist entry ID is required.", nil)
	}
	err := s.store.Delete(ctx, userID, entryID)
	if errors.Is(err, repositories.ErrWatchlistNotFound) {
		return newError("WATCHLIST_NOT_FOUND", "Watchlist entry not found.", err)
	}
	if err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Could not remove the stock from the watchlist.", err)
	}
	return nil
}

func (s *WatchlistService) quote(ctx context.Context, symbol string) (models.StockQuote, error) {
	if quote, found, err := s.cache.Get(ctx, symbol); err == nil && found {
		return quote, nil
	}
	quote, err := s.provider.Quote(ctx, symbol)
	if err != nil {
		return models.StockQuote{}, err
	}
	_ = s.cache.Set(ctx, quote, s.cacheTTL)
	return quote, nil
}

func mapStockProviderError(err error) error {
	if errors.Is(err, ErrStockNotFound) {
		return newError("STOCK_NOT_FOUND", "The US stock symbol was not found.", err)
	}
	if errors.Is(err, ErrStockProviderRateLimit) {
		return newError("STOCK_PROVIDER_RATE_LIMIT", "The stock data provider rate limit was reached. Please try again shortly.", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newError("STOCK_PROVIDER_TIMEOUT", "The stock data provider timed out.", err)
	}
	return newError("STOCK_PROVIDER_UNAVAILABLE", "Stock data is temporarily unavailable.", err)
}
