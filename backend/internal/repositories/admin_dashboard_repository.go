package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"stock_linebot/backend/internal/models"
)

type AdminDashboardRepository struct{ db *pgxpool.Pool }

func NewAdminDashboardRepository(db *pgxpool.Pool) *AdminDashboardRepository {
	return &AdminDashboardRepository{db: db}
}

func (r *AdminDashboardRepository) GetDashboard(ctx context.Context) (models.AdminDashboard, error) {
	var dashboard models.AdminDashboard
	err := r.db.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM users),
		  (SELECT COUNT(*) FROM users WHERE status = 'ACTIVE'),
		  (SELECT COUNT(*) FROM users WHERE status = 'DISABLED'),
		  (SELECT COUNT(*) FROM alerts WHERE status = 'ACTIVE'),
		  (SELECT COUNT(*) FROM alerts WHERE status = 'TRIGGERED'),
		  (SELECT COUNT(DISTINCT user_id) FROM user_identities WHERE provider = 'LINE'),
		  (SELECT COUNT(*) FROM alert_logs WHERE channel = 'LINE' AND status = 'FAILED')
	`).Scan(
		&dashboard.Users.Total, &dashboard.Users.Active, &dashboard.Users.Disabled,
		&dashboard.Alerts.Active, &dashboard.Alerts.Triggered,
		&dashboard.Line.ConnectedUsers, &dashboard.Line.FailedMessages,
	)
	if err != nil {
		return models.AdminDashboard{}, fmt.Errorf("get admin dashboard: %w", err)
	}
	return dashboard, nil
}
