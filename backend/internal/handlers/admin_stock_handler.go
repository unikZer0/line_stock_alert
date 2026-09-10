package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type AdminStockHandler struct{ service *services.AdminStockService }

func NewAdminStockHandler(service *services.AdminStockService) *AdminStockHandler {
	return &AdminStockHandler{service: service}
}
func (h *AdminStockHandler) List(c echo.Context) error {
	page, err := h.service.List(c.Request().Context(), c.QueryParam("search"), c.QueryParam("status"), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": page.Stocks, "meta": map[string]any{"page": page.Page, "limit": page.Limit, "total": page.Total}})
}
func (h *AdminStockHandler) Get(c echo.Context) error {
	stock, err := h.service.Get(c.Request().Context(), c.Param("symbol"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, stock, "")
}
func (h *AdminStockHandler) Enable(c echo.Context) error {
	if err := h.service.Enable(c.Request().Context(), adminAction(c), c.Param("symbol")); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, nil, "Stock enabled successfully.")
}
func (h *AdminStockHandler) Disable(c echo.Context) error {
	var request models.DisableStockRequest
	if err := c.Bind(&request); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	if err := h.service.Disable(c.Request().Context(), adminAction(c), c.Param("symbol"), request.Reason); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, nil, "Stock disabled successfully.")
}
