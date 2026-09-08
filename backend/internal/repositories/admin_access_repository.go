package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminAccessRepository struct{ db *pgxpool.Pool }

func NewAdminAccessRepository(db *pgxpool.Pool) *AdminAccessRepository {
	return &AdminAccessRepository{db: db}
}

func (r *AdminAccessRepository) FindUserAccess(ctx context.Context, userID string) (role, status string, found bool, err error) {
	err = r.db.QueryRow(ctx, `SELECT role, status FROM users WHERE id::text = $1`, userID).Scan(&role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("find admin access user: %w", err)
	}
	return role, status, true, nil
}
