package services

import (
	"context"
	"errors"
	"testing"

	"stock_linebot/backend/internal/models"
)

type rearmAlertStore struct {
	AlertStore
	alert       models.Alert
	activeCount int
	rearmed     bool
}

func (s *rearmAlertStore) Get(context.Context, string, string) (models.Alert, error) {
	return s.alert, nil
}

func (s *rearmAlertStore) CountActive(context.Context, string) (int, error) {
	return s.activeCount, nil
}

func (s *rearmAlertStore) Rearm(context.Context, string, string) (models.Alert, error) {
	s.rearmed = true
	s.alert.Status = "ACTIVE"
	s.alert.TriggeredAt = nil
	return s.alert, nil
}

type rearmStockProvider struct {
	StockProvider
	price float64
}

func (p rearmStockProvider) Quote(context.Context, string) (models.StockQuote, error) {
	return models.StockQuote{Symbol: "NVDA", Price: p.price}, nil
}

func TestRearmTriggeredAlertAfterPriceMovesAway(t *testing.T) {
	store := &rearmAlertStore{alert: models.Alert{
		ID: "alert-1", Symbol: "NVDA", Condition: "ABOVE", TargetPrice: 230, Status: "TRIGGERED",
	}}
	service := NewAlertService(store, rearmStockProvider{price: 225}, 100)

	alert, err := service.Rearm(context.Background(), "user-1", "alert-1")
	if err != nil {
		t.Fatalf("Rearm returned error: %v", err)
	}
	if !store.rearmed || alert.Status != "ACTIVE" {
		t.Fatalf("alert was not re-armed: %#v", alert)
	}
}

func TestRearmRejectsConditionStillMet(t *testing.T) {
	store := &rearmAlertStore{alert: models.Alert{
		ID: "alert-1", Symbol: "NVDA", Condition: "ABOVE", TargetPrice: 230, Status: "TRIGGERED",
	}}
	service := NewAlertService(store, rearmStockProvider{price: 233}, 100)

	_, err := service.Rearm(context.Background(), "user-1", "alert-1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "ALERT_CONDITION_STILL_MET" {
		t.Fatalf("Rearm error = %#v, want ALERT_CONDITION_STILL_MET", err)
	}
	if store.rearmed {
		t.Fatal("alert was re-armed while its condition was still met")
	}
}

func TestRearmRejectsActiveAlert(t *testing.T) {
	store := &rearmAlertStore{alert: models.Alert{
		ID: "alert-1", Symbol: "NVDA", Condition: "ABOVE", TargetPrice: 230, Status: "ACTIVE",
	}}
	service := NewAlertService(store, rearmStockProvider{price: 225}, 100)

	_, err := service.Rearm(context.Background(), "user-1", "alert-1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "ALERT_NOT_TRIGGERED" {
		t.Fatalf("Rearm error = %#v, want ALERT_NOT_TRIGGERED", err)
	}
}
