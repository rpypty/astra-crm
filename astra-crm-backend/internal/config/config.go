package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvTest        = "test"
)

type Config struct {
	AppEnv                 string
	HTTPAddr               string
	DatabaseURL            string
	SessionCookieName      string
	SessionSecure          bool
	CSRFAllowedOrigins     []string
	LoginRateLimitRequests int
	LoginRateLimitWindow   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             getString("APP_ENV", EnvDevelopment),
		HTTPAddr:           getString("HTTP_ADDR", ":8080"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SessionCookieName:  getString("SESSION_COOKIE_NAME", "p2p_crm_session"),
		CSRFAllowedOrigins: getCSV("CSRF_ALLOWED_ORIGINS"),
	}

	sessionSecure, err := getBool("SESSION_SECURE", defaultSessionSecure(cfg.AppEnv))
	if err != nil {
		return Config{}, err
	}
	cfg.SessionSecure = sessionSecure
	loginRateLimitRequests, err := getInt("LOGIN_RATE_LIMIT_REQUESTS", 10)
	if err != nil {
		return Config{}, err
	}
	cfg.LoginRateLimitRequests = loginRateLimitRequests
	loginRateLimitWindow, err := getDuration("LOGIN_RATE_LIMIT_WINDOW", time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.LoginRateLimitWindow = loginRateLimitWindow

	if strings.TrimSpace(cfg.HTTPAddr) == "" {
		return Config{}, errors.New("config: HTTP_ADDR must not be empty")
	}

	if strings.TrimSpace(cfg.SessionCookieName) == "" {
		return Config{}, errors.New("config: SESSION_COOKIE_NAME must not be empty")
	}
	if cfg.LoginRateLimitRequests <= 0 {
		return Config{}, errors.New("config: LOGIN_RATE_LIMIT_REQUESTS must be positive")
	}
	if cfg.LoginRateLimitWindow <= 0 {
		return Config{}, errors.New("config: LOGIN_RATE_LIMIT_WINDOW must be positive")
	}

	if cfg.DatabaseURL == "" && !allowsMissingDatabase(cfg.AppEnv) {
		return Config{}, errors.New("config: DATABASE_URL is required outside development and test")
	}

	return cfg, nil
}

func getString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getBool(key string, fallback bool) (bool, error) {
	raw, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("config: %s must be a boolean: %w", key, err)
	}

	return value, nil
}

func getCSV(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func getInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", key, err)
	}
	return value, nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a Go duration: %w", key, err)
	}
	return value, nil
}

func defaultSessionSecure(appEnv string) bool {
	return !allowsMissingDatabase(appEnv)
}

func allowsMissingDatabase(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case EnvDevelopment, EnvTest:
		return true
	default:
		return false
	}
}
