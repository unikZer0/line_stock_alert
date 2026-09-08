package utils

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL         string
	JWTAccessSecret     string
	JWTRefreshSecret    string
	JWTIssuer           string
	AccessTTL           time.Duration
	RefreshTTL          time.Duration
	BcryptCost          int
	OTPTTL              time.Duration
	OTPResendCooldown   time.Duration
	OTPMaxAttempts      int
	SMTPHost            string
	SMTPPort            int
	SMTPUser            string
	SMTPPassword        string
	SMTPUseTLS          bool
	FromEmail           string
	FromName            string
	FrontendURL         string
	LineChannelID       string
	LineChannelSecret   string
	LineCallbackURL     string
	LineLinkCallbackURL string
	LineStateSecret     string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTAccessSecret:     os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:    os.Getenv("JWT_REFRESH_SECRET"),
		JWTIssuer:           envOr("JWT_ISSUER", "stock-alert-api"),
		SMTPHost:            os.Getenv("SMTP_HOST"),
		SMTPUser:            os.Getenv("SMTP_USER"),
		SMTPPassword:        os.Getenv("SMTP_PASSWORD"),
		FromEmail:           os.Getenv("FROM_EMAIL"),
		FromName:            envOr("FROM_NAME", "Stock Alert"),
		FrontendURL:         os.Getenv("FRONTEND_URL"),
		LineChannelID:       os.Getenv("LINE_LOGIN_CHANNEL_ID"),
		LineChannelSecret:   os.Getenv("LINE_LOGIN_CHANNEL_SECRET"),
		LineCallbackURL:     os.Getenv("LINE_LOGIN_CALLBACK_URL"),
		LineLinkCallbackURL: os.Getenv("LINE_LINK_CALLBACK_URL"),
		LineStateSecret:     os.Getenv("LINE_OAUTH_STATE_SECRET"),
	}

	var err error
	if cfg.AccessTTL, err = envDuration("JWT_ACCESS_TTL_MINUTES", 60, time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.RefreshTTL, err = envDuration("JWT_REFRESH_TTL_HOURS", 720, time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.OTPTTL, err = envDuration("OTP_TTL_MINUTES", 10, time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.OTPResendCooldown, err = envDuration("OTP_RESEND_COOLDOWN_SECONDS", 60, time.Second); err != nil {
		return Config{}, err
	}
	if cfg.BcryptCost, err = envInt("PASSWORD_BCRYPT_COST", 12); err != nil {
		return Config{}, err
	}
	if cfg.OTPMaxAttempts, err = envInt("OTP_MAX_ATTEMPTS", 5); err != nil {
		return Config{}, err
	}
	if cfg.SMTPPort, err = envInt("SMTP_PORT", 587); err != nil {
		return Config{}, err
	}
	if cfg.SMTPUseTLS, err = envBool("SMTP_USE_TLS", false); err != nil {
		return Config{}, err
	}

	for name, value := range map[string]string{
		"DATABASE_URL": cfg.DatabaseURL, "JWT_ACCESS_SECRET": cfg.JWTAccessSecret,
		"JWT_REFRESH_SECRET": cfg.JWTRefreshSecret, "SMTP_HOST": cfg.SMTPHost,
		"FROM_EMAIL":   cfg.FromEmail,
		"FRONTEND_URL": cfg.FrontendURL, "LINE_LOGIN_CHANNEL_ID": cfg.LineChannelID,
		"LINE_LOGIN_CHANNEL_SECRET": cfg.LineChannelSecret, "LINE_LOGIN_CALLBACK_URL": cfg.LineCallbackURL,
		"LINE_LINK_CALLBACK_URL":  cfg.LineLinkCallbackURL,
		"LINE_OAUTH_STATE_SECRET": cfg.LineStateSecret,
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
