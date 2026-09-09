package services

import (
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"stock_linebot/backend/internal/models"
)

type richMenuAuditStub struct{ id string }

func (s *richMenuAuditStub) RecordPublish(_ context.Context, _ models.AdminActionContext, id string) error {
	s.id = id
	return nil
}

func TestRichMenuPublishWorkflow(t *testing.T) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/v2/bot/richmenu" {
			_, _ = w.Write([]byte(`{"richMenuId":"richmenu-test"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	audit := &richMenuAuditStub{}
	service := NewRichMenuService(server.Client(), "token", "https://example.com", audit)
	service.apiBase, service.dataBase = server.URL, server.URL
	result, err := service.Publish(context.Background(), models.AdminActionContext{}, testPNG(2500, 843))
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if result.RichMenuID != "richmenu-test" || audit.id != "richmenu-test" || len(requests) != 3 {
		t.Fatalf("unexpected result: %+v requests=%v audit=%s", result, requests, audit.id)
	}
}

func TestRichMenuRejectsWrongDimensions(t *testing.T) {
	service := NewRichMenuService(http.DefaultClient, "token", "https://example.com", &richMenuAuditStub{})
	_, err := service.Publish(context.Background(), models.AdminActionContext{}, testPNG(100, 100))
	appErr, ok := err.(*Error)
	if !ok || appErr.Code != "INVALID_RICH_MENU_IMAGE" {
		t.Fatalf("expected invalid image error, got %v", err)
	}
}

func testPNG(width, height int) []byte {
	data := make([]byte, 24)
	copy(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10})
	copy(data[12:16], "IHDR")
	binary.BigEndian.PutUint32(data[16:20], uint32(width))
	binary.BigEndian.PutUint32(data[20:24], uint32(height))
	return data
}
