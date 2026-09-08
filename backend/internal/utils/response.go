package utils

import "github.com/labstack/echo/v4"

type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   ErrorBody `json:"error"`
}

func JSONSuccess(c echo.Context, status int, data any, message string) error {
	return c.JSON(status, SuccessResponse{Success: true, Data: data, Message: message})
}

func JSONError(c echo.Context, status int, code, message string, details any) error {
	return c.JSON(status, ErrorResponse{
		Success: false,
		Error:   ErrorBody{Code: code, Message: message, Details: details},
	})
}
