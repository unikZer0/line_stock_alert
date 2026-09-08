package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stock_linebot/backend/internal/models"
)

var (
	ErrStockNotFound          = errors.New("stock not found")
	ErrStockProviderRateLimit = errors.New("stock provider rate limit exceeded")
)

type StockProvider interface {
	ValidateUSStock(context.Context, string) (models.StockMetadata, error)
	Quote(context.Context, string) (models.StockQuote, error)
}

type FinnhubClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewFinnhubClient(client *http.Client, baseURL, apiKey string) *FinnhubClient {
	return &FinnhubClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}

func (c *FinnhubClient) ValidateUSStock(ctx context.Context, symbol string) (models.StockMetadata, error) {
	query := url.Values{"q": {symbol}, "exchange": {"US"}}
	var response struct {
		Result []struct {
			Description   string `json:"description"`
			DisplaySymbol string `json:"displaySymbol"`
			Symbol        string `json:"symbol"`
		} `json:"result"`
	}
	if err := c.get(ctx, "/search?"+query.Encode(), &response); err != nil {
		return models.StockMetadata{}, err
	}
	for _, result := range response.Result {
		if strings.EqualFold(result.Symbol, symbol) || strings.EqualFold(result.DisplaySymbol, symbol) {
			return models.StockMetadata{Symbol: strings.ToUpper(result.Symbol), Name: result.Description, Exchange: "US", Currency: "USD"}, nil
		}
	}
	return models.StockMetadata{}, ErrStockNotFound
}

func (c *FinnhubClient) Quote(ctx context.Context, symbol string) (models.StockQuote, error) {
	var response struct {
		Current, Change, ChangePercent, High, Low, Open, PreviousClose float64
		Timestamp                                                      int64
	}
	var raw struct {
		Current       float64 `json:"c"`
		Change        float64 `json:"d"`
		ChangePercent float64 `json:"dp"`
		High          float64 `json:"h"`
		Low           float64 `json:"l"`
		Open          float64 `json:"o"`
		PreviousClose float64 `json:"pc"`
		Timestamp     int64   `json:"t"`
	}
	if err := c.get(ctx, "/quote?symbol="+url.QueryEscape(symbol), &raw); err != nil {
		return models.StockQuote{}, err
	}
	response.Current, response.Change, response.ChangePercent = raw.Current, raw.Change, raw.ChangePercent
	response.High, response.Low, response.Open, response.PreviousClose, response.Timestamp = raw.High, raw.Low, raw.Open, raw.PreviousClose, raw.Timestamp
	if response.Current <= 0 || response.Timestamp <= 0 {
		return models.StockQuote{}, ErrStockNotFound
	}
	return models.StockQuote{Symbol: symbol, Price: response.Current, Change: response.Change,
		ChangePercent: response.ChangePercent, Open: response.Open, High: response.High, Low: response.Low,
		PreviousClose: response.PreviousClose, Currency: "USD", UpdatedAt: time.Unix(response.Timestamp, 0).UTC()}, nil
}

func (c *FinnhubClient) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create Finnhub request: %w", err)
	}
	req.Header.Set("X-Finnhub-Token", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call Finnhub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrStockProviderRateLimit
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Finnhub returned status %d", resp.StatusCode)
	}
	if err = json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode Finnhub response: %w", err)
	}
	return nil
}
