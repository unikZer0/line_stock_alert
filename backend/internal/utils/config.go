package utils

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL                string
	JWTAccessSecret            string
	JWTRefreshSecret           string
	JWTIssuer                  string
	AccessTTL                  time.Duration
	RefreshTTL                 time.Duration
	FrontendURL                string
	LineChannelID              string
	LineChannelSecret          string
	LineCallbackURL            string
	LineStateSecret            string
	LineMessagingSecret        string
	LineMessagingToken         string
	RedisURL                   string
	StockAPIBaseURL            string
	StockAPIKey                string
	StockRequestTimeout        time.Duration
	StockQuoteCacheTTL         time.Duration
	TwelveDataAPIBaseURL       string
	TwelveDataAPIKey           string
	StockCandleCacheTTL        time.Duration
	AlertLimit                 int
	AlertWorkerEnabled         bool
	AlertCheckInterval         time.Duration
	AlertWorkerBatchSize       int
	AlertWorkerConcurrency     int
	MarketNotificationsEnabled bool
	MarketNotificationInterval time.Duration
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTAccessSecret:      os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:     os.Getenv("JWT_REFRESH_SECRET"),
		JWTIssuer:            envOr("JWT_ISSUER", "stock-alert-api"),
		FrontendURL:          os.Getenv("FRONTEND_URL"),
		LineChannelID:        os.Getenv("LINE_LOGIN_CHANNEL_ID"),
		LineChannelSecret:    os.Getenv("LINE_LOGIN_CHANNEL_SECRET"),
		LineCallbackURL:      os.Getenv("LINE_LOGIN_CALLBACK_URL"),
		LineStateSecret:      os.Getenv("LINE_OAUTH_STATE_SECRET"),
		LineMessagingSecret:  os.Getenv("LINE_MESSAGING_CHANNEL_SECRET"),
		LineMessagingToken:   os.Getenv("LINE_MESSAGING_CHANNEL_ACCESS_TOKEN"),
		RedisURL:             os.Getenv("REDIS_URL"),
		StockAPIBaseURL:      envOr("STOCK_API_BASE_URL", "https://finnhub.io/api/v1"),
		StockAPIKey:          os.Getenv("STOCK_API_KEY"),
		TwelveDataAPIBaseURL: envOr("TWELVE_DATA_API_BASE_URL", "https://api.twelvedata.com"),
		TwelveDataAPIKey:     os.Getenv("TWELVE_DATA_API_KEY"),
	}

	var err error
	if cfg.AccessTTL, err = envDuration("JWT_ACCESS_TTL_MINUTES", 60, time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.RefreshTTL, err = envDuration("JWT_REFRESH_TTL_HOURS", 720, time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.StockRequestTimeout, err = envDuration("STOCK_REQUEST_TIMEOUT_SECONDS", 10, time.Second); err != nil {
		return Config{}, err
	}
	if cfg.StockQuoteCacheTTL, err = envDuration("STOCK_QUOTE_CACHE_TTL_SECONDS", 15, time.Second); err != nil {
		return Config{}, err
	}
	if cfg.StockCandleCacheTTL, err = envDuration("STOCK_CANDLE_CACHE_TTL_SECONDS", 60, time.Second); err != nil {
		return Config{}, err
	}
	if cfg.AlertLimit, err = envInt("ALERT_LIMIT_PER_USER", 100); err != nil {
		return Config{}, err
	}
	if cfg.AlertWorkerEnabled, err = envBool("ALERT_WORKER_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.AlertCheckInterval, err = envDuration("ALERT_CHECK_INTERVAL_SECONDS", 15, time.Second); err != nil {
		return Config{}, err
	}
	if cfg.AlertWorkerBatchSize, err = envInt("ALERT_WORKER_BATCH_SIZE", 100); err != nil {
		return Config{}, err
	}
	if cfg.AlertWorkerConcurrency, err = envInt("ALERT_WORKER_CONCURRENCY", 5); err != nil {
		return Config{}, err
	}
	if cfg.MarketNotificationsEnabled, err = envBool("MARKET_NOTIFICATIONS_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.MarketNotificationInterval, err = envDuration("MARKET_NOTIFICATION_CHECK_SECONDS", 30, time.Second); err != nil {
		return Config{}, err
	}
	if cfg.AlertWorkerBatchSize <= 0 || cfg.AlertWorkerConcurrency <= 0 {
		return Config{}, fmt.Errorf("alert worker batch size and concurrency must be greater than zero")
	}
	if cfg.AlertLimit <= 0 {
		return Config{}, fmt.Errorf("ALERT_LIMIT_PER_USER must be greater than zero")
	}

	for name, value := range map[string]string{
		"DATABASE_URL": cfg.DatabaseURL, "JWT_ACCESS_SECRET": cfg.JWTAccessSecret,
		"JWT_REFRESH_SECRET": cfg.JWTRefreshSecret,
		"FRONTEND_URL":       cfg.FrontendURL, "LINE_LOGIN_CHANNEL_ID": cfg.LineChannelID,
		"LINE_LOGIN_CHANNEL_SECRET": cfg.LineChannelSecret, "LINE_LOGIN_CALLBACK_URL": cfg.LineCallbackURL,
		"LINE_OAUTH_STATE_SECRET":             cfg.LineStateSecret,
		"LINE_MESSAGING_CHANNEL_SECRET":       cfg.LineMessagingSecret,
		"LINE_MESSAGING_CHANNEL_ACCESS_TOKEN": cfg.LineMessagingToken,
		"REDIS_URL":                           cfg.RedisURL, "STOCK_API_BASE_URL": cfg.StockAPIBaseURL,
		"STOCK_API_KEY":            cfg.StockAPIKey,
		"TWELVE_DATA_API_BASE_URL": cfg.TwelveDataAPIBaseURL, "TWELVE_DATA_API_KEY": cfg.TwelveDataAPIKey,
	} {
		if value == "" {
			return Config{}, fmt.Errorf("%s is required", name)
		}
	}
	if len(cfg.JWTAccessSecret) < 32 || len(cfg.JWTRefreshSecret) < 32 {
		return Config{}, fmt.Errorf("JWT secrets must each contain at least 32 characters")
	}
	if len(cfg.LineStateSecret) < 32 {
		return Config{}, fmt.Errorf("LINE_OAUTH_STATE_SECRET must contain at least 32 characters")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func envBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
}

func envDuration(key string, fallback int, unit time.Duration) (time.Duration, error) {
	value, err := envInt(key, fallback)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return time.Duration(value) * unit, nil
}
