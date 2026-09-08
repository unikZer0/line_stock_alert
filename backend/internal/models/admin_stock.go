package models

import "time"

type AdminStock struct {
	Symbol         string     `json:"symbol"`
	Name           string     `json:"name"`
	Provider       string     `json:"provider"`
	Exchange       string     `json:"exchange"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	DisabledReason *string    `json:"disabled_reason,omitempty"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
	WatchingUsers  int64      `json:"watching_users"`
	Alerts         int64      `json:"alerts"`
	ActiveAlerts   int64      `json:"active_alerts"`
	LastQuotePrice *float64   `json:"last_quote_price,omitempty"`
	LastQuoteAt    *time.Time `json:"last_quote_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AdminStockFilter struct {
	Search, Status string
	Limit, Offset  int
}
type AdminStockPage struct {
	Stocks      []AdminStock
	Page, Limit int
	Total       int64
}
type DisableStockRequest struct {
	Reason string `json:"reason"`
}
