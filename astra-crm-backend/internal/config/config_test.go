package config

import "testing"

func TestLoadDefaultsForDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("SESSION_SECURE", "")

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
