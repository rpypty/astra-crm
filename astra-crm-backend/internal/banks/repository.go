package banks

import (
	"context"
	"time"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

type Bank struct {
	ID        int64
	Code      string
	Name      string
	Status    string
	SortOrder int64
	CreatedAt time.Time
}

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) ListActive(ctx context.Context) ([]Bank, error) {
	rows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]Bank, 0, len(rows))
	for _, row := range rows {
		items = append(items, Bank{
			ID:        row.ID,
			Code:      row.Code,
			Name:      row.Name,
			Status:    row.Status,
			SortOrder: row.SortOrder,
			CreatedAt: row.CreatedAt.Time,
		})
	}

	return items, nil
}
