package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	lineTokenURL  = "https://api.line.me/oauth2/v2.1/token"
	lineVerifyURL = "https://api.line.me/oauth2/v2.1/verify"
)

type LineIdentity struct {
	UserID      string
	DisplayName string
}

type LineProvider interface {
	ExchangeAndVerify(context.Context, string, string, string) (LineIdentity, error)
}

type LineClient struct {
	httpClient               *http.Client
	channelID, channelSecret string
}

func NewLineClient(client *http.Client, channelID, channelSecret string) *LineClient {
	return &LineClient{httpClient: client, channelID: channelID, channelSecret: channelSecret}
}

func (c *LineClient) ExchangeAndVerify(ctx context.Context, code, nonce, callbackURL string) (LineIdentity, error) {
	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {callbackURL},
		"client_id": {c.channelID}, "client_secret": {c.channelSecret},
	}
	var tokenResponse struct {
		IDToken string `json:"id_token"`
	}
	if err := c.postForm(ctx, lineTokenURL, form, &tokenResponse); err != nil {
		return LineIdentity{}, fmt.Errorf("exchange LINE authorization code: %w", err)
	}
	if tokenResponse.IDToken == "" {
		return LineIdentity{}, fmt.Errorf("LINE token response did not include an ID token")
	}

	form = url.Values{"id_token": {tokenResponse.IDToken}, "client_id": {c.channelID}, "nonce": {nonce}}
	var identity struct {
		Subject string `json:"sub"`
		Name    string `json:"name"`
	}
	if err := c.postForm(ctx, lineVerifyURL, form, &identity); err != nil {
		return LineIdentity{}, fmt.Errorf("verify LINE ID token: %w", err)
	}
	if identity.Subject == "" {
		return LineIdentity{}, fmt.Errorf("verified LINE ID token did not include a subject")
	}
	return LineIdentity{UserID: identity.Subject, DisplayName: identity.Name}, nil
}

func (c *LineClient) postForm(ctx context.Context, endpoint string, form url.Values, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("LINE returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode LINE response: %w", err)
	}
	return nil
}
