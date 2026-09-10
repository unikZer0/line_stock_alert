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
	ErrAdminUserNotFound     = errors.New("admin target user not found")
	ErrAdminActionNotAllowed = errors.New("admin action not allowed")
)

type AdminUserRepository struct{ db *pgxpool.Pool }

func NewAdminUserRepository(db *pgxpool.Pool) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

func (r *AdminUserRepository) List(ctx context.Context, filter models.AdminUserFilter) ([]models.AdminUser, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users
		WHERE ($1 = '' OR COALESCE(email::text, '') ILIKE '%' || $1 || '%' OR COALESCE(display_name, '') ILIKE '%' || $1 || '%')
		AND ($2 = '' OR status = $2)`, filter.Search, filter.Status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin users: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.email::text, COALESCE(u.display_name, ''), u.role, u.status,
		       u.email_verified_at IS NOT NULL,
		       EXISTS(SELECT 1 FROM user_identities ui WHERE ui.user_id = u.id AND ui.provider = 'LINE'),
		       u.disabled_reason, u.disabled_at, u.created_at, u.updated_at
		FROM users u
		WHERE ($1 = '' OR COALESCE(u.email::text, '') ILIKE '%' || $1 || '%' OR COALESCE(u.display_name, '') ILIKE '%' || $1 || '%')
		AND ($2 = '' OR u.status = $2)
		ORDER BY u.created_at DESC LIMIT $3 OFFSET $4
	`, filter.Search, filter.Status, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()
	users := make([]models.AdminUser, 0)
	for rows.Next() {
		var user models.AdminUser
		if err = rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status,
			&user.EmailVerified, &user.LineConnected, &user.DisabledReason, &user.DisabledAt,
			&user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan admin user: %w", err)
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("read admin users: %w", err)
	}
	return users, total, nil
}

func (r *AdminUserRepository) Get(ctx context.Context, userID string) (models.AdminUserDetail, error) {
	var user models.AdminUserDetail
	err := r.db.QueryRow(ctx, `
		SELECT u.id::text, u.email::text, COALESCE(u.display_name, ''), u.role, u.status,
		       u.email_verified_at IS NOT NULL,
		       EXISTS(SELECT 1 FROM user_identities ui WHERE ui.user_id = u.id AND ui.provider = 'LINE'),
		       u.disabled_reason, u.disabled_at, u.created_at, u.updated_at
		FROM users u WHERE u.id::text = $1
	`, userID).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status,
		&user.EmailVerified, &user.LineConnected, &user.DisabledReason, &user.DisabledAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.AdminUserDetail{}, ErrAdminUserNotFound
	}
	if err != nil {
		return models.AdminUserDetail{}, fmt.Errorf("get admin user: %w", err)
	}
	user.Providers = make([]string, 0, 2)
	if user.Email != nil {
		user.Providers = append(user.Providers, "EMAIL")
	}
	rows, err := r.db.Query(ctx, `SELECT provider FROM user_identities WHERE user_id::text = $1 ORDER BY provider`, userID)
	if err != nil {
		return models.AdminUserDetail{}, fmt.Errorf("list admin user providers: %w", err)
	}
	for rows.Next() {
		var provider string
		if err = rows.Scan(&provider); err != nil {
			rows.Close()
			return models.AdminUserDetail{}, err
		}
		user.Providers = append(user.Providers, provider)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return models.AdminUserDetail{}, fmt.Errorf("read admin user providers: %w", err)
	}
	user.Alerts, err = r.listAlerts(ctx, userID)
	if err != nil {
		return models.AdminUserDetail{}, err
	}
	return user, nil
}

func (r *AdminUserRepository) listAlerts(ctx context.Context, userID string) ([]models.Alert, error) {
	rows, err := r.db.Query(ctx, `SELECT a.id::text,s.symbol,a.condition,a.target_price,a.status,a.triggered_at,a.disabled_reason,a.created_at,a.updated_at
		FROM alerts a JOIN stocks s ON s.id=a.stock_id WHERE a.user_id::text=$1 ORDER BY a.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list admin user alerts: %w", err)
	}
	defer rows.Close()
	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var alert models.Alert
		if err = rows.Scan(&alert.ID, &alert.Symbol, &alert.Condition, &alert.TargetPrice, &alert.Status, &alert.TriggeredAt, &alert.DisabledReason, &alert.CreatedAt, &alert.UpdatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

func (r *AdminUserRepository) SetStatus(ctx context.Context, action models.AdminActionContext, targetID, status, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin admin user status: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('admin-user-status'))`); err != nil {
		return fmt.Errorf("lock admin user status: %w", err)
	}
	var role, current string
	err = tx.QueryRow(ctx, `SELECT role,status FROM users WHERE id::text=$1 FOR UPDATE`, targetID).Scan(&role, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdminUserNotFound
	}
	if err != nil {
		return fmt.Errorf("find admin target: %w", err)
	}
	if current == status {
		return ErrAdminActionNotAllowed
	}
	if status == "DISABLED" && role == "ADMIN" {
		var count int
		if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role='ADMIN' AND status='ACTIVE'`).Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return ErrAdminActionNotAllowed
		}
	}
	if status == "DISABLED" {
		_, err = tx.Exec(ctx, `UPDATE users SET status='DISABLED',disabled_reason=$2,disabled_at=NOW() WHERE id::text=$1`, targetID, reason)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW() WHERE user_id::text=$1 AND revoked_at IS NULL`, targetID)
		}
	} else {
		_, err = tx.Exec(ctx, `UPDATE users SET status='ACTIVE',disabled_reason=NULL,disabled_at=NULL WHERE id::text=$1`, targetID)
	}
	if err != nil {
		return fmt.Errorf("update admin target status: %w", err)
	}
	auditAction := "USER_ENABLED"
	if status == "DISABLED" {
		auditAction = "USER_DISABLED"
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit_logs(admin_user_id,action,target_type,target_id,ip_address,user_agent,details)
		VALUES($1,$2,'USER',$3,NULLIF($4,'')::inet,$5,jsonb_build_object('previous_status',$6,'new_status',$7,'reason',$8))`,
		action.AdminUserID, auditAction, targetID, action.IPAddress, action.UserAgent, current, status, reason)
	if err != nil {
		return fmt.Errorf("audit admin user status: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit admin user status: %w", err)
	}
	return nil
}
