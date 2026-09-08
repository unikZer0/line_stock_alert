package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"stock_linebot/backend/internal/models"
)

type QuoteCache interface {
	Get(context.Context, string) (models.StockQuote, bool, error)
	Set(context.Context, models.StockQuote, time.Duration) error
}

type RedisQuoteCache struct{ client *redis.Client }

func NewRedisQuoteCache(client *redis.Client) *RedisQuoteCache {
	return &RedisQuoteCache{client: client}
}

func (c *RedisQuoteCache) Get(ctx context.Context, symbol string) (models.StockQuote, bool, error) {
	value, err := c.client.Get(ctx, "stock:quote:"+symbol).Bytes()
	if err == redis.Nil {
		return models.StockQuote{}, false, nil
	}
	if err != nil {
		return models.StockQuote{}, false, fmt.Errorf("read quote cache: %w", err)
	}
	var quote models.StockQuote
	if err = json.Unmarshal(value, &quote); err != nil {
		return models.StockQuote{}, false, fmt.Errorf("decode quote cache: %w", err)
	}
	quote.Cached = true
	return quote, true, nil
}

func (c *RedisQuoteCache) Set(ctx context.Context, quote models.StockQuote, ttl time.Duration) error {
	quote.Cached = false
	value, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("encode quote cache: %w", err)
	}
	if err = c.client.Set(ctx, "stock:quote:"+quote.Symbol, value, ttl).Err(); err != nil {
		return fmt.Errorf("write quote cache: %w", err)
	}
	return nil
}
