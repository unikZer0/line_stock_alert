package models

type StockPage struct {
	Stocks []StockMetadata
	Page   int
	Limit  int
	Total  int
}

type StockQuoteDetail struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Open          float64 `json:"open"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	PreviousClose float64 `json:"previous_close"`
	Currency      string  `json:"currency"`
	MarketStatus  string  `json:"market_status"`
	UpdatedAt     string  `json:"updated_at"`
	Cached        bool    `json:"cached"`
}

type QuoteError struct {
	Symbol  string `json:"symbol"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MultipleQuotesResult struct {
	Quotes []StockQuote `json:"quotes"`
	Errors []QuoteError `json:"errors"`
}
