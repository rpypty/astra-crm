package httpserver

import (
	"errors"
	"net/http"

	"github.com/ashpak/astra-crm-backend/internal/auth"
)

func AuthMiddleware(authenticator Authenticator, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator == nil {
				RespondError(w, ServiceUnavailableError())
				return
			}

			token := cookieValue(r, cookieName)
			user, err := authenticator.AuthenticateToken(r.Context(), token)
			if errors.Is(err, auth.ErrUnauthenticated) || errors.Is(err, auth.ErrInactiveUser) {
				RespondError(w, UnauthorizedError())
				return
			}
			if err != nil {
				RespondError(w, err)
				return
			}

			SetRequestLogUser(r.Context(), user.ID, user.TeamID)
			next.ServeHTTP(w, r.WithContext(ContextWithCurrentUser(r.Context(), user)))
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				RespondError(w, UnauthorizedError())
				return
			}

			if _, ok := allowed[user.Role]; !ok {
				RespondError(w, ForbiddenError())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
