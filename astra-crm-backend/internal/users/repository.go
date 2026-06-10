package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var ErrNotFound = errors.New("user not found")

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}

	return fromSQLC(row), nil
}

func (r *Repository) GetByLogin(ctx context.Context, login string) (User, error) {
	row, err := r.queries.GetUserByLogin(ctx, login)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}

	return fromSQLC(row), nil
}

func (r *Repository) ListTradersByTeam(ctx context.Context, teamID int64) ([]User, error) {
	rows, err := r.queries.ListTradersByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}

	items := make([]User, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromSQLC(row))
	}

	return items, nil
}

func (r *Repository) CreateTrader(ctx context.Context, params CreateTraderRecord) (Trader, error) {
	row, err := r.queries.CreateTrader(ctx, db.CreateTraderParams{
		TeamID:             params.TeamID,
		Login:              params.Login,
		PasswordHash:       params.PasswordHash,
		SalaryRateBps:      params.SalaryRateBps,
		ExternalWorkerName: params.ExternalWorkerName,
	})
	if err != nil {
		return Trader{}, mapTraderWriteError(err)
	}

	return fromCreateTraderRow(row), nil
}

func (r *Repository) GetTraderByID(ctx context.Context, teamID int64, traderID int64) (Trader, error) {
	row, err := r.queries.GetTraderByIDForTeam(ctx, db.GetTraderByIDForTeamParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Trader{}, ErrTraderNotFound
	}
	if err != nil {
		return Trader{}, err
	}

	return fromGetTraderByIDForTeamRow(row), nil
}

func (r *Repository) ListTraderDetailsByTeam(ctx context.Context, teamID int64) ([]Trader, error) {
	rows, err := r.queries.ListTraderDetailsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}

	items := make([]Trader, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListTraderDetailsByTeamRow(row))
	}

	return items, nil
}

func (r *Repository) UpdateTrader(ctx context.Context, params UpdateTraderRecord) (Trader, error) {
	row, err := r.queries.UpdateTrader(ctx, db.UpdateTraderParams{
		TeamID:             params.TeamID,
		TraderID:           params.TraderID,
		Status:             params.Status,
		SalaryRateBps:      params.SalaryRateBps,
		ExternalWorkerName: params.ExternalWorkerName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Trader{}, ErrTraderNotFound
	}
	if err != nil {
		return Trader{}, mapTraderWriteError(err)
	}

	return fromUpdateTraderRow(row), nil
}

func (r *Repository) UpdateTraderPasswordHash(ctx context.Context, teamID int64, traderID int64, passwordHash string) error {
	return r.queries.UpdateTraderPasswordHash(ctx, db.UpdateTraderPasswordHashParams{
		TeamID:       teamID,
		TraderID:     traderID,
		PasswordHash: passwordHash,
	})
}

func fromSQLC(row db.User) User {
	return User{
		ID:           row.ID,
		TeamID:       row.TeamID,
		Role:         row.Role,
		Login:        row.Login,
		PasswordHash: row.PasswordHash,
		Status:       row.Status,
	}
}

func fromCreateTraderRow(row db.CreateTraderRow) Trader {
	return Trader{
		ID:                 row.ID,
		TeamID:             row.TeamID,
		Role:               row.Role,
		Login:              row.Login,
		Status:             row.Status,
		SalaryRateBps:      row.SalaryRateBps,
		ExternalWorkerName: row.ExternalWorkerName,
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}

func fromGetTraderByIDForTeamRow(row db.GetTraderByIDForTeamRow) Trader {
	return Trader{
		ID:                 row.ID,
		TeamID:             row.TeamID,
		Role:               row.Role,
		Login:              row.Login,
		Status:             row.Status,
		SalaryRateBps:      row.SalaryRateBps,
		ExternalWorkerName: row.ExternalWorkerName,
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}

func fromListTraderDetailsByTeamRow(row db.ListTraderDetailsByTeamRow) Trader {
	return Trader{
		ID:                 row.ID,
		TeamID:             row.TeamID,
		Role:               row.Role,
		Login:              row.Login,
		Status:             row.Status,
		SalaryRateBps:      row.SalaryRateBps,
		ExternalWorkerName: row.ExternalWorkerName,
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}

func fromUpdateTraderRow(row db.UpdateTraderRow) Trader {
	return Trader{
		ID:                 row.ID,
		TeamID:             row.TeamID,
		Role:               row.Role,
		Login:              row.Login,
		Status:             row.Status,
		SalaryRateBps:      row.SalaryRateBps,
		ExternalWorkerName: row.ExternalWorkerName,
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}

func mapTraderWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "users_team_id_login_key":
		return ErrDuplicateLogin
	case "trader_profiles_external_worker_name_key":
		return ErrDuplicateWorkerName
	default:
		return err
	}
}
