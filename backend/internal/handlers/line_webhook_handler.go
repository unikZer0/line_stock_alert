package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type LineWebhookHandler struct{ service *services.LineWebhookService }

func NewLineWebhookHandler(service *services.LineWebhookService) *LineWebhookHandler {
	return &LineWebhookHandler{service: service}
}

func (h *LineWebhookHandler) Receive(c echo.Context) error {
	body, ok := c.Get(authmw.LineWebhookBodyKey).([]byte)
	if !ok {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_WEBHOOK_PAYLOAD", "The LINE webhook payload is invalid.", nil)
	}
	var request models.LineWebhookRequest
	if err := json.Unmarshal(body, &request); err != nil || request.Events == nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_WEBHOOK_PAYLOAD", "The LINE webhook payload is invalid.", nil)
	}
	if err := h.service.Process(c.Request().Context(), request); err != nil {
		return handleServiceError(c, err)
	}
	return c.NoContent(http.StatusOK)
}
