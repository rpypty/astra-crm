package teams

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var ErrNotFound = errors.New("team not found")

type Team struct {
	ID     int64
	Name   string
	Status string
}

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Team, error) {
	row, err := r.queries.GetTeamByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Team{}, ErrNotFound
	}
	if err != nil {
		return Team{}, err
	}

	return Team{
		ID:     row.ID,
		Name:   row.Name,
		Status: row.Status,
	}, nil
}
