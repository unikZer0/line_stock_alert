package services

import (
	"context"
	"errors"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type CurrentUserStore interface {
	FindCurrentUser(context.Context, string) (models.CurrentUser, error)
}

type UserService struct {
	store CurrentUserStore
}

func NewUserService(store CurrentUserStore) *UserService {
	return &UserService{store: store}
}

func (s *UserService) CurrentUser(ctx context.Context, userID string) (models.CurrentUser, error) {
	user, err := s.store.FindCurrentUser(ctx, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return models.CurrentUser{}, newError("USER_NOT_FOUND", "User was not found.", err)
	}
	if err != nil {
		return models.CurrentUser{}, newError("INTERNAL_SERVER_ERROR", "Unable to retrieve the user profile.", err)
	}
	return user, nil
}
