package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"stock_linebot/backend/internal/models"
)

var (
	ErrAlertAlreadyExists = errors.New("active alert already exists")
	ErrAlertNotFound      = errors.New("alert not found")
	ErrAlertNotTriggered  = errors.New("alert is not triggered")
	ErrStockDisabled      = errors.New("stock is administratively disabled")
)

type AlertRepository struct{ db *pgxpool.Pool }

func NewAlertRepository(db *pgxpool.Pool) *AlertRepository { return &AlertRepository{db: db} }

func (r *AlertRepository) List(ctx context.Context, userID string, filter models.AlertFilter) ([]models.Alert, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, s.symbol, a.condition, a.target_price, a.status,
		       a.triggered_at, a.disabled_reason, a.created_at, a.updated_at
		FROM alerts a JOIN stocks s ON s.id = a.stock_id
		WHERE a.user_id = $1 AND ($2 = '' OR s.symbol = $2) AND ($3 = '' OR a.status = $3)
		ORDER BY a.created_at DESC
	`, userID, filter.Symbol, filter.Status)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()
	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var alert models.Alert
		if err = rows.Scan(&alert.ID, &alert.Symbol, &alert.Condition, &alert.TargetPrice, &alert.Status,
			&alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read alerts: %w", err)
	}
	return alerts, nil
}

func (r *AlertRepository) CountActive(ctx context.Context, userID string) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE user_id = $1 AND status = 'ACTIVE'`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active alerts: %w", err)
	}
	return count, nil
}

func (r *AlertRepository) Get(ctx context.Context, userID, alertID string) (models.Alert, error) {
	var alert models.Alert
	err := r.db.QueryRow(ctx, `
		SELECT a.id::text, s.symbol, a.condition, a.target_price, a.status,
		       a.triggered_at, a.disabled_reason, a.created_at, a.updated_at
		FROM alerts a JOIN stocks s ON s.id = a.stock_id
		WHERE a.id::text = $1 AND a.user_id = $2
	`, alertID, userID).Scan(&alert.ID, &alert.Symbol, &alert.Condition, &alert.TargetPrice,
		&alert.Status, &alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Alert{}, ErrAlertNotFound
	}
	if err != nil {
		return models.Alert{}, fmt.Errorf("get alert: %w", err)
	}
	return alert, nil
}

func (r *AlertRepository) Create(ctx context.Context, userID string, stock models.StockMetadata, condition string, targetPrice float64) (models.Alert, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Alert{}, fmt.Errorf("begin create alert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var stockID, stockStatus string
	err = tx.QueryRow(ctx, `
		INSERT INTO stocks (symbol, provider, provider_symbol, name, exchange, currency)
		VALUES ($1, 'FINNHUB', $1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''))
		ON CONFLICT (provider, symbol) DO UPDATE SET name = COALESCE(EXCLUDED.name, stocks.name),
		  exchange = COALESCE(EXCLUDED.exchange, stocks.exchange), currency = COALESCE(EXCLUDED.currency, stocks.currency)
		RETURNING id::text, status
	`, stock.Symbol, stock.Name, stock.Exchange, stock.Currency).Scan(&stockID, &stockStatus)
	if err != nil {
		return models.Alert{}, fmt.Errorf("upsert alert stock: %w", err)
	}
	if stockStatus != "ACTIVE" {
		return models.Alert{}, ErrStockDisabled
	}
	var alert models.Alert
	err = tx.QueryRow(ctx, `
		INSERT INTO alerts (user_id, stock_id, condition, target_price) VALUES ($1, $2, $3, $4)
		RETURNING id::text, condition, target_price, status, triggered_at, disabled_reason, created_at, updated_at
	`, userID, stockID, condition, targetPrice).Scan(&alert.ID, &alert.Condition, &alert.TargetPrice,
		&alert.Status, &alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt)
	if isAlertUniqueViolation(err) {
		return models.Alert{}, ErrAlertAlreadyExists
	}
	if err != nil {
		return models.Alert{}, fmt.Errorf("insert alert: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Alert{}, fmt.Errorf("commit alert: %w", err)
	}
	alert.Symbol = stock.Symbol
	return alert, nil
}

func (r *AlertRepository) Update(ctx context.Context, userID, alertID, condition string, targetPrice float64) (models.Alert, error) {
	var alert models.Alert
	err := r.db.QueryRow(ctx, `
		UPDATE alerts a SET condition = $3, target_price = $4
		FROM stocks s WHERE a.id::text = $1 AND a.user_id = $2 AND a.stock_id = s.id
		RETURNING a.id::text, s.symbol, a.condition, a.target_price, a.status,
		          a.triggered_at, a.disabled_reason, a.created_at, a.updated_at
	`, alertID, userID, condition, targetPrice).Scan(&alert.ID, &alert.Symbol, &alert.Condition,
		&alert.TargetPrice, &alert.Status, &alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt)
	if isAlertUniqueViolation(err) {
		return models.Alert{}, ErrAlertAlreadyExists
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Alert{}, ErrAlertNotFound
		}
		return models.Alert{}, fmt.Errorf("update alert: %w", err)
	}
	return alert, nil
}

func (r *AlertRepository) Delete(ctx context.Context, userID, alertID string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM alerts WHERE id::text = $1 AND user_id = $2`, alertID, userID)
	if err != nil {
		return fmt.Errorf("delete alert: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

func (r *AlertRepository) Rearm(ctx context.Context, userID, alertID string) (models.Alert, error) {
	var alert models.Alert
	err := r.db.QueryRow(ctx, `
		UPDATE alerts a
		SET status = 'ACTIVE', triggered_at = NULL, disabled_reason = NULL, disabled_at = NULL
		FROM stocks s
		WHERE a.id::text = $1 AND a.user_id = $2 AND a.status = 'TRIGGERED' AND a.stock_id = s.id
		RETURNING a.id::text, s.symbol, a.condition, a.target_price, a.status,
		          a.triggered_at, a.disabled_reason, a.created_at, a.updated_at
	`, alertID, userID).Scan(&alert.ID, &alert.Symbol, &alert.Condition, &alert.TargetPrice,
		&alert.Status, &alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt)
	if isAlertUniqueViolation(err) {
		return models.Alert{}, ErrAlertAlreadyExists
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Alert{}, ErrAlertNotTriggered
	}
	if err != nil {
		return models.Alert{}, fmt.Errorf("re-arm alert: %w", err)
	}
	return alert, nil
}

func isAlertUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
