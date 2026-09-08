package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type accessStoreStub struct {
	claimed   bool
	userID    string
	status    string
	lookupErr error
	processed bool
	failed    bool
}

func (s *accessStoreStub) ClaimEvent(context.Context, string, string, string, json.RawMessage) (bool, error) {
	return s.claimed, nil
}
func (s *accessStoreStub) FindUserByLineID(context.Context, string) (string, string, error) {
	return s.userID, s.status, s.lookupErr
}
func (s *accessStoreStub) MarkProcessed(context.Context, string) error {
	s.processed = true
	return nil
}
func (s *accessStoreStub) MarkFailed(context.Context, string, string) error {
	s.failed = true
	return nil
}

type accessMessengerStub struct{ text string }

func (m *accessMessengerStub) ReplyText(_ context.Context, _ string, text string) error {
	m.text = text
	return nil
}

func accessWebhookRequest() models.LineWebhookRequest {
	return models.LineWebhookRequest{Events: []models.LineWebhookEvent{{
		Type: "message", WebhookEventID: "event-access", ReplyToken: "reply-token",
		Source:  models.LineEventSource{UserID: "line-user"},
		Message: &models.LineMessage{Type: "text", Text: "HELP"},
	}}}
}

func TestLineAccessUnlinkedFriendMustRegisterOrConnect(t *testing.T) {
	store := &accessStoreStub{claimed: true, lookupErr: repositories.ErrNotFound}
	messenger := &accessMessengerStub{}
	err := NewLineWebhookService(store, messenger, "http://localhost:3000").Process(context.Background(), accessWebhookRequest())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(messenger.text, "register") || !strings.Contains(messenger.text, "connect LINE") {
		t.Fatalf("reply = %q", messenger.text)
	}
	if !store.processed || store.failed {
		t.Fatalf("processed=%v failed=%v", store.processed, store.failed)
	}
}

func TestLineAccessDisabledUserIsDenied(t *testing.T) {
	store := &accessStoreStub{claimed: true, userID: "user-1", status: "DISABLED"}
	messenger := &accessMessengerStub{}
	err := NewLineWebhookService(store, messenger, "http://localhost:3000").Process(context.Background(), accessWebhookRequest())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(strings.ToLower(messenger.text), "disabled") {
		t.Fatalf("reply = %q", messenger.text)
	}
}

func TestLineAccessActiveLinkedUserIsAllowed(t *testing.T) {
	store := &accessStoreStub{claimed: true, userID: "user-1", status: "ACTIVE"}
	messenger := &accessMessengerStub{}
	err := NewLineWebhookService(store, messenger, "http://localhost:3000").Process(context.Background(), accessWebhookRequest())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(messenger.text, "ABOVE/BELOW") {
		t.Fatalf("reply = %q", messenger.text)
	}
}

func TestLineAccessUnexpectedLookupFailureIsNotGranted(t *testing.T) {
	store := &accessStoreStub{claimed: true, lookupErr: errors.New("database unavailable")}
	messenger := &accessMessengerStub{}
	err := NewLineWebhookService(store, messenger, "http://localhost:3000").Process(context.Background(), accessWebhookRequest())
	if err == nil || messenger.text != "" || !store.failed {
		t.Fatalf("error=%v reply=%q failed=%v", err, messenger.text, store.failed)
	}
}
