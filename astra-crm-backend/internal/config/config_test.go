package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsForDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("SESSION_SECURE", "")
	t.Setenv("CSRF_ALLOWED_ORIGINS", "")
	t.Setenv("LOGIN_RATE_LIMIT_REQUESTS", "")
	t.Setenv("LOGIN_RATE_LIMIT_WINDOW", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != EnvDevelopment {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, EnvDevelopment)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}

	if cfg.SessionCookieName != "p2p_crm_session" {
		t.Fatalf("SessionCookieName = %q, want p2p_crm_session", cfg.SessionCookieName)
	}

	if cfg.SessionSecure {
		t.Fatal("SessionSecure = true, want false for development default")
	}
	if cfg.LoginRateLimitRequests != 10 {
		t.Fatalf("LoginRateLimitRequests = %d, want 10", cfg.LoginRateLimitRequests)
	}
	if cfg.LoginRateLimitWindow != time.Minute {
		t.Fatalf("LoginRateLimitWindow = %s, want 1m", cfg.LoginRateLimitWindow)
	}
}

func TestLoadRequiresDatabaseOutsideDevelopmentAndTest(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "p2p_crm_session")
	t.Setenv("SESSION_SECURE", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want DATABASE_URL error")
	}
}

func TestLoadParsesSecuritySettings(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "p2p_crm_session")
	t.Setenv("SESSION_SECURE", "")
	t.Setenv("CSRF_ALLOWED_ORIGINS", "http://localhost:5173, https://crm.example.test ")
	t.Setenv("LOGIN_RATE_LIMIT_REQUESTS", "5")
	t.Setenv("LOGIN_RATE_LIMIT_WINDOW", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.CSRFAllowedOrigins) != 2 || cfg.CSRFAllowedOrigins[0] != "http://localhost:5173" || cfg.CSRFAllowedOrigins[1] != "https://crm.example.test" {
		t.Fatalf("CSRFAllowedOrigins = %#v", cfg.CSRFAllowedOrigins)
	}
	if cfg.LoginRateLimitRequests != 5 {
		t.Fatalf("LoginRateLimitRequests = %d, want 5", cfg.LoginRateLimitRequests)
	}
	if cfg.LoginRateLimitWindow != 30*time.Second {
		t.Fatalf("LoginRateLimitWindow = %s, want 30s", cfg.LoginRateLimitWindow)
	}
}

func TestLoadRejectsInvalidSecuritySettings(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "p2p_crm_session")
	t.Setenv("SESSION_SECURE", "")
	t.Setenv("LOGIN_RATE_LIMIT_REQUESTS", "many")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid rate limit error")
	}

	t.Setenv("LOGIN_RATE_LIMIT_REQUESTS", "5")
	t.Setenv("LOGIN_RATE_LIMIT_WINDOW", "soon")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid rate limit window error")
	}
}

func TestLoadParsesSessionSecure(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "p2p_crm_session")
	t.Setenv("SESSION_SECURE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.SessionSecure {
		t.Fatal("SessionSecure = false, want true")
	}
}

func TestLoadRejectsInvalidSessionSecure(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "p2p_crm_session")
	t.Setenv("SESSION_SECURE", "sometimes")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid boolean error")
	}
}
