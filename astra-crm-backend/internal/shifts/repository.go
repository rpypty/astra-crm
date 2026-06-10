package shifts

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
	ErrCurrentShiftNotFound     = errors.New("current shift not found")
	ErrActiveAssignmentNotFound = errors.New("active requisite assignment not found")
	ErrShiftRequisiteNotFound   = errors.New("shift requisite not found")
	ErrShiftRequisiteExists     = errors.New("shift requisite already exists")
	ErrTurnoverTargetNotFound   = errors.New("turnover target not found")
	ErrShiftCannotBeClosed      = errors.New("shift cannot be closed")
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) CurrentShift(ctx context.Context, teamID int64, traderID int64) (Shift, error) {
	row, err := r.queries.GetCurrentTraderShift(ctx, db.GetCurrentTraderShiftParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Shift{}, ErrCurrentShiftNotFound
	}
	if err != nil {
		return Shift{}, err
	}

	return fromDBShift(row), nil
}

func (r *Repository) CreateShift(ctx context.Context, teamID int64, traderID int64) (Shift, error) {
	row, err := r.queries.CreateTraderShift(ctx, db.CreateTraderShiftParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return Shift{}, err
	}

	return fromDBShift(row), nil
}

func (r *Repository) ActiveAssignment(ctx context.Context, teamID int64, traderID int64, requisiteID int64) (int64, error) {
	row, err := r.queries.GetActiveAssignmentForTraderRequisite(ctx, db.GetActiveAssignmentForTraderRequisiteParams{
		TeamID:      teamID,
		TraderID:    traderID,
		RequisiteID: requisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrActiveAssignmentNotFound
	}
	if err != nil {
		return 0, err
	}

	return row.ID, nil
}

func (r *Repository) AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	rows, err := r.queries.ListAssignedRequisitesForTrader(ctx, db.ListAssignedRequisitesForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromAssignedRow(row))
	}

	return items, nil
}

func (r *Repository) CreateShiftRequisite(ctx context.Context, params CreateShiftRequisiteRecord) (ShiftRequisite, error) {
	row, err := r.queries.CreateShiftRequisite(ctx, db.CreateShiftRequisiteParams{
		TeamID:       params.TeamID,
		ShiftID:      params.ShiftID,
		TraderID:     params.TraderID,
		RequisiteID:  params.RequisiteID,
		AssignmentID: pgtype.Int8{Int64: params.AssignmentID, Valid: true},
		CardNumber:   params.CardNumber,
		HolderName:   params.HolderName,
	})
	if err != nil {
		return ShiftRequisite{}, mapShiftWriteError(err)
	}

	return fromDBShiftRequisite(row), nil
}

func (r *Repository) ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]ShiftRequisite, error) {
	rows, err := r.queries.ListShiftRequisitesByTrader(ctx, db.ListShiftRequisitesByTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]ShiftRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBShiftRequisite(row))
	}

	return items, nil
}

func (r *Repository) UpdateShiftRequisiteDetails(ctx context.Context, params UpdateShiftRequisiteDetailsRecord) (ShiftRequisite, error) {
	row, err := r.queries.UpdateShiftRequisiteDetails(ctx, db.UpdateShiftRequisiteDetailsParams{
		TeamID:           params.TeamID,
		TraderID:         params.TraderID,
		ShiftRequisiteID: params.ShiftRequisiteID,
		CardNumber:       params.CardNumber,
		HolderName:       params.HolderName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, ErrShiftRequisiteNotFound
	}
	if err != nil {
		return ShiftRequisite{}, err
	}

	return fromDBShiftRequisite(row), nil
}

func (r *Repository) CreateTurnoverEntry(ctx context.Context, params CreateTurnoverEntryRecord) (TurnoverEntry, error) {
	row, err := r.queries.CreateTurnoverEntry(ctx, db.CreateTurnoverEntryParams{
		TeamID:           params.TeamID,
		TraderID:         params.TraderID,
		ShiftRequisiteID: params.ShiftRequisiteID,
		AmountMinor:      params.AmountMinor,
		CreatedBy:        params.CreatedBy,
		Comment:          textValue(params.Comment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return TurnoverEntry{}, ErrTurnoverTargetNotFound
	}
	if err != nil {
		return TurnoverEntry{}, err
	}

	return fromDBTurnover(row), nil
}

func (r *Repository) LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]TurnoverEntry, error) {
	rows, err := r.queries.ListLatestTurnoversForCurrentShift(ctx, db.ListLatestTurnoversForCurrentShiftParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]TurnoverEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBTurnover(row))
	}

	return items, nil
}

func (r *Repository) TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]TurnoverEntry, error) {
	rows, err := r.queries.ListTurnoversByShiftRequisite(ctx, db.ListTurnoversByShiftRequisiteParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]TurnoverEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBTurnover(row))
	}

	return items, nil
}

func (r *Repository) CurrentShiftChecklist(ctx context.Context, teamID int64, traderID int64) (CloseChecklist, error) {
	row, err := r.queries.GetCurrentShiftChecklist(ctx, db.GetCurrentShiftChecklistParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return CloseChecklist{}, ErrCurrentShiftNotFound
	}
	if err != nil {
		return CloseChecklist{}, err
	}

	return fromChecklistRow(row), nil
}

