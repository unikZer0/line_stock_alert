package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) CurrentUser(c echo.Context) error {
	userID, ok := c.Get(authmw.UserIDKey).(string)
	if !ok || userID == "" {
		return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	}
	user, err := h.service.CurrentUser(c.Request().Context(), userID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, user, "")
}
