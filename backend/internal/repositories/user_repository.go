package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stock_linebot/backend/internal/models"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindCurrentUser(ctx context.Context, userID string) (models.CurrentUser, error) {
	var user models.CurrentUser
	var hasEmailProvider bool
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email::text, COALESCE(display_name, ''),
		       email_verified_at IS NOT NULL, password_hash IS NOT NULL
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.EmailVerified, &hasEmailProvider,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CurrentUser{}, ErrNotFound
	}
	if err != nil {
		return models.CurrentUser{}, fmt.Errorf("find current user: %w", err)
	}

	user.Providers = make([]string, 0, 2)
	if hasEmailProvider {
		user.Providers = append(user.Providers, "EMAIL")
	}
	rows, err := r.db.Query(ctx, `
		SELECT provider FROM user_identities WHERE user_id = $1 ORDER BY provider
	`, userID)
	if err != nil {
		return models.CurrentUser{}, fmt.Errorf("find current user identities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var provider string
		if err = rows.Scan(&provider); err != nil {
			return models.CurrentUser{}, fmt.Errorf("scan current user identity: %w", err)
		}
		user.Providers = append(user.Providers, provider)
		if provider == "LINE" {
			user.LineConnected = true
		}
	}
	if err = rows.Err(); err != nil {
		return models.CurrentUser{}, fmt.Errorf("read current user identities: %w", err)
	}
	return user, nil
}
