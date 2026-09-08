package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"stock_linebot/backend/internal/models"
)

type AlertWorkerStore interface {
	ActiveSymbols(context.Context, int, int) ([]string, error)
	ActiveAlerts(context.Context, string) ([]models.ActiveAlertJob, error)
	ClaimTriggered(context.Context, string, time.Time) (bool, error)
	UpdateStockQuote(context.Context, string, float64, time.Time) error
	RecordLineDelivery(context.Context, models.ActiveAlertJob, string, string, float64, error) error
	LogError(context.Context, string, string, map[string]any) error
}

type AlertWorkerConfig struct {
	Interval, QuoteTTL     time.Duration
	BatchSize, Concurrency int
}
type AlertWorker struct {
	store    AlertWorkerStore
	provider StockProvider
	cache    QuoteCache
	pusher   LinePusher
	cfg      AlertWorkerConfig
	offset   int
}

func NewAlertWorker(store AlertWorkerStore, provider StockProvider, cache QuoteCache, pusher LinePusher, cfg AlertWorkerConfig) *AlertWorker {
	return &AlertWorker{store: store, provider: provider, cache: cache, pusher: pusher, cfg: cfg}
}

func (w *AlertWorker) Run(ctx context.Context) {
	w.runCycle(ctx)
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}
func (w *AlertWorker) runCycle(ctx context.Context) {
	symbols, err := w.store.ActiveSymbols(ctx, w.cfg.BatchSize, w.offset)
	if err != nil {
		w.logError(ctx, "ALERT_SYMBOLS_FAILED", "Unable to load active alert symbols.", map[string]any{"error": err.Error()})
		return
	}
	if len(symbols) == 0 {
		w.offset = 0
		return
	}
	if len(symbols) < w.cfg.BatchSize {
		w.offset = 0
	} else {
		w.offset += len(symbols)
	}
	semaphore := make(chan struct{}, w.cfg.Concurrency)
	var wg sync.WaitGroup
	for _, symbol := range symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			w.processSymbol(ctx, symbol)
		}(symbol)
	}
	wg.Wait()
}
func (w *AlertWorker) processSymbol(ctx context.Context, symbol string) {
	quote, err := w.quote(ctx, symbol)
	if err != nil {
		w.logError(ctx, "STOCK_QUOTE_FAILED", "Unable to fetch an alert quote.", map[string]any{"symbol": symbol, "error": err.Error()})
		return
	}
	if err = w.store.UpdateStockQuote(ctx, symbol, quote.Price, quote.UpdatedAt); err != nil {
		w.logError(ctx, "STOCK_QUOTE_SAVE_FAILED", "Unable to save the latest stock quote.", map[string]any{"symbol": symbol, "error": err.Error()})
	}
	alerts, err := w.store.ActiveAlerts(ctx, symbol)
	if err != nil {
		w.logError(ctx, "ALERT_LOAD_FAILED", "Unable to load alerts for a symbol.", map[string]any{"symbol": symbol, "error": err.Error()})
		return
	}
	for _, alert := range alerts {
		if !alertMatches(alert, quote.Price) {
			continue
		}
		claimed, claimErr := w.store.ClaimTriggered(ctx, alert.ID, quote.UpdatedAt)
		if claimErr != nil {
			w.logError(ctx, "ALERT_CLAIM_FAILED", "Unable to claim a triggered alert.", map[string]any{"alert_id": alert.ID, "error": claimErr.Error()})
			continue
		}
		if !claimed {
			continue
		}
		w.notify(ctx, alert, quote)
	}
}
func (w *AlertWorker) notify(ctx context.Context, alert models.ActiveAlertJob, quote models.StockQuote) {
	message := formatAlertMessage(alert, quote)
	var deliveryErr error
	if alert.LineUserID == nil || *alert.LineUserID == "" {
		deliveryErr = fmt.Errorf("LINE account is not connected")
	} else {
		deliveryErr = w.pusher.PushText(ctx, *alert.LineUserID, message)
	}
	status := "SENT"
	if deliveryErr != nil {
		status = "FAILED"
	}
	if err := w.store.RecordLineDelivery(ctx, alert, status, message, quote.Price, deliveryErr); err != nil {
		w.logError(ctx, "ALERT_LOG_FAILED", "Unable to record LINE delivery.", map[string]any{"alert_id": alert.ID, "error": err.Error()})
	}
}
func (w *AlertWorker) quote(ctx context.Context, symbol string) (models.StockQuote, error) {
	if quote, found, err := w.cache.Get(ctx, symbol); err == nil && found {
		return quote, nil
	}
	quote, err := w.provider.Quote(ctx, symbol)
	if err != nil {
		return models.StockQuote{}, err
	}
	_ = w.cache.Set(ctx, quote, w.cfg.QuoteTTL)
	return quote, nil
}
func (w *AlertWorker) logError(ctx context.Context, code, message string, details map[string]any) {
	log.Printf("alert worker: %s: %v", message, details)
	_ = w.store.LogError(ctx, code, message, details)
}
func alertMatches(alert models.ActiveAlertJob, price float64) bool {
	return alert.Condition == "ABOVE" && price >= alert.TargetPrice || alert.Condition == "BELOW" && price <= alert.TargetPrice
}
func formatAlertMessage(alert models.ActiveAlertJob, quote models.StockQuote) string {
	return fmt.Sprintf("Stock Alert Triggered\n\n%s is now $%.2f\nCondition: %s $%.2f\nPrice updated: %s", alert.Symbol, quote.Price, alert.Condition, alert.TargetPrice, quote.UpdatedAt.UTC().Format(time.RFC3339))
}
