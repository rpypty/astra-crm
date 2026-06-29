package shifts

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var (
	ErrCurrentShiftNotFound           = errors.New("current shift not found")
	ErrActiveAssignmentNotFound       = errors.New("active requisite assignment not found")
	ErrShiftRequisiteNotFound         = errors.New("shift requisite not found")
	ErrShiftRequisiteExists           = errors.New("shift requisite already exists")
	ErrTurnoverTargetNotFound         = errors.New("turnover target not found")
	ErrShiftCannotBeClosed            = errors.New("shift cannot be closed")
	ErrInternalTransferTargetNotFound = errors.New("internal transfer target not found")
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

func (r *Repository) ShiftHistory(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[Shift], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListTraderShiftHistory(ctx, db.ListTraderShiftHistoryParams{
		TeamID:      teamID,
		TraderID:    traderID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[Shift]{}, err
	}

	items := make([]Shift, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBShift(row))
	}

	total, err := r.queries.CountTraderShiftHistory(ctx, db.CountTraderShiftHistoryParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return pagination.Result[Shift]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) TeamShiftHistory(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[Shift], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListTeamShiftHistory(ctx, db.ListTeamShiftHistoryParams{
		TeamID:      teamID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[Shift]{}, err
	}

	items := make([]Shift, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBShift(row))
	}

	total, err := r.queries.CountTeamShiftHistory(ctx, teamID)
	if err != nil {
		return pagination.Result[Shift]{}, err
	}

	return pagination.NewResult(items, page, total), nil
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
	var (
		requisiteRows []db.ListShiftReportRequisitesRow
		inboundRows   []db.ListShiftReportInboundScopeItemsRow
		transferRows  []db.ListShiftReportOutboundTransfersRow
		bankRows      []db.Bank
		inbound       *ShiftReportReconciliation
		outbound      *ShiftReportReconciliation
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		requisiteRows, err = r.queries.ListShiftReportRequisites(groupCtx, db.ListShiftReportRequisitesParams{
			TeamID:  shift.TeamID,
			ShiftID: shift.ID,
		})
		return err
	})
	g.Go(func() error {
		var err error
		inboundRows, err = r.queries.ListShiftReportInboundScopeItems(groupCtx, db.ListShiftReportInboundScopeItemsParams{
			TeamID:  shift.TeamID,
			ShiftID: pgtype.Int8{Int64: shift.ID, Valid: true},
		})
		return err
	})
	g.Go(func() error {
		var err error
		transferRows, err = r.queries.ListShiftReportOutboundTransfers(groupCtx, db.ListShiftReportOutboundTransfersParams{
			TeamID:  shift.TeamID,
			ShiftID: shift.ID,
		})
		return err
	})
	g.Go(func() error {
		var err error
		bankRows, err = r.queries.ListActiveBanks(groupCtx)
		return err
	})
	g.Go(func() error {
		var err error
		inbound, err = r.latestReportReconciliation(groupCtx, shift.TeamID, shift.ID, "inbound")
		return err
	})
	g.Go(func() error {
		var err error
		outbound, err = r.latestReportReconciliation(groupCtx, shift.TeamID, shift.ID, "outbound")
		return err
	})
	if err := g.Wait(); err != nil {
		return ShiftReportDetails{}, err
	}

	return ShiftReportDetails{
		Shift:    shift,
		Inbound:  inbound,
		Outbound: outbound,
		Rows: buildShiftReportRows(
			shiftReportRequisitesFromDB(requisiteRows),
			shiftReportInboundItemsFromDB(inboundRows),
			shiftReportOutboundTransfersFromDB(transferRows),
			bankNamesByCode(bankRows),
		),
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

func (r *Repository) AssignedRequisites(ctx context.Context, teamID int64, traderID int64, filters AssignedRequisitesFilters, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	page = pagination.Normalize(page)
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}
	bankNames := bankNamesByCode(bankRows)
	bankSearchCodes := matchingBankCodes(bankNames, filters.Search)

	rows, err := r.queries.ListAssignedRequisitesForTrader(ctx, db.ListAssignedRequisitesForTraderParams{
		TeamID:          teamID,
		TraderID:        traderID,
		Today:           currentBusinessDate(),
		ExcludeID:       optionalInt8(filters.ExcludeID),
		Statuses:        filters.Statuses,
		Search:          filters.Search,
		BankSearchCodes: bankSearchCodes,
		SearchDigits:    digitsOnly(filters.Search),
		OffsetCount:     paginationOffset32(page),
		LimitCount:      paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromAssignedRow(row, bankNames))
	}

	total, err := r.queries.CountAssignedRequisitesForTrader(ctx, db.CountAssignedRequisitesForTraderParams{
		TeamID:          teamID,
		TraderID:        traderID,
		Today:           currentBusinessDate(),
		ExcludeID:       optionalInt8(filters.ExcludeID),
		Statuses:        filters.Statuses,
		Search:          filters.Search,
		BankSearchCodes: bankSearchCodes,
		SearchDigits:    digitsOnly(filters.Search),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	page = pagination.Normalize(page)
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	rows, err := r.queries.ListFutureAssignedRequisitesForTrader(ctx, db.ListFutureAssignedRequisitesForTraderParams{
		TeamID:      teamID,
		TraderID:    traderID,
		Today:       currentBusinessDate(),
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromFutureAssignedRow(row, bankNames))
	}

	total, err := r.queries.CountFutureAssignedRequisitesForTrader(ctx, db.CountFutureAssignedRequisitesForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
		Today:    currentBusinessDate(),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	page = pagination.Normalize(page)
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	rows, err := r.queries.ListHistoricalAssignedRequisitesForTrader(ctx, db.ListHistoricalAssignedRequisitesForTraderParams{
		TeamID:      teamID,
		TraderID:    traderID,
		Today:       currentBusinessDate(),
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromHistoricalAssignedRow(row, bankNames))
	}

	total, err := r.queries.CountHistoricalAssignedRequisitesForTrader(ctx, db.CountHistoricalAssignedRequisitesForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
		Today:    currentBusinessDate(),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	return pagination.NewResult(items, page, total), nil
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

func (r *Repository) AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	page = pagination.Normalize(page)
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	rows, err := r.queries.ListAssignedRequisitesForShift(ctx, db.ListAssignedRequisitesForShiftParams{
		TeamID:      teamID,
		TraderID:    traderID,
		ShiftID:     shiftID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromAssignedShiftRow(row, bankNames))
	}

	total, err := r.queries.CountAssignedRequisitesForShift(ctx, db.CountAssignedRequisitesForShiftParams{
		TeamID:   teamID,
		TraderID: traderID,
		ShiftID:  shiftID,
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	page = pagination.Normalize(page)
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	rows, err := r.queries.ListAssignedRequisitesForTeamShift(ctx, db.ListAssignedRequisitesForTeamShiftParams{
		TeamID:      teamID,
		ShiftID:     shiftID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	items := make([]AssignedRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromAssignedTeamShiftRow(row, bankNames))
	}

	total, err := r.queries.CountAssignedRequisitesForTeamShift(ctx, db.CountAssignedRequisitesForTeamShiftParams{
		TeamID:  teamID,
		ShiftID: shiftID,
	})
	if err != nil {
		return pagination.Result[AssignedRequisite]{}, err
	}

	return pagination.NewResult(items, page, total), nil
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

func (r *Repository) ShiftRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[ShiftRequisite], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListShiftRequisitesByTrader(ctx, db.ListShiftRequisitesByTraderParams{
		TeamID:      teamID,
		TraderID:    traderID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[ShiftRequisite]{}, err
	}

	items := make([]ShiftRequisite, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListShiftRequisiteRow(row))
	}

	total, err := r.queries.CountShiftRequisitesByTrader(ctx, db.CountShiftRequisitesByTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return pagination.Result[ShiftRequisite]{}, err
	}

	return pagination.NewResult(items, page, total), nil
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
		ReleasedAt:            timePtrValue(params.ReleasedAt),
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
		CompletedAt:      timePtrValue(params.ReleasedAt),
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

func (r *Repository) TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64, page pagination.Params) (pagination.Result[TurnoverEntry], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListTurnoversByShiftRequisite(ctx, db.ListTurnoversByShiftRequisiteParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
		OffsetCount:      paginationOffset32(page),
		LimitCount:       paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[TurnoverEntry]{}, err
	}

	items := make([]TurnoverEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBTurnover(row))
	}

	total, err := r.queries.CountTurnoversByShiftRequisite(ctx, db.CountTurnoversByShiftRequisiteParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
	})
	if err != nil {
		return pagination.Result[TurnoverEntry]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) InternalTransfersByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64, page pagination.Params) (pagination.Result[InternalTransfer], error) {
	page = pagination.Normalize(page)
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return pagination.Result[InternalTransfer]{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	rows, err := r.queries.ListInternalTransfersForShiftRequisite(ctx, db.ListInternalTransfersForShiftRequisiteParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
		OffsetCount:      paginationOffset32(page),
		LimitCount:       paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[InternalTransfer]{}, err
	}

	items := make([]InternalTransfer, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListInternalTransferRow(row, bankNames))
	}

	total, err := r.queries.CountInternalTransfersForShiftRequisite(ctx, db.CountInternalTransfersForShiftRequisiteParams{
		TeamID:           teamID,
		TraderID:         traderID,
		ShiftRequisiteID: shiftRequisiteID,
	})
	if err != nil {
		return pagination.Result[InternalTransfer]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) CreateInternalTransfer(ctx context.Context, params CreateInternalTransferRecord) (InternalTransfer, error) {
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return InternalTransfer{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	row, err := r.queries.CreateInternalTransfer(ctx, db.CreateInternalTransferParams{
		TeamID:                      params.TeamID,
		TraderID:                    params.TraderID,
		SourceShiftRequisiteID:      params.SourceShiftRequisiteID,
		DestinationShiftRequisiteID: params.DestinationShiftRequisiteID,
		AmountMinor:                 params.AmountMinor,
		CreatedBy:                   params.CreatedBy,
		Comment:                     textValue(params.Comment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return InternalTransfer{}, ErrInternalTransferTargetNotFound
	}
	if err != nil {
		return InternalTransfer{}, err
	}

	return fromCreateInternalTransferRow(row, bankNames), nil
}

func (r *Repository) CancelInternalTransfer(ctx context.Context, params CancelInternalTransferRecord) (InternalTransfer, error) {
	bankRows, err := r.queries.ListActiveBanks(ctx)
	if err != nil {
		return InternalTransfer{}, err
	}
	bankNames := bankNamesByCode(bankRows)

	row, err := r.queries.CancelInternalTransfer(ctx, db.CancelInternalTransferParams{
		TeamID:      params.TeamID,
		TraderID:    params.TraderID,
		TransferID:  params.TransferID,
		CancelledBy: pgtype.Int8{Int64: params.CancelledBy, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return InternalTransfer{}, ErrInternalTransferNotFound
	}
	if err != nil {
		return InternalTransfer{}, err
	}

	return fromCancelInternalTransferRow(row, bankNames), nil
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
	ReleasedAt            *time.Time
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

type CreateInternalTransferRecord struct {
	TeamID                      int64
	TraderID                    int64
	SourceShiftRequisiteID      int64
	DestinationShiftRequisiteID int64
	AmountMinor                 int64
	CreatedBy                   int64
	Comment                     *string
}

type CancelInternalTransferRecord struct {
	TeamID      int64
	TraderID    int64
	TransferID  int64
	CancelledBy int64
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
		TLStatus:                     row.TlReconciliationStatus,
		LastTeamleadReconciliationID: int64Ptr(row.LastTeamleadReconciliationID),
		TLReconciledAt:               timePtr(row.TlReconciledAt),
		CloseComment:                 textPtr(row.CloseComment),
		CreatedAt:                    row.CreatedAt.Time,
		UpdatedAt:                    row.UpdatedAt.Time,
		ClosedAt:                     timePtr(row.ClosedAt),
	}
}

func fromAssignedRow(row db.ListAssignedRequisitesForTraderRow, bankNames map[string]string) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              bankNames[row.BankCode],
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
		ReleasedAt:            timePtr(row.ReleasedAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromFutureAssignedRow(row db.ListFutureAssignedRequisitesForTraderRow, bankNames map[string]string) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              bankNames[row.BankCode],
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
		ReleasedAt:            timePtr(row.ReleasedAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromHistoricalAssignedRow(row db.ListHistoricalAssignedRequisitesForTraderRow, bankNames map[string]string) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              bankNames[row.BankCode],
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
		ReleasedAt:            timePtr(row.ReleasedAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromAssignedShiftRow(row db.ListAssignedRequisitesForShiftRow, bankNames map[string]string) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              bankNames[row.BankCode],
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
		ReleasedAt:            timePtr(row.ReleasedAt),
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
	}
}

func fromAssignedTeamShiftRow(row db.ListAssignedRequisitesForTeamShiftRow, bankNames map[string]string) AssignedRequisite {
	return AssignedRequisite{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		Phone:                 row.Phone,
		MethodType:            row.MethodType,
		BankCode:              row.BankCode,
		BankName:              bankNames[row.BankCode],
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
		ReleasedAt:            timePtr(row.ReleasedAt),
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

type shiftReportRequisiteRow struct {
	ShiftRequisiteID       int64
	RequisiteID            int64
	Phone                  string
	MethodType             string
	BankCode               string
	Proxy                  *string
	EmployeeComment        *string
	CardNumber             string
	RequisiteCardNumber    *string
	HolderName             string
	Status                 string
	TLReconciliationStatus string
	InboundTurnoverMinor   int64
	OutboundTurnoverMinor  int64
	ClosingBalanceMinor    int64
	TargetTurnoverMinor    int64
}

type shiftReportInboundItem struct {
	CSVRequisite     string
	NormalizedStatus string
	AmountMinor      int64
}

type shiftReportOutboundTransfer struct {
	SourceShiftRequisiteID int64
	AmountMinor            int64
}

type shiftReportInboundAggregate struct {
	CSVRequisite    string
	CSVInboundMinor int64
}

func buildShiftReportRows(requisites []shiftReportRequisiteRow, inboundItems []shiftReportInboundItem, transfers []shiftReportOutboundTransfer, bankNames map[string]string) []ShiftReportRow {
	crmByMatchKey := make(map[string]shiftReportRequisiteRow, len(requisites))
	lookupCandidates := map[string]map[int64]string{}
	for _, requisite := range requisites {
		matchKey := crmMatchKey(requisite.ShiftRequisiteID)
		crmByMatchKey[matchKey] = requisite

		if phoneKey := rightDigits(requisite.Phone, 10); phoneKey != "" {
			addReportLookupCandidate(lookupCandidates, "phone:"+phoneKey, requisite.ShiftRequisiteID, matchKey)
		}
		if cardKey := digitsOnly(firstNonEmpty(requisite.CardNumber, stringPtrValue(requisite.RequisiteCardNumber))); cardKey != "" {
			addReportLookupCandidate(lookupCandidates, "card:"+cardKey, requisite.ShiftRequisiteID, matchKey)
		}
	}

	uniqueLookup := make(map[string]string, len(lookupCandidates))
	for lookupKey, candidates := range lookupCandidates {
		if len(candidates) != 1 {
			continue
		}
		for _, matchKey := range candidates {
			uniqueLookup[lookupKey] = matchKey
		}
	}

	inboundByMatchKey := map[string]shiftReportInboundAggregate{}
	for _, item := range inboundItems {
		matchKey := inboundReportMatchKey(item.CSVRequisite, uniqueLookup)
		aggregate := inboundByMatchKey[matchKey]
		if item.CSVRequisite > aggregate.CSVRequisite {
			aggregate.CSVRequisite = item.CSVRequisite
		}
		if item.NormalizedStatus == "success" || item.NormalizedStatus == "corrected" {
			aggregate.CSVInboundMinor += item.AmountMinor
		}
		inboundByMatchKey[matchKey] = aggregate
	}

	outboundByShiftRequisiteID := map[int64]int64{}
	for _, transfer := range transfers {
		outboundByShiftRequisiteID[transfer.SourceShiftRequisiteID] += transfer.AmountMinor
	}

	allKeys := map[string]bool{}
	for key := range crmByMatchKey {
		allKeys[key] = true
	}
	for key := range inboundByMatchKey {
		allKeys[key] = true
	}

	rows := make([]ShiftReportRow, 0, len(allKeys))
	for key := range allKeys {
		requisite, hasCRM := crmByMatchKey[key]
		inbound := inboundByMatchKey[key]
		rows = append(rows, buildShiftReportRow(key, requisite, hasCRM, inbound, outboundByShiftRequisiteID, bankNames))
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.HasMismatch != right.HasMismatch {
			return left.HasMismatch
		}
		if left.CSVOnly != right.CSVOnly {
			return left.CSVOnly
		}
		leftDiff := absInt64(left.InboundDiffMinor) + absInt64(left.OutboundDiffMinor)
		rightDiff := absInt64(right.InboundDiffMinor) + absInt64(right.OutboundDiffMinor)
		if leftDiff != rightDiff {
			return leftDiff > rightDiff
		}
		if left.Phone != right.Phone {
			return left.Phone < right.Phone
		}
		return left.RowKey < right.RowKey
	})

	return rows
}

func buildShiftReportRow(matchKey string, requisite shiftReportRequisiteRow, hasCRM bool, inbound shiftReportInboundAggregate, outboundByShiftRequisiteID map[int64]int64, bankNames map[string]string) ShiftReportRow {
	if !hasCRM {
		csvInboundMinor := inbound.CSVInboundMinor
		phone := inbound.CSVRequisite
		if phone == "" {
			phone = "Без реквизита"
		}
		return ShiftReportRow{
			RowKey:           "csv:" + matchKey,
			Phone:            phone,
			Status:           "csv_only",
			TLStatus:         "not_checked",
			CSVInboundMinor:  csvInboundMinor,
			InboundDiffMinor: -csvInboundMinor,
			HasMismatch:      csvInboundMinor != 0,
			CSVOnly:          true,
		}
	}

	csvInboundMinor := inbound.CSVInboundMinor
	csvOutboundMinor := outboundByShiftRequisiteID[requisite.ShiftRequisiteID]
	inboundDiffMinor := requisite.InboundTurnoverMinor - csvInboundMinor
	outboundDiffMinor := requisite.OutboundTurnoverMinor - csvOutboundMinor
	return ShiftReportRow{
		RowKey:                crmMatchKey(requisite.ShiftRequisiteID),
		ShiftRequisiteID:      &requisite.ShiftRequisiteID,
		RequisiteID:           &requisite.RequisiteID,
		Phone:                 requisite.Phone,
		MethodType:            requisite.MethodType,
		BankCode:              requisite.BankCode,
		BankName:              bankNames[requisite.BankCode],
		Proxy:                 requisite.Proxy,
		EmployeeComment:       requisite.EmployeeComment,
		CardNumber:            &requisite.CardNumber,
		HolderName:            &requisite.HolderName,
		Status:                requisite.Status,
		TLStatus:              defaultString(requisite.TLReconciliationStatus, "not_checked"),
		InboundTurnoverMinor:  requisite.InboundTurnoverMinor,
		OutboundTurnoverMinor: requisite.OutboundTurnoverMinor,
		ClosingBalanceMinor:   requisite.ClosingBalanceMinor,
		TargetTurnoverMinor:   requisite.TargetTurnoverMinor,
		CSVInboundMinor:       csvInboundMinor,
		CSVOutboundMinor:      csvOutboundMinor,
		InboundDiffMinor:      inboundDiffMinor,
		OutboundDiffMinor:     outboundDiffMinor,
		HasMismatch:           inboundDiffMinor != 0 || outboundDiffMinor != 0,
	}
}

func shiftReportRequisitesFromDB(rows []db.ListShiftReportRequisitesRow) []shiftReportRequisiteRow {
	items := make([]shiftReportRequisiteRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, shiftReportRequisiteRow{
			ShiftRequisiteID:       row.ShiftRequisiteID,
			RequisiteID:            row.RequisiteID,
			Phone:                  row.Phone,
			MethodType:             row.MethodType,
			BankCode:               row.BankCode,
			Proxy:                  textPtr(row.Proxy),
			EmployeeComment:        textPtr(row.EmployeeComment),
			CardNumber:             row.CardNumber,
			RequisiteCardNumber:    textPtr(row.RequisiteCardNumber),
			HolderName:             row.HolderName,
			Status:                 row.Status,
			TLReconciliationStatus: row.TlReconciliationStatus,
			InboundTurnoverMinor:   row.InboundTurnoverMinor,
			OutboundTurnoverMinor:  row.OutboundTurnoverMinor,
			ClosingBalanceMinor:    row.ClosingBalanceMinor,
			TargetTurnoverMinor:    row.TargetTurnoverMinor,
		})
	}
	return items
}

func shiftReportInboundItemsFromDB(rows []db.ListShiftReportInboundScopeItemsRow) []shiftReportInboundItem {
	items := make([]shiftReportInboundItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, shiftReportInboundItem{
			CSVRequisite:     row.CsvRequisite,
			NormalizedStatus: row.NormalizedStatus,
			AmountMinor:      row.AmountMinor,
		})
	}
	return items
}

func shiftReportOutboundTransfersFromDB(rows []db.ListShiftReportOutboundTransfersRow) []shiftReportOutboundTransfer {
	items := make([]shiftReportOutboundTransfer, 0, len(rows))
	for _, row := range rows {
		items = append(items, shiftReportOutboundTransfer{
			SourceShiftRequisiteID: row.SourceShiftRequisiteID,
			AmountMinor:            row.AmountMinor,
		})
	}
	return items
}

func bankNamesByCode(rows []db.Bank) map[string]string {
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.Code] = row.Name
	}
	return names
}

func matchingBankCodes(bankNames map[string]string, search string) []string {
	search = strings.TrimSpace(strings.ToLower(search))
	if search == "" {
		return nil
	}

	codes := make([]string, 0)
	for code, name := range bankNames {
		lowerCode := strings.ToLower(code)
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerCode, search) || strings.Contains(lowerName, search) {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return nil
	}
	sort.Strings(codes)
	return codes
}

func addReportLookupCandidate(candidates map[string]map[int64]string, lookupKey string, shiftRequisiteID int64, matchKey string) {
	if candidates[lookupKey] == nil {
		candidates[lookupKey] = map[int64]string{}
	}
	candidates[lookupKey][shiftRequisiteID] = matchKey
}

func inboundReportMatchKey(csvRequisite string, uniqueLookup map[string]string) string {
	csvDigits := digitsOnly(csvRequisite)
	if len(csvDigits) >= 10 && len(csvDigits) <= 11 {
		phoneLookup := "phone:" + rightString(csvDigits, 10)
		if matchKey, ok := uniqueLookup[phoneLookup]; ok {
			return matchKey
		}
		return "csv_phone:" + rightString(csvDigits, 10)
	}
	if len(csvDigits) >= 12 {
		cardLookup := "card:" + csvDigits
		if matchKey, ok := uniqueLookup[cardLookup]; ok {
			return matchKey
		}
		return "csv_card:" + csvDigits
	}
	return "csv:" + strings.ToLower(strings.TrimSpace(defaultString(csvRequisite, "unknown")))
}

func crmMatchKey(shiftRequisiteID int64) string {
	return "crm:" + strconv.FormatInt(shiftRequisiteID, 10)
}

func rightDigits(value string, count int) string {
	digits := digitsOnly(value)
	if digits == "" {
		return ""
	}
	return rightString(digits, count)
}

func rightString(value string, count int) string {
	if len(value) <= count {
		return value
	}
	return value[len(value)-count:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
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

func fromListInternalTransferRow(row db.ListInternalTransfersForShiftRequisiteRow, bankNames map[string]string) InternalTransfer {
	return InternalTransfer{
		ID:                          row.ID,
		TeamID:                      row.TeamID,
		ShiftID:                     row.ShiftID,
		TraderID:                    row.TraderID,
		SourceShiftRequisiteID:      row.SourceShiftRequisiteID,
		SourceRequisiteID:           row.SourceRequisiteID,
		SourcePhone:                 row.SourcePhone,
		SourceBankCode:              row.SourceBankCode,
		SourceBankName:              bankNames[row.SourceBankCode],
		DestinationShiftRequisiteID: row.DestinationShiftRequisiteID,
		DestinationRequisiteID:      row.DestinationRequisiteID,
		DestinationPhone:            row.DestinationPhone,
		DestinationBankCode:         row.DestinationBankCode,
		DestinationBankName:         bankNames[row.DestinationBankCode],
		AmountMinor:                 row.AmountMinor,
		Status:                      row.Status,
		CreatedBy:                   row.CreatedBy,
		CreatedAt:                   row.CreatedAt.Time,
		CancelledBy:                 int64Ptr(row.CancelledBy),
		CancelledAt:                 timePtr(row.CancelledAt),
		Comment:                     textPtr(row.Comment),
	}
}

func fromCreateInternalTransferRow(row db.CreateInternalTransferRow, bankNames map[string]string) InternalTransfer {
	return InternalTransfer{
		ID:                          row.ID,
		TeamID:                      row.TeamID,
		ShiftID:                     row.ShiftID,
		TraderID:                    row.TraderID,
		SourceShiftRequisiteID:      row.SourceShiftRequisiteID,
		SourceRequisiteID:           row.SourceRequisiteID,
		SourcePhone:                 row.SourcePhone,
		SourceBankCode:              row.SourceBankCode,
		SourceBankName:              bankNames[row.SourceBankCode],
		DestinationShiftRequisiteID: row.DestinationShiftRequisiteID,
		DestinationRequisiteID:      row.DestinationRequisiteID,
		DestinationPhone:            row.DestinationPhone,
		DestinationBankCode:         row.DestinationBankCode,
		DestinationBankName:         bankNames[row.DestinationBankCode],
		AmountMinor:                 row.AmountMinor,
		Status:                      row.Status,
		CreatedBy:                   row.CreatedBy,
		CreatedAt:                   row.CreatedAt.Time,
		CancelledBy:                 int64Ptr(row.CancelledBy),
		CancelledAt:                 timePtr(row.CancelledAt),
		Comment:                     textPtr(row.Comment),
	}
}

func fromCancelInternalTransferRow(row db.CancelInternalTransferRow, bankNames map[string]string) InternalTransfer {
	return InternalTransfer{
		ID:                          row.ID,
		TeamID:                      row.TeamID,
		ShiftID:                     row.ShiftID,
		TraderID:                    row.TraderID,
		SourceShiftRequisiteID:      row.SourceShiftRequisiteID,
		SourceRequisiteID:           row.SourceRequisiteID,
		SourcePhone:                 row.SourcePhone,
		SourceBankCode:              row.SourceBankCode,
		SourceBankName:              bankNames[row.SourceBankCode],
		DestinationShiftRequisiteID: row.DestinationShiftRequisiteID,
		DestinationRequisiteID:      row.DestinationRequisiteID,
		DestinationPhone:            row.DestinationPhone,
		DestinationBankCode:         row.DestinationBankCode,
		DestinationBankName:         bankNames[row.DestinationBankCode],
		AmountMinor:                 row.AmountMinor,
		Status:                      row.Status,
		CreatedBy:                   row.CreatedBy,
		CreatedAt:                   row.CreatedAt.Time,
		CancelledBy:                 int64Ptr(row.CancelledBy),
		CancelledAt:                 timePtr(row.CancelledAt),
		Comment:                     textPtr(row.Comment),
	}
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
		OpenRequisiteCount:  row.OpenRequisiteCount,
		AllRequisitesClosed: row.OpenRequisiteCount == 0,
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

func timePtrValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func optionalInt8(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: *value, Valid: true}
}

func digitsOnly(value string) string {
	var digits []rune
	for _, char := range value {
		if unicode.IsDigit(char) {
			digits = append(digits, char)
		}
	}

	return string(digits)
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
