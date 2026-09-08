package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const lineReplyURL = "https://api.line.me/v2/bot/message/reply"
const linePushURL = "https://api.line.me/v2/bot/message/push"

type LineMessenger interface {
	ReplyText(context.Context, string, string) error
}

type LinePusher interface {
	PushText(context.Context, string, string) error
}

type LineMessagingClient struct {
	httpClient  *http.Client
	accessToken string
	replyURL    string
}

func NewLineMessagingClient(client *http.Client, accessToken string) *LineMessagingClient {
	return &LineMessagingClient{httpClient: client, accessToken: accessToken, replyURL: lineReplyURL}
}

func (c *LineMessagingClient) ReplyText(ctx context.Context, replyToken, text string) error {
	return c.sendText(ctx, c.replyURL, map[string]any{"replyToken": replyToken}, text)
}

func (c *LineMessagingClient) PushText(ctx context.Context, to, text string) error {
	return c.sendText(ctx, linePushURL, map[string]any{"to": to}, text)
}

func (c *LineMessagingClient) sendText(ctx context.Context, endpoint string, envelope map[string]any, text string) error {
	envelope["messages"] = []map[string]string{{"type": "text", "text": text}}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode LINE reply: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create LINE reply request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send LINE reply: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("LINE Messaging API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
