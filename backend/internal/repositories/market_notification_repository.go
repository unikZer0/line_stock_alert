package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketNotificationRepository struct{ db *pgxpool.Pool }

func NewMarketNotificationRepository(db *pgxpool.Pool) *MarketNotificationRepository {
	return &MarketNotificationRepository{db: db}
}

func (r *MarketNotificationRepository) ActiveLineUserIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT ui.provider_user_id FROM user_identities ui
		JOIN users u ON u.id=ui.user_id
		WHERE ui.provider='LINE' AND u.status='ACTIVE' ORDER BY ui.provider_user_id`)
	if err != nil {
		return nil, fmt.Errorf("list market notification recipients: %w", err)
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan market notification recipient: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *MarketNotificationRepository) LogError(ctx context.Context, code, message string, details map[string]any) error {
	_, err := r.db.Exec(ctx, `INSERT INTO application_logs(level,service,code,message,details)
		VALUES('ERROR','market-notification-worker',$1,$2,$3)`, code, message, details)
	return err
}
