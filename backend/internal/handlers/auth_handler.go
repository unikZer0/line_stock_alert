package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	result, err := h.service.Login(c.Request().Context(), req, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, result, "")
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req models.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	result, err := h.service.Refresh(c.Request().Context(), req.RefreshToken, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, result, "")
}

func (h *AuthHandler) Logout(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	if err := h.service.Logout(c.Request().Context(), userID); err != nil {
		return handleServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func handleServiceError(c echo.Context, err error) error {
	var appErr *services.Error
	if !errors.As(err, &appErr) {
		return utils.JSONError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Something went wrong. Please try again later.", nil)
	}
	status := map[string]int{
		"INVALID_REQUEST":     http.StatusBadRequest,
		"VALIDATION_ERROR":    http.StatusUnprocessableEntity,
		"INVALID_CREDENTIALS": http.StatusUnauthorized, "INVALID_REFRESH_TOKEN": http.StatusUnauthorized,
		"REFRESH_TOKEN_EXPIRED": http.StatusUnauthorized, "REFRESH_TOKEN_REVOKED": http.StatusUnauthorized,
		"EMAIL_NOT_VERIFIED": http.StatusForbidden, "ACCOUNT_DISABLED": http.StatusForbidden,
		"USER_NOT_FOUND": http.StatusNotFound, "RATE_LIMIT_EXCEEDED": http.StatusTooManyRequests,
		"LOGIN_RATE_LIMITED":             http.StatusTooManyRequests,
		"ADMIN_LOGIN_REQUIRED":           http.StatusForbidden,
		"INVALID_OAUTH_STATE":            http.StatusBadRequest,
		"LINE_AUTH_CANCELLED":            http.StatusBadRequest,
		"INVALID_OAUTH_CODE":             http.StatusBadRequest,
		"LINE_AUTH_PROVIDER_ERROR":       http.StatusBadGateway,
		"LINE_AUTH_PROVIDER_UNAVAILABLE": http.StatusServiceUnavailable,
		"STOCK_NOT_FOUND":                http.StatusNotFound,
		"STOCK_PROVIDER_RATE_LIMIT":      http.StatusTooManyRequests,
		"STOCK_PROVIDER_UNAVAILABLE":     http.StatusServiceUnavailable,
		"STOCK_PROVIDER_TIMEOUT":         http.StatusGatewayTimeout,
		"INVALID_CANDLE_RANGE":           http.StatusUnprocessableEntity,
		"CANDLE_DATA_NOT_FOUND":          http.StatusNotFound,
		"CANDLE_PROVIDER_RATE_LIMIT":     http.StatusTooManyRequests,
		"CANDLE_PROVIDER_UNAVAILABLE":    http.StatusServiceUnavailable,
		"CANDLE_PROVIDER_TIMEOUT":        http.StatusGatewayTimeout,
		"INVALID_FILTER":                 http.StatusUnprocessableEntity,
		"ALERT_LIMIT_REACHED":            http.StatusForbidden,
		"ALERT_ALREADY_EXISTS":           http.StatusConflict,
		"ALERT_NOT_FOUND":                http.StatusNotFound,
		"ALERT_NOT_TRIGGERED":            http.StatusConflict,
		"ALERT_CONDITION_STILL_MET":      http.StatusConflict,
		"INVALID_ALERT_CONDITION":        http.StatusUnprocessableEntity,
		"INVALID_TARGET_PRICE":           http.StatusUnprocessableEntity,
		"INVALID_WEBHOOK_PAYLOAD":        http.StatusBadRequest,
		"ADMIN_FORBIDDEN":                http.StatusForbidden,
		"ADMIN_USER_NOT_FOUND":           http.StatusNotFound,
		"ADMIN_ACTION_NOT_ALLOWED":       http.StatusBadRequest,
		"ADMIN_ALERT_NOT_FOUND":          http.StatusNotFound,
		"ADMIN_STOCK_NOT_FOUND":          http.StatusNotFound,
	}[appErr.Code]
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return utils.JSONError(c, status, appErr.Code, appErr.Message, nil)
}
