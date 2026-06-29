package reconciliation

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
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
	requisiteMismatchCount, err := queries.CountTraderInboundRequisiteMismatches(ctx, db.CountTraderInboundRequisiteMismatchesParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	status := StatusMismatch
	if diff == 0 && requisiteMismatchCount == 0 {
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

	if diff != 0 {
		if err := createTotalMismatchItem(ctx, tx, run.ID, expected.ExpectedAmountMinor, actualAmount, diff, "Trader invoice CSV total differs from closed requisite turnover"); err != nil {
			return Run{}, err
		}
	}

	if _, err := queries.CreateTraderInboundRequisiteMismatchItems(ctx, db.CreateTraderInboundRequisiteMismatchItemsParams{
		RunID:    run.ID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	}); err != nil {
		return Run{}, err
	}

	if err := queries.UpdateTraderInboundRequisiteReviewStatuses(ctx, db.UpdateTraderInboundRequisiteReviewStatusesParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	}); err != nil {
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

	run, err := queries.CreateTraderOutboundReconciliationRun(ctx, db.CreateTraderOutboundReconciliationRunParams{
		TeamID:              record.TeamID,
		ShiftID:             int8Value(&record.ShiftID),
		TraderID:            int8Value(&record.TraderID),
		ImportBatchID:       int8Value(record.ImportBatchID),
		ExpectedAmountMinor: expected.ExpectedAmountMinor,
		ActualAmountMinor:   actualAmount,
		DiffAmountMinor:     diff,
		SuccessCount:        expected.SuccessCount,
		FailedAmountMinor:   expected.FailedAmountMinor,
		FailedCount:         expected.FailedCount,
		TotalAmountMinor:    expected.TotalAmountMinor,
		TotalCount:          expected.TotalCount,
		Status:              StatusMismatch,
	})
	if err != nil {
		return Run{}, err
	}

	if diff != 0 {
		if err := createTotalMismatchItem(ctx, tx, run.ID, expected.ExpectedAmountMinor, actualAmount, diff, "Trader payout CSV total differs from manual payout transfers"); err != nil {
			return Run{}, err
		}
	}

	itemIDs, err := queries.CreateTraderOutboundReconciliationItems(ctx, db.CreateTraderOutboundReconciliationItemsParams{
		RunID:    run.ID,
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	status := StatusMismatch
	if diff == 0 && len(itemIDs) == 0 {
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

func (r *Repository) RecalculateTeamleadPeriodOutbound(ctx context.Context, record RecalculateTeamleadPeriodOutboundRecord) (Run, error) {
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
	summary, err := queries.CalculateTeamleadPeriodOutboundSummary(ctx, db.CalculateTeamleadPeriodOutboundSummaryParams{
		TeamID:             record.TeamID,
		AccountingPeriodID: record.AccountingPeriodID,
	})
	if err != nil {
		return Run{}, err
	}

	diff := summary.ActualAmountMinor - summary.ExpectedAmountMinor
	run, err := queries.CreateTeamleadPeriodOutboundReconciliationRun(ctx, db.CreateTeamleadPeriodOutboundReconciliationRunParams{
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

	itemIDs, err := queries.CreateTeamleadPeriodOutboundReconciliationItems(ctx, db.CreateTeamleadPeriodOutboundReconciliationItemsParams{
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

func (r *Repository) RecalculateTeamleadCurrent(ctx context.Context, record RecalculateTeamleadCurrentRecord) (Run, error) {
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
	latestImport, err := queries.LatestActiveTeamleadCurrentImportBatch(ctx, db.LatestActiveTeamleadCurrentImportBatchParams{
		TeamID:     record.TeamID,
		Direction:  record.Direction,
		UploadedBy: record.ActorID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}

	summary, err := queries.CalculateTeamleadCurrentSummary(ctx, db.CalculateTeamleadCurrentSummaryParams{
		TeamID:     record.TeamID,
		Direction:  record.Direction,
		UploadedBy: record.ActorID,
	})
	if err != nil {
		return Run{}, err
	}

	diff := summary.ActualAmountMinor - summary.ExpectedAmountMinor
	runType := TypeTeamleadPeriodInbound
	if record.Direction == "outbound" {
		runType = TypeTeamleadPeriodOutbound
	}

	run, err := queries.CreateTeamleadCurrentReconciliationRun(ctx, db.CreateTeamleadCurrentReconciliationRunParams{
		TeamID:              record.TeamID,
		Type:                runType,
		ImportBatchID:       int8Value(&latestImport.ID),
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

	itemIDs, err := queries.CreateTeamleadCurrentReconciliationItems(ctx, db.CreateTeamleadCurrentReconciliationItemsParams{
		RunID:         run.ID,
		TeamID:        record.TeamID,
		Direction:     record.Direction,
		ImportBatchID: latestImport.ID,
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

func (r *Repository) LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error) {
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

	run, err := db.New(tx).LatestTraderInboundReconciliationRunByShift(ctx, db.LatestTraderInboundReconciliationRunByShiftParams{
		TeamID:  teamID,
		ShiftID: int8Value(&shiftID),
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

func (r *Repository) LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error) {
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

	run, err := db.New(tx).LatestTraderOutboundReconciliationRunByShift(ctx, db.LatestTraderOutboundReconciliationRunByShiftParams{
		TeamID:  teamID,
		ShiftID: int8Value(&shiftID),
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

func (r *Repository) GetTraderInboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error) {
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

	run, err := db.New(tx).GetTraderInboundReconciliationRun(ctx, db.GetTraderInboundReconciliationRunParams{
		RunID:    runID,
		TeamID:   teamID,
		TraderID: int8Value(&traderID),
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

func (r *Repository) GetTraderOutboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error) {
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

	run, err := db.New(tx).GetTraderOutboundReconciliationRun(ctx, db.GetTraderOutboundReconciliationRunParams{
		RunID:    runID,
		TeamID:   teamID,
		TraderID: int8Value(&traderID),
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

func (r *Repository) LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
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

	run, err := db.New(tx).LatestTeamleadPeriodOutboundReconciliationRun(ctx, db.LatestTeamleadPeriodOutboundReconciliationRunParams{
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

func (r *Repository) LatestTeamleadInbound(ctx context.Context, teamID int64, actorID int64) (Run, error) {
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

	run, err := db.New(tx).LatestTeamleadInboundReconciliationRun(ctx, db.LatestTeamleadInboundReconciliationRunParams{
		TeamID:     teamID,
		UploadedBy: actorID,
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

func (r *Repository) LatestTeamleadOutbound(ctx context.Context, teamID int64, actorID int64) (Run, error) {
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

	run, err := db.New(tx).LatestTeamleadOutboundReconciliationRun(ctx, db.LatestTeamleadOutboundReconciliationRunParams{
		TeamID:     teamID,
		UploadedBy: actorID,
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

func (r *Repository) ListTeamleadCurrentRuns(ctx context.Context, teamID int64, actorID int64, direction string, page pagination.Params) (pagination.Result[Run], error) {
	if r.db == nil {
		return pagination.Result[Run]{}, ErrRepositoryNotConfigured
	}
	page = pagination.Normalize(page)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return pagination.Result[Run]{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	rows, err := db.New(tx).ListTeamleadCurrentReconciliationRuns(ctx, db.ListTeamleadCurrentReconciliationRunsParams{
		TeamID:      teamID,
		Direction:   direction,
		UploadedBy:  actorID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[Run]{}, err
	}

	runs := make([]Run, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, fromDBRun(row))
	}

	total, err := db.New(tx).CountTeamleadCurrentReconciliationRuns(ctx, db.CountTeamleadCurrentReconciliationRunsParams{
		TeamID:     teamID,
		Direction:  direction,
		UploadedBy: actorID,
	})
	if err != nil {
		return pagination.Result[Run]{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return pagination.Result[Run]{}, err
	}
	committed = true

	return pagination.NewResult(runs, page, total), nil
}

func (r *Repository) GetTeamleadCurrentRun(ctx context.Context, teamID int64, actorID int64, direction string, runID int64) (Run, error) {
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

	run, err := db.New(tx).GetTeamleadCurrentReconciliationRun(ctx, db.GetTeamleadCurrentReconciliationRunParams{
		RunID:      runID,
		TeamID:     teamID,
		Direction:  direction,
		UploadedBy: actorID,
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

func (r *Repository) ListItems(ctx context.Context, runID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	if r.db == nil {
		return pagination.Result[Item]{}, ErrRepositoryNotConfigured
	}
	page = pagination.Normalize(page)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return pagination.Result[Item]{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	rows, err := db.New(tx).ListReconciliationItemsForRun(ctx, db.ListReconciliationItemsForRunParams{
		RunID:        runID,
		OnlyMismatch: filters.OnlyMismatch,
		Status:       filters.Status,
		OffsetCount:  paginationOffset32(page),
		LimitCount:   paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBItem(row))
	}

	total, err := db.New(tx).CountReconciliationItemsForRun(ctx, db.CountReconciliationItemsForRunParams{
		RunID:        runID,
		OnlyMismatch: filters.OnlyMismatch,
		Status:       filters.Status,
	})
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return pagination.Result[Item]{}, err
	}
	committed = true

	return pagination.NewResult(items, page, total), nil
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

func (r *Repository) ListActiveTeamleadOutboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadOutboundPeriodScope, error) {
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

	rows, err := db.New(tx).ListActiveTeamleadOutboundPeriodScopes(ctx, teamID)
	if err != nil {
		return nil, err
	}

	scopes := make([]TeamleadOutboundPeriodScope, 0, len(rows))
	for _, row := range rows {
		if !row.AccountingPeriodID.Valid {
			continue
		}
		scopes = append(scopes, TeamleadOutboundPeriodScope{
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

func createTotalMismatchItem(ctx context.Context, tx pgx.Tx, runID int64, csvAmountMinor int64, crmAmountMinor int64, diffAmountMinor int64, message string) error {
	csvValue, err := json.Marshal(map[string]int64{
		"amountMinor": csvAmountMinor,
	})
	if err != nil {
		return err
	}
	crmValue, err := json.Marshal(map[string]int64{
		"amountMinor":     crmAmountMinor,
		"diffAmountMinor": diffAmountMinor,
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO reconciliation_items (
			reconciliation_run_id,
			issue_type,
			teamlead_value_json,
			trader_value_json,
			message
		)
		VALUES ($1, 'total_mismatch', $2::jsonb, $3::jsonb, $4)
	`, runID, csvValue, crmValue, message)

	return err
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

func (r *Repository) AcceptTeamleadCurrent(ctx context.Context, record AcceptTeamleadCurrentRecord) (Run, error) {
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

	run, err := db.New(tx).AcceptTeamleadCurrentReconciliationRun(ctx, db.AcceptTeamleadCurrentReconciliationRunParams{
		RunID:       record.RunID,
		TeamID:      record.TeamID,
		ActorID:     record.ActorID,
		ConfirmedBy: int8Value(&record.ActorID),
		Comment:     textValue(&record.Comment),
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
