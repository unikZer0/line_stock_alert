package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"stock_linebot/backend/internal/models"
)

var (
	ErrAdminAlertNotFound         = errors.New("admin alert not found")
	ErrAdminAlertActionNotAllowed = errors.New("admin alert action not allowed")
)

type AdminAlertRepository struct{ db *pgxpool.Pool }

func NewAdminAlertRepository(db *pgxpool.Pool) *AdminAlertRepository {
	return &AdminAlertRepository{db: db}
}

func (r *AdminAlertRepository) List(ctx context.Context, filter models.AdminAlertFilter) ([]models.AdminAlert, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM alerts a JOIN stocks s ON s.id=a.stock_id
		WHERE ($1='' OR s.symbol=$1) AND ($2='' OR a.status=$2) AND ($3='' OR a.user_id::text=$3)`,
		filter.Symbol, filter.Status, filter.UserID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin alerts: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text,s.symbol,a.condition,a.target_price,a.status,a.triggered_at,a.disabled_reason,
		       a.created_at,a.updated_at,a.user_id::text,u.email::text
		FROM alerts a JOIN stocks s ON s.id=a.stock_id JOIN users u ON u.id=a.user_id
		WHERE ($1='' OR s.symbol=$1) AND ($2='' OR a.status=$2) AND ($3='' OR a.user_id::text=$3)
		ORDER BY a.created_at DESC LIMIT $4 OFFSET $5
	`, filter.Symbol, filter.Status, filter.UserID, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin alerts: %w", err)
	}
	defer rows.Close()
	alerts := make([]models.AdminAlert, 0)
	for rows.Next() {
		var alert models.AdminAlert
		if err = scanAdminAlert(rows, &alert); err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, alert)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("read admin alerts: %w", err)
	}
	return alerts, total, nil
}

func (r *AdminAlertRepository) Get(ctx context.Context, alertID string) (models.AdminAlert, error) {
	var alert models.AdminAlert
	err := scanAdminAlert(r.db.QueryRow(ctx, `
		SELECT a.id::text,s.symbol,a.condition,a.target_price,a.status,a.triggered_at,a.disabled_reason,
		       a.created_at,a.updated_at,a.user_id::text,u.email::text
		FROM alerts a JOIN stocks s ON s.id=a.stock_id JOIN users u ON u.id=a.user_id WHERE a.id::text=$1
	`, alertID), &alert)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.AdminAlert{}, ErrAdminAlertNotFound
	}
	if err != nil {
		return models.AdminAlert{}, fmt.Errorf("get admin alert: %w", err)
	}
	return alert, nil
}

type rowScanner interface{ Scan(...any) error }

func scanAdminAlert(row rowScanner, alert *models.AdminAlert) error {
	return row.Scan(&alert.ID, &alert.Symbol, &alert.Condition, &alert.TargetPrice, &alert.Status,
		&alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt, &alert.UserID, &alert.UserEmail)
}

func (r *AdminAlertRepository) Disable(ctx context.Context, action models.AdminActionContext, alertID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin disable admin alert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current string
	err = tx.QueryRow(ctx, `SELECT status FROM alerts WHERE id::text=$1 FOR UPDATE`, alertID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdminAlertNotFound
	}
	if err != nil {
		return fmt.Errorf("find admin alert: %w", err)
	}
	if current == "DISABLED" {
		return ErrAdminAlertActionNotAllowed
	}
	_, err = tx.Exec(ctx, `UPDATE alerts SET status='DISABLED',triggered_at=NULL,disabled_reason=$2,disabled_at=NOW() WHERE id::text=$1`, alertID, reason)
	if err != nil {
		return fmt.Errorf("disable admin alert: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit_logs(admin_user_id,action,target_type,target_id,ip_address,user_agent,details)
		VALUES($1,'ALERT_DISABLED','ALERT',$2,NULLIF($3,'')::inet,$4,jsonb_build_object('previous_status',$5,'reason',$6))`,
		action.AdminUserID, alertID, action.IPAddress, action.UserAgent, current, reason)
	if err != nil {
		return fmt.Errorf("audit disabled alert: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit disabled alert: %w", err)
	}
	return nil
}
