package models

import "encoding/json"

type LineWebhookRequest struct {
	Destination string             `json:"destination"`
	Events      []LineWebhookEvent `json:"events"`
}

type LineWebhookEvent struct {
	Type           string          `json:"type"`
	WebhookEventID string          `json:"webhookEventId"`
	ReplyToken     string          `json:"replyToken"`
	Timestamp      int64           `json:"timestamp"`
	Source         LineEventSource `json:"source"`
	Message        *LineMessage    `json:"message,omitempty"`
	Raw            json.RawMessage `json:"-"`
}

type LineEventSource struct {
	Type   string `json:"type"`
	UserID string `json:"userId"`
}

type LineMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}
