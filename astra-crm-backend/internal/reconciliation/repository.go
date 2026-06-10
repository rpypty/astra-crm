package reconciliation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var (
	ErrRepositoryNotConfigured = errors.New("reconciliation repository is not configured")
	ErrRunNotFound             = errors.New("reconciliation run not found")
)

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	db txBeginner
}

func NewRepository(db txBeginner) *Repository {
	return &Repository{db: db}
}

func (r *Repository) RecalculateTraderInbound(ctx context.Context, record RecalculateTraderInboundRecord) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	expected, err := queries.CalculateTraderInboundExpected(ctx, db.CalculateTraderInboundExpectedParams{
		TeamID:  record.TeamID,
		ShiftID: int8Value(&record.ShiftID),
	})
	if err != nil {
		return Run{}, err
	}

	actualAmount, err := queries.CalculateTraderInboundActual(ctx, db.CalculateTraderInboundActualParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	diff := actualAmount - expected.ExpectedAmountMinor
	status := StatusMismatch
	if diff == 0 {
		status = StatusMatched
	}

	run, err := queries.CreateTraderInboundReconciliationRun(ctx, db.CreateTraderInboundReconciliationRunParams{
		TeamID:              record.TeamID,
		ShiftID:             int8Value(&record.ShiftID),
		TraderID:            int8Value(&record.TraderID),
		ImportBatchID:       int8Value(record.ImportBatchID),
		ExpectedAmountMinor: expected.ExpectedAmountMinor,
		ActualAmountMinor:   actualAmount,
		DiffAmountMinor:     diff,
		SuccessAmountMinor:  expected.ExpectedAmountMinor,
		SuccessCount:        expected.SuccessCount,
		FailedAmountMinor:   expected.FailedAmountMinor,
		FailedCount:         expected.FailedCount,
		TotalAmountMinor:    expected.TotalAmountMinor,
		TotalCount:          expected.TotalCount,
		Status:              status,
	})
	if err != nil {
		return Run{}, err
	}

	if err := queries.UpdateTraderShiftInboundReconciliationStatus(ctx, db.UpdateTraderShiftInboundReconciliationStatusParams{
		Status:   status,
		ShiftID:  record.ShiftID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
	}); err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) RecalculateTraderOutbound(ctx context.Context, record RecalculateTraderOutboundRecord) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	expected, err := queries.CalculateTraderOutboundExpected(ctx, db.CalculateTraderOutboundExpectedParams{
		TeamID:  record.TeamID,
		ShiftID: int8Value(&record.ShiftID),
	})
	if err != nil {
		return Run{}, err
	}

	actualAmount, err := queries.CalculateTraderOutboundActual(ctx, db.CalculateTraderOutboundActualParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	diff := actualAmount - expected.ExpectedAmountMinor
	status := StatusMismatch
	if diff == 0 {
		status = StatusMatched
	}

	run, err := queries.CreateTraderOutboundReconciliationRun(ctx, db.CreateTraderOutboundReconciliationRunParams{
		TeamID:              record.TeamID,
		ShiftID:             int8Value(&record.ShiftID),
		TraderID:            int8Value(&record.TraderID),
		ImportBatchID:       int8Value(record.ImportBatchID),
		ExpectedAmountMinor: expected.ExpectedAmountMinor,
		ActualAmountMinor:   actualAmount,
		DiffAmountMinor:     diff,
		OrderCount:          expected.OrderCount,
		Status:              status,
	})
	if err != nil {
		return Run{}, err
	}

	if err := queries.CreateTraderOutboundUnpaidPayoutItems(ctx, db.CreateTraderOutboundUnpaidPayoutItemsParams{
		RunID:    run.ID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	}); err != nil {
		return Run{}, err
	}

	if err := queries.UpdateTraderShiftOutboundReconciliationStatus(ctx, db.UpdateTraderShiftOutboundReconciliationStatusParams{
		Status:   status,
		ShiftID:  record.ShiftID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
	}); err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) RecalculateTeamleadPeriodInbound(ctx context.Context, record RecalculateTeamleadPeriodInboundRecord) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	summary, err := queries.CalculateTeamleadPeriodInboundSummary(ctx, db.CalculateTeamleadPeriodInboundSummaryParams{
		TeamID:             record.TeamID,
		AccountingPeriodID: record.AccountingPeriodID,
	})
	if err != nil {
		return Run{}, err
	}

	diff := summary.ActualAmountMinor - summary.ExpectedAmountMinor
	run, err := queries.CreateTeamleadPeriodInboundReconciliationRun(ctx, db.CreateTeamleadPeriodInboundReconciliationRunParams{
		TeamID:              record.TeamID,
		AccountingPeriodID:  int8Value(&record.AccountingPeriodID),
		ImportBatchID:       int8Value(record.ImportBatchID),
		ExpectedAmountMinor: summary.ExpectedAmountMinor,
		ActualAmountMinor:   summary.ActualAmountMinor,
		DiffAmountMinor:     diff,
		SuccessAmountMinor:  summary.ExpectedAmountMinor,
		SuccessCount:        summary.ExpectedSuccessCount,
		FailedAmountMinor:   summary.FailedAmountMinor,
		FailedCount:         summary.FailedCount,
		TotalAmountMinor:    summary.TotalAmountMinor,
		TotalCount:          summary.TotalCount,
		Status:              StatusMismatch,
	})
	if err != nil {
		return Run{}, err
	}

	itemIDs, err := queries.CreateTeamleadPeriodInboundReconciliationItems(ctx, db.CreateTeamleadPeriodInboundReconciliationItemsParams{
		RunID:              run.ID,
		TeamID:             record.TeamID,
		AccountingPeriodID: record.AccountingPeriodID,
	})
	if err != nil {
		return Run{}, err
	}

	status := StatusMismatch
	if diff == 0 && summary.ExpectedSuccessCount == summary.ActualSuccessCount && len(itemIDs) == 0 {
		status = StatusMatched
	}

	run, err = queries.UpdateReconciliationRunStatus(ctx, db.UpdateReconciliationRunStatusParams{
		Status: status,
		RunID:  run.ID,
		TeamID: record.TeamID,
	})
	if err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	run, err := db.New(tx).LatestTraderInboundReconciliationRun(ctx, db.LatestTraderInboundReconciliationRunParams{
		TeamID:   teamID,
		TraderID: int8Value(&traderID),
		ShiftID:  int8Value(&shiftID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	run, err := db.New(tx).LatestTraderOutboundReconciliationRun(ctx, db.LatestTraderOutboundReconciliationRunParams{
		TeamID:   teamID,
		TraderID: int8Value(&traderID),
		ShiftID:  int8Value(&shiftID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	run, err := db.New(tx).LatestTeamleadPeriodInboundReconciliationRun(ctx, db.LatestTeamleadPeriodInboundReconciliationRunParams{
		TeamID:             teamID,
		AccountingPeriodID: int8Value(&accountingPeriodID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) LatestTeamleadInbound(ctx context.Context, teamID int64) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	run, err := db.New(tx).LatestTeamleadInboundReconciliationRun(ctx, teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) ListItems(ctx context.Context, runID int64) ([]Item, error) {
	if r.db == nil {
		return nil, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	rows, err := db.New(tx).ListReconciliationItemsForRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBItem(row))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	committed = true

	return items, nil
}

func (r *Repository) ListActiveTeamleadInboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadInboundPeriodScope, error) {
	if r.db == nil {
		return nil, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	rows, err := db.New(tx).ListActiveTeamleadInboundPeriodScopes(ctx, teamID)
	if err != nil {
		return nil, err
	}

	scopes := make([]TeamleadInboundPeriodScope, 0, len(rows))
	for _, row := range rows {
		if !row.AccountingPeriodID.Valid {
			continue
		}
		scopes = append(scopes, TeamleadInboundPeriodScope{
			AccountingPeriodID: row.AccountingPeriodID.Int64,
			ImportBatchID:      row.ImportBatchID,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	committed = true

	return scopes, nil
}

func (r *Repository) AcceptTraderInbound(ctx context.Context, record AcceptTraderInboundRecord) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	run, err := queries.AcceptTraderInboundReconciliationRun(ctx, db.AcceptTraderInboundReconciliationRunParams{
		RunID:       record.RunID,
		TeamID:      record.TeamID,
		TraderID:    int8Value(&record.TraderID),
		ConfirmedBy: int8Value(&record.ActorID),
		Comment:     textValue(&record.Comment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	shiftID := run.ShiftID.Int64
	if err := queries.UpdateTraderShiftInboundReconciliationStatus(ctx, db.UpdateTraderShiftInboundReconciliationStatusParams{
		Status:   StatusAcceptedWithComment,
		ShiftID:  shiftID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
	}); err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func (r *Repository) AcceptTraderOutbound(ctx context.Context, record AcceptTraderOutboundRecord) (Run, error) {
	if r.db == nil {
		return Run{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	run, err := queries.AcceptTraderOutboundReconciliationRun(ctx, db.AcceptTraderOutboundReconciliationRunParams{
		RunID:       record.RunID,
		TeamID:      record.TeamID,
		TraderID:    int8Value(&record.TraderID),
		ConfirmedBy: int8Value(&record.ActorID),
		Comment:     textValue(&record.Comment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	shiftID := run.ShiftID.Int64
	if err := queries.UpdateTraderShiftOutboundReconciliationStatus(ctx, db.UpdateTraderShiftOutboundReconciliationStatusParams{
		Status:   StatusAcceptedWithComment,
		ShiftID:  shiftID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
	}); err != nil {
		return Run{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	committed = true

	return fromDBRun(run), nil
}

func fromDBRun(row db.ReconciliationRun) Run {
	return Run{
		ID:                  row.ID,
		TeamID:              row.TeamID,
		Type:                row.Type,
		ScopeType:           row.ScopeType,
		ShiftID:             int8Ptr(row.ShiftID),
		AccountingPeriodID:  int8Ptr(row.AccountingPeriodID),
		TraderID:            int8Ptr(row.TraderID),
		ImportBatchID:       int8Ptr(row.ImportBatchID),
		ExpectedAmountMinor: row.ExpectedAmountMinor,
		ActualAmountMinor:   row.ActualAmountMinor,
		DiffAmountMinor:     row.DiffAmountMinor,
		SuccessAmountMinor:  row.SuccessAmountMinor,
		SuccessCount:        row.SuccessCount,
		FailedAmountMinor:   row.FailedAmountMinor,
		FailedCount:         row.FailedCount,
		TotalAmountMinor:    row.TotalAmountMinor,
		TotalCount:          row.TotalCount,
		Status:              row.Status,
		Comment:             textPtr(row.Comment),
		ConfirmedBy:         int8Ptr(row.ConfirmedBy),
		ConfirmedAt:         timePtr(row.ConfirmedAt),
		CreatedAt:           row.CreatedAt.Time,
	}
}

func fromDBItem(row db.ReconciliationItem) Item {
	return Item{
		ID:                  row.ID,
		ReconciliationRunID: row.ReconciliationRunID,
		IssueType:           row.IssueType,
		ExternalOrderID:     int8Ptr(row.ExternalOrderID),
		ExternalInnerID:     textPtr(row.ExternalInnerID),
		TeamleadValueJSON:   row.TeamleadValueJson,
		TraderValueJSON:     row.TraderValueJson,
		Message:             textPtr(row.Message),
		CreatedAt:           row.CreatedAt.Time,
	}
}

func int8Value(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: *value, Valid: true}
}

func int8Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	return &value.Int64
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

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
