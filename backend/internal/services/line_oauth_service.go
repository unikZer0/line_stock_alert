package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

const lineAuthorizationURL = "https://access.line.me/oauth2/v2.1/authorize"

type LineUserStore interface {
	FindOrCreateLineUser(context.Context, string, string) (models.User, error)
	UserHasLineIdentity(context.Context, string) (bool, error)
	LinkLineIdentity(context.Context, string, string) error
	UnlinkLineIdentity(context.Context, string) error
}

type LineOAuthConfig struct {
	ChannelID, CallbackURL, LinkCallbackURL, StateSecret, Issuer string
	StateTTL                                                     time.Duration
}

type LineOAuthService struct {
	store    LineUserStore
	provider LineProvider
	auth     *AuthService
	cfg      LineOAuthConfig
	now      func() time.Time
}

type lineStateClaims struct {
	Purpose string `json:"purpose"`
	Nonce   string `json:"nonce"`
	UserID  string `json:"user_id,omitempty"`
	jwt.RegisteredClaims
}

func NewLineOAuthService(store LineUserStore, provider LineProvider, auth *AuthService, cfg LineOAuthConfig) *LineOAuthService {
	return &LineOAuthService{store: store, provider: provider, auth: auth, cfg: cfg, now: time.Now}
}

func (s *LineOAuthService) AuthorizationURL() (string, error) {
	return s.authorizationURL("LINE_LOGIN", "", s.cfg.CallbackURL)
}

func (s *LineOAuthService) ConnectAuthorizationURL(ctx context.Context, userID string) (string, error) {
	connected, err := s.store.UserHasLineIdentity(ctx, userID)
	if err != nil {
		return "", newError("INTERNAL_SERVER_ERROR", "Unable to start LINE account linking.", err)
	}
	if connected {
		return "", newError("USER_ALREADY_HAS_LINE", "This account is already connected to LINE.", nil)
	}
	return s.authorizationURL("LINE_LINK", userID, s.cfg.LinkCallbackURL)
}

func (s *LineOAuthService) authorizationURL(purpose, userID, callbackURL string) (string, error) {
	nonce, err := randomToken()
	if err != nil {
		return "", newError("INTERNAL_SERVER_ERROR", "Unable to start LINE login.", err)
	}
	now := s.now()
	claims := lineStateClaims{
		Purpose: purpose, Nonce: nonce, UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: s.cfg.Issuer, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.StateTTL)),
		},
	}
	state, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.StateSecret))
	if err != nil {
		return "", newError("INTERNAL_SERVER_ERROR", "Unable to start LINE login.", err)
	}
	params := url.Values{
		"response_type": {"code"}, "client_id": {s.cfg.ChannelID}, "redirect_uri": {callbackURL},
		"state": {state}, "scope": {"openid profile"}, "nonce": {nonce},
	}
	return lineAuthorizationURL + "?" + params.Encode(), nil
}

func (s *LineOAuthService) Login(ctx context.Context, code, rawState, ip, userAgent string) (models.TokenResponse, error) {
	if code == "" {
		return models.TokenResponse{}, newError("INVALID_OAUTH_CODE", "The LINE authorization code is invalid.", nil)
	}
	claims, err := s.parseState(rawState, "LINE_LOGIN")
	if err != nil {
		return models.TokenResponse{}, err
	}
	identity, err := s.provider.ExchangeAndVerify(ctx, code, claims.Nonce, s.cfg.CallbackURL)
	if err != nil {
		return models.TokenResponse{}, lineProviderError(err)
	}
	user, err := s.store.FindOrCreateLineUser(ctx, identity.UserID, identity.DisplayName)
	if err != nil {
		return models.TokenResponse{}, newError("INTERNAL_SERVER_ERROR", "Unable to complete LINE login.", err)
	}
	result, err := s.auth.IssueTokens(ctx, user, ip, userAgent)
	if err != nil {
		return models.TokenResponse{}, fmt.Errorf("issue LINE login tokens: %w", err)
	}
	return result, nil
}

func (s *LineOAuthService) Link(ctx context.Context, code, rawState string) error {
	if code == "" {
		return newError("INVALID_OAUTH_CODE", "The LINE authorization code is invalid.", nil)
	}
	claims, err := s.parseState(rawState, "LINE_LINK")
	if err != nil || claims.UserID == "" {
		return newError("INVALID_OAUTH_STATE", "The OAuth state is invalid or expired.", err)
	}
	identity, err := s.provider.ExchangeAndVerify(ctx, code, claims.Nonce, s.cfg.LinkCallbackURL)
	if err != nil {
		return lineProviderError(err)
	}
	if err = s.store.LinkLineIdentity(ctx, claims.UserID, identity.UserID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrUserAlreadyHasLine):
			return newError("USER_ALREADY_HAS_LINE", "This account is already connected to LINE.", err)
		case errors.Is(err, repositories.ErrLineAlreadyLinked):
			return newError("LINE_ALREADY_LINKED", "This LINE account belongs to another user.", err)
		default:
			return newError("INTERNAL_SERVER_ERROR", "Unable to connect the LINE account.", err)
		}
	}
	return nil
}

func (s *LineOAuthService) Unlink(ctx context.Context, userID string) error {
	err := s.store.UnlinkLineIdentity(ctx, userID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repositories.ErrCannotUnlinkOnlyLogin):
		return newError("CANNOT_UNLINK_ONLY_LOGIN_METHOD", "Add and verify an email login before disconnecting LINE.", err)
	case errors.Is(err, repositories.ErrNotFound):
		return newError("LINE_USER_NOT_FOUND", "No LINE account is connected.", err)
	default:
		return newError("INTERNAL_SERVER_ERROR", "Unable to disconnect the LINE account.", err)
	}
}

func (s *LineOAuthService) parseState(rawState, purpose string) (*lineStateClaims, error) {
	claims := &lineStateClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(s.cfg.Issuer),
		jwt.WithExpirationRequired(), jwt.WithTimeFunc(s.now),
	)
	token, err := parser.ParseWithClaims(rawState, claims, func(*jwt.Token) (any, error) {
		return []byte(s.cfg.StateSecret), nil
	})
	if err != nil || !token.Valid || claims.Purpose != purpose || claims.Nonce == "" {
		return nil, newError("INVALID_OAUTH_STATE", "The OAuth state is invalid or expired.", err)
	}
	return claims, nil
}

func lineProviderError(err error) error {
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return newError("LINE_AUTH_PROVIDER_UNAVAILABLE", "LINE Login is temporarily unavailable.", err)
	}
	return newError("LINE_AUTH_PROVIDER_ERROR", "LINE could not authenticate this request.", err)
}
