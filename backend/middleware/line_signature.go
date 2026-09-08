package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"stock_linebot/backend/internal/utils"
)

const LineWebhookBodyKey = "line_webhook_body"
const maxLineWebhookBody = 1 << 20

func VerifyLineSignature(channelSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxLineWebhookBody+1))
			if err != nil || len(body) > maxLineWebhookBody {
				return utils.JSONError(c, http.StatusBadRequest, "INVALID_WEBHOOK_PAYLOAD", "The LINE webhook payload is invalid.", nil)
			}
			signature, err := base64.StdEncoding.DecodeString(c.Request().Header.Get("X-Line-Signature"))
			mac := hmac.New(sha256.New, []byte(channelSecret))
			_, _ = mac.Write(body)
			if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
				return utils.JSONError(c, http.StatusUnauthorized, "INVALID_LINE_SIGNATURE", "The LINE signature is invalid.", nil)
			}
			c.Set(LineWebhookBodyKey, body)
			c.Request().Body = io.NopCloser(bytes.NewReader(body))
			return next(c)
		}
	}
}
