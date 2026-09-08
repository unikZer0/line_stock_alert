package services

import (
	"context"
	"errors"
	"strings"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type AlertStore interface {
	List(context.Context, string, models.AlertFilter) ([]models.Alert, error)
	CountActive(context.Context, string) (int, error)
	Create(context.Context, string, models.StockMetadata, string, float64) (models.Alert, error)
	Update(context.Context, string, string, string, float64) (models.Alert, error)
	Delete(context.Context, string, string) error
}

type AlertService struct {
	store    AlertStore
	provider StockProvider
	limit    int
}

func NewAlertService(store AlertStore, provider StockProvider, limit int) *AlertService {
	return &AlertService{store: store, provider: provider, limit: limit}
}

func (s *AlertService) List(ctx context.Context, userID, symbol, status string) ([]models.Alert, error) {
	symbol, status = strings.ToUpper(strings.TrimSpace(symbol)), strings.ToUpper(strings.TrimSpace(status))
	if symbol != "" && !stockSymbolPattern.MatchString(symbol) {
		return nil, newError("INVALID_FILTER", "The symbol filter is invalid.", nil)
	}
	if status != "" && status != "ACTIVE" && status != "TRIGGERED" && status != "DISABLED" {
		return nil, newError("INVALID_FILTER", "The status filter is invalid.", nil)
	}
	alerts, err := s.store.List(ctx, userID, models.AlertFilter{Symbol: symbol, Status: status})
	if err != nil {
		return nil, newError("INTERNAL_SERVER_ERROR", "Could not load alerts.", err)
	}
	return alerts, nil
}

func (s *AlertService) Create(ctx context.Context, userID string, request models.CreateAlertRequest) (models.Alert, error) {
	symbol := strings.ToUpper(strings.TrimSpace(request.Symbol))
	condition := strings.ToUpper(strings.TrimSpace(request.Condition))
	if !stockSymbolPattern.MatchString(symbol) {
		return models.Alert{}, newError("STOCK_NOT_FOUND", "The US stock symbol was not found.", nil)
	}
	if err := validateAlert(condition, request.TargetPrice); err != nil {
		return models.Alert{}, err
	}
	count, err := s.store.CountActive(ctx, userID)
	if err != nil {
		return models.Alert{}, newError("INTERNAL_SERVER_ERROR", "Could not check the alert limit.", err)
	}
	if count >= s.limit {
		return models.Alert{}, newError("ALERT_LIMIT_REACHED", "The active alert limit has been reached.", nil)
	}
	stock, err := s.provider.ValidateUSStock(ctx, symbol)
	if err != nil {
		return models.Alert{}, mapStockProviderError(err)
	}
	alert, err := s.store.Create(ctx, userID, stock, condition, request.TargetPrice)
	return alert, mapAlertRepositoryError(err, "Could not create the alert.")
}

func (s *AlertService) Update(ctx context.Context, userID, alertID string, request models.UpdateAlertRequest) (models.Alert, error) {
	condition := strings.ToUpper(strings.TrimSpace(request.Condition))
	if err := validateAlert(condition, request.TargetPrice); err != nil {
		return models.Alert{}, err
	}
	alert, err := s.store.Update(ctx, userID, strings.TrimSpace(alertID), condition, request.TargetPrice)
	return alert, mapAlertRepositoryError(err, "Could not update the alert.")
}

func (s *AlertService) Delete(ctx context.Context, userID, alertID string) error {
	err := s.store.Delete(ctx, userID, strings.TrimSpace(alertID))
	return mapAlertRepositoryError(err, "Could not delete the alert.")
}

func validateAlert(condition string, targetPrice float64) error {
	if condition != "ABOVE" && condition != "BELOW" {
		return newError("INVALID_ALERT_CONDITION", "Condition must be ABOVE or BELOW.", nil)
	}
	if targetPrice <= 0 {
		return newError("INVALID_TARGET_PRICE", "Target price must be greater than zero.", nil)
	}
	return nil
}

func mapAlertRepositoryError(err error, fallback string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repositories.ErrAlertAlreadyExists) {
		return newError("ALERT_ALREADY_EXISTS", "An identical active alert already exists.", err)
	}
	if errors.Is(err, repositories.ErrStockDisabled) {
		return newError("STOCK_NOT_FOUND", "This stock is not currently supported.", err)
	}
	if errors.Is(err, repositories.ErrAlertNotFound) {
		return newError("ALERT_NOT_FOUND", "Alert not found.", err)
	}
	return newError("INTERNAL_SERVER_ERROR", fallback, err)
}
