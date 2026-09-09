package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type AuthStore interface {
	FindUserByEmail(context.Context, string) (models.User, error)
	CreateRefreshToken(context.Context, string, string, time.Time, string, string) error
	FindRefreshToken(context.Context, string) (models.RefreshToken, models.User, error)
	RotateRefreshToken(context.Context, string, string, string, string, time.Time, string, string) error
	RevokeTokenFamily(context.Context, string) error
	RevokeAllUserTokens(context.Context, string) error
	FindOrCreateLineUser(context.Context, string, string) (models.User, error)
}

type AuthConfig struct {
	AccessSecret  string
	RefreshSecret string
	Issuer        string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type AuthService struct {
	store AuthStore
	cfg   AuthConfig
	now   func() time.Time
}

func NewAuthService(store AuthStore, cfg AuthConfig) *AuthService {
	return &AuthService{store: store, cfg: cfg, now: time.Now}
}

func (s *AuthService) IssueTokens(ctx context.Context, user models.User, ip, userAgent string) (models.TokenResponse, error) {
	if user.Status == "DISABLED" {
		return models.TokenResponse{}, newError("ACCOUNT_DISABLED", "This account is disabled.", nil)
	}
	return s.issueTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest, ip, userAgent string) (models.TokenResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return models.TokenResponse{}, newError("INVALID_CREDENTIALS", "Email or password is incorrect.", err)
	}
	user, err := s.store.FindUserByEmail(ctx, email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return models.TokenResponse{}, newError("INVALID_CREDENTIALS", "Email or password is incorrect.", errInvalidCredentials)
	}
	if user.Status == "DISABLED" {
		return models.TokenResponse{}, newError("ACCOUNT_DISABLED", "This account is disabled.", nil)
	}
	if user.EmailVerifiedAt == nil {
		return models.TokenResponse{}, newError("EMAIL_NOT_VERIFIED", "The administrator email is not verified.", nil)
	}
	if user.Role != "ADMIN" {
		return models.TokenResponse{}, newError("ADMIN_LOGIN_REQUIRED", "Password login is available to administrators only. Please use LINE Login.", nil)
	}
	return s.issueTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) Refresh(ctx context.Context, rawToken, ip, userAgent string) (models.TokenResponse, error) {
	if rawToken == "" {
		return models.TokenResponse{}, newError("INVALID_REFRESH_TOKEN", "The refresh token is invalid.", nil)
	}
	token, user, err := s.store.FindRefreshToken(ctx, hashToken(rawToken))
	if errors.Is(err, repositories.ErrNotFound) {
		return models.TokenResponse{}, newError("INVALID_REFRESH_TOKEN", "The refresh token is invalid.", err)
	}
	if err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to refresh token.", err)
	}
	if token.RevokedAt != nil {
		_ = s.store.RevokeTokenFamily(ctx, token.FamilyID)
		return models.TokenResponse{}, newError("REFRESH_TOKEN_REVOKED", "The refresh token has been revoked.", nil)
	}
	if !s.now().Before(token.ExpiresAt) {
		return models.TokenResponse{}, newError("REFRESH_TOKEN_EXPIRED", "The refresh token has expired.", nil)
	}
	if user.Status == "DISABLED" {
		return models.TokenResponse{}, newError("ACCOUNT_DISABLED", "This account is disabled.", nil)
	}
	access, err := s.createAccessToken(user)
	if err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to refresh token.", err)
	}
	newRaw, err := randomToken()
	if err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to refresh token.", err)
	}
	if err = s.store.RotateRefreshToken(ctx, token.ID, user.ID, token.FamilyID, hashToken(newRaw), s.now().Add(s.cfg.RefreshTTL), ip, userAgent); err != nil {
		return models.TokenResponse{}, newError("INVALID_REFRESH_TOKEN", "The refresh token is invalid.", err)
	}
	return models.TokenResponse{AccessToken: access, RefreshToken: newRaw, TokenType: "Bearer", ExpiresIn: int64(s.cfg.AccessTTL.Seconds())}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID string) error {
	if err := s.store.RevokeAllUserTokens(ctx, userID); err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Unable to log out.", err)
	}
	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, user models.User, ip, userAgent string) (models.TokenResponse, error) {
	access, err := s.createAccessToken(user)
	if err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to create tokens.", err)
	}
	refresh, err := randomToken()
	if err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to create tokens.", err)
	}
	if err = s.store.CreateRefreshToken(ctx, user.ID, hashToken(refresh), s.now().Add(s.cfg.RefreshTTL), ip, userAgent); err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to create tokens.", err)
	}
	return models.TokenResponse{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int64(s.cfg.AccessTTL.Seconds())}, nil
}

func (s *AuthService) createAccessToken(user models.User) (string, error) {
	now := s.now()
	claims := jwt.MapClaims{
		"sub": user.ID, "role": user.Role, "token_type": "access",
		"iss": s.cfg.Issuer, "iat": now.Unix(), "exp": now.Add(s.cfg.AccessTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.AccessSecret))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return "", errors.New("invalid email")
	}
	return value, nil
}
