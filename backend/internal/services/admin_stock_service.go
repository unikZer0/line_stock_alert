package services

import (
	"context"
	"errors"
	"strings"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type AdminStockStore interface {
	List(context.Context, models.AdminStockFilter) ([]models.AdminStock, int64, error)
	Get(context.Context, string) (models.AdminStock, error)
	SetStatus(context.Context, models.AdminActionContext, string, string, string) error
}
type AdminStockService struct{ store AdminStockStore }

func NewAdminStockService(store AdminStockStore) *AdminStockService {
	return &AdminStockService{store: store}
}

func (s *AdminStockService) List(ctx context.Context, search, status, pageValue, limitValue string) (models.AdminStockPage, error) {
	page, err := positiveInt(pageValue, 1, 1_000_000)
	if err != nil {
		return models.AdminStockPage{}, newError("INVALID_FILTER", "Page must be a positive integer.", err)
	}
	limit, err := positiveInt(limitValue, 20, 100)
	if err != nil {
		return models.AdminStockPage{}, newError("INVALID_FILTER", "Limit must be between 1 and 100.", err)
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != "ACTIVE" && status != "DISABLED" {
		return models.AdminStockPage{}, newError("INVALID_FILTER", "Status must be ACTIVE or DISABLED.", nil)
	}
	stocks, total, err := s.store.List(ctx, models.AdminStockFilter{Search: strings.TrimSpace(search), Status: status, Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return models.AdminStockPage{}, newError("INTERNAL_SERVER_ERROR", "Could not load stocks.", err)
	}
	return models.AdminStockPage{Stocks: stocks, Page: page, Limit: limit, Total: total}, nil
}

func (s *AdminStockService) Get(ctx context.Context, rawSymbol string) (models.AdminStock, error) {
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	if !stockSymbolPattern.MatchString(symbol) {
		return models.AdminStock{}, newError("ADMIN_STOCK_NOT_FOUND", "Stock not found.", nil)
	}
	stock, err := s.store.Get(ctx, symbol)
	if errors.Is(err, repositories.ErrAdminStockNotFound) {
		return models.AdminStock{}, newError("ADMIN_STOCK_NOT_FOUND", "Stock not found.", err)
	}
	if err != nil {
		return models.AdminStock{}, newError("INTERNAL_SERVER_ERROR", "Could not load the stock.", err)
	}
	return stock, nil
}

func (s *AdminStockService) Disable(ctx context.Context, action models.AdminActionContext, rawSymbol, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1000 {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "A disable reason between 1 and 1000 characters is required.", nil)
	}
	return s.setStatus(ctx, action, rawSymbol, "DISABLED", reason)
}
func (s *AdminStockService) Enable(ctx context.Context, action models.AdminActionContext, rawSymbol string) error {
	return s.setStatus(ctx, action, rawSymbol, "ACTIVE", "")
}
func (s *AdminStockService) setStatus(ctx context.Context, action models.AdminActionContext, rawSymbol, status, reason string) error {
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	if !stockSymbolPattern.MatchString(symbol) {
		return newError("ADMIN_STOCK_NOT_FOUND", "Stock not found.", nil)
	}
	err := s.store.SetStatus(ctx, action, symbol, status, reason)
	if errors.Is(err, repositories.ErrAdminStockNotFound) {
		return newError("ADMIN_STOCK_NOT_FOUND", "Stock not found.", err)
	}
	if errors.Is(err, repositories.ErrAdminStockActionNotAllowed) {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "The stock already has the requested status.", err)
	}
	if err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Could not update the stock.", err)
	}
	return nil
}
