package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestAuthHandlerLoginSetsHTTPOnlyCookie(t *testing.T) {
	service := &fakeAuthService{
		loginResult: auth.LoginResult{
			User: users.User{
				ID:     1,
				TeamID: 2,
				Role:   users.RoleTrader,
				Login:  "trader_1",
				Status: users.StatusActive,
			},
			Token:     "raw-session-token",
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	handler := NewAuthHandler(service, "session", true)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"trader_1","password":"secret"}`))

	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	cookie := response.Result().Cookies()[0]
	if cookie.Name != "session" {
		t.Fatalf("cookie name = %q, want session", cookie.Name)
	}
	if !cookie.HttpOnly {
		t.Fatal("session cookie is not httpOnly")
	}
	if !cookie.Secure {
		t.Fatal("session cookie is not secure")
	}
	if strings.Contains(response.Body.String(), "raw-session-token") {
		t.Fatal("raw session token leaked to response body")
	}
}

func TestAuthHandlerLoginRejectsInvalidCredentials(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginErr: auth.ErrInvalidCredentials,
	}, "session", false)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"trader_1","password":"wrong"}`))

	handler.Login(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandlerLogoutRevokesAndClearsCookie(t *testing.T) {
	service := &fakeAuthService{}
	handler := NewAuthHandler(service, "session", false)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "raw-session-token"})

	handler.Logout(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if service.logoutToken != "raw-session-token" {
		t.Fatalf("logout token = %q, want raw-session-token", service.logoutToken)
	}

	cookie := response.Result().Cookies()[0]
	if cookie.MaxAge != -1 {
		t.Fatalf("clear cookie MaxAge = %d, want -1", cookie.MaxAge)
	}
}

func TestAuthHandlerMeReturnsCurrentUser(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{}, "session", false)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Login:  "lead",
		Status: users.StatusActive,
	}))

	handler.Me(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"login":"lead"`) {
		t.Fatalf("response body does not contain public user: %s", response.Body.String())
	}
}

type fakeAuthService struct {
	loginResult auth.LoginResult
	loginErr    error
	logoutToken string
}

func (s *fakeAuthService) Login(ctx context.Context, params auth.LoginParams) (auth.LoginResult, error) {
	if s.loginErr != nil {
		return auth.LoginResult{}, s.loginErr
	}

	return s.loginResult, nil
}

func (s *fakeAuthService) Logout(ctx context.Context, token string) error {
	s.logoutToken = token
	return nil
}

func (s *fakeAuthService) AuthenticateToken(ctx context.Context, token string) (users.User, error) {
	return users.User{}, errors.New("not implemented")
}
