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
	ErrAdminStockNotFound         = errors.New("admin stock not found")
	ErrAdminStockActionNotAllowed = errors.New("admin stock action not allowed")
)

type AdminStockRepository struct{ db *pgxpool.Pool }

func NewAdminStockRepository(db *pgxpool.Pool) *AdminStockRepository {
	return &AdminStockRepository{db: db}
}

func (r *AdminStockRepository) List(ctx context.Context, filter models.AdminStockFilter) ([]models.AdminStock, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM stocks WHERE ($1='' OR symbol ILIKE '%'||$1||'%' OR COALESCE(name,'') ILIKE '%'||$1||'%') AND ($2='' OR status=$2)`, filter.Search, filter.Status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin stocks: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT s.symbol,COALESCE(s.name,''),s.provider,COALESCE(s.exchange,''),COALESCE(s.currency,'USD'),s.status,
		 s.disabled_reason,s.disabled_at,
		 (SELECT COUNT(*) FROM alerts a WHERE a.stock_id=s.id),
		 (SELECT COUNT(*) FROM alerts a WHERE a.stock_id=s.id AND a.status='ACTIVE'),
		 s.last_quote_price,s.last_quote_at,s.created_at,s.updated_at
		FROM stocks s WHERE ($1='' OR s.symbol ILIKE '%'||$1||'%' OR COALESCE(s.name,'') ILIKE '%'||$1||'%') AND ($2='' OR s.status=$2)
		ORDER BY s.symbol LIMIT $3 OFFSET $4`, filter.Search, filter.Status, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin stocks: %w", err)
	}
	defer rows.Close()
	stocks := make([]models.AdminStock, 0)
	for rows.Next() {
		var stock models.AdminStock
		if err = scanAdminStock(rows, &stock); err != nil {
			return nil, 0, err
		}
		stocks = append(stocks, stock)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("read admin stocks: %w", err)
	}
	return stocks, total, nil
}

func (r *AdminStockRepository) Get(ctx context.Context, symbol string) (models.AdminStock, error) {
	var stock models.AdminStock
	err := scanAdminStock(r.db.QueryRow(ctx, `
		SELECT s.symbol,COALESCE(s.name,''),s.provider,COALESCE(s.exchange,''),COALESCE(s.currency,'USD'),s.status,
		 s.disabled_reason,s.disabled_at,
		 (SELECT COUNT(*) FROM alerts a WHERE a.stock_id=s.id),
		 (SELECT COUNT(*) FROM alerts a WHERE a.stock_id=s.id AND a.status='ACTIVE'),
		 s.last_quote_price,s.last_quote_at,s.created_at,s.updated_at
		FROM stocks s WHERE s.symbol=$1`, symbol), &stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.AdminStock{}, ErrAdminStockNotFound
	}
	if err != nil {
		return models.AdminStock{}, fmt.Errorf("get admin stock: %w", err)
	}
	return stock, nil
}

func scanAdminStock(row rowScanner, stock *models.AdminStock) error {
	return row.Scan(&stock.Symbol, &stock.Name, &stock.Provider, &stock.Exchange, &stock.Currency, &stock.Status,
		&stock.DisabledReason, &stock.DisabledAt, &stock.Alerts, &stock.ActiveAlerts, &stock.LastQuotePrice, &stock.LastQuoteAt, &stock.CreatedAt, &stock.UpdatedAt)
}

func (r *AdminStockRepository) SetStatus(ctx context.Context, action models.AdminActionContext, symbol, status, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin admin stock status: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var stockID, current string
	err = tx.QueryRow(ctx, `SELECT id::text,status FROM stocks WHERE symbol=$1 FOR UPDATE`, symbol).Scan(&stockID, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdminStockNotFound
	}
	if err != nil {
		return fmt.Errorf("find admin stock: %w", err)
	}
	if current == status {
		return ErrAdminStockActionNotAllowed
	}
	affectedAlerts := int64(0)
	if status == "DISABLED" {
		_, err = tx.Exec(ctx, `UPDATE stocks SET status='DISABLED',disabled_reason=$2,disabled_at=NOW() WHERE id=$1`, stockID, reason)
		if err == nil {
			var result pgconn.CommandTag
			result, err = tx.Exec(ctx, `UPDATE alerts SET status='DISABLED',triggered_at=NULL,disabled_reason=$2,disabled_at=NOW() WHERE stock_id=$1 AND status='ACTIVE'`, stockID, "Stock disabled: "+reason)
			affectedAlerts = result.RowsAffected()
		}
	} else {
		_, err = tx.Exec(ctx, `UPDATE stocks SET status='ACTIVE',disabled_reason=NULL,disabled_at=NULL WHERE id=$1`, stockID)
	}
	if err != nil {
		return fmt.Errorf("update admin stock status: %w", err)
	}
	auditAction := "STOCK_ENABLED"
	if status == "DISABLED" {
		auditAction = "STOCK_DISABLED"
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit_logs(admin_user_id,action,target_type,target_id,ip_address,user_agent,details)
	 VALUES($1,$2,'STOCK',$3,NULLIF($4,'')::inet,$5,jsonb_build_object('previous_status',$6,'new_status',$7,'reason',$8,'affected_alerts',$9))`,
		action.AdminUserID, auditAction, symbol, action.IPAddress, action.UserAgent, current, status, reason, affectedAlerts)
	if err != nil {
		return fmt.Errorf("audit admin stock status: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit admin stock status: %w", err)
	}
	return nil
}
