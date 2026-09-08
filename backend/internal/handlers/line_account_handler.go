package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type LineAccountHandler struct {
	service *services.LineOAuthService
}

func NewLineAccountHandler(service *services.LineOAuthService) *LineAccountHandler {
	return &LineAccountHandler{service: service}
}

func (h *LineAccountHandler) Connect(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	authorizationURL, err := h.service.ConnectAuthorizationURL(c.Request().Context(), userID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, map[string]string{"authorization_url": authorizationURL}, "")
}

func (h *LineAccountHandler) Callback(c echo.Context) error {
	if c.QueryParam("error") != "" {
		return utils.JSONError(c, http.StatusBadRequest, "LINE_AUTH_CANCELLED", "LINE account linking was cancelled.", nil)
	}
	if err := h.service.Link(c.Request().Context(), c.QueryParam("code"), c.QueryParam("state")); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, map[string]any{"provider": "LINE", "linked": true}, "LINE account connected successfully.")
}

func (h *LineAccountHandler) Unlink(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	if err := h.service.Unlink(c.Request().Context(), userID); err != nil {
		return handleServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
