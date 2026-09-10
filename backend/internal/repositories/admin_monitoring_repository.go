package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"stock_linebot/backend/internal/models"
)

type AdminMonitoringRepository struct{ db *pgxpool.Pool }

func NewAdminMonitoringRepository(db *pgxpool.Pool) *AdminMonitoringRepository {
	return &AdminMonitoringRepository{db: db}
}

func (r *AdminMonitoringRepository) LineStats(ctx context.Context) (models.AdminLineStats, error) {
	var stats models.AdminLineStats
	err := r.db.QueryRow(ctx, `SELECT
	 (SELECT COUNT(DISTINCT user_id) FROM user_identities WHERE provider='LINE'),
	 (SELECT COUNT(*) FROM alert_logs WHERE channel='LINE' AND status='SENT'),
	 (SELECT COUNT(*) FROM alert_logs WHERE channel='LINE' AND status='FAILED'),
	 (SELECT COUNT(*) FROM line_webhook_events WHERE status='FAILED')`).Scan(&stats.ConnectedUsers, &stats.MessagesSent, &stats.FailedMessages, &stats.DeliveryFailures)
	if err != nil {
		return models.AdminLineStats{}, fmt.Errorf("get admin LINE stats: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT id::text,alert_id::text,user_id::text,status,COALESCE(message,''),triggered_price,
	 provider_message_id,attempted_at,sent_at,error_message FROM alert_logs WHERE channel='LINE' ORDER BY attempted_at DESC LIMIT 20`)
	if err != nil {
		return models.AdminLineStats{}, fmt.Errorf("get recent LINE activity: %w", err)
	}
	defer rows.Close()
	stats.RecentActivity = make([]models.AdminLineMessage, 0)
	for rows.Next() {
		var item models.AdminLineMessage
		if err = scanLineMessage(rows, &item); err != nil {
			return models.AdminLineStats{}, err
		}
		stats.RecentActivity = append(stats.RecentActivity, item)
	}
	if err = rows.Err(); err != nil {
		return models.AdminLineStats{}, fmt.Errorf("read recent LINE activity: %w", err)
	}
	return stats, nil
}

func (r *AdminMonitoringRepository) FailedLineMessages(ctx context.Context, limit, offset int) ([]models.AdminLineMessage, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_logs WHERE channel='LINE' AND status='FAILED'`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id::text,alert_id::text,user_id::text,status,COALESCE(message,''),triggered_price,
	 provider_message_id,attempted_at,sent_at,error_message FROM alert_logs WHERE channel='LINE' AND status='FAILED' ORDER BY attempted_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list failed LINE messages: %w", err)
	}
	defer rows.Close()
	items := make([]models.AdminLineMessage, 0)
	for rows.Next() {
		var item models.AdminLineMessage
		if err = scanLineMessage(rows, &item); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func scanLineMessage(row rowScanner, item *models.AdminLineMessage) error {
	return row.Scan(&item.ID, &item.AlertID, &item.UserID, &item.Status, &item.Message, &item.TriggeredPrice, &item.ProviderMessageID, &item.AttemptedAt, &item.SentAt, &item.ErrorMessage)
}

func (r *AdminMonitoringRepository) ApplicationLogs(ctx context.Context, filter models.ApplicationLogFilter) ([]models.ApplicationLog, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM application_logs WHERE ($1='' OR level=$1) AND ($2='' OR service=$2)`, filter.Level, filter.Service).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count application logs: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT id::text,level,service,code,message,details,request_id,created_at FROM application_logs
	 WHERE ($1='' OR level=$1) AND ($2='' OR service=$2) ORDER BY created_at DESC LIMIT $3 OFFSET $4`, filter.Level, filter.Service, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list application logs: %w", err)
	}
	defer rows.Close()
	logs := make([]models.ApplicationLog, 0)
	for rows.Next() {
		var item models.ApplicationLog
		if err = rows.Scan(&item.ID, &item.Level, &item.Service, &item.Code, &item.Message, &item.Details, &item.RequestID, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, item)
	}
	return logs, total, rows.Err()
}

func (r *AdminMonitoringRepository) AuditLogs(ctx context.Context, limit, offset int) ([]models.AdminAuditLog, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM admin_audit_logs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT id::text,admin_user_id::text,action,target_type,target_id,ip_address::text,user_agent,details,created_at
	 FROM admin_audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()
	logs := make([]models.AdminAuditLog, 0)
	for rows.Next() {
		var item models.AdminAuditLog
		if err = rows.Scan(&item.ID, &item.AdminUserID, &item.Action, &item.TargetType, &item.TargetID, &item.IPAddress, &item.UserAgent, &item.Details, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, item)
	}
	return logs, total, rows.Err()
}
