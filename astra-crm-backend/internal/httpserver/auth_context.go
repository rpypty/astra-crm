package httpserver

import (
	"context"

	"github.com/ashpak/astra-crm-backend/internal/users"
)

type currentUserContextKey struct{}

func ContextWithCurrentUser(ctx context.Context, user users.User) context.Context {
	return context.WithValue(ctx, currentUserContextKey{}, user)
}

func CurrentUser(ctx context.Context) (users.User, bool) {
	user, ok := ctx.Value(currentUserContextKey{}).(users.User)
	return user, ok
}
