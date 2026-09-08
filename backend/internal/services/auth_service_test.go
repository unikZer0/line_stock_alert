package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"stock_linebot/backend/internal/models"
)

type fakeAuthStore struct {
	AuthStore
	create func(context.Context, string, string, string, time.Time) (models.User, error)
}

func (f fakeAuthStore) CreateUserWithVerification(ctx context.Context, email, passwordHash, otpHash string, expiresAt time.Time) (models.User, error) {
	return f.create(ctx, email, passwordHash, otpHash, expiresAt)
}

type fakeMailer struct {
	to  string
	otp string
}

func (f *fakeMailer) SendVerificationOTP(to, otp string) error {
	f.to, f.otp = to, otp
	return nil
}

func TestRegisterHashesPasswordAndSendsOTP(t *testing.T) {
	fixedNow := time.Unix(1_700_000_000, 0)
	mailer := &fakeMailer{}
	store := fakeAuthStore{create: func(_ context.Context, email, passwordHash, otpHash string, expiresAt time.Time) (models.User, error) {
		if email != "user@example.com" {
			t.Fatalf("email = %q", email)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("Password123!")); err != nil {
			t.Fatalf("password was not hashed correctly: %v", err)
		}
		if otpHash == "" {
			t.Fatal("OTP hash is empty")
		}
		if !expiresAt.Equal(fixedNow.Add(10 * time.Minute)) {
			t.Fatalf("expiresAt = %v", expiresAt)
		}
		return models.User{ID: "user-1", Email: email}, nil
	}}
	service := NewAuthService(store, mailer, AuthConfig{
		RefreshSecret: strings.Repeat("r", 32), OTPTTL: 10 * time.Minute, BcryptCost: bcrypt.MinCost,
	})
	service.now = func() time.Time { return fixedNow }

	result, err := service.Register(context.Background(), models.RegisterRequest{
		Email: "User@Example.com", Password: "Password123!", ConfirmPassword: "Password123!",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if result.UserID != "user-1" || !result.VerificationRequired {
		t.Fatalf("unexpected result: %#v", result)
	}
	if mailer.to != "user@example.com" || len(mailer.otp) != 6 {
		t.Fatalf("unexpected mail: %#v", mailer)
	}
}

func TestValidPassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"Password123!", true}, {"short!1A", true}, {"password123!", false},
		{"PASSWORD123!", false}, {"Password!", false}, {"Password123", false},
		{strings.Repeat("A", 73), false},
	}
	for _, test := range tests {
		if got := validPassword(test.password); got != test.want {
			t.Errorf("validPassword(%q) = %v, want %v", test.password, got, test.want)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  User@Example.COM ")
	if err != nil {
		t.Fatalf("normalizeEmail returned error: %v", err)
	}
	if got != "user@example.com" {
		t.Fatalf("normalizeEmail = %q", got)
	}
	if _, err = normalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestOTPHashIsScopedToEmail(t *testing.T) {
	service := &AuthService{cfg: AuthConfig{RefreshSecret: strings.Repeat("r", 32)}}
	one := service.hashOTP("one@example.com", "123456")
	two := service.hashOTP("two@example.com", "123456")
	if one == two {
		t.Fatal("OTP hashes should differ across email addresses")
	}
	if one != service.hashOTP("one@example.com", "123456") {
		t.Fatal("OTP hash must be deterministic")
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
	token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return []byte(strings.Repeat("a", 32)), nil }, jwt.WithIssuer("test"), jwt.WithTimeFunc(service.now))
	if err != nil || !token.Valid {
		t.Fatalf("parse access token: %v", err)
	}
	if claims["sub"] != "user-id" || claims["token_type"] != "access" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
