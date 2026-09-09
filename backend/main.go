package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	authService := services.NewAuthService(authRepository, services.AuthConfig{
		AccessSecret: cfg.JWTAccessSecret, RefreshSecret: cfg.JWTRefreshSecret,
		Issuer: cfg.JWTIssuer, AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL,
	})
	authHandler := handlers.NewAuthHandler(authService)
	lineClient := services.NewLineClient(
		&http.Client{Timeout: 10 * time.Second}, cfg.LineChannelID, cfg.LineChannelSecret,
	)
	lineService := services.NewLineOAuthService(authRepository, lineClient, authService, services.LineOAuthConfig{
		ChannelID: cfg.LineChannelID, CallbackURL: cfg.LineCallbackURL,
		StateSecret: cfg.LineStateSecret, Issuer: cfg.JWTIssuer, StateTTL: 10 * time.Minute,
	})
	lineHandler := handlers.NewLineOAuthHandler(lineService, cfg.FrontendURL)
	userRepository := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepository)
	userHandler := handlers.NewUserHandler(userService)
	stockProvider := services.NewFinnhubClient(&http.Client{Timeout: cfg.StockRequestTimeout}, cfg.StockAPIBaseURL, cfg.StockAPIKey)
	quoteCache := services.NewRedisQuoteCache(redisClient)
	alertRepository := repositories.NewAlertRepository(db)
	alertService := services.NewAlertService(alertRepository, stockProvider, cfg.AlertLimit)
	alertHandler := handlers.NewAlertHandler(alertService)
	stockService := services.NewStockService(stockProvider, quoteCache, cfg.StockQuoteCacheTTL)
	candleProvider := services.NewTwelveDataClient(&http.Client{Timeout: cfg.StockRequestTimeout}, cfg.TwelveDataAPIBaseURL, cfg.TwelveDataAPIKey)
	stockService.ConfigureCandles(candleProvider, quoteCache, cfg.StockCandleCacheTTL)
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
	adminAlertRepository := repositories.NewAdminAlertRepository(db)
	adminAlertService := services.NewAdminAlertService(adminAlertRepository)
	adminAlertHandler := handlers.NewAdminAlertHandler(adminAlertService)
	adminStockRepository := repositories.NewAdminStockRepository(db)
	adminStockService := services.NewAdminStockService(adminStockRepository)
	adminStockHandler := handlers.NewAdminStockHandler(adminStockService)
	adminMonitoringRepository := repositories.NewAdminMonitoringRepository(db)
	adminMonitoringService := services.NewAdminMonitoringService(adminMonitoringRepository)
	adminMonitoringHandler := handlers.NewAdminMonitoringHandler(adminMonitoringService)
	alertWorkerRepository := repositories.NewAlertWorkerRepository(db)
	alertWorker := services.NewAlertWorker(alertWorkerRepository, stockProvider, quoteCache, lineMessenger, services.AlertWorkerConfig{
		Interval: cfg.AlertCheckInterval, QuoteTTL: cfg.StockQuoteCacheTTL,
		BatchSize: cfg.AlertWorkerBatchSize, Concurrency: cfg.AlertWorkerConcurrency,
	})
	marketNotificationRepository := repositories.NewMarketNotificationRepository(db)
	marketNotificationWorker := services.NewMarketNotificationWorker(marketNotificationRepository, lineMessenger, redisClient, cfg.MarketNotificationInterval)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.POST("/webhooks/line", lineWebhookHandler.Receive, authmw.VerifyLineSignature(cfg.LineMessagingSecret))

	api := e.Group("/api/v1")
	auth := api.Group("/auth")
	// LINE OAuth2 is the only login method for normal users. Password login is
	// retained for back-office administrators only.
	auth.POST("/login", authHandler.Login, authmw.RateLimit(5, time.Minute, "LOGIN_RATE_LIMITED"))
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	auth.GET("/line", lineHandler.Start, authmw.RateLimit(10, time.Minute, "RATE_LIMIT_EXCEEDED"))
	auth.GET("/line/callback", lineHandler.Callback, authmw.RateLimit(20, time.Minute, "RATE_LIMIT_EXCEEDED"))

	api.GET("/me", userHandler.CurrentUser, authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))

	alerts := api.Group("/alerts", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	alerts.GET("", alertHandler.List)
	alerts.POST("", alertHandler.Create, authmw.RateLimit(20, time.Minute, "RATE_LIMIT_EXCEEDED"))
	alerts.PATCH("/:id", alertHandler.Update)
	alerts.DELETE("/:id", alertHandler.Delete)
	alerts.POST("/:id/rearm", alertHandler.Rearm, authmw.RateLimit(20, time.Minute, "RATE_LIMIT_EXCEEDED"))

	stocks := api.Group("/stocks", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer))
	stocks.GET("", stockHandler.Search)
	stocks.GET("/quotes", stockHandler.Quotes)
	stocks.GET("/:symbol/candles", stockHandler.Candles)
	stocks.GET("/:symbol/quote", stockHandler.Quote)

	// All endpoints added under this group inherit JWT authentication and a
	// fresh database check for an active ADMIN role.
	admin := api.Group("/admin", authmw.RequireAuth(cfg.JWTAccessSecret, cfg.JWTIssuer), authmw.RequireAdmin(adminAccessRepository))
	admin.GET("/dashboard", adminDashboardHandler.Dashboard)
	admin.GET("/users", adminUserHandler.List)
	admin.GET("/users/:id", adminUserHandler.Get)
	admin.POST("/users/:id/disable", adminUserHandler.Disable)
	admin.POST("/users/:id/enable", adminUserHandler.Enable)
	admin.GET("/alerts", adminAlertHandler.List)
	admin.GET("/alerts/:id", adminAlertHandler.Get)
	admin.POST("/alerts/:id/disable", adminAlertHandler.Disable)
	admin.GET("/stocks", adminStockHandler.List)
	admin.GET("/stocks/:symbol", adminStockHandler.Get)
	admin.POST("/stocks/:symbol/enable", adminStockHandler.Enable)
	admin.POST("/stocks/:symbol/disable", adminStockHandler.Disable)
	admin.GET("/line/stats", adminMonitoringHandler.LineStats)
	admin.GET("/line/failed-messages", adminMonitoringHandler.FailedLineMessages)
	admin.GET("/logs", adminMonitoringHandler.ApplicationLogs)
	admin.GET("/audit-logs", adminMonitoringHandler.AuditLogs)

	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = defaultPort
	}

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.AlertWorkerEnabled {
		go alertWorker.Run(appCtx)
	}
	if cfg.MarketNotificationsEnabled {
		go marketNotificationWorker.Run(appCtx)
	}
	go func() {
		<-appCtx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if shutdownErr := e.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("shutdown HTTP server: %v", shutdownErr)
		}
	}()
	if err = e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("start HTTP server: %v", err)
	}
}
