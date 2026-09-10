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

type CandleCache interface {
	GetCandles(context.Context, string) (models.StockCandleSeries, bool, error)
	SetCandles(context.Context, string, models.StockCandleSeries, time.Duration) error
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

func (c *RedisQuoteCache) GetCandles(ctx context.Context, key string) (models.StockCandleSeries, bool, error) {
	value, err := c.client.Get(ctx, "stock:candles:"+key).Bytes()
	if err == redis.Nil {
		return models.StockCandleSeries{}, false, nil
	}
	if err != nil {
		return models.StockCandleSeries{}, false, fmt.Errorf("read candle cache: %w", err)
	}
	var series models.StockCandleSeries
	if err = json.Unmarshal(value, &series); err != nil {
		return models.StockCandleSeries{}, false, fmt.Errorf("decode candle cache: %w", err)
	}
	series.Cached = true
	return series, true, nil
}

func (c *RedisQuoteCache) SetCandles(ctx context.Context, key string, series models.StockCandleSeries, ttl time.Duration) error {
	series.Cached = false
	value, err := json.Marshal(series)
	if err != nil {
		return fmt.Errorf("encode candle cache: %w", err)
	}
	if err = c.client.Set(ctx, "stock:candles:"+key, value, ttl).Err(); err != nil {
		return fmt.Errorf("write candle cache: %w", err)
	}
	return nil
}
