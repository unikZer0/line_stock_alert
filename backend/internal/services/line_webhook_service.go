package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/repositories"
)

type LineWebhookStore interface {
	ClaimEvent(context.Context, string, string, string, json.RawMessage) (bool, error)
	FindUserByLineID(context.Context, string) (string, string, error)
	MarkProcessed(context.Context, string) error
	MarkFailed(context.Context, string, string) error
}

type LineWebhookService struct {
	store       LineWebhookStore
	messenger   LineMessenger
	frontendURL string
}

func NewLineWebhookService(store LineWebhookStore, messenger LineMessenger, frontendURL string) *LineWebhookService {
	return &LineWebhookService{store: store, messenger: messenger, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (s *LineWebhookService) Process(ctx context.Context, request models.LineWebhookRequest) error {
	for _, event := range request.Events {
		payload, err := json.Marshal(event)
		if err != nil {
			return newError("INVALID_WEBHOOK_PAYLOAD", "The LINE webhook payload is invalid.", err)
		}
		eventID := event.WebhookEventID
		if eventID == "" {
			sum := sha256.Sum256(payload)
			eventID = "legacy-" + hex.EncodeToString(sum[:])
		}
		claimed, err := s.store.ClaimEvent(ctx, eventID, event.Source.UserID, event.Type, payload)
		if err != nil {
			return newError("INTERNAL_SERVER_ERROR", "Could not record the LINE webhook event.", err)
		}
		if !claimed {
			continue
		}
		if err = s.processEvent(ctx, event); err != nil {
			_ = s.store.MarkFailed(ctx, eventID, truncateWebhookError(err.Error()))
			return newError("INTERNAL_SERVER_ERROR", "Could not process the LINE webhook event.", err)
		}
		if err = s.store.MarkProcessed(ctx, eventID); err != nil {
			return newError("INTERNAL_SERVER_ERROR", "Could not complete the LINE webhook event.", err)
		}
	}
	return nil
}

func (s *LineWebhookService) processEvent(ctx context.Context, event models.LineWebhookEvent) error {
	if event.ReplyToken == "" || (event.Type != "message" && event.Type != "follow") {
		return nil
	}
	if event.Type == "message" && (event.Message == nil || event.Message.Type != "text") {
		return nil
	}
	_, status, err := s.store.FindUserByLineID(ctx, event.Source.UserID)
	if errors.Is(err, repositories.ErrNotFound) {
		return s.messenger.ReplyText(ctx, event.ReplyToken, "Welcome to Stock Alert. Open Stocks from the rich menu to continue with LINE Login: "+s.frontendURL+"/stocks")
	}
	if err != nil {
		return fmt.Errorf("look up LINE account: %w", err)
	}
	if status != "ACTIVE" {
		return s.messenger.ReplyText(ctx, event.ReplyToken, "Your Stock Alert account is currently disabled.")
	}
	message := "Open Stocks or My Alerts from the rich menu. Stocks: " + s.frontendURL + "/stocks\nMy Alerts: " + s.frontendURL + "/alerts"
	if event.Type == "message" && strings.EqualFold(strings.TrimSpace(event.Message.Text), "HELP") {
		message = "Use Stocks to view US quotes and create ABOVE/BELOW alerts. Use My Alerts to edit or delete them.\nStocks: " + s.frontendURL + "/stocks\nMy Alerts: " + s.frontendURL + "/alerts"
	}
	return s.messenger.ReplyText(ctx, event.ReplyToken, message)
}

func truncateWebhookError(value string) string {
	const maximum = 2000
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}
