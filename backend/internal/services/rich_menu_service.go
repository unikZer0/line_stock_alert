package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"stock_linebot/backend/internal/models"
)

const maxRichMenuImageBytes = 1024 * 1024

type RichMenuAuditStore interface {
	RecordPublish(context.Context, models.AdminActionContext, string) error
}

type RichMenuService struct {
	client                                *http.Client
	token, frontendURL, apiBase, dataBase string
	audit                                 RichMenuAuditStore
}

func NewRichMenuService(client *http.Client, token, frontendURL string, audit RichMenuAuditStore) *RichMenuService {
	return &RichMenuService{client: client, token: token, frontendURL: strings.TrimRight(frontendURL, "/"), apiBase: "https://api.line.me", dataBase: "https://api-data.line.me", audit: audit}
}

func (s *RichMenuService) Publish(ctx context.Context, action models.AdminActionContext, imageBytes []byte) (models.RichMenuPublishResult, error) {
	if len(imageBytes) == 0 || len(imageBytes) > maxRichMenuImageBytes {
		return models.RichMenuPublishResult{}, newError("INVALID_RICH_MENU_IMAGE", "Rich menu image must be no larger than 1 MB.", nil)
	}
	contentType, width, height, err := richMenuImageInfo(imageBytes)
	if err != nil {
		return models.RichMenuPublishResult{}, newError("INVALID_RICH_MENU_IMAGE", "The file is not a valid PNG or JPEG image.", err)
	}
	if width != 2500 || height != 843 {
		return models.RichMenuPublishResult{}, newError("INVALID_RICH_MENU_IMAGE", fmt.Sprintf("Rich menu image is %d × %d; it must be exactly 2500 × 843 pixels.", width, height), nil)
	}
	frontend, err := url.Parse(s.frontendURL)
	if err != nil || frontend.Scheme != "https" || frontend.Host == "" {
		return models.RichMenuPublishResult{}, newError("INVALID_FRONTEND_URL", "FRONTEND_URL must be a public HTTPS URL before publishing a rich menu.", err)
	}
	richMenuID, err := s.create(ctx)
	if err != nil {
		return models.RichMenuPublishResult{}, lineRichMenuError(err)
	}
	if err = s.upload(ctx, richMenuID, imageBytes, contentType); err != nil {
		_ = s.delete(ctx, richMenuID)
		return models.RichMenuPublishResult{}, lineRichMenuError(err)
	}
	if err = s.setDefault(ctx, richMenuID); err != nil {
		_ = s.delete(ctx, richMenuID)
		return models.RichMenuPublishResult{}, lineRichMenuError(err)
	}
	if err = s.audit.RecordPublish(ctx, action, richMenuID); err != nil {
		return models.RichMenuPublishResult{}, newError("INTERNAL_SERVER_ERROR", "Rich menu was published, but its audit record failed.", err)
	}
	return models.RichMenuPublishResult{RichMenuID: richMenuID, Status: "DEFAULT"}, nil
}

func (s *RichMenuService) create(ctx context.Context) (string, error) {
	payload := map[string]any{"size": map[string]int{"width": 2500, "height": 843}, "selected": true, "name": "Alert Bot Main Menu", "chatBarText": "Alert Bot Menu", "areas": []any{
		map[string]any{"bounds": map[string]int{"x": 0, "y": 0, "width": 834, "height": 843}, "action": map[string]string{"type": "uri", "label": "Market", "uri": s.frontendURL + "/stocks"}},
		map[string]any{"bounds": map[string]int{"x": 834, "y": 0, "width": 833, "height": 843}, "action": map[string]string{"type": "uri", "label": "My Alerts", "uri": s.frontendURL + "/alerts"}},
		map[string]any{"bounds": map[string]int{"x": 1667, "y": 0, "width": 833, "height": 843}, "action": map[string]string{"type": "message", "label": "Help", "text": "HELP"}},
	}}
	body, _ := json.Marshal(payload)
	response, err := s.request(ctx, http.MethodPost, s.apiBase+"/v2/bot/richmenu", "application/json", body)
	if err != nil {
		return "", err
	}
	var result struct {
		RichMenuID string `json:"richMenuId"`
	}
	if err = json.Unmarshal(response, &result); err != nil {
		return "", fmt.Errorf("decode LINE rich menu response: %w", err)
	}
	if result.RichMenuID == "" {
		return "", fmt.Errorf("LINE rich menu response did not include an ID")
	}
	return result.RichMenuID, nil
}

func (s *RichMenuService) upload(ctx context.Context, id string, data []byte, contentType string) error {
	_, err := s.request(ctx, http.MethodPost, s.dataBase+"/v2/bot/richmenu/"+url.PathEscape(id)+"/content", contentType, data)
	return err
}
func (s *RichMenuService) setDefault(ctx context.Context, id string) error {
	_, err := s.request(ctx, http.MethodPost, s.apiBase+"/v2/bot/user/all/richmenu/"+url.PathEscape(id), "application/json", nil)
	return err
}
func (s *RichMenuService) delete(ctx context.Context, id string) error {
	_, err := s.request(ctx, http.MethodDelete, s.apiBase+"/v2/bot/richmenu/"+url.PathEscape(id), "application/json", nil)
	return err
}

func (s *RichMenuService) request(ctx context.Context, method, endpoint, contentType string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", contentType)
	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("LINE returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func lineRichMenuError(err error) error {
	return newError("LINE_RICH_MENU_PROVIDER_ERROR", "LINE could not publish the rich menu.", err)
}

func richMenuImageInfo(data []byte) (string, int, int, error) {
	if len(data) >= 24 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) && string(data[12:16]) == "IHDR" {
		return "image/png", int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24])), nil
	}
	if len(data) >= 4 && data[0] == 0xff && data[1] == 0xd8 {
		for offset := 2; offset+9 < len(data); {
			if data[offset] != 0xff {
				offset++
				continue
			}
			marker := data[offset+1]
			if marker == 0xd8 || marker == 0xd9 {
				offset += 2
				continue
			}
			if offset+4 > len(data) {
				break
			}
			length := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
			if length < 2 || offset+2+length > len(data) {
				break
			}
			if marker >= 0xc0 && marker <= 0xc3 && length >= 7 {
				return "image/jpeg", int(binary.BigEndian.Uint16(data[offset+7 : offset+9])), int(binary.BigEndian.Uint16(data[offset+5 : offset+7])), nil
			}
			offset += 2 + length
		}
	}
	return "", 0, 0, fmt.Errorf("unsupported or malformed image")
}
