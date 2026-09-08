package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"

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
	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()
	if err = redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect to Redis: %v", err)
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
	lineClient := services.NewLineClient(
		&http.Client{Timeout: 10 * time.Second}, cfg.LineChannelID, cfg.LineChannelSecret,
	)
	lineService := services.NewLineOAuthService(authRepository, lineClient, authService, services.LineOAuthConfig{
		ChannelID: cfg.LineChannelID, CallbackURL: cfg.LineCallbackURL, LinkCallbackURL: cfg.LineLinkCallbackURL,
		StateSecret: cfg.LineStateSecret, Issuer: cfg.JWTIssuer, StateTTL: 10 * time.Minute,
	})
	lineHandler := handlers.NewLineOAuthHandler(lineService, cfg.FrontendURL)
	lineAccountHandler := handlers.NewLineAccountHandler(lineService)
	userRepository := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepository)
	userHandler := handlers.NewUserHandler(userService)
	watchlistRepository := repositories.NewWatchlistRepository(db)
	stockProvider := services.NewFinnhubClient(&http.Client{Timeout: cfg.StockRequestTimeout}, cfg.StockAPIBaseURL, cfg.StockAPIKey)
	quoteCache := services.NewRedisQuoteCache(redisClient)
	watchlistService := services.NewWatchlistService(watchlistRepository, stockProvider, quoteCache, cfg.WatchlistLimit, cfg.StockQuoteCacheTTL)
	watchlistHandler := handlers.NewWatchlistHandler(watchlistService)
	alertRepository := repositories.NewAlertRepository(db)
	alertService := services.NewAlertService(alertRepository, stockProvider, cfg.AlertLimit)
	alertHandler := handlers.NewAlertHandler(alertService)
	stockService := services.NewStockService(stockProvider, quoteCache, cfg.StockQuoteCacheTTL)
	stockHandler := handlers.NewStockHandler(stockService)
	lineWebhookRepository := repositories.NewLineWebhookRepository(db)
	lineMessenger := services.NewLineMessagingClient(&http.Client{Timeout: 10 * time.Second}, cfg.LineMessagingToken)
	lineWebhookService := services.NewLineWebhookService(lineWebhookRepository, lineMessenger, cfg.FrontendURL)
	lineWebhookHandler := handlers.NewLineWebhookHandler(lineWebhookService)
	adminAccessRepository := repositories.NewAdminAccessRepository(db)
	adminDashboardRepository := repositories.NewAdminDashboardRepository(db)
	adminDashboardService := services.NewAdminDashboardService(adminDashboardRepository)
	adminDashboardHandler := handlers.NewAdminDashboardHandler(adminDashboardService)
	adminUserRepository := repositories.NewAdminUserRepository(db)
	adminUserService := services.NewAdminUserService(adminUserRepository)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserService)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.POST("/webhooks/line", lineWebhookHandler.Receive, authmw.VerifyLineSignature(cfg.LineMessagingSecret))

	api := e.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register, authmw.RateLimit(5, time.Minute, "RATE_LIMIT_EXCEEDED"))
	auth.POST("/verify-email", authHandler.VerifyEmail)
	auth.POST("/resend-otp", authHandler.ResendOTP, authmw.RateLimit(3, time.Minute, "OTP_RATE_LIMITED"))
	auth.POST("/login", authHandler.Login, authmw.RateLimit(5, time.Minute, "LOGIN_RATE_LIMITED"))
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	auth.GET("/line", lineHandler.Start, authmw.RateLimit(10, time.Minute, "RATE_LIMIT_EXCEEDED"))
	auth.GET("/line/callback", lineHandler.Callback, authmw.RateLimit(20, time.Minute, "RATE_LIMIT_EXCEEDED"))

	accounts := api.Group("/accounts")
	accounts.POST("/line/connect", lineAccountHandler.Connect, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	accounts.GET("/line/callback", lineAccountHandler.Callback, authmw.RateLimit(20, time.Minute, "RATE_LIMIT_EXCEEDED"))
	accounts.DELETE("/line", lineAccountHandler.Unlink, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	api.GET("/me", userHandler.CurrentUser, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))

	watchlists := api.Group("/watchlists", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	watchlists.GET("", watchlistHandler.List)
	watchlists.POST("", watchlistHandler.Add)
	watchlists.DELETE("/:id", watchlistHandler.Delete)

	alerts := api.Group("/alerts", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	alerts.GET("", alertHandler.List)
	alerts.POST("", alertHandler.Create, authmw.RateLimit(20, time.Minute, "RATE_LIMIT_EXCEEDED"))
	alerts.PATCH("/:id", alertHandler.Update)
	alerts.DELETE("/:id", alertHandler.Delete)

	stocks := api.Group("/stocks", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	stocks.GET("", stockHandler.Search)
	stocks.GET("/quotes", stockHandler.Quotes)
	stocks.GET("/:symbol/quote", stockHandler.Quote)

	// All endpoints added under this group inherit JWT authentication and a
	// fresh database check for an active ADMIN role.
	admin := api.Group("/admin", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer), authmw.RequireAdmin(adminAccessRepository))
	admin.GET("/dashboard", adminDashboardHandler.Dashboard)
	admin.GET("/users", adminUserHandler.List)
	admin.GET("/users/:id", adminUserHandler.Get)
	admin.POST("/users/:id/disable", adminUserHandler.Disable)
	admin.POST("/users/:id/enable", adminUserHandler.Enable)

	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = defaultPort
	}

	e.Logger.Fatal(e.Start(":" + port))
}
