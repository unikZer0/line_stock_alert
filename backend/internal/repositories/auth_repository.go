package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"stock_linebot/backend/internal/models"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrTokenConflict = errors.New("token already exists")
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email::text, password_hash, role, status, email_verified_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Status, &user.EmailVerifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) CreateRefreshToken(
	ctx context.Context, userID, tokenHash string, expiresAt time.Time, ip, userAgent string,
) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_ip, user_agent)
		VALUES ($1, $2, $3, NULLIF($4, '')::inet, NULLIF($5, ''))
	`, userID, tokenHash, expiresAt, ip, userAgent)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrTokenConflict
		}
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenHash string) (models.RefreshToken, models.User, error) {
	var token models.RefreshToken
	var user models.User
	err := r.db.QueryRow(ctx, `
		SELECT rt.id::text, rt.user_id::text, rt.family_id::text, rt.token_hash,
		       rt.expires_at, rt.revoked_at,
		       u.id::text, COALESCE(u.email::text, ''), COALESCE(u.password_hash, ''),
		       u.role, u.status, u.email_verified_at
		FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1
	`, tokenHash).Scan(
		&token.ID, &token.UserID, &token.FamilyID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt,
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Status, &user.EmailVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.RefreshToken{}, models.User{}, ErrNotFound
	}
	if err != nil {
		return models.RefreshToken{}, models.User{}, fmt.Errorf("find refresh token: %w", err)
	}
	return token, user, nil
}

func (r *AuthRepository) RotateRefreshToken(
	ctx context.Context, oldTokenID, userID, familyID, newHash string, expiresAt time.Time, ip, userAgent string,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin token rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at, created_ip, user_agent)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::inet, NULLIF($6, '')) RETURNING id::text
	`, userID, familyID, newHash, expiresAt, ip, userAgent).Scan(&newID)
	if err != nil {
		return fmt.Errorf("insert rotated token: %w", err)
	}
	result, err := tx.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW(), last_used_at = NOW(), replaced_by_token_id = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, oldTokenID, newID)
	if err != nil {
		return fmt.Errorf("revoke rotated token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit token rotation: %w", err)
	}
	return nil
}

func (r *AuthRepository) RevokeTokenFamily(ctx context.Context, familyID string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = COALESCE(revoked_at, NOW()) WHERE family_id = $1`, familyID)
	if err != nil {
		return fmt.Errorf("revoke token family: %w", err)
	}
	return nil
}

func (r *AuthRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = COALESCE(revoked_at, NOW()) WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("revoke user tokens: %w", err)
	}
	return nil
}

func (r *AuthRepository) FindOrCreateLineUser(ctx context.Context, lineUserID, displayName string) (models.User, error) {
	user, err := r.findUserByLineID(ctx, lineUserID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return models.User{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("begin LINE user creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx, `
		INSERT INTO users (display_name) VALUES (NULLIF($1, ''))
		RETURNING id::text, COALESCE(email::text, ''), COALESCE(password_hash, ''), role, status, email_verified_at
	`, displayName).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Status, &user.EmailVerifiedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("insert LINE user: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_identities (user_id, provider, provider_user_id) VALUES ($1, 'LINE', $2)
	`, user.ID, lineUserID); err != nil {
		return models.User{}, fmt.Errorf("insert LINE identity: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return models.User{}, fmt.Errorf("commit LINE user creation: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) findUserByLineID(ctx context.Context, lineUserID string) (models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx, `
		SELECT u.id::text, COALESCE(u.email::text, ''), COALESCE(u.password_hash, ''),
		       u.role, u.status, u.email_verified_at
		FROM users u JOIN user_identities ui ON ui.user_id = u.id
		WHERE ui.provider = 'LINE' AND ui.provider_user_id = $1
	`, lineUserID).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Status, &user.EmailVerifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("find LINE user: %w", err)
	}
	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
