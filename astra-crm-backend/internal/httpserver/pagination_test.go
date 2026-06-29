package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaginationFromRequestRejectsInvalidPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/items?page=abc&pageSize=10", nil)
	response := httptest.NewRecorder()

	_, ok := paginationFromRequest(response, request)
	if ok {
		t.Fatal("paginationFromRequest() ok = true, want false")
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "page") {
		t.Fatalf("response does not mention page: %s", response.Body.String())
	}
}

func TestPaginationFromRequestAppliesDefaults(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/items", nil)
	response := httptest.NewRecorder()

	params, ok := paginationFromRequest(response, request)
	if !ok {
		t.Fatalf("paginationFromRequest() ok = false; body: %s", response.Body.String())
	}
	if params.Page != 1 || params.PageSize != 50 {
		t.Fatalf("pagination = %d/%d, want 1/50", params.Page, params.PageSize)
	}
}
