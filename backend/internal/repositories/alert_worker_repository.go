package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"stock_linebot/backend/internal/models"
)

type AlertWorkerRepository struct{ db *pgxpool.Pool }

func NewAlertWorkerRepository(db *pgxpool.Pool) *AlertWorkerRepository {
	return &AlertWorkerRepository{db: db}
}

func (r *AlertWorkerRepository) ActiveSymbols(ctx context.Context, limit, offset int) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT s.symbol FROM alerts a JOIN stocks s ON s.id=a.stock_id
	 WHERE a.status='ACTIVE' AND s.status='ACTIVE' ORDER BY s.symbol LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list active alert symbols: %w", err)
	}
	defer rows.Close()
	symbols := make([]string, 0)
	for rows.Next() {
		var symbol string
		if err = rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, rows.Err()
}
func (r *AlertWorkerRepository) ActiveAlerts(ctx context.Context, symbol string) ([]models.ActiveAlertJob, error) {
	rows, err := r.db.Query(ctx, `SELECT a.id::text,a.user_id::text,ui.provider_user_id,s.symbol,a.condition,a.target_price
	 FROM alerts a JOIN stocks s ON s.id=a.stock_id JOIN users u ON u.id=a.user_id
	 LEFT JOIN user_identities ui ON ui.user_id=a.user_id AND ui.provider='LINE'
	 WHERE s.symbol=$1 AND s.status='ACTIVE' AND a.status='ACTIVE' AND u.status='ACTIVE'`, symbol)
	if err != nil {
		return nil, fmt.Errorf("list active alerts for symbol: %w", err)
	}
	defer rows.Close()
	alerts := make([]models.ActiveAlertJob, 0)
	for rows.Next() {
		var alert models.ActiveAlertJob
		if err = rows.Scan(&alert.ID, &alert.UserID, &alert.LineUserID, &alert.Symbol, &alert.Condition, &alert.TargetPrice); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}
func (r *AlertWorkerRepository) ClaimTriggered(ctx context.Context, alertID string, triggeredAt time.Time) (bool, error) {
	result, err := r.db.Exec(ctx, `UPDATE alerts SET status='TRIGGERED',triggered_at=$2,disabled_reason=NULL,disabled_at=NULL WHERE id::text=$1 AND status='ACTIVE'`, alertID, triggeredAt)
	if err != nil {
		return false, fmt.Errorf("claim triggered alert: %w", err)
	}
	return result.RowsAffected() == 1, nil
}
func (r *AlertWorkerRepository) UpdateStockQuote(ctx context.Context, symbol string, price float64, at time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE stocks SET last_quote_price=$2,last_quote_at=$3 WHERE symbol=$1`, symbol, price, at)
	if err != nil {
		return fmt.Errorf("update stock quote: %w", err)
	}
	return nil
}
func (r *AlertWorkerRepository) RecordLineDelivery(ctx context.Context, alert models.ActiveAlertJob, status, message string, price float64, deliveryErr error) error {
	var errorMessage *string
	var sentAt *time.Time
	if deliveryErr != nil {
		value := deliveryErr.Error()
		errorMessage = &value
	} else {
		now := time.Now().UTC()
		sentAt = &now
	}
	_, err := r.db.Exec(ctx, `INSERT INTO alert_logs(alert_id,user_id,channel,status,message,triggered_price,sent_at,error_message)
	 VALUES($1,$2,'LINE',$3,$4,$5,$6,$7)`, alert.ID, alert.UserID, status, message, price, sentAt, errorMessage)
	if err != nil {
		return fmt.Errorf("record LINE alert delivery: %w", err)
	}
	return nil
}
func (r *AlertWorkerRepository) LogError(ctx context.Context, code, message string, details map[string]any) error {
	_, err := r.db.Exec(ctx, `INSERT INTO application_logs(level,service,code,message,details) VALUES('ERROR','alert-worker',$1,$2,$3)`, code, message, details)
	return err
}
