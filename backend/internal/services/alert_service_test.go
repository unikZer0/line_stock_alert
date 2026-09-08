package services

import (
	"context"
	"errors"
	"testing"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type alertStoreStub struct {
	alerts                          []models.Alert
	count                           int
	createdCondition                string
	createdPrice                    float64
	createErr, updateErr, deleteErr error
}

func (s *alertStoreStub) List(context.Context, string, models.AlertFilter) ([]models.Alert, error) {
	return s.alerts, nil
}
func (s *alertStoreStub) CountActive(context.Context, string) (int, error) { return s.count, nil }
func (s *alertStoreStub) Create(_ context.Context, _ string, stock models.StockMetadata, condition string, price float64) (models.Alert, error) {
	s.createdCondition, s.createdPrice = condition, price
	return models.Alert{ID: "alert-1", Symbol: stock.Symbol, Condition: condition, TargetPrice: price, Status: "ACTIVE"}, s.createErr
}
func (s *alertStoreStub) Update(_ context.Context, _, id, condition string, price float64) (models.Alert, error) {
	return models.Alert{ID: id, Condition: condition, TargetPrice: price}, s.updateErr
}
func (s *alertStoreStub) Delete(context.Context, string, string) error { return s.deleteErr }

func TestAlertCreateNormalizesAndCreatesIndependentTarget(t *testing.T) {
	store := &alertStoreStub{}
	service := NewAlertService(store, &stockProviderStub{}, 100)
	alert, err := service.Create(context.Background(), "user-1", models.CreateAlertRequest{Symbol: " nvda ", Condition: " below ", TargetPrice: 190})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if alert.Symbol != "NVDA" || store.createdCondition != "BELOW" || store.createdPrice != 190 {
		t.Fatalf("alert = %#v", alert)
	}
}

func TestAlertCreateValidatesConditionAndPrice(t *testing.T) {
	service := NewAlertService(&alertStoreStub{}, &stockProviderStub{}, 100)
	tests := []struct {
		request models.CreateAlertRequest
		code    string
	}{
		{models.CreateAlertRequest{Symbol: "AAPL", Condition: "EQUAL", TargetPrice: 10}, "INVALID_ALERT_CONDITION"},
		{models.CreateAlertRequest{Symbol: "AAPL", Condition: "ABOVE", TargetPrice: 0}, "INVALID_TARGET_PRICE"},
	}
	for _, test := range tests {
		_, err := service.Create(context.Background(), "user-1", test.request)
		var appErr *Error
		if !errors.As(err, &appErr) || appErr.Code != test.code {
			t.Fatalf("error = %#v, want %s", err, test.code)
		}
	}
}

func TestAlertCreateEnforcesActiveLimit(t *testing.T) {
	service := NewAlertService(&alertStoreStub{count: 3}, &stockProviderStub{}, 3)
	_, err := service.Create(context.Background(), "user-1", models.CreateAlertRequest{Symbol: "AAPL", Condition: "ABOVE", TargetPrice: 200})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "ALERT_LIMIT_REACHED" {
		t.Fatalf("error = %#v", err)
	}
}

func TestAlertUpdateMapsDuplicate(t *testing.T) {
	service := NewAlertService(&alertStoreStub{updateErr: repositories.ErrAlertAlreadyExists}, &stockProviderStub{}, 100)
	_, err := service.Update(context.Background(), "user-1", "alert-1", models.UpdateAlertRequest{Condition: "ABOVE", TargetPrice: 200})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "ALERT_ALREADY_EXISTS" {
		t.Fatalf("error = %#v", err)
	}
}

func TestAlertListRejectsInvalidStatus(t *testing.T) {
	service := NewAlertService(&alertStoreStub{}, &stockProviderStub{}, 100)
	_, err := service.List(context.Background(), "user-1", "", "PENDING")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "INVALID_FILTER" {
		t.Fatalf("error = %#v", err)
	}
}
