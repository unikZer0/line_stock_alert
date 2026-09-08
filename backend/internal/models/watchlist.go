package models

import "time"

type AddWatchlistRequest struct {
	Symbol string `json:"symbol"`
}

type StockMetadata struct {
	Symbol   string
	Name     string
	Exchange string
	Currency string
}

type StockQuote struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	PreviousClose float64   `json:"previous_close"`
	Currency      string    `json:"currency"`
	UpdatedAt     time.Time `json:"updated_at"`
	Cached        bool      `json:"cached"`
}

type WatchlistItem struct {
	ID        string      `json:"id"`
	Symbol    string      `json:"symbol"`
	Name      string      `json:"name"`
	Exchange  string      `json:"exchange"`
	Currency  string      `json:"currency"`
	Quote     *StockQuote `json:"quote,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}
