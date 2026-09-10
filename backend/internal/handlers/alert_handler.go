package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type AlertHandler struct{ service *services.AlertService }

func NewAlertHandler(service *services.AlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

func (h *AlertHandler) List(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	alerts, err := h.service.List(c.Request().Context(), userID, c.QueryParam("symbol"), c.QueryParam("status"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, alerts, "")
}

func (h *AlertHandler) Create(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	var request models.CreateAlertRequest
	if err := c.Bind(&request); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	alert, err := h.service.Create(c.Request().Context(), userID, request)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusCreated, alert, "Alert created successfully.")
}

func (h *AlertHandler) Update(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	var request models.UpdateAlertRequest
	if err := c.Bind(&request); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	alert, err := h.service.Update(c.Request().Context(), userID, c.Param("id"), request)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, alert, "Alert updated successfully.")
}

func (h *AlertHandler) Delete(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	if err := h.service.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return handleServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AlertHandler) Rearm(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	alert, err := h.service.Rearm(c.Request().Context(), userID, c.Param("id"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, alert, "Alert re-armed successfully.")
}
