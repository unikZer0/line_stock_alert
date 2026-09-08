package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"stock_linebot/backend/internal/utils"
)

func RateLimit(requests int, window time.Duration, code string) echo.MiddlewareFunc {
	store := echomw.NewRateLimiterMemoryStoreWithConfig(echomw.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(float64(requests) / window.Seconds()),
		Burst:     requests,
		ExpiresIn: 3 * window,
	})
	return echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Store:               store,
		IdentifierExtractor: func(c echo.Context) (string, error) { return c.RealIP(), nil },
		ErrorHandler: func(c echo.Context, _ error) error {
			return utils.JSONError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Unable to apply rate limit.", nil)
		},
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			return utils.JSONError(c, http.StatusTooManyRequests, code, "Too many requests. Please try again later.", nil)
		},
	})
}
