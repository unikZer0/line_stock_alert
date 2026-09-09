package services

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"stock_linebot/backend/internal/models"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  Admin@Example.COM ")
	if err != nil {
		t.Fatalf("normalizeEmail returned error: %v", err)
	}
	if got != "admin@example.com" {
		t.Fatalf("normalizeEmail = %q", got)
	}
	if _, err = normalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestCreateAccessToken(t *testing.T) {
	service := &AuthService{cfg: AuthConfig{
		AccessSecret: strings.Repeat("a", 32), Issuer: "test", AccessTTL: time.Hour,
	}, now: func() time.Time { return time.Unix(1_700_000_000, 0) }}
	raw, err := service.createAccessToken(models.User{ID: "user-id", Role: "USER"})
	if err != nil {
		t.Fatalf("createAccessToken: %v", err)
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(strings.Repeat("a", 32)), nil
	}, jwt.WithIssuer("test"), jwt.WithTimeFunc(service.now))
	if err != nil || !token.Valid {
		t.Fatalf("parse access token: %v", err)
	}
	if claims["sub"] != "user-id" || claims["token_type"] != "access" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
