package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"stock_linebot/backend/internal/models"
)

type alertWorkerStoreStub struct {
	mu              sync.Mutex
	symbols         []string
	alerts          []models.ActiveAlertJob
	claimed, logged bool
	deliveryStatus  string
}

func (s *alertWorkerStoreStub) ActiveSymbols(context.Context, int, int) ([]string, error) {
	return s.symbols, nil
}
func (s *alertWorkerStoreStub) ActiveAlerts(context.Context, string) ([]models.ActiveAlertJob, error) {
	return s.alerts, nil
}
func (s *alertWorkerStoreStub) ClaimTriggered(context.Context, string, time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claimed = true
	return true, nil
}
func (s *alertWorkerStoreStub) UpdateStockQuote(context.Context, string, float64, time.Time) error {
	return nil
}
func (s *alertWorkerStoreStub) RecordLineDelivery(_ context.Context, _ models.ActiveAlertJob, status, _ string, _ float64, _ error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deliveryStatus = status
	return nil
}
func (s *alertWorkerStoreStub) LogError(context.Context, string, string, map[string]any) error {
	s.logged = true
	return nil
}

type alertWorkerProviderStub struct {
	quote models.StockQuote
	err   error
}

func (p alertWorkerProviderStub) SearchUSStocks(context.Context) ([]models.StockMetadata, error) {
	return nil, nil
}
func (p alertWorkerProviderStub) ValidateUSStock(context.Context, string) (models.StockMetadata, error) {
	return models.StockMetadata{}, nil
}
func (p alertWorkerProviderStub) Quote(context.Context, string) (models.StockQuote, error) {
	return p.quote, p.err
}

type alertWorkerCacheStub struct{}

func (alertWorkerCacheStub) Get(context.Context, string) (models.StockQuote, bool, error) {
	return models.StockQuote{}, false, nil
}
func (alertWorkerCacheStub) Set(context.Context, models.StockQuote, time.Duration) error { return nil }

type alertWorkerPusherStub struct {
	err  error
	sent bool
}

func (p *alertWorkerPusherStub) PushText(context.Context, string, string) error {
	p.sent = true
	return p.err
}

func TestAlertWorkerTriggersAboveAndSendsLine(t *testing.T) {
	lineID := "line-1"
	store := &alertWorkerStoreStub{symbols: []string{"NVDA"}, alerts: []models.ActiveAlertJob{{ID: "a1", UserID: "u1", LineUserID: &lineID, Symbol: "NVDA", Condition: "ABOVE", TargetPrice: 232}}}
	pusher := &alertWorkerPusherStub{}
	worker := NewAlertWorker(store, alertWorkerProviderStub{quote: models.StockQuote{Symbol: "NVDA", Price: 233, UpdatedAt: time.Now()}}, alertWorkerCacheStub{}, pusher, AlertWorkerConfig{Interval: time.Second, QuoteTTL: time.Second, BatchSize: 100, Concurrency: 1})
	worker.runCycle(context.Background())
	if !store.claimed || !pusher.sent || store.deliveryStatus != "SENT" {
		t.Fatalf("store=%#v pusher=%#v", store, pusher)
	}
}
func TestAlertWorkerDoesNotTriggerUnmatchedAlert(t *testing.T) {
	store := &alertWorkerStoreStub{symbols: []string{"NVDA"}, alerts: []models.ActiveAlertJob{{ID: "a1", Symbol: "NVDA", Condition: "ABOVE", TargetPrice: 250}}}
	worker := NewAlertWorker(store, alertWorkerProviderStub{quote: models.StockQuote{Symbol: "NVDA", Price: 233, UpdatedAt: time.Now()}}, alertWorkerCacheStub{}, &alertWorkerPusherStub{}, AlertWorkerConfig{Interval: time.Second, QuoteTTL: time.Second, BatchSize: 100, Concurrency: 1})
	worker.runCycle(context.Background())
	if store.claimed {
		t.Fatal("unmatched alert was triggered")
	}
}
func TestAlertWorkerRecordsFailedPush(t *testing.T) {
	lineID := "line-1"
	store := &alertWorkerStoreStub{symbols: []string{"NVDA"}, alerts: []models.ActiveAlertJob{{ID: "a1", UserID: "u1", LineUserID: &lineID, Symbol: "NVDA", Condition: "BELOW", TargetPrice: 240}}}
	worker := NewAlertWorker(store, alertWorkerProviderStub{quote: models.StockQuote{Symbol: "NVDA", Price: 233, UpdatedAt: time.Now()}}, alertWorkerCacheStub{}, &alertWorkerPusherStub{err: errors.New("LINE down")}, AlertWorkerConfig{Interval: time.Second, QuoteTTL: time.Second, BatchSize: 100, Concurrency: 1})
	worker.runCycle(context.Background())
	if store.deliveryStatus != "FAILED" {
		t.Fatalf("status=%s", store.deliveryStatus)
	}
}
