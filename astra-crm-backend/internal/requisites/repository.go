package requisites

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var (
	ErrNotFound           = errors.New("requisite not found")
	ErrAssignmentNotFound = errors.New("requisite assignment not found")
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusArchived = "archived"
)

type Requisite struct {
	ID         int64
	TeamID     int64
	Phone      string
	MethodType string
	Proxy      *string
	Status     string
	CreatedBy  int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type RequisiteDetails struct {
	Requisite
	ActiveAssignmentID  *int64
	AssignedTraderID    *int64
	AssignedTraderLogin *string
}

type Assignment struct {
	ID           int64
	TeamID       int64
	RequisiteID  int64
	TraderID     int64
	AssignedBy   int64
	AssignedAt   time.Time
	UnassignedAt *time.Time
	Comment      *string
	WasReassign  bool
}

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetByIDForTeam(ctx context.Context, id int64, teamID int64) (Requisite, error) {
	row, err := r.queries.GetRequisiteByIDForTeam(ctx, db.GetRequisiteByIDForTeamParams{
		ID:     id,
		TeamID: teamID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Requisite{}, ErrNotFound
	}
	if err != nil {
		return Requisite{}, err
	}

	return Requisite{
		ID:         row.ID,
		TeamID:     row.TeamID,
		Phone:      row.Phone,
		MethodType: row.MethodType,
		Proxy:      textPtr(row.Proxy),
		Status:     row.Status,
		CreatedBy:  row.CreatedBy,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) Create(ctx context.Context, params CreateRecord) (Requisite, error) {
	row, err := r.queries.CreateRequisite(ctx, db.CreateRequisiteParams{
		TeamID:     params.TeamID,
		Phone:      params.Phone,
		MethodType: params.MethodType,
		Proxy:      textValue(params.Proxy),
		CreatedBy:  params.CreatedBy,
	})
	if err != nil {
		return Requisite{}, err
	}

	return fromDBRequisite(row), nil
}

func (r *Repository) GetDetails(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error) {
	row, err := r.queries.GetRequisiteDetailsByIDForTeam(ctx, db.GetRequisiteDetailsByIDForTeamParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return RequisiteDetails{}, ErrNotFound
	}
	if err != nil {
		return RequisiteDetails{}, err
	}

	return fromDetailsRow(row), nil
}

func (r *Repository) ListDetails(ctx context.Context, teamID int64) ([]RequisiteDetails, error) {
	rows, err := r.queries.ListRequisiteDetailsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}

	items := make([]RequisiteDetails, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListDetailsRow(row))
	}

	return items, nil
}

func (r *Repository) Update(ctx context.Context, params UpdateRecord) (Requisite, error) {
	row, err := r.queries.UpdateRequisite(ctx, db.UpdateRequisiteParams{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
		Phone:       params.Phone,
		MethodType:  params.MethodType,
		Proxy:       textValue(params.Proxy),
		Status:      params.Status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Requisite{}, ErrNotFound
	}
	if err != nil {
		return Requisite{}, err
	}

	return fromDBRequisite(row), nil
}

func (r *Repository) Assign(ctx context.Context, params AssignRecord) (Assignment, error) {
	row, err := r.queries.AssignRequisite(ctx, db.AssignRequisiteParams{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
		TraderID:    params.TraderID,
		AssignedBy:  params.AssignedBy,
		Comment:     textValue(params.Comment),
	})
	if err != nil {
		return Assignment{}, err
	}

	return fromAssignRow(row), nil
}

func (r *Repository) Unassign(ctx context.Context, teamID int64, requisiteID int64) (Assignment, error) {
	row, err := r.queries.UnassignRequisite(ctx, db.UnassignRequisiteParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound
	}
	if err != nil {
		return Assignment{}, err
	}

	return fromDBAssignment(row), nil
}

func (r *Repository) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64) ([]Assignment, error) {
	rows, err := r.queries.ListRequisiteAssignmentHistory(ctx, db.ListRequisiteAssignmentHistoryParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBAssignment(row))
	}

	return items, nil
}

