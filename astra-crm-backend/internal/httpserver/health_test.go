package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	assertHealthStatus(t, response, "ok")
}

func TestReadyHandlerWithoutDatabase(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)

	ReadyHandler(nil).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}

	assertHealthStatus(t, response, "not_ready")
}

func TestReadyHandlerWithHealthyDatabase(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)

	ReadyHandler(pingFunc(func(ctx context.Context) error {
		return nil
	})).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	assertHealthStatus(t, response, "ok")
}

func TestReadyHandlerWithUnhealthyDatabase(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)

	ReadyHandler(pingFunc(func(ctx context.Context) error {
		return errors.New("connection refused")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}

	assertHealthStatus(t, response, "not_ready")
}

func assertHealthStatus(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()

	var payload HealthResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Status != want {
		t.Fatalf("status payload = %q, want %q", payload.Status, want)
	}
}

type pingFunc func(ctx context.Context) error

func (f pingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}
