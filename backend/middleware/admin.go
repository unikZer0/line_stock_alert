package middleware

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/utils"
)

type AdminAccessStore interface {
	FindUserAccess(context.Context, string) (role, status string, found bool, err error)
}

func RequireAdmin(store AdminAccessStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := c.Get(UserIDKey).(string)
			if !ok || userID == "" {
				return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
			}
			role, status, found, err := store.FindUserAccess(c.Request().Context(), userID)
			if err != nil {
				return utils.JSONError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Could not verify administrator access.", nil)
			}
			if !found {
				return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "The authenticated user no longer exists.", nil)
			}
			if status != "ACTIVE" {
				return utils.JSONError(c, http.StatusForbidden, "ACCOUNT_DISABLED", "This account is disabled.", nil)
			}
			if role != "ADMIN" {
				return utils.JSONError(c, http.StatusForbidden, "ADMIN_FORBIDDEN", "Administrator access is required.", nil)
			}
			c.Set(UserRoleKey, role)
			return next(c)
		}
	}
}
