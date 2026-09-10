package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type AdminMonitoringHandler struct {
	service *services.AdminMonitoringService
}

func NewAdminMonitoringHandler(service *services.AdminMonitoringService) *AdminMonitoringHandler {
	return &AdminMonitoringHandler{service: service}
}
func (h *AdminMonitoringHandler) LineStats(c echo.Context) error {
	stats, err := h.service.LineStats(c.Request().Context())
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, stats, "")
}
func (h *AdminMonitoringHandler) FailedLineMessages(c echo.Context) error {
	items, page, limit, total, err := h.service.FailedLineMessages(c.Request().Context(), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return paginatedAdminResponse(c, items, page, limit, total)
}
func (h *AdminMonitoringHandler) ApplicationLogs(c echo.Context) error {
	page, err := h.service.ApplicationLogs(c.Request().Context(), c.QueryParam("level"), c.QueryParam("service"), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return paginatedAdminResponse(c, page.Logs, page.Page, page.Limit, page.Total)
}
func (h *AdminMonitoringHandler) AuditLogs(c echo.Context) error {
	page, err := h.service.AuditLogs(c.Request().Context(), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return paginatedAdminResponse(c, page.Logs, page.Page, page.Limit, page.Total)
}
func paginatedAdminResponse(c echo.Context, data any, page, limit int, total int64) error {
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": data, "meta": map[string]any{"page": page, "limit": limit, "total": total}})
}
