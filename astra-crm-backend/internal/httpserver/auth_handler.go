package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

type AuthService interface {
	Login(ctx context.Context, params auth.LoginParams) (auth.LoginResult, error)
	Logout(ctx context.Context, token string) error
}

type Authenticator interface {
	AuthenticateToken(ctx context.Context, token string) (users.User, error)
}

type AuthHandler struct {
	service    AuthService
	cookieName string
	secure     bool
}

func NewAuthHandler(service AuthService, cookieName string, secure bool) *AuthHandler {
	return &AuthHandler{
		service:    service,
		cookieName: cookieName,
		secure:     secure,
	}
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	User users.PublicUser `json:"user"`
}

type meResponse struct {
	User users.PublicUser `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request loginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		RespondError(w, ValidationError(map[string]string{
			"body": "Некорректный JSON",
		}))
		return
	}

	fields := map[string]string{}
	if strings.TrimSpace(request.Login) == "" {
		fields["login"] = "Логин обязателен"
	}
	if request.Password == "" {
		fields["password"] = "Пароль обязателен"
	}
	if len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	userAgent := strings.TrimSpace(r.UserAgent())
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}

	result, err := h.service.Login(r.Context(), auth.LoginParams{
		Login:     request.Login,
		Password:  request.Password,
		UserAgent: userAgentPtr,
		IP:        requestIP(r),
	})
	if errors.Is(err, auth.ErrInvalidCredentials) {
		RespondError(w, UnauthorizedError())
		return
	}
	if err != nil {
		RespondError(w, err)
		return
	}

	http.SetCookie(w, h.sessionCookie(result.Token, result.ExpiresAt))
	WriteJSON(w, http.StatusOK, loginResponse{
		User: users.ToPublic(result.User),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	token := cookieValue(r, h.cookieName)
	if err := h.service.Logout(r.Context(), token); err != nil {
		RespondError(w, err)
		return
	}

	http.SetCookie(w, h.clearSessionCookie())
	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}

	WriteJSON(w, http.StatusOK, meResponse{
		User: users.ToPublic(user),
	})
}

func (h *AuthHandler) sessionCookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func requestIP(r *http.Request) *netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}

	return &addr
}
