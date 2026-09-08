package services

import (
	"context"
	"errors"
	"strings"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type AdminUserStore interface {
	List(context.Context, models.AdminUserFilter) ([]models.AdminUser, int64, error)
	Get(context.Context, string) (models.AdminUserDetail, error)
	SetStatus(context.Context, models.AdminActionContext, string, string, string) error
}

type AdminUserService struct{ store AdminUserStore }

func NewAdminUserService(store AdminUserStore) *AdminUserService {
	return &AdminUserService{store: store}
}

func (s *AdminUserService) List(ctx context.Context, search, status, pageValue, limitValue string) (models.AdminUserPage, error) {
	page, err := positiveInt(pageValue, 1, 1_000_000)
	if err != nil {
		return models.AdminUserPage{}, newError("INVALID_FILTER", "Page must be a positive integer.", err)
	}
	limit, err := positiveInt(limitValue, 20, 100)
	if err != nil {
		return models.AdminUserPage{}, newError("INVALID_FILTER", "Limit must be between 1 and 100.", err)
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != "ACTIVE" && status != "DISABLED" {
		return models.AdminUserPage{}, newError("INVALID_FILTER", "Status must be ACTIVE or DISABLED.", nil)
	}
	users, total, err := s.store.List(ctx, models.AdminUserFilter{Search: strings.TrimSpace(search), Status: status, Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return models.AdminUserPage{}, newError("INTERNAL_SERVER_ERROR", "Could not load users.", err)
	}
	return models.AdminUserPage{Users: users, Page: page, Limit: limit, Total: total}, nil
}

func (s *AdminUserService) Get(ctx context.Context, userID string) (models.AdminUserDetail, error) {
	user, err := s.store.Get(ctx, strings.TrimSpace(userID))
	if errors.Is(err, repositories.ErrAdminUserNotFound) {
		return models.AdminUserDetail{}, newError("ADMIN_USER_NOT_FOUND", "User not found.", err)
	}
	if err != nil {
		return models.AdminUserDetail{}, newError("INTERNAL_SERVER_ERROR", "Could not load the user.", err)
	}
	return user, nil
}

func (s *AdminUserService) Disable(ctx context.Context, action models.AdminActionContext, targetID, reason string) error {
	targetID, reason = strings.TrimSpace(targetID), strings.TrimSpace(reason)
	if targetID == action.AdminUserID {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "Administrators cannot disable their own account.", nil)
	}
	if reason == "" || len(reason) > 1000 {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "A disable reason between 1 and 1000 characters is required.", nil)
	}
	return mapAdminUserMutationError(s.store.SetStatus(ctx, action, targetID, "DISABLED", reason))
}

func (s *AdminUserService) Enable(ctx context.Context, action models.AdminActionContext, targetID string) error {
	return mapAdminUserMutationError(s.store.SetStatus(ctx, action, strings.TrimSpace(targetID), "ACTIVE", ""))
}

func mapAdminUserMutationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repositories.ErrAdminUserNotFound) {
		return newError("ADMIN_USER_NOT_FOUND", "User not found.", err)
	}
	if errors.Is(err, repositories.ErrAdminActionNotAllowed) {
		return newError("ADMIN_ACTION_NOT_ALLOWED", "The requested account action is not allowed.", err)
	}
	return newError("INTERNAL_SERVER_ERROR", "Could not update the user account.", err)
}
