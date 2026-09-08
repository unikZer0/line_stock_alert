package services

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"stock_linebot/backend/internal/models"
)

const stockCatalogTTL = time.Hour

type StockService struct {
	provider       StockProvider
	cache          QuoteCache
	quoteTTL       time.Duration
	flights        singleflight.Group
	catalogMu      sync.RWMutex
	catalog        []models.StockMetadata
	catalogExpires time.Time
}

func NewStockService(provider StockProvider, cache QuoteCache, quoteTTL time.Duration) *StockService {
	return &StockService{provider: provider, cache: cache, quoteTTL: quoteTTL}
}

func (s *StockService) Search(ctx context.Context, search, pageValue, limitValue string) (models.StockPage, error) {
	page, err := positiveInt(pageValue, 1, 0)
	if err != nil {
		return models.StockPage{}, newError("INVALID_FILTER", "Page must be a positive integer.", err)
	}
	limit, err := positiveInt(limitValue, 20, 100)
	if err != nil {
		return models.StockPage{}, newError("INVALID_FILTER", "Limit must be between 1 and 100.", err)
	}
	stocks, err := s.stockCatalog(ctx)
	if err != nil {
		return models.StockPage{}, mapStockProviderError(err)
	}
	query := strings.ToUpper(strings.TrimSpace(search))
	filtered := make([]models.StockMetadata, 0)
	for _, stock := range stocks {
		if query == "" || strings.Contains(strings.ToUpper(stock.Symbol), query) || strings.Contains(strings.ToUpper(stock.Name), query) {
			filtered = append(filtered, stock)
		}
	}
	start := len(filtered)
	if page <= len(filtered)/limit+1 {
		start = (page - 1) * limit
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return models.StockPage{Stocks: filtered[start:end], Page: page, Limit: limit, Total: len(filtered)}, nil
}

func (s *StockService) Quote(ctx context.Context, rawSymbol string) (models.StockQuoteDetail, error) {
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	if !stockSymbolPattern.MatchString(symbol) {
		return models.StockQuoteDetail{}, newError("STOCK_NOT_FOUND", "The US stock symbol was not found.", nil)
	}
	stock, err := s.provider.ValidateUSStock(ctx, symbol)
	if err != nil {
		return models.StockQuoteDetail{}, mapStockProviderError(err)
	}
	quote, err := s.cachedQuote(ctx, stock.Symbol)
	if err != nil {
		return models.StockQuoteDetail{}, mapStockProviderError(err)
	}
	return quoteDetail(stock.Name, quote), nil
}

func (s *StockService) Quotes(ctx context.Context, rawSymbols string) (models.MultipleQuotesResult, error) {
	parts := strings.Split(rawSymbols, ",")
	symbols, seen := make([]string, 0, len(parts)), make(map[string]struct{})
	for _, part := range parts {
		symbol := strings.ToUpper(strings.TrimSpace(part))
		if symbol == "" || !stockSymbolPattern.MatchString(symbol) {
			return models.MultipleQuotesResult{}, newError("INVALID_FILTER", "Symbols must be valid comma-separated US stock symbols.", nil)
		}
		if _, exists := seen[symbol]; !exists {
			seen[symbol] = struct{}{}
			symbols = append(symbols, symbol)
		}
	}
	if rawSymbols == "" || len(symbols) > 50 {
		return models.MultipleQuotesResult{}, newError("INVALID_FILTER", "Provide between 1 and 50 unique symbols.", nil)
	}
	catalog, err := s.stockCatalog(ctx)
	if err != nil {
		return models.MultipleQuotesResult{}, mapStockProviderError(err)
	}
	supported := make(map[string]struct{}, len(catalog))
	for _, stock := range catalog {
		supported[stock.Symbol] = struct{}{}
	}
	result := models.MultipleQuotesResult{Quotes: make([]models.StockQuote, 0, len(symbols)), Errors: make([]models.QuoteError, 0)}
	type outcome struct {
		quote  models.StockQuote
		err    error
		symbol string
	}
	results := make(chan outcome, len(symbols))
	semaphore := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, symbol := range symbols {
		if _, exists := supported[symbol]; !exists {
			result.Errors = append(result.Errors, models.QuoteError{Symbol: symbol, Code: "STOCK_NOT_FOUND", Message: "The US stock symbol was not found."})
			continue
		}
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			quote, err := s.cachedQuote(ctx, symbol)
			results <- outcome{quote: quote, err: err, symbol: symbol}
		}(symbol)
	}
	wg.Wait()
	close(results)
	for item := range results {
		if item.err == nil {
			result.Quotes = append(result.Quotes, item.quote)
			continue
		}
		appErr := mapStockProviderError(item.err).(*Error)
		result.Errors = append(result.Errors, models.QuoteError{Symbol: item.symbol, Code: appErr.Code, Message: appErr.Message})
	}
	sort.Slice(result.Quotes, func(i, j int) bool { return result.Quotes[i].Symbol < result.Quotes[j].Symbol })
	sort.Slice(result.Errors, func(i, j int) bool { return result.Errors[i].Symbol < result.Errors[j].Symbol })
	return result, nil
}

func (s *StockService) cachedQuote(ctx context.Context, symbol string) (models.StockQuote, error) {
	if quote, found, err := s.cache.Get(ctx, symbol); err == nil && found {
		return quote, nil
	}
	value, err, _ := s.flights.Do(symbol, func() (any, error) {
		if quote, found, cacheErr := s.cache.Get(ctx, symbol); cacheErr == nil && found {
			return quote, nil
		}
		quote, providerErr := s.provider.Quote(ctx, symbol)
		if providerErr != nil {
			return models.StockQuote{}, providerErr
		}
		_ = s.cache.Set(ctx, quote, s.quoteTTL)
		return quote, nil
	})
	if err != nil {
		return models.StockQuote{}, err
	}
	return value.(models.StockQuote), nil
}

func (s *StockService) stockCatalog(ctx context.Context) ([]models.StockMetadata, error) {
	s.catalogMu.RLock()
	if time.Now().Before(s.catalogExpires) {
		stocks := s.catalog
		s.catalogMu.RUnlock()
		return stocks, nil
	}
	s.catalogMu.RUnlock()
	value, err, _ := s.flights.Do("stock-catalog", func() (any, error) { return s.provider.SearchUSStocks(ctx) })
	if err != nil {
		return nil, err
	}
	stocks := value.([]models.StockMetadata)
	sort.Slice(stocks, func(i, j int) bool { return stocks[i].Symbol < stocks[j].Symbol })
	s.catalogMu.Lock()
	s.catalog, s.catalogExpires = stocks, time.Now().Add(stockCatalogTTL)
	s.catalogMu.Unlock()
	return stocks, nil
}

func positiveInt(value string, fallback, maximum int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 || (maximum > 0 && number > maximum) {
		return 0, errors.New("invalid positive integer")
	}
	return number, nil
}

func quoteDetail(name string, quote models.StockQuote) models.StockQuoteDetail {
	return models.StockQuoteDetail{Symbol: quote.Symbol, Name: name, Price: quote.Price, Change: quote.Change,
		ChangePercent: quote.ChangePercent, Open: quote.Open, High: quote.High, Low: quote.Low,
		PreviousClose: quote.PreviousClose, Currency: quote.Currency, MarketStatus: quote.MarketStatus,
		UpdatedAt: quote.UpdatedAt.Format(time.RFC3339), Cached: quote.Cached}
}
