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
	ErrNotFound              = errors.New("record not found")
	ErrEmailExists           = errors.New("email already exists")
	ErrTokenConflict         = errors.New("token already exists")
	ErrUserAlreadyHasLine    = errors.New("user already has LINE identity")
	ErrLineAlreadyLinked     = errors.New("LINE identity already linked")
	ErrCannotUnlinkOnlyLogin = errors.New("cannot unlink only login method")
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

func (r *AuthRepository) UserHasLineIdentity(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_identities WHERE user_id = $1 AND provider = 'LINE')
	`, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check LINE identity: %w", err)
	}
	return exists, nil
}

func (r *AuthRepository) LinkLineIdentity(ctx context.Context, userID, lineUserID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin LINE identity link: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userConnected, lineConnected bool
	if err = tx.QueryRow(ctx, `
		SELECT
		  EXISTS (SELECT 1 FROM user_identities WHERE user_id = $1 AND provider = 'LINE'),
		  EXISTS (SELECT 1 FROM user_identities WHERE provider = 'LINE' AND provider_user_id = $2)
	`, userID, lineUserID).Scan(&userConnected, &lineConnected); err != nil {
		return fmt.Errorf("check LINE identity conflicts: %w", err)
	}
	if userConnected {
		return ErrUserAlreadyHasLine
	}
	if lineConnected {
		return ErrLineAlreadyLinked
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_identities (user_id, provider, provider_user_id) VALUES ($1, 'LINE', $2)
	`, userID, lineUserID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "user_identities_user_id_provider_key" {
				return ErrUserAlreadyHasLine
			}
			return ErrLineAlreadyLinked
		}
		return fmt.Errorf("link LINE identity: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit LINE identity link: %w", err)
	}
	return nil
}

func (r *AuthRepository) UnlinkLineIdentity(ctx context.Context, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin LINE identity unlink: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var passwordHash *string
	var emailVerifiedAt *time.Time
	if err = tx.QueryRow(ctx, `
		SELECT password_hash, email_verified_at FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&passwordHash, &emailVerifiedAt); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("find user for LINE unlink: %w", err)
	}
	if passwordHash == nil || *passwordHash == "" || emailVerifiedAt == nil {
		return ErrCannotUnlinkOnlyLogin
	}
	result, err := tx.Exec(ctx, `DELETE FROM user_identities WHERE user_id = $1 AND provider = 'LINE'`, userID)
	if err != nil {
		return fmt.Errorf("unlink LINE identity: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit LINE identity unlink: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
