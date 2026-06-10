package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestAuthMiddlewareRejectsUnauthenticatedRequest(t *testing.T) {
	handler := AuthMiddleware(&fakeAuthenticator{}, "session")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareStoresCurrentUser(t *testing.T) {
	wantUser := users.User{ID: 1, TeamID: 2, Role: users.RoleTrader, Login: "trader_1", Status: users.StatusActive}
	handler := AuthMiddleware(&fakeAuthenticator{user: wantUser}, "session")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, ok := CurrentUser(r.Context())
		if !ok {
			t.Fatal("current user missing from request context")
		}
		if gotUser.ID != wantUser.ID {
			t.Fatalf("current user ID = %d, want %d", gotUser.ID, wantUser.ID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRequireRoleRejectsWrongRole(t *testing.T) {
	handler := RequireRole(users.RoleTeamlead)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/traders", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

type fakeAuthenticator struct {
	user users.User
}

func (a *fakeAuthenticator) AuthenticateToken(ctx context.Context, token string) (users.User, error) {
	if token == "" {
		return users.User{}, auth.ErrUnauthenticated
	}

	return a.user, nil
}
