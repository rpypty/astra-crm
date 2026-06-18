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

func (r *Repository) ShiftHistory(ctx context.Context, teamID int64, traderID int64, limit int32) ([]Shift, error) {
	rows, err := r.queries.ListTraderShiftHistory(ctx, db.ListTraderShiftHistoryParams{
		TeamID:     teamID,
		TraderID:   traderID,
		LimitCount: limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Shift, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBShift(row))
	}

	return items, nil
}

func (r *Repository) TeamShiftHistory(ctx context.Context, teamID int64, limit int32) ([]Shift, error) {
	rows, err := r.queries.ListTeamShiftHistory(ctx, db.ListTeamShiftHistoryParams{
		TeamID:     teamID,
		LimitCount: limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Shift, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBShift(row))
	}

	return items, nil
}

func (r *Repository) ShiftReport(ctx context.Context, teamID int64, traderID int64, shiftID int64) (ShiftReportDetails, error) {
	shift, err := r.queries.GetTraderShiftByIDForTrader(ctx, db.GetTraderShiftByIDForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
		ShiftID:  shiftID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftReportDetails{}, ErrCurrentShiftNotFound
	}
	if err != nil {
		return ShiftReportDetails{}, err
	}

	return r.shiftReport(ctx, fromDBShift(shift))
}

func (r *Repository) TeamShiftReport(ctx context.Context, teamID int64, shiftID int64) (ShiftReportDetails, error) {
	shift, err := r.queries.GetTraderShiftByIDForTeam(ctx, db.GetTraderShiftByIDForTeamParams{
		TeamID:  teamID,
		ShiftID: shiftID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftReportDetails{}, ErrCurrentShiftNotFound
	}
	if err != nil {
		return ShiftReportDetails{}, err
	}

	return r.shiftReport(ctx, fromDBShift(shift))
}

func (r *Repository) shiftReport(ctx context.Context, shift Shift) (ShiftReportDetails, error) {
	rows, err := r.queries.ListShiftReportRows(ctx, db.ListShiftReportRowsParams{
		TeamID:  shift.TeamID,
		ShiftID: shift.ID,
	})
	if err != nil {
		return ShiftReportDetails{}, err
	}

	inbound, err := r.latestReportReconciliation(ctx, shift.TeamID, shift.ID, "inbound")
	if err != nil {
		return ShiftReportDetails{}, err
	}
	outbound, err := r.latestReportReconciliation(ctx, shift.TeamID, shift.ID, "outbound")
	if err != nil {
		return ShiftReportDetails{}, err
	}

	items := make([]ShiftReportRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromShiftReportRow(row))
	}

	return ShiftReportDetails{
		Shift:    shift,
		Inbound:  inbound,
		Outbound: outbound,
		Rows:     items,
	}, nil
}

func (r *Repository) latestReportReconciliation(ctx context.Context, teamID int64, shiftID int64, direction string) (*ShiftReportReconciliation, error) {
	var (
		run db.ReconciliationRun
		err error
	)
	switch direction {
	case "inbound":
		run, err = r.queries.LatestTraderInboundReconciliationRunByShift(ctx, db.LatestTraderInboundReconciliationRunByShiftParams{
			TeamID:  teamID,
			ShiftID: pgtype.Int8{Int64: shiftID, Valid: true},
		})
	case "outbound":
		run, err = r.queries.LatestTraderOutboundReconciliationRunByShift(ctx, db.LatestTraderOutboundReconciliationRunByShiftParams{
			TeamID:  teamID,
			ShiftID: pgtype.Int8{Int64: shiftID, Valid: true},
		})
	default:
		return nil, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return fromDBReportReconciliation(run), nil
}

func (r *Repository) ActiveAssignment(ctx context.Context, teamID int64, traderID int64, requisiteID int64) (int64, error) {
	row, err := r.queries.GetActiveAssignmentForTraderRequisite(ctx, db.GetActiveAssignmentForTraderRequisiteParams{
		TeamID:      teamID,
		TraderID:    traderID,
		RequisiteID: requisiteID,
		Today:       currentBusinessDate(),
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
		Today:    currentBusinessDate(),
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

func (r *Repository) FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	rows, err := r.queries.ListFutureAssignedRequisitesForTrader(ctx, db.ListFutureAssignedRequisitesForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
		Today:    currentBusinessDate(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromFutureAssignedRow(row))
	}

	return items, nil
}

func (r *Repository) HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	rows, err := r.queries.ListHistoricalAssignedRequisitesForTrader(ctx, db.ListHistoricalAssignedRequisitesForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
		Today:    currentBusinessDate(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromHistoricalAssignedRow(row))
	}

	return items, nil
}

func currentBusinessDate() pgtype.Date {
	now := time.Now().In(businessLocation())
	return pgtype.Date{Time: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), Valid: true}
}

func businessLocation() *time.Location {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("Europe/Moscow", 3*60*60)
	}

	return location
}

