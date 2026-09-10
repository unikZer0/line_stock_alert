package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"stock_linebot/backend/internal/models"
)

type RichMenuRepository struct{ db *pgxpool.Pool }

func NewRichMenuRepository(db *pgxpool.Pool) *RichMenuRepository { return &RichMenuRepository{db: db} }

func (r *RichMenuRepository) RecordPublish(ctx context.Context, action models.AdminActionContext, richMenuID string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO admin_audit_logs(admin_user_id,action,target_type,target_id,ip_address,user_agent,details)
		VALUES($1,'RICH_MENU_PUBLISHED','LINE_RICH_MENU',$2,NULLIF($3,'')::inet,$4,jsonb_build_object('status','DEFAULT'))`,
		action.AdminUserID, richMenuID, action.IPAddress, action.UserAgent)
	if err != nil {
		return fmt.Errorf("audit rich menu publish: %w", err)
	}
	return nil
}
