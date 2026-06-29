package users

import (
	"context"
	"errors"
	"math"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
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

func (r *Repository) ListTraderDetailsByTeam(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[Trader], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListTraderDetailsByTeam(ctx, db.ListTraderDetailsByTeamParams{
		TeamID:      teamID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[Trader]{}, err
	}

	items := make([]Trader, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListTraderDetailsByTeamRow(row))
	}

	total, err := r.queries.CountTraderDetailsByTeam(ctx, teamID)
	if err != nil {
		return pagination.Result[Trader]{}, err
	}

	return pagination.NewResult(items, page, total), nil
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
	case "users_login_key", "users_team_id_login_key":
		return ErrDuplicateLogin
	case "trader_profiles_external_worker_name_key":
		return ErrDuplicateWorkerName
	default:
		return err
	}
}

func paginationOffset32(params pagination.Params) int32 {
	offset := pagination.Offset(params)
	if offset > math.MaxInt32 {
		return math.MaxInt32
	}

	return int32(offset)
}

func paginationLimit32(params pagination.Params) int32 {
	params = pagination.Normalize(params)
	return int32(params.PageSize)
}
