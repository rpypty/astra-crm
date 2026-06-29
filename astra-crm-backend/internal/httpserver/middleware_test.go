package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func TestRecovererReturnsSafeInternalError(t *testing.T) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(Recoverer(slog.New(slog.NewTextHandler(io.Discard, nil))))
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("database password leaked in panic")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Error.Code != CodeInternal {
		t.Fatalf("code = %q, want %q", payload.Error.Code, CodeInternal)
	}

	if payload.Error.Message == "database password leaked in panic" {
		t.Fatal("panic value leaked to response")
	}
}

func TestRecovererRedactsPanicValueInLogs(t *testing.T) {
	var logs bytes.Buffer
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(Recoverer(slog.New(slog.NewTextHandler(&logs, nil))))
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("database password leaked in panic")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)

	router.ServeHTTP(response, request)

	if strings.Contains(logs.String(), "database password leaked in panic") {
		t.Fatalf("panic value leaked to logs: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "[REDACTED]") {
		t.Fatalf("redacted panic marker missing from logs: %s", logs.String())
	}
}

func TestRequestLoggerIncludesAuthenticatedUserMetadata(t *testing.T) {
	var logs bytes.Buffer
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(RequestLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	router.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		SetRequestLogUser(r.Context(), 42, 7)
		w.WriteHeader(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/me", nil)

	router.ServeHTTP(response, request)

	if !strings.Contains(logs.String(), "user_id=42") || !strings.Contains(logs.String(), "team_id=7") {
		t.Fatalf("user metadata missing from request log: %s", logs.String())
	}
}

func TestRequestLoggerLogsUnknownErrorDetails(t *testing.T) {
	var logs bytes.Buffer
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(RequestLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	router.Get("/broken", func(w http.ResponseWriter, r *http.Request) {
		RespondError(w, errors.New("select trader dashboard: connection refused"))
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/broken", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	logText := logs.String()
	for _, want := range []string{
		"level=ERROR",
		"error_code=INTERNAL_ERROR",
		"error=\"select trader dashboard: connection refused\"",
		"error_type=*errors.errorString",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("log %q missing %q", logText, want)
		}
	}
}

func TestRequestLoggerLogsAPIErrorCause(t *testing.T) {
	var logs bytes.Buffer
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(RequestLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	router.Get("/blocked", func(w http.ResponseWriter, r *http.Request) {
		RespondError(w, DomainError("SHIFT_CANNOT_BE_CLOSED", "Смену нельзя закрыть").WithCause(errors.New("shift close is blocked")))
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/blocked", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	logText := logs.String()
	for _, want := range []string{
		"level=WARN",
		"error_code=SHIFT_CANNOT_BE_CLOSED",
		"error=\"shift close is blocked\"",
		"error_cause=\"shift close is blocked\"",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("log %q missing %q", logText, want)
		}
	}
}

func TestCSRFOriginGuardRejectsCrossOriginUnsafeRequest(t *testing.T) {
	handler := CSRFOriginGuard([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://api.example.test/api/v1/trader/payouts", nil)
	request.Header.Set("Origin", "http://evil.example.test")

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCSRFOriginGuardAllowsSafeMissingAndAllowedOrigins(t *testing.T) {
	calls := 0
	handler := CSRFOriginGuard([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "http://api.example.test/api/v1/auth/me", nil),
		httptest.NewRequest(http.MethodPost, "http://api.example.test/api/v1/auth/logout", nil),
		httptest.NewRequest(http.MethodPost, "http://api.example.test/api/v1/auth/logout", nil),
	} {
		if request.Method == http.MethodPost && request.Header.Get("Origin") == "" && calls == 2 {
			request.Header.Set("Origin", "http://localhost:5173")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s with origin %q status = %d, want %d", request.Method, request.Header.Get("Origin"), response.Code, http.StatusNoContent)
		}
	}

	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestLoginRateLimiterRejectsAfterLimit(t *testing.T) {
	limiter := NewLoginRateLimiter(2, time.Minute)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		request.RemoteAddr = "192.0.2.10:12345"
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("attempt %d status = %d, want %d", i+1, response.Code, http.StatusNoContent)
		}
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	request.RemoteAddr = "192.0.2.10:12345"
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}
