package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type WatchlistHandler struct{ service *services.WatchlistService }

func NewWatchlistHandler(service *services.WatchlistService) *WatchlistHandler {
	return &WatchlistHandler{service: service}
}

func (h *WatchlistHandler) List(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	items, err := h.service.List(c.Request().Context(), userID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, items, "")
}

func (h *WatchlistHandler) Add(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	var request models.AddWatchlistRequest
	if err := c.Bind(&request); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	item, err := h.service.Add(c.Request().Context(), userID, request)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusCreated, item, "Stock added to watchlist.")
}

func (h *WatchlistHandler) Delete(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	if err := h.service.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return handleServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
