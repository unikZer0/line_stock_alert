package services

import (
	"context"
	"errors"
	"testing"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type fakeCurrentUserStore struct {
	user models.CurrentUser
	err  error
}

func (f fakeCurrentUserStore) FindCurrentUser(context.Context, string) (models.CurrentUser, error) {
	return f.user, f.err
}

func TestCurrentUser(t *testing.T) {
	email := "user@example.com"
	want := models.CurrentUser{
		ID: "user-1", Email: &email, DisplayName: "User", EmailVerified: true,
		Providers: []string{"EMAIL", "LINE"}, LineConnected: true,
	}
	service := NewUserService(fakeCurrentUserStore{user: want})
	got, err := service.CurrentUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("CurrentUser returned error: %v", err)
	}
	if got.ID != want.ID || got.Email == nil || *got.Email != email || !got.LineConnected || len(got.Providers) != 2 {
		t.Fatalf("unexpected current user: %#v", got)
	}
}

func TestCurrentUserNotFound(t *testing.T) {
	service := NewUserService(fakeCurrentUserStore{err: repositories.ErrNotFound})
	_, err := service.CurrentUser(context.Background(), "missing")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Fatalf("unexpected error: %v", err)
	}
}
