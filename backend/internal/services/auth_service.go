package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type AuthStore interface {
	CreateUserWithVerification(context.Context, string, string, string, time.Time) (models.User, error)
	FindUserByEmail(context.Context, string) (models.User, error)
	LatestVerification(context.Context, string) (models.VerificationToken, error)
	IncrementVerificationAttempts(context.Context, string) error
	MarkEmailVerified(context.Context, string, string) error
	ReplaceVerification(context.Context, string, string, time.Time) error
	CreateRefreshToken(context.Context, string, string, time.Time, string, string) error
	FindRefreshToken(context.Context, string) (models.RefreshToken, models.User, error)
	RotateRefreshToken(context.Context, string, string, string, string, time.Time, string, string) error
	RevokeTokenFamily(context.Context, string) error
	RevokeAllUserTokens(context.Context, string) error
	FindOrCreateLineUser(context.Context, string, string) (models.User, error)
}

func (s *AuthService) IssueTokens(ctx context.Context, user models.User, ip, userAgent string) (models.TokenResponse, error) {
	if user.Status == "DISABLED" {
		return models.TokenResponse{}, newError("ACCOUNT_DISABLED", "This account is disabled.", nil)
	}
	return s.issueTokens(ctx, user, ip, userAgent)
}

type AuthConfig struct {
	AccessSecret, RefreshSecret, Issuer              string
	AccessTTL, RefreshTTL, OTPTTL, OTPResendCooldown time.Duration
	BcryptCost, OTPMaxAttempts                       int
}

type AuthService struct {
	store  AuthStore
	mailer Mailer
	cfg    AuthConfig
	now    func() time.Time
}

func NewAuthService(store AuthStore, mailer Mailer, cfg AuthConfig) *AuthService {
	return &AuthService{store: store, mailer: mailer, cfg: cfg, now: time.Now}
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (models.RegisterResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil || !validPassword(req.Password) || req.Password != req.ConfirmPassword {
		return models.RegisterResponse{}, newError("VALIDATION_ERROR", "The registration details are invalid.", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
	if err != nil {
		return models.RegisterResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to register account.", err)
	}
	otp, err := generateOTP()
	if err != nil {
		return models.RegisterResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to register account.", err)
	}
	user, err := s.store.CreateUserWithVerification(ctx, email, string(hash), s.hashOTP(email, otp), s.now().Add(s.cfg.OTPTTL))
	if errors.Is(err, repositories.ErrEmailExists) {
		return models.RegisterResponse{}, newError("EMAIL_ALREADY_EXISTS", "Email is already registered.", err)
	}
	if err != nil {
		return models.RegisterResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to register account.", err)
	}
	if err = s.mailer.SendVerificationOTP(email, otp); err != nil {
		return models.RegisterResponse{}, newError("EMAIL_SERVICE_UNAVAILABLE", "Unable to send the verification email.", err)
	}
	return models.RegisterResponse{UserID: user.ID, Email: email, VerificationRequired: true}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req models.VerifyEmailRequest) error {
	email, err := normalizeEmail(req.Email)
	if err != nil || len(req.OTP) != 6 {
		return newError("INVALID_OTP", "The verification code is invalid.", err)
	}
	user, err := s.store.FindUserByEmail(ctx, email)
	if errors.Is(err, repositories.ErrNotFound) {
		return newError("USER_NOT_FOUND", "User was not found.", err)
	}
	if err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Unable to verify email.", err)
	}
	if user.EmailVerifiedAt != nil {
		return newError("OTP_ALREADY_USED", "The verification code has already been used.", nil)
	}
	token, err := s.store.LatestVerification(ctx, user.ID)
	if errors.Is(err, repositories.ErrNotFound) {
		return newError("INVALID_OTP", "The verification code is invalid.", err)
	}
	if err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Unable to verify email.", err)
	}
	if token.Attempts >= s.cfg.OTPMaxAttempts {
		return newError("OTP_MAX_ATTEMPTS_EXCEEDED", "Too many verification attempts.", nil)
	}
	if !s.now().Before(token.ExpiresAt) {
		return newError("OTP_EXPIRED", "The verification code has expired.", nil)
	}
	if !hmac.Equal([]byte(token.OTPHash), []byte(s.hashOTP(email, req.OTP))) {
		if incrementErr := s.store.IncrementVerificationAttempts(ctx, token.ID); incrementErr != nil {
			return newError("INTERNAL_SERVER_ERROR", "Unable to verify email.", incrementErr)
		}
		if token.Attempts+1 >= s.cfg.OTPMaxAttempts {
			return newError("OTP_MAX_ATTEMPTS_EXCEEDED", "Too many verification attempts.", nil)
		}
		return newError("INVALID_OTP", "The verification code is invalid.", errInvalidCredentials)
	}
	if err = s.store.MarkEmailVerified(ctx, user.ID, token.ID); err != nil {
		return newError("INTERNAL_SERVER_ERROR", "Unable to verify email.", err)
	}
	return nil
}

func (s *AuthService) ResendOTP(ctx context.Context, req models.ResendOTPRequest) (int64, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return 0, newError("VALIDATION_ERROR", "A valid email is required.", err)
	}
	user, err := s.store.FindUserByEmail(ctx, email)
	if errors.Is(err, repositories.ErrNotFound) {
		return 0, newError("USER_NOT_FOUND", "User was not found.", err)
	}
	if err != nil {
		return 0, newError("INTERNAL_SERVER_ERROR", "Unable to resend verification code.", err)
	}
	if user.EmailVerifiedAt != nil {
		return 0, newError("EMAIL_ALREADY_VERIFIED", "Email is already verified.", nil)
	}
	if current, lookupErr := s.store.LatestVerification(ctx, user.ID); lookupErr == nil && s.now().Before(current.CreatedAt.Add(s.cfg.OTPResendCooldown)) {
		return 0, newError("OTP_RATE_LIMITED", "Please wait before requesting another verification code.", nil)
	} else if lookupErr != nil && !errors.Is(lookupErr, repositories.ErrNotFound) {
		return 0, newError("INTERNAL_SERVER_ERROR", "Unable to resend verification code.", lookupErr)
	}
	otp, err := generateOTP()
	if err != nil {
		return 0, newError("INTERNAL_SERVER_ERROR", "Unable to resend verification code.", err)
	}
	if err = s.store.ReplaceVerification(ctx, user.ID, s.hashOTP(email, otp), s.now().Add(s.cfg.OTPTTL)); err != nil {
		return 0, newError("INTERNAL_SERVER_ERROR", "Unable to resend verification code.", err)
	}
	if err = s.mailer.SendVerificationOTP(email, otp); err != nil {
		return 0, newError("EMAIL_SERVICE_UNAVAILABLE", "Unable to send the verification email.", err)
	}
	return int64(s.cfg.OTPTTL.Seconds()), nil
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
		return models.TokenResponse{}, newError("EMAIL_NOT_VERIFIED", "Please verify your email before logging in.", nil)
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

func (s *AuthService) hashOTP(email, otp string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.RefreshSecret))
	_, _ = mac.Write([]byte("email-verification:" + email + ":" + otp))
	return hex.EncodeToString(mac.Sum(nil))
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

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return "", errors.New("invalid email")
	}
	return value, nil
}

func validPassword(value string) bool {
	if len(value) < 8 || len(value) > 72 {
		return false
	}
	var upper, lower, digit, special bool
	for _, r := range value {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			special = true
		}
	}
	return upper && lower && digit && special
}
