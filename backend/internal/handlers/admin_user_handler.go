package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/models"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

type AdminUserHandler struct{ service *services.AdminUserService }

func NewAdminUserHandler(service *services.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{service: service}
}

func (h *AdminUserHandler) List(c echo.Context) error {
	page, err := h.service.List(c.Request().Context(), c.QueryParam("search"), c.QueryParam("status"), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": page.Users,
		"meta": map[string]any{"page": page.Page, "limit": page.Limit, "total": page.Total}})
}

func (h *AdminUserHandler) Get(c echo.Context) error {
	user, err := h.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, user, "")
}

func (h *AdminUserHandler) Disable(c echo.Context) error {
	var request models.DisableUserRequest
	if err := c.Bind(&request); err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.", nil)
	}
	if err := h.service.Disable(c.Request().Context(), adminAction(c), c.Param("id"), request.Reason); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, nil, "User disabled successfully.")
}

func (h *AdminUserHandler) Enable(c echo.Context) error {
	if err := h.service.Enable(c.Request().Context(), adminAction(c), c.Param("id")); err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, nil, "User enabled successfully.")
}

func adminAction(c echo.Context) models.AdminActionContext {
	adminID, _ := c.Get(authmw.UserIDKey).(string)
	return models.AdminActionContext{AdminUserID: adminID, IPAddress: c.RealIP(), UserAgent: c.Request().UserAgent()}
}
