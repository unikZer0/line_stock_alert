package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"stock_linebot/backend/internal/models"
)

type loginOnlyStore struct {
	AuthStore
	user models.User
}

func (s loginOnlyStore) FindUserByEmail(context.Context, string) (models.User, error) {
	return s.user, nil
}

func TestPasswordLoginRejectsNonAdminUsers(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	verified := time.Now()
	service := NewAuthService(loginOnlyStore{user: models.User{
		ID: "user-1", Email: "user@example.com", PasswordHash: string(hash),
		Role: "USER", Status: "ACTIVE", EmailVerifiedAt: &verified,
	}}, AuthConfig{RefreshSecret: strings.Repeat("r", 32)})

	_, err = service.Login(context.Background(), models.LoginRequest{
		Email: "user@example.com", Password: "Password123!",
	}, "", "")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "ADMIN_LOGIN_REQUIRED" {
		t.Fatalf("Login error = %#v, want ADMIN_LOGIN_REQUIRED", err)
	}
}
