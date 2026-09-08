package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"stock_linebot/backend/internal/utils"
)

const (
	UserIDKey   = "user_id"
	UserRoleKey = "user_role"
)

func RequireAuth(secret, issuer string) echo.MiddlewareFunc {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer))
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			parts := strings.Fields(c.Request().Header.Get(echo.HeaderAuthorization))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
			}
			claims := jwt.MapClaims{}
			token, err := parser.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid || claims["token_type"] != "access" {
				return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "The access token is invalid or expired.", nil)
			}
			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				return utils.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "The access token is invalid.", nil)
			}
			c.Set(UserIDKey, userID)
			if role, ok := claims["role"].(string); ok {
				c.Set(UserRoleKey, role)
			}
			return next(c)
		}
	}
}
