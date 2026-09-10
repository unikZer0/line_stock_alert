package handlers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type RichMenuHandler struct{ service *services.RichMenuService }

func NewRichMenuHandler(service *services.RichMenuService) *RichMenuHandler {
	return &RichMenuHandler{service: service}
}

func (h *RichMenuHandler) Publish(c echo.Context) error {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_RICH_MENU_IMAGE", "An image file is required.", nil)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_RICH_MENU_IMAGE", "The image could not be read.", nil)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil {
		return utils.JSONError(c, http.StatusBadRequest, "INVALID_RICH_MENU_IMAGE", "The image could not be read.", nil)
	}
	result, err := h.service.Publish(c.Request().Context(), adminAction(c), data)
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusCreated, result, "Rich menu published successfully.")
}
