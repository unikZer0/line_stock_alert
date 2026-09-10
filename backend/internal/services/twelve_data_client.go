package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"stock_linebot/backend/internal/models"
)

var (
	ErrCandleNotFound          = errors.New("candle data not found")
	ErrCandleProviderRateLimit = errors.New("candle provider rate limit")
)

type CandleProvider interface {
	Candles(context.Context, string, string, int) (models.StockCandleSeries, error)
}

type TwelveDataClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewTwelveDataClient(httpClient *http.Client, baseURL, apiKey string) *TwelveDataClient {
	return &TwelveDataClient{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}

func (c *TwelveDataClient) Candles(ctx context.Context, symbol, interval string, outputSize int) (models.StockCandleSeries, error) {
	endpoint, err := url.Parse(c.baseURL + "/time_series")
	if err != nil {
		return models.StockCandleSeries{}, fmt.Errorf("build Twelve Data URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("symbol", symbol)
	query.Set("interval", interval)
	query.Set("outputsize", strconv.Itoa(outputSize))
	query.Set("timezone", "UTC")
	query.Set("order", "ASC")
	query.Set("apikey", c.apiKey)
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return models.StockCandleSeries{}, fmt.Errorf("create Twelve Data request: %w", err)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return models.StockCandleSeries{}, fmt.Errorf("request Twelve Data candles: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return models.StockCandleSeries{}, ErrCandleProviderRateLimit
	}
	if response.StatusCode != http.StatusOK {
		return models.StockCandleSeries{}, fmt.Errorf("Twelve Data returned HTTP %d", response.StatusCode)
	}
	var payload twelveDataResponse
	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return models.StockCandleSeries{}, fmt.Errorf("decode Twelve Data response: %w", err)
	}
	if payload.Status == "error" {
		if payload.Code == 429 {
			return models.StockCandleSeries{}, ErrCandleProviderRateLimit
		}
		if payload.Code == 400 || payload.Code == 404 {
			return models.StockCandleSeries{}, ErrCandleNotFound
		}
		return models.StockCandleSeries{}, fmt.Errorf("Twelve Data error %d: %s", payload.Code, payload.Message)
	}
	candles := make([]models.StockCandle, 0, len(payload.Values))
	for _, value := range payload.Values {
		candle, parseErr := value.candle()
		if parseErr != nil {
			return models.StockCandleSeries{}, parseErr
		}
		candles = append(candles, candle)
	}
	if len(candles) == 0 {
		return models.StockCandleSeries{}, ErrCandleNotFound
	}
	return models.StockCandleSeries{Symbol: symbol, Interval: interval, Currency: payload.Meta.Currency, Timezone: "UTC", Candles: candles}, nil
}

type twelveDataResponse struct {
	Meta struct {
		Currency string `json:"currency"`
	} `json:"meta"`
	Values  []twelveDataValue `json:"values"`
	Status  string            `json:"status"`
	Code    int               `json:"code"`
	Message string            `json:"message"`
}

type twelveDataValue struct {
	Datetime string `json:"datetime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume"`
}

func (v twelveDataValue) candle() (models.StockCandle, error) {
	timestamp, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
	if err != nil {
		timestamp, err = time.Parse("2006-01-02", v.Datetime)
	}
	if err != nil {
		return models.StockCandle{}, fmt.Errorf("parse candle timestamp: %w", err)
	}
	values := []string{v.Open, v.High, v.Low, v.Close, v.Volume}
	parsed := make([]float64, len(values))
	for i, raw := range values {
		parsed[i], err = strconv.ParseFloat(raw, 64)
		if err != nil {
			return models.StockCandle{}, fmt.Errorf("parse candle number: %w", err)
		}
	}
	return models.StockCandle{Timestamp: timestamp.UTC(), Open: parsed[0], High: parsed[1], Low: parsed[2], Close: parsed[3], Volume: parsed[4]}, nil
}
