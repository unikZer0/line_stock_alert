package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type StockHandler struct{ service *services.StockService }

func NewStockHandler(service *services.StockService) *StockHandler {
	return &StockHandler{service: service}
}

func (h *StockHandler) Search(c echo.Context) error {
	page, err := h.service.Search(c.Request().Context(), c.QueryParam("search"), c.QueryParam("page"), c.QueryParam("limit"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(http.StatusOK, struct {
		Success bool `json:"success"`
		Data    any  `json:"data"`
		Meta    any  `json:"meta"`
	}{Success: true, Data: page.Stocks, Meta: map[string]int{"page": page.Page, "limit": page.Limit, "total": page.Total}})
}

func (h *StockHandler) Quote(c echo.Context) error {
	quote, err := h.service.Quote(c.Request().Context(), c.Param("symbol"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, quote, "")
}

func (h *StockHandler) Quotes(c echo.Context) error {
	quotes, err := h.service.Quotes(c.Request().Context(), c.QueryParam("symbols"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, quotes, "")
}

func (h *StockHandler) Candles(c echo.Context) error {
	series, err := h.service.Candles(c.Request().Context(), c.Param("symbol"), c.QueryParam("range"))
	if err != nil {
		return handleServiceError(c, err)
	}
	return utils.JSONSuccess(c, http.StatusOK, series, "")
}
