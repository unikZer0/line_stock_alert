package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type AdminDashboardHandler struct {
	service *services.AdminDashboardService
}

func NewAdminDashboardHandler(service *services.AdminDashboardService) *AdminDashboardHandler {
	return &AdminDashboardHandler{service: service}
}

func (h *AdminDashboardHandler) Dashboard(c echo.Context) error {
	dashboard, err := h.service.Dashboard(c.Request().Context())
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, dashboard, "")
}
