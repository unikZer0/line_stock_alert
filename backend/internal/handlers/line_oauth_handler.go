package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
)

type LineOAuthHandler struct {
	service     *services.LineOAuthService
	frontendURL string
}

func NewLineOAuthHandler(service *services.LineOAuthService, frontendURL string) *LineOAuthHandler {
	return &LineOAuthHandler{service: service, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (h *LineOAuthHandler) Start(c echo.Context) error {
	authorizationURL, err := h.service.AuthorizationURL()
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Redirect(http.StatusFound, authorizationURL)
}

func (h *LineOAuthHandler) Callback(c echo.Context) error {
	if c.QueryParam("error") != "" {
		return utils.JSONError(c, http.StatusBadRequest, "LINE_AUTH_CANCELLED", "LINE login was cancelled.", nil)
	}
	result, err := h.service.Login(
		c.Request().Context(), c.QueryParam("code"), c.QueryParam("state"),
		c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		return handleServiceError(c, err)
	}
	callbackURL, err := url.Parse(h.frontendURL + "/auth/callback")
	if err != nil {
		return utils.JSONError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Unable to complete LINE login.", nil)
	}
	callbackURL.Fragment = url.Values{
		"access_token": {result.AccessToken}, "refresh_token": {result.RefreshToken},
		"token_type": {result.TokenType}, "expires_in": {strconv.FormatInt(result.ExpiresIn, 10)},
	}.Encode()
	return c.Redirect(http.StatusFound, callbackURL.String())
}
