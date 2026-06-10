package auth

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/users"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInactiveUser       = errors.New("inactive user")
)

const DefaultSessionTTL = 24 * time.Hour

type UserStore interface {
	GetByID(ctx context.Context, id int64) (users.User, error)
	GetByLogin(ctx context.Context, login string) (users.User, error)
}

type SessionStore interface {
	Create(ctx context.Context, params CreateSessionParams) (Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
}

type Service struct {
	users    UserStore
	sessions SessionStore
	now      func() time.Time
}

func NewService(userStore UserStore, sessionStore SessionStore) *Service {
	return &Service{
		users:    userStore,
		sessions: sessionStore,
		now:      time.Now,
	}
}

type LoginParams struct {
	Login     string
	Password  string
	UserAgent *string
	IP        *netip.Addr
}

type LoginResult struct {
	User      users.User
	Token     string
	ExpiresAt time.Time
}

type CreateSessionParams struct {
	UserID    int64
	TokenHash string
	UserAgent *string
	IP        *netip.Addr
	ExpiresAt time.Time
}

type Session struct {
	ID        int64
	UserID    int64
	ExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, params LoginParams) (LoginResult, error) {
	login := strings.TrimSpace(params.Login)
	password := params.Password

	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	if !user.IsActive() {
		return LoginResult{}, ErrInvalidCredentials
	}

	if !CheckPassword(password, user.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := NewSessionToken()
	if err != nil {
		return LoginResult{}, err
	}

	expiresAt := s.now().Add(DefaultSessionTTL)
	if _, err := s.sessions.Create(ctx, CreateSessionParams{
		UserID:    user.ID,
		TokenHash: HashSessionToken(token),
		UserAgent: params.UserAgent,
		IP:        params.IP,
		ExpiresAt: expiresAt,
	}); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) AuthenticateToken(ctx context.Context, token string) (users.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return users.User{}, ErrUnauthenticated
	}

	session, err := s.sessions.GetByTokenHash(ctx, HashSessionToken(token))
	if err != nil {
		return users.User{}, ErrUnauthenticated
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return users.User{}, ErrUnauthenticated
	}

	if !user.IsActive() {
		return users.User{}, ErrInactiveUser
	}

	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	return s.sessions.RevokeByTokenHash(ctx, HashSessionToken(token))
}