func (r *Repository) CloseCurrentShift(ctx context.Context, params CloseShiftRecord) (Shift, error) {
	row, err := r.queries.CloseCurrentTraderShift(ctx, db.CloseCurrentTraderShiftParams{
		TeamID:       params.TeamID,
		TraderID:     params.TraderID,
		ShiftID:      params.ShiftID,
		CloseComment: textValue(params.CloseComment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Shift{}, ErrShiftCannotBeClosed
	}
	if err != nil {
		return Shift{}, err
	}

	return fromDBShift(row), nil
}

type CreateShiftRequisiteRecord struct {
	TeamID       int64
	ShiftID      int64
	TraderID     int64
	RequisiteID  int64
	AssignmentID int64
	CardNumber   string
	HolderName   string
}

type CreateTurnoverEntryRecord struct {
	TeamID           int64
	TraderID         int64
	ShiftRequisiteID int64
	AmountMinor      int64
	CreatedBy        int64
	Comment          *string
}

type CloseShiftRecord struct {
	TeamID       int64
	TraderID     int64
	ShiftID      int64
	CloseComment *string
}

type UpdateShiftRequisiteDetailsRecord struct {
	TeamID           int64
	TraderID         int64
	ShiftRequisiteID int64
	CardNumber       string
	HolderName       string
}

func fromDBShift(row db.TraderShift) Shift {
	return Shift{
		ID:                           row.ID,
		TeamID:                       row.TeamID,
		TraderID:                     row.TraderID,
		StartedAt:                    row.StartedAt.Time,
		EndedAt:                      timePtr(row.EndedAt),
		Status:                       row.Status,
		InboundReconciliationStatus:  row.InboundReconciliationStatus,
		OutboundReconciliationStatus: row.OutboundReconciliationStatus,
		CloseComment:                 textPtr(row.CloseComment),
		CreatedAt:                    row.CreatedAt.Time,
		UpdatedAt:                    row.UpdatedAt.Time,
		ClosedAt:                     timePtr(row.ClosedAt),
	}
}

func fromAssignedRow(row db.ListAssignedRequisitesForTraderRow) AssignedRequisite {
	return AssignedRequisite{
		ID:                   row.ID,
		TeamID:               row.TeamID,
		Phone:                row.Phone,
		MethodType:           row.MethodType,
		Proxy:                textPtr(row.Proxy),
		Status:               row.Status,
		AssignmentID:         row.AssignmentID,
		ShiftRequisiteID:     int64Ptr(row.ShiftRequisiteID),
		CardNumber:           textPtr(row.CardNumber),
		HolderName:           textPtr(row.HolderName),
		ShiftRequisiteStatus: textPtr(row.ShiftRequisiteStatus),
		TakenAt:              timePtr(row.TakenAt),
	}
}

func fromDBShiftRequisite(row db.ShiftRequisite) ShiftRequisite {
	return ShiftRequisite{
		ID:           row.ID,
		TeamID:       row.TeamID,
		ShiftID:      row.ShiftID,
		TraderID:     row.TraderID,
		RequisiteID:  row.RequisiteID,
		AssignmentID: int64Ptr(row.AssignmentID),
		CardNumber:   row.CardNumber,
		HolderName:   row.HolderName,
		TakenAt:      row.TakenAt.Time,
		ReleasedAt:   timePtr(row.ReleasedAt),
		Status:       row.Status,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}

func fromDBTurnover(row db.RequisiteTurnoverEntry) TurnoverEntry {
	return TurnoverEntry{
		ID:               row.ID,
		TeamID:           row.TeamID,
		ShiftID:          row.ShiftID,
		ShiftRequisiteID: row.ShiftRequisiteID,
		RequisiteID:      row.RequisiteID,
		TraderID:         row.TraderID,
		AmountMinor:      row.AmountMinor,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt.Time,
		Comment:          textPtr(row.Comment),
	}
}

func fromChecklistRow(row db.GetCurrentShiftChecklistRow) CloseChecklist {
	checklist := CloseChecklist{
		Shift: Shift{
			ID:                           row.ID,
			TeamID:                       row.TeamID,
			TraderID:                     row.TraderID,
			StartedAt:                    row.StartedAt.Time,
			EndedAt:                      timePtr(row.EndedAt),
			Status:                       row.Status,
			InboundReconciliationStatus:  row.InboundReconciliationStatus,
			OutboundReconciliationStatus: row.OutboundReconciliationStatus,
			CloseComment:                 textPtr(row.CloseComment),
			CreatedAt:                    row.CreatedAt.Time,
			UpdatedAt:                    row.UpdatedAt.Time,
			ClosedAt:                     timePtr(row.ClosedAt),
		},
		InboundImported:     row.InboundImported,
		InboundOk:           row.InboundOk,
		OutboundImported:    row.OutboundImported,
		OutboundOk:          row.OutboundOk,
		AllPayoutsFullyPaid: row.AllPayoutsFullyPaid,
		UnpaidPayoutCount:   row.UnpaidPayoutCount,
	}
	checklist.CanClose = checklist.InboundImported &&
		checklist.InboundOk &&
		checklist.OutboundImported &&
		checklist.OutboundOk &&
		checklist.AllPayoutsFullyPaid

	return checklist
}

func mapShiftWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}

	if pgErr.ConstraintName == "uq_shift_requisite_once" {
		return ErrShiftRequisiteExists
	}

	return err
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func textValue(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
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
