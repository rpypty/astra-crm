package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
