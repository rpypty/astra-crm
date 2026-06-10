package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondErrorValidation(t *testing.T) {
	response := httptest.NewRecorder()

	RespondError(response, ValidationError(map[string]string{
		"comment": "Комментарий обязателен",
	}))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Error.Code != CodeValidation {
		t.Fatalf("code = %q, want %q", payload.Error.Code, CodeValidation)
	}

	if payload.Error.Fields["comment"] == "" {
		t.Fatal("expected comment validation field")
	}
}

func TestRespondErrorHidesUnknownError(t *testing.T) {
	response := httptest.NewRecorder()

	RespondError(response, assertError("pq: duplicate key violates constraint users_password_hash_key"))

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

	if payload.Error.Message == "pq: duplicate key violates constraint users_password_hash_key" {
		t.Fatal("technical error leaked to response")
	}
}

type assertError string

func (e assertError) Error() string {
	return string(e)
}
