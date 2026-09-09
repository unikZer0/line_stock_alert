package services

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"stock_linebot/backend/internal/models"
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
	auth := NewAuthService(fakeRefreshStore{}, AuthConfig{
		AccessSecret: strings.Repeat("a", 32), RefreshSecret: strings.Repeat("r", 32),
		Issuer: "test", AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour,
	})
	auth.now = func() time.Time { return now }
	provider := &fakeLineProvider{identity: LineIdentity{UserID: "U123", DisplayName: "Line User"}}
	store := &fakeLineStore{user: models.User{ID: "user-1", Role: "USER", Status: "ACTIVE"}}
	service := NewLineOAuthService(store, provider, auth, LineOAuthConfig{
		ChannelID: "channel-id", CallbackURL: "http://localhost:8080/api/v1/auth/line/callback",
		StateSecret: secret, Issuer: "test", StateTTL: 10 * time.Minute,
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
