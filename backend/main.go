package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"stock_linebot/backend/internal/handlers"
	"stock_linebot/backend/internal/repositories"
	"stock_linebot/backend/internal/services"
	"stock_linebot/backend/internal/utils"
	authmw "stock_linebot/backend/middleware"
)

const defaultPort = "8080"

func main() {
	cfg, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database pool: %v", err)
	}
	defer db.Close()
	if err = db.Ping(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	authRepository := repositories.NewAuthRepository(db)
	mailer := services.NewSMTPMailer(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword,
		cfg.SMTPUseTLS, cfg.FromEmail, cfg.FromName,
	)
	authService := services.NewAuthService(authRepository, mailer, services.AuthConfig{
		AccessSecret: cfg.JWTAccessSecret, RefreshSecret: cfg.JWTRefreshSecret,
		Issuer: cfg.JWTIssuer, AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL,
		OTPTTL: cfg.OTPTTL, OTPResendCooldown: cfg.OTPResendCooldown,
		BcryptCost: cfg.BcryptCost, OTPMaxAttempts: cfg.OTPMaxAttempts,
	})
	authHandler := handlers.NewAuthHandler(authService)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api := e.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register, authmw.RateLimit(5, time.Minute, "RATE_LIMIT_EXCEEDED"))
	auth.POST("/verify-email", authHandler.VerifyEmail)
	auth.POST("/resend-otp", authHandler.ResendOTP, authmw.RateLimit(3, time.Minute, "OTP_RATE_LIMITED"))
	auth.POST("/login", authHandler.Login, authmw.RateLimit(5, time.Minute, "LOGIN_RATE_LIMITED"))
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))

	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = defaultPort
	}

	e.Logger.Fatal(e.Start(":" + port))
}
