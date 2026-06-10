package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestServiceLoginCreatesSessionWithTokenHash(t *testing.T) {
	passwordHash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	userStore := &fakeUserStore{
		byLogin: map[string]users.User{
			"trader_1": {
				ID:           10,
				TeamID:       20,
				Role:         users.RoleTrader,
				Login:        "trader_1",
				PasswordHash: passwordHash,
				Status:       users.StatusActive,
			},
		},
	}
	sessionStore := &fakeSessionStore{}
	service := NewService(userStore, sessionStore)
	service.now = func() time.Time {
		return time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	}

	result, err := service.Login(context.Background(), LoginParams{
		Login:    "trader_1",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if result.Token == "" {
		t.Fatal("Login() returned empty token")
	}

	if sessionStore.created.TokenHash == "" {
		t.Fatal("session token hash was not stored")
	}

	if sessionStore.created.TokenHash == result.Token {
		t.Fatal("raw session token was stored instead of token hash")
	}

	if result.ExpiresAt.Sub(service.now()) != DefaultSessionTTL {
		t.Fatalf("session ttl = %s, want %s", result.ExpiresAt.Sub(service.now()), DefaultSessionTTL)
	}
}

func TestServiceLoginRejectsDisabledUser(t *testing.T) {
	passwordHash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	service := NewService(&fakeUserStore{
		byLogin: map[string]users.User{
			"trader_1": {
				ID:           10,
				Role:         users.RoleTrader,
				Login:        "trader_1",
				PasswordHash: passwordHash,
				Status:       users.StatusDisabled,
			},
		},
	}, &fakeSessionStore{})

	_, err = service.Login(context.Background(), LoginParams{
		Login:    "trader_1",
		Password: "secret",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceAuthenticateTokenLoadsActiveUser(t *testing.T) {
	token := "raw-token"
	tokenHash := HashSessionToken(token)
	userStore := &fakeUserStore{
		byID: map[int64]users.User{
			10: {
				ID:     10,
				TeamID: 20,
				Role:   users.RoleTeamlead,
				Login:  "lead",
				Status: users.StatusActive,
			},
		},
	}
	sessionStore := &fakeSessionStore{
		byHash: map[string]Session{
			tokenHash: {
				ID:     1,
				UserID: 10,
			},
		},
	}
	service := NewService(userStore, sessionStore)

	user, err := service.AuthenticateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("AuthenticateToken() error = %v", err)
	}

	if user.ID != 10 {
		t.Fatalf("user ID = %d, want 10", user.ID)
	}
}

type fakeUserStore struct {
	byID    map[int64]users.User
	byLogin map[string]users.User
}

func (s *fakeUserStore) GetByID(ctx context.Context, id int64) (users.User, error) {
	user, ok := s.byID[id]
	if !ok {
		return users.User{}, users.ErrNotFound
	}

	return user, nil
}

func (s *fakeUserStore) GetByLogin(ctx context.Context, login string) (users.User, error) {
	user, ok := s.byLogin[login]
	if !ok {
		return users.User{}, users.ErrNotFound
	}

	return user, nil
}

type fakeSessionStore struct {
	created CreateSessionParams
	byHash  map[string]Session
}

func (s *fakeSessionStore) Create(ctx context.Context, params CreateSessionParams) (Session, error) {
	s.created = params
	return Session{
		ID:        1,
		UserID:    params.UserID,
		ExpiresAt: params.ExpiresAt,
	}, nil
}

func (s *fakeSessionStore) GetByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	session, ok := s.byHash[tokenHash]
	if !ok {
		return Session{}, ErrSessionNotFound
	}

	return session, nil
}

func (s *fakeSessionStore) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
