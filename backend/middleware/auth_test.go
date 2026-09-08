package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func TestRequireAuth(t *testing.T) {
	secret := strings.Repeat("s", 32)
	claims := jwt.MapClaims{"sub": "user-1", "role": "USER", "token_type": "access", "iss": "test", "exp": time.Now().Add(time.Hour).Unix()}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	e := echo.New()
	e.GET("/private", func(c echo.Context) error {
		if c.Get(UserIDKey) != "user-1" {
			t.Fatalf("unexpected user ID: %v", c.Get(UserIDKey))
		}
		return c.NoContent(http.StatusNoContent)
	}, RequireAuth(secret, "test"))

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+raw)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	e := echo.New()
	e.GET("/private", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, RequireAuth(strings.Repeat("s", 32), "test"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/private", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}
