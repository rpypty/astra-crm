package banks

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var (
	ErrBankNotFound      = errors.New("bank not found")
	ErrDuplicateCSVAlias = errors.New("bank csv alias already exists")
	ErrInvalidBankInput  = errors.New("invalid bank input")
)

type Bank struct {
	ID        int64
	Code      string
	Name      string
	CSVAlias  *string
	Status    string
	SortOrder int64
	CreatedAt time.Time
	UpdatedAt time.Time
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
			CSVAlias:  textPtr(row.CsvAlias),
			Status:    row.Status,
			SortOrder: row.SortOrder,
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		})
	}

	return items, nil
}

func (r *Repository) UpdateCSVAlias(ctx context.Context, code string, alias string) (Bank, error) {
	row, err := r.queries.UpdateBankCSVAlias(ctx, db.UpdateBankCSVAliasParams{
		Code:     code,
		CsvAlias: alias,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Bank{}, ErrBankNotFound
	}
	if err != nil {
		return Bank{}, mapBankWriteError(err)
	}

	return Bank{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		CSVAlias:  textPtr(row.CsvAlias),
		Status:    row.Status,
		SortOrder: row.SortOrder,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func mapBankWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	if pgErr.ConstraintName == "uq_banks_csv_alias_normalized" {
		return ErrDuplicateCSVAlias
	}
	return err
}