func (r *Repository) ListActiveAssignmentsByTrader(ctx context.Context, teamID int64, traderID int64) ([]Assignment, error) {
	rows, err := r.queries.ListActiveRequisiteAssignmentsByTrader(ctx, db.ListActiveRequisiteAssignmentsByTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBAssignment(row))
	}

	return items, nil
}

type CreateRecord struct {
	TeamID     int64
	Phone      string
	MethodType string
	Proxy      *string
	CreatedBy  int64
}

type UpdateRecord struct {
	TeamID      int64
	RequisiteID int64
	Phone       string
	MethodType  string
	Proxy       *string
	Status      string
}

type AssignRecord struct {
	TeamID      int64
	RequisiteID int64
	TraderID    int64
	AssignedBy  int64
	Comment     *string
}

func fromDBRequisite(row db.Requisite) Requisite {
	return Requisite{
		ID:         row.ID,
		TeamID:     row.TeamID,
		Phone:      row.Phone,
		MethodType: row.MethodType,
		Proxy:      textPtr(row.Proxy),
		Status:     row.Status,
		CreatedBy:  row.CreatedBy,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func fromDetailsRow(row db.GetRequisiteDetailsByIDForTeamRow) RequisiteDetails {
	return RequisiteDetails{
		Requisite: Requisite{
			ID:         row.ID,
			TeamID:     row.TeamID,
			Phone:      row.Phone,
			MethodType: row.MethodType,
			Proxy:      textPtr(row.Proxy),
			Status:     row.Status,
			CreatedBy:  row.CreatedBy,
			CreatedAt:  row.CreatedAt.Time,
			UpdatedAt:  row.UpdatedAt.Time,
		},
		ActiveAssignmentID:  int64Ptr(row.ActiveAssignmentID),
		AssignedTraderID:    int64Ptr(row.AssignedTraderID),
		AssignedTraderLogin: textPtr(row.AssignedTraderLogin),
	}
}

func fromListDetailsRow(row db.ListRequisiteDetailsByTeamRow) RequisiteDetails {
	return RequisiteDetails{
		Requisite: Requisite{
			ID:         row.ID,
			TeamID:     row.TeamID,
			Phone:      row.Phone,
			MethodType: row.MethodType,
			Proxy:      textPtr(row.Proxy),
			Status:     row.Status,
			CreatedBy:  row.CreatedBy,
			CreatedAt:  row.CreatedAt.Time,
			UpdatedAt:  row.UpdatedAt.Time,
		},
		ActiveAssignmentID:  int64Ptr(row.ActiveAssignmentID),
		AssignedTraderID:    int64Ptr(row.AssignedTraderID),
		AssignedTraderLogin: textPtr(row.AssignedTraderLogin),
	}
}

func fromAssignRow(row db.AssignRequisiteRow) Assignment {
	assignment := fromAssignmentFields(row.ID, row.TeamID, row.RequisiteID, row.TraderID, row.AssignedBy, row.AssignedAt, row.UnassignedAt, row.Comment)
	assignment.WasReassign = row.WasReassign
	return assignment
}

func fromDBAssignment(row db.RequisiteAssignment) Assignment {
	return fromAssignmentFields(row.ID, row.TeamID, row.RequisiteID, row.TraderID, row.AssignedBy, row.AssignedAt, row.UnassignedAt, row.Comment)
}

func fromAssignmentFields(id int64, teamID int64, requisiteID int64, traderID int64, assignedBy int64, assignedAt pgtype.Timestamptz, unassignedAt pgtype.Timestamptz, comment pgtype.Text) Assignment {
	return Assignment{
		ID:           id,
		TeamID:       teamID,
		RequisiteID:  requisiteID,
		TraderID:     traderID,
		AssignedBy:   assignedBy,
		AssignedAt:   assignedAt.Time,
		UnassignedAt: timePtr(unassignedAt),
		Comment:      textPtr(comment),
	}
}

func textValue(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func int64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	return &value.Int64
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
