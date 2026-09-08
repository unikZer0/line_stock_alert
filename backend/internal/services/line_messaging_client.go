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

type LineMessenger interface {
	ReplyText(context.Context, string, string) error
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
	payload, err := json.Marshal(map[string]any{"replyToken": replyToken, "messages": []map[string]string{{"type": "text", "text": text}}})
	if err != nil {
		return fmt.Errorf("encode LINE reply: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.replyURL, bytes.NewReader(payload))
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
