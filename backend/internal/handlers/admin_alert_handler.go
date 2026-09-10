package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type AdminAlertHandler struct{ service *services.AdminAlertService }

func NewAdminAlertHandler(service *services.AdminAlertService) *AdminAlertHandler {
	return &AdminAlertHandler{service: service}
}

func (h *AdminAlertHandler) List(c echo.Context) error {
	page, err := h.service.List(c.Request().Context(), c.QueryParam("symbol"), c.QueryParam("status"), c.QueryParam("user_id"), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": page.Alerts, "meta": map[string]any{"page": page.Page, "limit": page.Limit, "total": page.Total}})
}

func (h *AdminAlertHandler) Get(c echo.Context) error {
	alert, err := h.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, alert, "")
}

func (h *AdminAlertHandler) Disable(c echo.Context) error {
	var request models.DisableAlertRequest
	if err := c.Bind(&request); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	if err := h.service.Disable(c.Request().Context(), adminAction(c), c.Param("id"), request.Reason); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, nil, "Alert disabled successfully.")
}
