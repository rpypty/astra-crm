package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionRepository struct {
	queries *db.Queries
}

func NewSessionRepository(queries *db.Queries) *SessionRepository {
	return &SessionRepository{queries: queries}
}

func (r *SessionRepository) Create(ctx context.Context, params CreateSessionParams) (Session, error) {
	userAgent := pgtype.Text{}
	if params.UserAgent != nil {
		userAgent = pgtype.Text{String: *params.UserAgent, Valid: true}
	}

	row, err := r.queries.CreateAuthSession(ctx, db.CreateAuthSessionParams{
		UserID:    params.UserID,
		TokenHash: params.TokenHash,
		UserAgent: userAgent,
		Ip:        params.IP,
		ExpiresAt: pgtype.Timestamptz{Time: params.ExpiresAt, Valid: true},
	})
	if err != nil {
		return Session{}, err
	}

	return Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	row, err := r.queries.GetValidAuthSessionByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}

	return Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.queries.RevokeAuthSessionByTokenHash(ctx, tokenHash)
	return err
}
