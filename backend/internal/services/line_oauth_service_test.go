package services

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type fakeLineProvider struct {
	identity LineIdentity
	nonce    string
}

func (f *fakeLineProvider) ExchangeAndVerify(_ context.Context, _ string, nonce, _ string) (LineIdentity, error) {
	f.nonce = nonce
	return f.identity, nil
}

type fakeLineStore struct {
	user       models.User
	lineUserID string
	connected  bool
	linkedID   string
	linkErr    error
	unlinkErr  error
}

func (f *fakeLineStore) UserHasLineIdentity(context.Context, string) (bool, error) {
	return f.connected, nil
}

func (f *fakeLineStore) LinkLineIdentity(_ context.Context, _, lineUserID string) error {
	f.linkedID = lineUserID
	return f.linkErr
}

func (f *fakeLineStore) UnlinkLineIdentity(context.Context, string) error {
	return f.unlinkErr
}

func (f *fakeLineStore) FindOrCreateLineUser(_ context.Context, lineUserID, _ string) (models.User, error) {
	f.lineUserID = lineUserID
	return f.user, nil
}

type fakeRefreshStore struct{ AuthStore }

func (fakeRefreshStore) CreateRefreshToken(context.Context, string, string, time.Time, string, string) error {
	return nil
}

func TestLineAuthorizationURLAndLogin(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	secret := strings.Repeat("l", 32)
	auth := NewAuthService(fakeRefreshStore{}, nil, AuthConfig{
		AccessSecret: strings.Repeat("a", 32), RefreshSecret: strings.Repeat("r", 32),
		Issuer: "test", AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour,
	})
	auth.now = func() time.Time { return now }
	provider := &fakeLineProvider{identity: LineIdentity{UserID: "U123", DisplayName: "Line User"}}
	store := &fakeLineStore{user: models.User{ID: "user-1", Role: "USER", Status: "ACTIVE"}}
	service := NewLineOAuthService(store, provider, auth, LineOAuthConfig{
		ChannelID: "channel-id", CallbackURL: "http://localhost:8080/api/v1/auth/line/callback",
		LinkCallbackURL: "http://localhost:8080/api/v1/accounts/line/callback",
		StateSecret:     secret, Issuer: "test", StateTTL: 10 * time.Minute,
	})
	service.now = func() time.Time { return now }

	authorizationURL, err := service.AuthorizationURL()
	if err != nil {
		t.Fatalf("AuthorizationURL: %v", err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("client_id") != "channel-id" || query.Get("scope") != "openid profile" {
		t.Fatalf("unexpected authorization query: %v", query)
	}
	if query.Get("nonce") == "" || query.Get("state") == "" {
		t.Fatal("nonce and state are required")
	}

	result, err := service.Login(context.Background(), "authorization-code", query.Get("state"), "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if store.lineUserID != "U123" {
		t.Fatalf("LINE user ID = %q", store.lineUserID)
	}
	if provider.nonce != query.Get("nonce") {
		t.Fatal("ID token nonce did not match authorization nonce")
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("missing application tokens: %#v", result)
	}
}

func TestLineAccountLinking(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	provider := &fakeLineProvider{identity: LineIdentity{UserID: "U-linked"}}
	store := &fakeLineStore{}
	service := NewLineOAuthService(store, provider, nil, LineOAuthConfig{
		ChannelID: "channel-id", LinkCallbackURL: "http://localhost:8080/api/v1/accounts/line/callback",
		StateSecret: strings.Repeat("l", 32), Issuer: "test", StateTTL: 10 * time.Minute,
	})
	service.now = func() time.Time { return now }
	authorizationURL, err := service.ConnectAuthorizationURL(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ConnectAuthorizationURL: %v", err)
	}
	parsed, _ := url.Parse(authorizationURL)
	if parsed.Query().Get("redirect_uri") != service.cfg.LinkCallbackURL {
		t.Fatal("link callback URL was not used")
	}
	if err = service.Link(context.Background(), "code", parsed.Query().Get("state")); err != nil {
		t.Fatalf("Link: %v", err)
	}
	if store.linkedID != "U-linked" {
		t.Fatalf("linked LINE ID = %q", store.linkedID)
	}
}

func TestConnectRejectsExistingLineIdentity(t *testing.T) {
	service := NewLineOAuthService(&fakeLineStore{connected: true}, nil, nil, LineOAuthConfig{})
	_, err := service.ConnectAuthorizationURL(context.Background(), "user-1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "USER_ALREADY_HAS_LINE" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnlinkProtectsOnlyLoginMethod(t *testing.T) {
	service := NewLineOAuthService(&fakeLineStore{unlinkErr: repositories.ErrCannotUnlinkOnlyLogin}, nil, nil, LineOAuthConfig{})
	err := service.Unlink(context.Background(), "user-1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != "CANNOT_UNLINK_ONLY_LOGIN_METHOD" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLineLoginRejectsInvalidState(t *testing.T) {
	service := NewLineOAuthService(nil, nil, nil, LineOAuthConfig{
		StateSecret: strings.Repeat("l", 32), Issuer: "test", StateTTL: 10 * time.Minute,
	})
	_, err := service.Login(context.Background(), "code", "invalid", "", "")
	if err == nil {
		t.Fatal("expected invalid state error")
	}
	appErr, ok := err.(*Error)
	if !ok || appErr.Code != "INVALID_OAUTH_STATE" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
