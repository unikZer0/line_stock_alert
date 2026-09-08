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
	ErrEmailExists   = errors.New("email already exists")
	ErrTokenConflict = errors.New("token already exists")
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUserWithVerification(
	ctx context.Context, email, passwordHash, otpHash string, expiresAt time.Time,
) (models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("begin registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user models.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, email::text, password_hash, role, status, email_verified_at
	`, email, passwordHash).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Status, &user.EmailVerifiedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrEmailExists
		}
		return models.User{}, fmt.Errorf("insert user: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, otp_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, otpHash, expiresAt); err != nil {
		return models.User{}, fmt.Errorf("insert verification token: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return models.User{}, fmt.Errorf("commit registration: %w", err)
	}
	return user, nil
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

func (r *AuthRepository) LatestVerification(ctx context.Context, userID string) (models.VerificationToken, error) {
	var token models.VerificationToken
	err := r.db.QueryRow(ctx, `
		SELECT id::text, user_id::text, otp_hash, expires_at, used_at, attempts, created_at
		FROM email_verification_tokens
		WHERE user_id = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&token.ID, &token.UserID, &token.OTPHash, &token.ExpiresAt, &token.UsedAt, &token.Attempts, &token.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.VerificationToken{}, ErrNotFound
	}
	if err != nil {
		return models.VerificationToken{}, fmt.Errorf("find verification token: %w", err)
	}
	return token, nil
}

func (r *AuthRepository) IncrementVerificationAttempts(ctx context.Context, tokenID string) error {
	result, err := r.db.Exec(ctx, `UPDATE email_verification_tokens SET attempts = attempts + 1 WHERE id = $1`, tokenID)
	if err != nil {
		return fmt.Errorf("increment verification attempts: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AuthRepository) MarkEmailVerified(ctx context.Context, userID, tokenID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `
		UPDATE email_verification_tokens SET used_at = NOW()
		WHERE id = $1 AND user_id = $2 AND used_at IS NULL
	`, tokenID, userID)
	if err != nil {
		return fmt.Errorf("consume verification token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET email_verified_at = NOW() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("verify user email: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit email verification: %w", err)
	}
	return nil
}

func (r *AuthRepository) ReplaceVerification(ctx context.Context, userID, otpHash string, expiresAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `
		UPDATE email_verification_tokens SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID); err != nil {
		return fmt.Errorf("invalidate verification tokens: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, otp_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, otpHash, expiresAt); err != nil {
		return fmt.Errorf("insert verification token: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replacement verification: %w", err)
	}
	return nil
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
