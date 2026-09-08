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

func (h *AuthHandler) Register(c echo.Context) error {
	var req models.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	result, err := h.service.Register(c.Request().Context(), req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusCreated, result, "Registration successful. Please verify your email.")
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req models.VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	if err := h.service.VerifyEmail(c.Request().Context(), req); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, map[string]bool{"email_verified": true}, "Email verified successfully.")
}

func (h *AuthHandler) ResendOTP(c echo.Context) error {
	var req models.ResendOTPRequest
	if err := c.Bind(&req); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	expiresIn, err := h.service.ResendOTP(c.Request().Context(), req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, map[string]int64{"expires_in": expiresIn}, "Verification code sent successfully.")
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
		"INVALID_REQUEST": http.StatusBadRequest, "INVALID_OTP": http.StatusBadRequest,
		"OTP_EXPIRED": http.StatusBadRequest, "OTP_ALREADY_USED": http.StatusBadRequest,
		"VALIDATION_ERROR": http.StatusUnprocessableEntity, "EMAIL_ALREADY_EXISTS": http.StatusConflict,
		"INVALID_CREDENTIALS": http.StatusUnauthorized, "INVALID_REFRESH_TOKEN": http.StatusUnauthorized,
		"REFRESH_TOKEN_EXPIRED": http.StatusUnauthorized, "REFRESH_TOKEN_REVOKED": http.StatusUnauthorized,
		"EMAIL_NOT_VERIFIED": http.StatusForbidden, "ACCOUNT_DISABLED": http.StatusForbidden,
		"USER_NOT_FOUND": http.StatusNotFound, "OTP_MAX_ATTEMPTS_EXCEEDED": http.StatusTooManyRequests,
		"OTP_RATE_LIMITED": http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED": http.StatusTooManyRequests,
		"LOGIN_RATE_LIMITED":              http.StatusTooManyRequests,
		"EMAIL_SERVICE_UNAVAILABLE":       http.StatusServiceUnavailable,
		"INVALID_OAUTH_STATE":             http.StatusBadRequest,
		"LINE_AUTH_CANCELLED":             http.StatusBadRequest,
		"INVALID_OAUTH_CODE":              http.StatusBadRequest,
		"LINE_AUTH_PROVIDER_ERROR":        http.StatusBadGateway,
		"LINE_AUTH_PROVIDER_UNAVAILABLE":  http.StatusServiceUnavailable,
		"USER_ALREADY_HAS_LINE":           http.StatusConflict,
		"LINE_ALREADY_LINKED":             http.StatusConflict,
		"CANNOT_UNLINK_ONLY_LOGIN_METHOD": http.StatusBadRequest,
		"LINE_USER_NOT_FOUND":             http.StatusNotFound,
		"WATCHLIST_LIMIT_REACHED":         http.StatusForbidden,
		"STOCK_NOT_FOUND":                 http.StatusNotFound,
		"WATCHLIST_NOT_FOUND":             http.StatusNotFound,
		"STOCK_ALREADY_IN_WATCHLIST":      http.StatusConflict,
		"STOCK_PROVIDER_RATE_LIMIT":       http.StatusTooManyRequests,
		"STOCK_PROVIDER_UNAVAILABLE":      http.StatusServiceUnavailable,
		"STOCK_PROVIDER_TIMEOUT":          http.StatusGatewayTimeout,
		"INVALID_FILTER":                  http.StatusUnprocessableEntity,
		"ALERT_LIMIT_REACHED":             http.StatusForbidden,
		"ALERT_ALREADY_EXISTS":            http.StatusConflict,
		"ALERT_NOT_FOUND":                 http.StatusNotFound,
		"INVALID_ALERT_CONDITION":         http.StatusUnprocessableEntity,
		"INVALID_TARGET_PRICE":            http.StatusUnprocessableEntity,
	}[appErr.Code]
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return utils.JSONError(c, status, appErr.Code, appErr.Message, nil)
}
