package services

import (
	"context"

	"stock_linebot/backend/internal/models"
)

type AdminDashboardStore interface {
	GetDashboard(context.Context) (models.AdminDashboard, error)
}

type AdminDashboardService struct{ store AdminDashboardStore }

func NewAdminDashboardService(store AdminDashboardStore) *AdminDashboardService {
	return &AdminDashboardService{store: store}
}

func (s *AdminDashboardService) Dashboard(ctx context.Context) (models.AdminDashboard, error) {
	dashboard, err := s.store.GetDashboard(ctx)
	if err != nil {
		return models.AdminDashboard{}, newError("INTERNAL_SERVER_ERROR", "Could not load the admin dashboard.", err)
	}
	return dashboard, nil
}