func (r *Repository) AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64) ([]AssignedRequisite, error) {
	rows, err := r.queries.ListAssignedRequisitesForShift(ctx, db.ListAssignedRequisitesForShiftParams{
		TeamID:   teamID,
		TraderID: traderID,
		ShiftID:  shiftID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromAssignedShiftRow(row))
	}

	return items, nil
}

func (r *Repository) AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64) ([]AssignedRequisite, error) {
	rows, err := r.queries.ListAssignedRequisitesForTeamShift(ctx, db.ListAssignedRequisitesForTeamShiftParams{
		TeamID:  teamID,
		ShiftID: shiftID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromAssignedTeamShiftRow(row))
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
	if err := r.queries.FillRequisiteInitialDetails(ctx, db.FillRequisiteInitialDetailsParams{
		TeamID:          params.TeamID,
		RequisiteID:     params.RequisiteID,
		HolderName:      pgtype.Text{String: params.HolderName, Valid: true},
		CardNumber:      pgtype.Text{String: params.CardNumber, Valid: true},
		DetailsFilledBy: pgtype.Int8{Int64: params.TraderID, Valid: true},
	}); err != nil {
		return ShiftRequisite{}, err
	}
	if _, err := r.queries.StartRequisiteAssignmentWork(ctx, db.StartRequisiteAssignmentWorkParams{
		TeamID:           params.TeamID,
		AssignmentID:     params.AssignmentID,
		ShiftRequisiteID: pgtype.Int8{Int64: row.ID, Valid: true},
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, err
	}

	return fromCreateShiftRequisiteRow(row), nil
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
		items = append(items, fromListShiftRequisiteRow(row))
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

	return fromUpdateShiftRequisiteRow(row), nil
}

func (r *Repository) CloseShiftRequisite(ctx context.Context, params CloseShiftRequisiteRecord) (ShiftRequisite, error) {
	row, err := r.queries.CloseShiftRequisite(ctx, db.CloseShiftRequisiteParams{
		TeamID:                params.TeamID,
		TraderID:              params.TraderID,
		ShiftRequisiteID:      params.ShiftRequisiteID,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		Blocked:               params.Blocked,
		CreatedBy:             params.CreatedBy,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, ErrShiftRequisiteNotFound
	}
	if err != nil {
		return ShiftRequisite{}, err
	}
	if _, err := r.queries.CompleteRequisiteAssignmentWork(ctx, db.CompleteRequisiteAssignmentWorkParams{
		TeamID:           params.TeamID,
		ShiftRequisiteID: pgtype.Int8{Int64: row.ID, Valid: true},
		Blocked:          params.Blocked,
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, err
	}

	return fromCloseShiftRequisiteRow(row), nil
}

func (r *Repository) CorrectClosedShiftRequisiteTurnovers(ctx context.Context, params CorrectShiftRequisiteTurnoversRecord) (ShiftRequisite, error) {
	row, err := r.queries.CorrectClosedShiftRequisiteTurnovers(ctx, db.CorrectClosedShiftRequisiteTurnoversParams{
		TeamID:                params.TeamID,
		TraderID:              params.TraderID,
		ShiftRequisiteID:      params.ShiftRequisiteID,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		CreatedBy:             params.CreatedBy,
		Comment:               pgtype.Text{String: params.Comment, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, ErrShiftRequisiteNotFound
	}
	if err != nil {
		return ShiftRequisite{}, err
	}

	return fromCorrectShiftRequisiteRow(row), nil
}

func (r *Repository) ReturnShiftRequisiteToWork(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) (ShiftRequisite, error) {
	row, err := r.queries.ReturnShiftRequisiteToWork(ctx, db.ReturnShiftRequisiteToWorkParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, ErrShiftRequisiteNotFound
	}
	if err != nil {
		return ShiftRequisite{}, err
	}

	return fromReturnShiftRequisiteToWorkRow(row), nil
}

func (r *Repository) GetShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) (ShiftRequisite, error) {
	row, err := r.queries.GetShiftRequisiteForTrader(ctx, db.GetShiftRequisiteForTraderParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ShiftRequisite{}, ErrShiftRequisiteNotFound
	}
	if err != nil {
		return ShiftRequisite{}, err
	}

	return fromGetShiftRequisiteForTraderRow(row), nil
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

type CloseShiftRequisiteRecord struct {
	TeamID                int64
	TraderID              int64
	ShiftRequisiteID      int64
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
	Blocked               bool
	CreatedBy             int64
}

type CorrectShiftRequisiteTurnoversRecord struct {
	TeamID                int64
	TraderID              int64
	ShiftRequisiteID      int64
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
	CreatedBy             int64
	Comment               string
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
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		EmployeeComment:       textPtr(row.EmployeeComment),
		Status:                row.Status,
		AssignmentID:          row.AssignmentID,
		AssignmentStatus:      row.AssignmentStatus,
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		ShiftRequisiteID:      positiveInt64Ptr(row.ShiftRequisiteID),
		CardNumber:            stringPtr(row.CardNumber),
		HolderName:            stringPtr(row.HolderName),
		ShiftRequisiteStatus:  stringPtr(row.ShiftRequisiteStatus),
		TakenAt:               timePtr(row.TakenAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromFutureAssignedRow(row db.ListFutureAssignedRequisitesForTraderRow) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		EmployeeComment:       textPtr(row.EmployeeComment),
		Status:                row.Status,
		AssignmentID:          row.AssignmentID,
		AssignmentStatus:      row.AssignmentStatus,
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		ShiftRequisiteID:      positiveInt64Ptr(row.ShiftRequisiteID),
		CardNumber:            stringPtr(row.CardNumber),
		HolderName:            stringPtr(row.HolderName),
		ShiftRequisiteStatus:  stringPtr(row.ShiftRequisiteStatus),
		TakenAt:               timePtr(row.TakenAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromHistoricalAssignedRow(row db.ListHistoricalAssignedRequisitesForTraderRow) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		EmployeeComment:       textPtr(row.EmployeeComment),
		Status:                row.Status,
		AssignmentID:          row.AssignmentID,
		AssignmentStatus:      row.AssignmentStatus,
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		ShiftRequisiteID:      positiveInt64Ptr(row.ShiftRequisiteID),
		CardNumber:            stringPtr(row.CardNumber),
		HolderName:            stringPtr(row.HolderName),
		ShiftRequisiteStatus:  stringPtr(row.ShiftRequisiteStatus),
		TakenAt:               timePtr(row.TakenAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromAssignedShiftRow(row db.ListAssignedRequisitesForShiftRow) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		EmployeeComment:       textPtr(row.EmployeeComment),
		Status:                row.Status,
		AssignmentID:          int64Value(row.AssignmentID),
		AssignmentStatus:      textString(row.AssignmentStatus),
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   int64Value(row.TargetTurnoverMinor),
		ShiftRequisiteID:      positiveInt64Ptr(row.ShiftRequisiteID),
		CardNumber:            stringPtr(row.CardNumber),
		HolderName:            stringPtr(row.HolderName),
		ShiftRequisiteStatus:  stringPtr(row.ShiftRequisiteStatus),
		TakenAt:               timePtr(row.TakenAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromAssignedTeamShiftRow(row db.ListAssignedRequisitesForTeamShiftRow) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		EmployeeComment:       textPtr(row.EmployeeComment),
		Status:                row.Status,
		AssignmentID:          int64Value(row.AssignmentID),
		AssignmentStatus:      textString(row.AssignmentStatus),
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   int64Value(row.TargetTurnoverMinor),
		ShiftRequisiteID:      positiveInt64Ptr(row.ShiftRequisiteID),
		CardNumber:            stringPtr(row.CardNumber),
		HolderName:            stringPtr(row.HolderName),
		ShiftRequisiteStatus:  stringPtr(row.ShiftRequisiteStatus),
		TakenAt:               timePtr(row.TakenAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromCreateShiftRequisiteRow(row db.CreateShiftRequisiteRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromListShiftRequisiteRow(row db.ListShiftRequisitesByTraderRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromUpdateShiftRequisiteRow(row db.UpdateShiftRequisiteDetailsRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromCloseShiftRequisiteRow(row db.CloseShiftRequisiteRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromShiftReportRow(row db.ListShiftReportRowsRow) ShiftReportRow {
	return ShiftReportRow{
		RowKey:                row.RowKey,
		ShiftRequisiteID:      int64Ptr(row.ShiftRequisiteID),
		RequisiteID:           int64Ptr(row.RequisiteID),
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		EmployeeComment:       textPtr(row.EmployeeComment),
		CardNumber:            textPtr(row.CardNumber),
		HolderName:            textPtr(row.HolderName),
		Status:                row.Status,
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		CSVInboundMinor:       row.CsvInboundMinor,
		CSVOutboundMinor:      row.CsvOutboundMinor,
		InboundDiffMinor:      row.InboundDiffMinor,
		OutboundDiffMinor:     row.OutboundDiffMinor,
		HasMismatch:           row.HasMismatch,
		CSVOnly:               row.CsvOnly,
	}
}

func fromDBReportReconciliation(row db.ReconciliationRun) *ShiftReportReconciliation {
	return &ShiftReportReconciliation{
		ID:                  row.ID,
		Status:              row.Status,
		ExpectedAmountMinor: row.ExpectedAmountMinor,
		ActualAmountMinor:   row.ActualAmountMinor,
		DiffAmountMinor:     row.DiffAmountMinor,
		Comment:             textPtr(row.Comment),
		CreatedAt:           row.CreatedAt.Time,
	}
}

func fromCorrectShiftRequisiteRow(row db.CorrectClosedShiftRequisiteTurnoversRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromReturnShiftRequisiteToWorkRow(row db.ReturnShiftRequisiteToWorkRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromGetShiftRequisiteForTraderRow(row db.GetShiftRequisiteForTraderRow) ShiftRequisite {
	return fromShiftRequisiteFields(row.ID, row.TeamID, row.ShiftID, row.TraderID, row.RequisiteID, row.AssignmentID, row.CardNumber, row.HolderName, row.TakenAt, row.ReleasedAt, row.Status, row.InboundTurnoverMinor, row.OutboundTurnoverMinor, row.ClosingBalanceMinor, row.CreatedAt, row.UpdatedAt)
}

func fromShiftRequisiteFields(id int64, teamID int64, shiftID int64, traderID int64, requisiteID int64, assignmentID pgtype.Int8, cardNumber string, holderName string, takenAt pgtype.Timestamptz, releasedAt pgtype.Timestamptz, status string, inboundTurnoverMinor int64, outboundTurnoverMinor int64, closingBalanceMinor int64, createdAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz) ShiftRequisite {
	return ShiftRequisite{
		ID:                    id,
		TeamID:                teamID,
		ShiftID:               shiftID,
		TraderID:              traderID,
		RequisiteID:           requisiteID,
		AssignmentID:          int64Ptr(assignmentID),
		CardNumber:            cardNumber,
		HolderName:            holderName,
		TakenAt:               takenAt.Time,
		ReleasedAt:            timePtr(releasedAt),
		Status:                status,
		InboundTurnoverMinor:  inboundTurnoverMinor,
		OutboundTurnoverMinor: outboundTurnoverMinor,
		ClosingBalanceMinor:   closingBalanceMinor,
		CreatedAt:             createdAt.Time,
		UpdatedAt:             updatedAt.Time,
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
		AllRequisitesClosed: true,
		AllPayoutsFullyPaid: row.AllPayoutsFullyPaid,
		UnpaidPayoutCount:   row.UnpaidPayoutCount,
	}
	checklist.CanClose = checklist.InboundImported &&
		checklist.InboundOk &&
		checklist.OutboundImported &&
		checklist.OutboundOk &&
		checklist.AllRequisitesClosed &&
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

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
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

func int64Value(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}

	return value.Int64
}

func positiveInt64Ptr(value int64) *int64 {
	if value <= 0 {
		return nil
	}

	return &value
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
