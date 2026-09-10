package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type LineWebhookRepository struct{ db DBTX }

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewLineWebhookRepository(db DBTX) *LineWebhookRepository { return &LineWebhookRepository{db: db} }

func (r *LineWebhookRepository) ClaimEvent(ctx context.Context, eventID, lineUserID, eventType string, payload json.RawMessage) (bool, error) {
	var claimed string
	err := r.db.QueryRow(ctx, `
		INSERT INTO line_webhook_events (line_event_id, line_user_id, event_type, payload)
		VALUES ($1, NULLIF($2, ''), $3, $4)
		ON CONFLICT (line_event_id) DO UPDATE SET status = 'RECEIVED', error_message = NULL
		WHERE line_webhook_events.status = 'FAILED'
		RETURNING id::text
	`, eventID, lineUserID, eventType, payload).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim LINE webhook event: %w", err)
	}
	return true, nil
}

func (r *LineWebhookRepository) FindUserByLineID(ctx context.Context, lineUserID string) (string, string, error) {
	var userID, status string
	err := r.db.QueryRow(ctx, `
		SELECT u.id::text, u.status FROM users u
		JOIN user_identities ui ON ui.user_id = u.id
		WHERE ui.provider = 'LINE' AND ui.provider_user_id = $1
	`, lineUserID).Scan(&userID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("find webhook LINE user: %w", err)
	}
	return userID, status, nil
}

func (r *LineWebhookRepository) MarkProcessed(ctx context.Context, eventID string) error {
	_, err := r.db.Exec(ctx, `UPDATE line_webhook_events SET status = 'PROCESSED', processed_at = NOW(), error_message = NULL WHERE line_event_id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("mark LINE webhook processed: %w", err)
	}
	return nil
}

func (r *LineWebhookRepository) MarkFailed(ctx context.Context, eventID, message string) error {
	_, err := r.db.Exec(ctx, `UPDATE line_webhook_events SET status = 'FAILED', processed_at = NULL, error_message = $2 WHERE line_event_id = $1`, eventID, message)
	if err != nil {
		return fmt.Errorf("mark LINE webhook failed: %w", err)
	}
	return nil
}
