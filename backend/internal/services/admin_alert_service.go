package services

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

var uuidFilterPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type AdminAlertStore interface {
	List(context.Context, models.AdminAlertFilter) ([]models.AdminAlert, int64, error)
	Get(context.Context, string) (models.AdminAlert, error)
	Disable(context.Context, models.AdminActionContext, string, string) error
}

type AdminAlertService struct{ store AdminAlertStore }

func NewAdminAlertService(store AdminAlertStore) *AdminAlertService {
	return &AdminAlertService{store: store}
}

func (s *AdminAlertService) List(ctx context.Context, symbol, status, userID, pageValue, limitValue string) (models.AdminAlertPage, error) {
	page, err := positiveInt(pageValue, 1, 1_000_000)
	if err != nil {
		return models.AdminAlertPage{}, newError("INVALID_FILTER", "Page must be a positive integer.", err)
	}
	limit, err := positiveInt(limitValue, 20, 100)
	if err != nil {
		return models.AdminAlertPage{}, newError("INVALID_FILTER", "Limit must be between 1 and 100.", err)
	}
	symbol, status, userID = strings.ToUpper(strings.TrimSpace(symbol)), strings.ToUpper(strings.TrimSpace(status)), strings.TrimSpace(userID)
	if symbol != "" && !stockSymbolPattern.MatchString(symbol) {
		return models.AdminAlertPage{}, newError("INVALID_FILTER", "The symbol filter is invalid.", nil)
	}
	if status != "" && status != "ACTIVE" && status != "TRIGGERED" && status != "DISABLED" {
		return models.AdminAlertPage{}, newError("INVALID_FILTER", "The status filter is invalid.", nil)
	}
	if userID != "" && !uuidFilterPattern.MatchString(userID) {
		return models.AdminAlertPage{}, newError("INVALID_FILTER", "The user_id filter is invalid.", nil)
	}
	alerts, total, err := s.store.List(ctx, models.AdminAlertFilter{Symbol: symbol, Status: status, UserID: userID, Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return models.AdminAlertPage{}, newError("INTERNAL_SERVER_ERROR", "Could not load alerts.", err)
	}
	return models.AdminAlertPage{Alerts: alerts, Page: page, Limit: limit, Total: total}, nil
}

func (s *AdminAlertService) Get(ctx context.Context, id string) (models.AdminAlert, error) {
	alert, err := s.store.Get(ctx, strings.TrimSpace(id))
	if errors.Is(err, repositories.ErrAdminAlertNotFound) {
		return models.AdminAlert{}, newError("ADMIN_ALERT_NOT_FOUND", "Alert not found.", err)
	}
	if err != nil {
		return models.AdminAlert{}, newError("INTERNAL_SERVER_ERROR", "Could not load the alert.", err)
	}
	return alert, nil
}

func (s *AdminAlertService) Disable(ctx context.Context, action models.AdminActionContext, id, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1000 {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "A disable reason between 1 and 1000 characters is required.", nil)
	}
	err := s.store.Disable(ctx, action, strings.TrimSpace(id), reason)
	if errors.Is(err, repositories.ErrAdminAlertNotFound) {
		return newError("ADMIN_ALERT_NOT_FOUND", "Alert not found.", err)
	}
	if errors.Is(err, repositories.ErrAdminAlertActionNotAllowed) {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "The alert is already disabled.", err)
	}
	if err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Could not disable the alert.", err)
	}
	return nil
}
