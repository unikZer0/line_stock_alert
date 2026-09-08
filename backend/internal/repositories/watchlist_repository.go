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
	ErrStockAlreadyWatched = errors.New("stock already in watchlist")
	ErrWatchlistNotFound   = errors.New("watchlist entry not found")
)

type WatchlistRepository struct {
	db *pgxpool.Pool
}

func NewWatchlistRepository(db *pgxpool.Pool) *WatchlistRepository {
	return &WatchlistRepository{db: db}
}

func (r *WatchlistRepository) List(ctx context.Context, userID string) ([]models.WatchlistItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ws.id::text, s.symbol, COALESCE(s.name, ''), COALESCE(s.exchange, ''),
		       COALESCE(s.currency, 'USD'), ws.created_at
		FROM watchlist_stocks ws
		JOIN watchlists w ON w.id = ws.watchlist_id
		JOIN stocks s ON s.id = ws.stock_id
		WHERE w.user_id = $1 AND s.status = 'ACTIVE'
		ORDER BY ws.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list watchlist: %w", err)
	}
	defer rows.Close()
	items := make([]models.WatchlistItem, 0)
	for rows.Next() {
		var item models.WatchlistItem
		if err = rows.Scan(&item.ID, &item.Symbol, &item.Name, &item.Exchange, &item.Currency, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan watchlist: %w", err)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read watchlist: %w", err)
	}
	return items, nil
}

func (r *WatchlistRepository) Count(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM watchlist_stocks ws
		JOIN watchlists w ON w.id = ws.watchlist_id WHERE w.user_id = $1
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count watchlist: %w", err)
	}
	return count, nil
}

func (r *WatchlistRepository) Add(ctx context.Context, userID string, stock models.StockMetadata) (models.WatchlistItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.WatchlistItem{}, fmt.Errorf("begin add watchlist: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var stockID string
	err = tx.QueryRow(ctx, `
		INSERT INTO stocks (symbol, provider, provider_symbol, name, exchange, currency)
		VALUES ($1, 'FINNHUB', $1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''))
		ON CONFLICT (provider, symbol) DO UPDATE SET
		  name = COALESCE(EXCLUDED.name, stocks.name), exchange = COALESCE(EXCLUDED.exchange, stocks.exchange),
		  currency = COALESCE(EXCLUDED.currency, stocks.currency)
		RETURNING id::text
	`, stock.Symbol, stock.Name, stock.Exchange, stock.Currency).Scan(&stockID)
	if err != nil {
		return models.WatchlistItem{}, fmt.Errorf("upsert stock: %w", err)
	}

	var watchlistID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM watchlists WHERE user_id = $1 ORDER BY created_at LIMIT 1`, userID).Scan(&watchlistID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO watchlists (user_id) VALUES ($1) RETURNING id::text`, userID).Scan(&watchlistID)
	}
	if err != nil {
		return models.WatchlistItem{}, fmt.Errorf("find or create watchlist: %w", err)
	}

	var item models.WatchlistItem
	err = tx.QueryRow(ctx, `
		INSERT INTO watchlist_stocks (watchlist_id, stock_id) VALUES ($1, $2)
		RETURNING id::text, created_at
	`, watchlistID, stockID).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.WatchlistItem{}, ErrStockAlreadyWatched
		}
		return models.WatchlistItem{}, fmt.Errorf("insert watchlist stock: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return models.WatchlistItem{}, fmt.Errorf("commit add watchlist: %w", err)
	}
	item.Symbol, item.Name, item.Exchange, item.Currency = stock.Symbol, stock.Name, stock.Exchange, stock.Currency
	return item, nil
}

func (r *WatchlistRepository) Delete(ctx context.Context, userID, entryID string) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM watchlist_stocks ws USING watchlists w
		WHERE ws.id = $1 AND ws.watchlist_id = w.id AND w.user_id = $2
	`, entryID, userID)
	if err != nil {
		return fmt.Errorf("delete watchlist stock: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}
	return nil
}
