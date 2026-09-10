package services

import (
	"context"
	"errors"
	"regexp"
)

var stockSymbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,19}$`)

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
