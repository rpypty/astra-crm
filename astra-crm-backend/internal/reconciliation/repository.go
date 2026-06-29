package reconciliation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/imports"
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
	db.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	db txBeginner
}

func NewRepository(db txBeginner) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error) {
	if r.db == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}
	row, err := db.New(r.db).GetTeamleadReconciliation(ctx, db.GetTeamleadReconciliationParams{
		TeamID: teamID,
		ID:     runID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamleadRun{}, ErrRunNotFound
		}
		return TeamleadRun{}, err
	}
	return fromDBTeamleadRun(row), nil
}

func (r *Repository) ListTeamleadReconciliations(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[TeamleadRun], error) {
	if r.db == nil {
		return pagination.Result[TeamleadRun]{}, ErrRepositoryNotConfigured
	}
	page = pagination.Normalize(page)
	queries := db.New(r.db)
	rows, err := queries.ListTeamleadReconciliations(ctx, db.ListTeamleadReconciliationsParams{
		TeamID:      teamID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[TeamleadRun]{}, err
	}
	items := make([]TeamleadRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBTeamleadRun(row))
	}
	total, err := queries.CountTeamleadReconciliations(ctx, teamID)
	if err != nil {
		return pagination.Result[TeamleadRun]{}, err
	}
	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) ListTeamleadReconciliationItems(ctx context.Context, teamID int64, runID int64, filters TeamleadItemFilters, page pagination.Params) (pagination.Result[TeamleadItem], error) {
	if r.db == nil {
		return pagination.Result[TeamleadItem]{}, ErrRepositoryNotConfigured
	}
	page = pagination.Normalize(page)
	params := db.ListTeamleadReconciliationItemsParams{
		TeamID:                   teamID,
		TeamleadReconciliationID: runID,
		Direction:                optionalText(filters.Direction),
		Stage:                    optionalText(filters.Stage),
		IssueType:                optionalText(filters.IssueType),
		Severity:                 optionalText(filters.Severity),
		TraderID:                 int8Value(filters.TraderID),
		RequisiteID:              int8Value(filters.RequisiteID),
		OnlyMismatch:             filters.OnlyMismatch,
		OffsetCount:              paginationOffset32(page),
		LimitCount:               paginationLimit32(page),
	}
	queries := db.New(r.db)
	rows, err := queries.ListTeamleadReconciliationItems(ctx, params)
	if err != nil {
		return pagination.Result[TeamleadItem]{}, err
	}
	items := make([]TeamleadItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBTeamleadItem(row))
	}
	total, err := queries.CountTeamleadReconciliationItems(ctx, db.CountTeamleadReconciliationItemsParams{
		TeamID:                   params.TeamID,
		TeamleadReconciliationID: params.TeamleadReconciliationID,
		Direction:                params.Direction,
		Stage:                    params.Stage,
		IssueType:                params.IssueType,
		Severity:                 params.Severity,
		TraderID:                 params.TraderID,
		RequisiteID:              params.RequisiteID,
		OnlyMismatch:             params.OnlyMismatch,
	})
	if err != nil {
		return pagination.Result[TeamleadItem]{}, err
	}
	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) QueueTeamleadReconciliationApply(ctx context.Context, record QueueTeamleadApplyRecord) (TeamleadRun, error) {
	if r.db == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return TeamleadRun{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	runRow, err := queries.QueueTeamleadReconciliationApply(ctx, db.QueueTeamleadReconciliationApplyParams{
		ActorID: int8Value(&record.ActorID),
		Comment: textValue(record.Comment),
		TeamID:  record.TeamID,
		ID:      record.RunID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamleadRun{}, ErrInvalidState
		}
		return TeamleadRun{}, err
	}
	if err := queries.UpsertTeamleadReconciliationApplyJobQueued(ctx, db.UpsertTeamleadReconciliationApplyJobQueuedParams{
		TeamleadReconciliationID: record.RunID,
		TeamID:                   record.TeamID,
	}); err != nil {
		return TeamleadRun{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TeamleadRun{}, err
	}
	committed = true

	return fromDBTeamleadRun(runRow), nil
}

func (r *Repository) RejectTeamleadReconciliation(ctx context.Context, record RejectTeamleadReconciliationRecord) (TeamleadRun, error) {
	if r.db == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}
	runRow, err := db.New(r.db).RejectTeamleadReconciliation(ctx, db.RejectTeamleadReconciliationParams{
		ActorID: int8Value(&record.ActorID),
		Comment: textValue(&record.Comment),
		TeamID:  record.TeamID,
		ID:      record.RunID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamleadRun{}, ErrInvalidState
		}
		return TeamleadRun{}, err
	}
	return fromDBTeamleadRun(runRow), nil
}

func (r *Repository) ApplyQueuedTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error) {
	if r.db == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}
	queries := db.New(r.db)
	runRow, err := queries.MarkTeamleadReconciliationApplying(ctx, db.MarkTeamleadReconciliationApplyingParams{
		TeamID: teamID,
		ID:     runID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamleadRun{}, ErrInvalidState
		}
		return TeamleadRun{}, err
	}
	if err := queries.MarkTeamleadReconciliationApplyJobRunning(ctx, db.MarkTeamleadReconciliationApplyJobRunningParams{
		TeamleadReconciliationID: runID,
		TeamID:                   teamID,
	}); err != nil {
		return TeamleadRun{}, err
	}

	run := fromDBTeamleadRun(runRow)
	result := TeamleadApplyResult{RunID: runID}
	var applyErr error
	for _, direction := range teamleadApplyDirections(run) {
		directionResult, err := r.applyTeamleadDirection(ctx, run, direction.direction, direction.batchID)
		result.Directions = append(result.Directions, directionResult)
		result.CreatedOrders += directionResult.CreatedOrders
		result.UpdatedOrders += directionResult.UpdatedOrders
		result.ConfirmedOrders += directionResult.ConfirmedOrders
		result.DiscrepancyOrders += directionResult.DiscrepancyOrders
		if err != nil {
			applyErr = err
			break
		}
	}

	resultPayload, err := marshalJSON(result)
	if err != nil {
		return TeamleadRun{}, err
	}
	if applyErr != nil {
		errorMessage := applyErr.Error()
		failedRow, markErr := queries.MarkTeamleadReconciliationApplyFailed(ctx, db.MarkTeamleadReconciliationApplyFailedParams{
			ApplyResultJson: resultPayload,
			ErrorMessage:    textValue(&errorMessage),
			TeamID:          teamID,
			ID:              runID,
		})
		if markErr != nil {
			return TeamleadRun{}, markErr
		}
		if err := queries.MarkTeamleadReconciliationApplyJobFailed(ctx, db.MarkTeamleadReconciliationApplyJobFailedParams{
			ResultJson:               resultPayload,
			ErrorMessage:             textValue(&errorMessage),
			TeamleadReconciliationID: runID,
			TeamID:                   teamID,
		}); err != nil {
			return TeamleadRun{}, err
		}
		return fromDBTeamleadRun(failedRow), nil
	}

	appliedRow, err := queries.MarkTeamleadReconciliationApplied(ctx, db.MarkTeamleadReconciliationAppliedParams{
		ApplyResultJson: resultPayload,
		TeamID:          teamID,
		ID:              runID,
	})
	if err != nil {
		return TeamleadRun{}, err
	}
	if err := queries.MarkTeamleadReconciliationApplyJobSucceeded(ctx, db.MarkTeamleadReconciliationApplyJobSucceededParams{
		ResultJson:               resultPayload,
		TeamleadReconciliationID: runID,
		TeamID:                   teamID,
	}); err != nil {
		return TeamleadRun{}, err
	}
	return fromDBTeamleadRun(appliedRow), nil
}

func (r *Repository) CreateTeamleadAnalysis(ctx context.Context, record CreateTeamleadAnalysisRecord) (TeamleadRun, error) {
	if r.db == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return TeamleadRun{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	var inboundBatchID *int64
	var outboundBatchID *int64
	for _, direction := range record.Directions {
		batch, err := queries.CreateImportBatch(ctx, db.CreateImportBatchParams{
			TeamID:             record.TeamID,
			UploadedBy:         record.ActorID,
			ScopeType:          imports.ScopeTypeTeamleadPeriod,
			Direction:          direction.Direction,
			ShiftID:            pgtype.Int8{},
			AccountingPeriodID: pgtype.Int8{},
			TraderID:           pgtype.Int8{},
			FileName:           direction.FileName,
			FileHash:           direction.FileHash,
			RowsCount:          int64(len(direction.Rows)),
		})
		if err != nil {
			return TeamleadRun{}, err
		}
		for _, row := range direction.Rows {
			if _, err := insertTeamleadImportRow(ctx, queries, batch.ID, row); err != nil {
				return TeamleadRun{}, err
			}
		}
		batchID := batch.ID
		if direction.Direction == imports.DirectionInbound {
			inboundBatchID = &batchID
		} else {
			outboundBatchID = &batchID
		}
	}

	runRow, err := queries.CreateTeamleadReconciliation(ctx, db.CreateTeamleadReconciliationParams{
		TeamID:                record.TeamID,
		DateFrom:              dateValue(record.DateFrom),
		DateTo:                dateValue(record.DateTo),
		Status:                record.Status,
		CreatedBy:             record.ActorID,
		InboundImportBatchID:  int8Value(inboundBatchID),
		OutboundImportBatchID: int8Value(outboundBatchID),
		PipelineJson:          record.PipelineJSON,
		InboundSummaryJson:    record.InboundSummary,
		OutboundSummaryJson:   record.OutboundSummary,
		PreviewJson:           record.PreviewJSON,
	})
	if err != nil {
		return TeamleadRun{}, err
	}

	runRow, err = queries.UpdateTeamleadReconciliationAnalysisResult(ctx, db.UpdateTeamleadReconciliationAnalysisResultParams{
		ID:            runRow.ID,
		TeamID:        record.TeamID,
		Status:        record.Status,
		MismatchCount: record.MismatchCount,
		ConflictCount: record.ConflictCount,
		BlockedCount:  record.BlockedCount,
	})
	if err != nil {
		return TeamleadRun{}, err
	}

	for _, item := range record.Items {
		if _, err := queries.CreateTeamleadReconciliationItem(ctx, db.CreateTeamleadReconciliationItemParams{
			TeamleadReconciliationID: runRow.ID,
			TeamID:                   record.TeamID,
			Direction:                item.Direction,
			Stage:                    item.Stage,
			IssueType:                item.IssueType,
			Severity:                 item.Severity,
			ExternalOrderID:          int8Value(item.ExternalOrderID),
			ExternalInnerID:          textValue(item.ExternalInnerID),
			TraderID:                 int8Value(item.TraderID),
			RequisiteID:              int8Value(item.RequisiteID),
			ShiftID:                  int8Value(item.ShiftID),
			BeforeJson:               jsonbValue(item.BeforeJSON),
			AfterJson:                jsonbValue(item.AfterJSON),
			Message:                  textValue(item.Message),
			IsBlocking:               item.IsBlocking,
		}); err != nil {
			return TeamleadRun{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return TeamleadRun{}, err
	}
	committed = true

	return fromDBTeamleadRun(runRow), nil
}

type teamleadApplyDirection struct {
	direction string
	batchID   int64
}

func teamleadApplyDirections(run TeamleadRun) []teamleadApplyDirection {
	result := make([]teamleadApplyDirection, 0, 2)
	if run.InboundImportBatchID != nil {
		result = append(result, teamleadApplyDirection{direction: imports.DirectionInbound, batchID: *run.InboundImportBatchID})
	}
	if run.OutboundImportBatchID != nil {
		result = append(result, teamleadApplyDirection{direction: imports.DirectionOutbound, batchID: *run.OutboundImportBatchID})
	}
	return result
}

func (r *Repository) applyTeamleadDirection(ctx context.Context, run TeamleadRun, direction string, batchID int64) (TeamleadDirectionApplyResult, error) {
	result := TeamleadDirectionApplyResult{Direction: direction}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	lockKey := fmt.Sprintf("teamlead-reconciliation:%d:%s:%s:%s", run.TeamID, run.DateFrom.Format("2006-01-02"), run.DateTo.Format("2006-01-02"), direction)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", lockKey); err != nil {
		return result, err
	}

	queries := db.New(tx)
	traderRows, err := queries.ListTeamleadReconciliationTraders(ctx, run.TeamID)
	if err != nil {
		return result, err
	}
	requisiteRows, err := queries.ListTeamleadReconciliationRequisites(ctx, run.TeamID)
	if err != nil {
		return result, err
	}
	lookups := newTeamleadAnalysisLookups(fromDBTeamleadTraderMatches(traderRows), fromDBTeamleadRequisiteMatches(requisiteRows))

	rows, err := loadTeamleadImportRows(ctx, tx, batchID)
	if err != nil {
		return result, err
	}

	confirmedOrderIDs := make([]int64, 0)
	updatedOrderIDs := make([]int64, 0)
	seenInnerIDs := make([]string, 0, len(rows))
	seenInnerIDSet := map[string]struct{}{}
	for _, importRow := range rows {
		parsedRow, err := parsedOrderRowFromImportRow(importRow)
		if err != nil {
			return result, err
		}
		if rowDate := dateKey(parsedRow.CreatedAtExternal); rowDate < dateKey(run.DateFrom) || rowDate > dateKey(run.DateTo) {
			continue
		}
		if _, ok := seenInnerIDSet[parsedRow.ExternalInnerID]; !ok {
			seenInnerIDSet[parsedRow.ExternalInnerID] = struct{}{}
			seenInnerIDs = append(seenInnerIDs, parsedRow.ExternalInnerID)
		}

		traderID := lookups.matchTrader(parsedRow.WorkerName)
		if traderID == nil {
			return result, fmt.Errorf("row %d: unmatched trader %q", parsedRow.RowNumber, parsedRow.WorkerName)
		}
		requisiteMatch := lookups.matchRequisite(parsedRow)
		if requisiteMatch.issueType != "" || requisiteMatch.requisiteID == nil {
			return result, fmt.Errorf("row %d: %s", parsedRow.RowNumber, requisiteMatch.message)
		}

		existing, err := selectTeamleadExternalOrderSnapshot(ctx, tx, run.TeamID, direction, parsedRow.ExternalInnerID)
		if err != nil {
			return result, err
		}
		tlStatus := TLStatusUpdatedByTL
		if existing != nil && !teamleadExternalOrderChanged(*existing, parsedRow, traderID, requisiteMatch.requisiteID) {
			tlStatus = TLStatusConfirmedByTL
		}
		orderID, err := upsertTeamleadExternalOrder(ctx, tx, teamleadExternalOrderApplyRecord{
			RunID:       run.ID,
			TeamID:      run.TeamID,
			BatchID:     batchID,
			Direction:   direction,
			Row:         parsedRow,
			TraderID:    traderID,
			RequisiteID: requisiteMatch.requisiteID,
			TLStatus:    tlStatus,
		})
		if err != nil {
			return result, err
		}

		result.RowsApplied++
		if existing == nil {
			result.CreatedOrders++
			updatedOrderIDs = append(updatedOrderIDs, orderID)
			continue
		}
		if tlStatus == TLStatusConfirmedByTL {
			result.ConfirmedOrders++
			confirmedOrderIDs = append(confirmedOrderIDs, orderID)
		} else {
			result.UpdatedOrders++
			updatedOrderIDs = append(updatedOrderIDs, orderID)
		}
	}

	discrepancyOrderIDs, err := markMissingTeamleadOrdersDiscrepant(ctx, tx, run, direction, seenInnerIDs)
	if err != nil {
		return result, err
	}
	result.DiscrepancyOrders = int64(len(discrepancyOrderIDs))
	if err := markAffectedShiftTLStatus(ctx, tx, run.TeamID, run.ID, confirmedOrderIDs, TLStatusConfirmedByTL); err != nil {
		return result, err
	}
	if err := markAffectedShiftTLStatus(ctx, tx, run.TeamID, run.ID, updatedOrderIDs, TLStatusUpdatedByTL); err != nil {
		return result, err
	}
	if err := markAffectedShiftTLStatus(ctx, tx, run.TeamID, run.ID, discrepancyOrderIDs, TLStatusDiscrepancy); err != nil {
		return result, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE teamlead_reconciliation_items
		SET applied_at = now()
		WHERE team_id = $1
		  AND teamlead_reconciliation_id = $2
		  AND direction = $3
		  AND applied_at IS NULL
	`, run.TeamID, run.ID, direction); err != nil {
		return result, err
	}

	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	committed = true
	return result, nil
}

func (r *Repository) ListTeamleadReconciliationTraders(ctx context.Context, teamID int64) ([]TeamleadTraderMatch, error) {
	if r.db == nil {
		return nil, ErrRepositoryNotConfigured
	}
	rows, err := db.New(r.db).ListTeamleadReconciliationTraders(ctx, teamID)
	if err != nil {
		return nil, err
	}
	result := make([]TeamleadTraderMatch, 0, len(rows))
	for _, row := range rows {
		result = append(result, TeamleadTraderMatch{
			TraderID:           row.TraderID,
			ExternalWorkerName: row.ExternalWorkerName,
		})
	}
	return result, nil
}

func (r *Repository) ListTeamleadReconciliationRequisites(ctx context.Context, teamID int64) ([]TeamleadRequisiteMatch, error) {
	if r.db == nil {
		return nil, ErrRepositoryNotConfigured
	}
	rows, err := db.New(r.db).ListTeamleadReconciliationRequisites(ctx, teamID)
	if err != nil {
		return nil, err
	}
	result := make([]TeamleadRequisiteMatch, 0, len(rows))
	for _, row := range rows {
		result = append(result, TeamleadRequisiteMatch{
			ID:                   row.ID,
			BankCode:             row.BankCode,
			Phone:                row.Phone,
			CardNumber:           textPtr(row.CardNumber),
			NormalizedPhone:      textString(row.NormalizedPhone),
			NormalizedCardNumber: textString(row.NormalizedCardNumber),
		})
	}
	return result, nil
}

func (r *Repository) ListTeamleadReconciliationExternalOrders(ctx context.Context, teamID int64, direction string, innerIDs []string) ([]TeamleadExternalOrderSnapshot, error) {
	if r.db == nil {
		return nil, ErrRepositoryNotConfigured
	}
	if len(innerIDs) == 0 {
		return []TeamleadExternalOrderSnapshot{}, nil
	}
	rows, err := db.New(r.db).ListTeamleadReconciliationExternalOrders(ctx, db.ListTeamleadReconciliationExternalOrdersParams{
		TeamID:    teamID,
		Direction: direction,
		InnerIds:  innerIDs,
	})
	if err != nil {
		return nil, err
	}
	return fromDBExternalOrderSnapshots(rows), nil
}

func (r *Repository) ListTeamleadReconciliationExternalOrdersInPeriod(ctx context.Context, teamID int64, direction string, dateFrom time.Time, dateTo time.Time) ([]TeamleadExternalOrderSnapshot, error) {
	if r.db == nil {
		return nil, ErrRepositoryNotConfigured
	}
	dateToExclusive := dateTo.AddDate(0, 0, 1)
	rows, err := db.New(r.db).ListTeamleadReconciliationExternalOrdersInPeriod(ctx, db.ListTeamleadReconciliationExternalOrdersInPeriodParams{
		TeamID:          teamID,
		Direction:       direction,
		DateFrom:        timestamptzValueFromTime(dateFrom),
		DateToExclusive: timestamptzValueFromTime(dateToExclusive),
	})
	if err != nil {
		return nil, err
	}
	return fromDBExternalOrderPeriodSnapshots(rows), nil
}

func (r *Repository) CalculateTeamleadV2InboundTurnover(ctx context.Context, teamID int64, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error) {
	if r.db == nil {
		return TeamleadTurnoverSnapshot{}, ErrRepositoryNotConfigured
	}
	row, err := db.New(r.db).CalculateTeamleadV2InboundTurnover(ctx, db.CalculateTeamleadV2InboundTurnoverParams{
		TeamID:   teamID,
		DateFrom: dateValue(dateFrom),
		DateTo:   dateValue(dateTo),
	})
	if err != nil {
		return TeamleadTurnoverSnapshot{}, err
	}
	return TeamleadTurnoverSnapshot{AmountMinor: row.AmountMinor, Count: row.RequisitesCount}, nil
}

func (r *Repository) CalculateTeamleadV2OutboundTransfers(ctx context.Context, teamID int64, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error) {
	if r.db == nil {
		return TeamleadTurnoverSnapshot{}, ErrRepositoryNotConfigured
	}
	row, err := db.New(r.db).CalculateTeamleadV2OutboundTransfers(ctx, db.CalculateTeamleadV2OutboundTransfersParams{
		TeamID:   teamID,
		DateFrom: dateValue(dateFrom),
		DateTo:   dateValue(dateTo),
	})
	if err != nil {
		return TeamleadTurnoverSnapshot{}, err
	}
	return TeamleadTurnoverSnapshot{AmountMinor: row.AmountMinor, Count: row.TransfersCount}, nil
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

	inboundRequisites, err := queries.ListTraderInboundReconciliationRequisites(ctx, db.ListTraderInboundReconciliationRequisitesParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	inboundReviewRequisites, err := queries.ListTraderInboundReconciliationReviewRequisites(ctx, db.ListTraderInboundReconciliationReviewRequisitesParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	inboundScopeItems, err := queries.ListTraderInboundReconciliationScopeItems(ctx, db.ListTraderInboundReconciliationScopeItemsParams{
		TeamID:  record.TeamID,
		ShiftID: int8Value(&record.ShiftID),
	})
	if err != nil {
		return Run{}, err
	}

	bankRows, err := queries.ListActiveBanks(ctx)
	if err != nil {
		return Run{}, err
	}
	bankNames := reconciliationBankNamesByCode(bankRows)

	inboundPlan, err := buildTraderInboundReconciliationPlan(
		traderInboundRequisitesFromDB(inboundRequisites, bankNames),
		traderInboundReviewRequisitesFromDB(inboundReviewRequisites),
		traderInboundScopeItemsFromDB(inboundScopeItems),
	)
	if err != nil {
		return Run{}, err
	}

	status := traderReconciliationStatus(diff, len(inboundPlan.items))

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

	if err := createReconciliationItems(ctx, queries, run.ID, inboundPlan.items); err != nil {
		return Run{}, err
	}

	for _, reviewStatus := range inboundPlan.statuses {
		if err := queries.UpdateShiftRequisiteInboundReviewStatus(ctx, db.UpdateShiftRequisiteInboundReviewStatusParams{
			Status:           reviewStatus.status,
			ShiftRequisiteID: reviewStatus.shiftRequisiteID,
			TeamID:           record.TeamID,
			TraderID:         record.TraderID,
			ShiftID:          record.ShiftID,
		}); err != nil {
			return Run{}, err
		}
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

	outboundSourceRequisites, err := queries.ListTraderOutboundReconciliationSourceRequisites(ctx, db.ListTraderOutboundReconciliationSourceRequisitesParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	outboundScopeItems, err := queries.ListTraderOutboundReconciliationScopeItems(ctx, db.ListTraderOutboundReconciliationScopeItemsParams{
		TeamID:  record.TeamID,
		ShiftID: int8Value(&record.ShiftID),
	})
	if err != nil {
		return Run{}, err
	}

	outboundPayoutOrders, err := queries.ListTraderOutboundReconciliationPayoutOrders(ctx, db.ListTraderOutboundReconciliationPayoutOrdersParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	outboundTransfers, err := queries.ListTraderOutboundReconciliationTransfers(ctx, db.ListTraderOutboundReconciliationTransfersParams{
		TeamID:   record.TeamID,
		TraderID: record.TraderID,
		ShiftID:  record.ShiftID,
	})
	if err != nil {
		return Run{}, err
	}

	bankRows, err := queries.ListActiveBanks(ctx)
	if err != nil {
		return Run{}, err
	}
	bankNames := reconciliationBankNamesByCode(bankRows)

	outboundItems, err := buildTraderOutboundReconciliationItems(
		traderOutboundSourceRequisitesFromDB(outboundSourceRequisites, bankNames),
		traderOutboundScopeItemsFromDB(outboundScopeItems),
		traderOutboundPayoutOrdersFromDB(outboundPayoutOrders),
		traderOutboundTransfersFromDB(outboundTransfers),
	)
	if err != nil {
		return Run{}, err
	}

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

	if err := createReconciliationItems(ctx, queries, run.ID, outboundItems); err != nil {
		return Run{}, err
	}

	status := traderReconciliationStatus(diff, len(outboundItems))

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
	dateFrom, dateToExclusive, err := teamleadPeriodDateRange(ctx, queries, record.TeamID, record.AccountingPeriodID)
	if err != nil {
		return Run{}, err
	}

	teamleadRows, err := queries.ListTeamleadPeriodReconciliationScopeItems(ctx, db.ListTeamleadPeriodReconciliationScopeItemsParams{
		TeamID:             record.TeamID,
		AccountingPeriodID: int8Value(&record.AccountingPeriodID),
		Direction:          imports.DirectionInbound,
	})
	if err != nil {
		return Run{}, err
	}

	traderRows, err := queries.ListTeamleadPeriodInboundShiftRequisites(ctx, db.ListTeamleadPeriodInboundShiftRequisitesParams{
		TeamID:          record.TeamID,
		DateFrom:        timestamptzValueFromTime(dateFrom),
		DateToExclusive: timestamptzValueFromTime(dateToExclusive),
	})
	if err != nil {
		return Run{}, err
	}

	result, err := buildTeamleadPeriodInboundReconciliation(
		teamleadScopeOrdersFromPeriodDB(teamleadRows),
		teamleadPeriodInboundShiftRequisitesFromDB(traderRows),
	)
	if err != nil {
		return Run{}, err
	}
	summary := result.summary
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

	if err := createReconciliationItems(ctx, queries, run.ID, result.items); err != nil {
		return Run{}, err
	}

	status := teamleadReconciliationStatus(summary, len(result.items))

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
	dateFrom, dateToExclusive, err := teamleadPeriodDateRange(ctx, queries, record.TeamID, record.AccountingPeriodID)
	if err != nil {
		return Run{}, err
	}

	teamleadRows, err := queries.ListTeamleadPeriodReconciliationScopeItems(ctx, db.ListTeamleadPeriodReconciliationScopeItemsParams{
		TeamID:             record.TeamID,
		AccountingPeriodID: int8Value(&record.AccountingPeriodID),
		Direction:          imports.DirectionOutbound,
	})
	if err != nil {
		return Run{}, err
	}

	payoutRows, err := queries.ListTeamleadPeriodPayoutOrders(ctx, db.ListTeamleadPeriodPayoutOrdersParams{
		TeamID:          record.TeamID,
		DateFrom:        timestamptzValueFromTime(dateFrom),
		DateToExclusive: timestamptzValueFromTime(dateToExclusive),
	})
	if err != nil {
		return Run{}, err
	}

	transferRows, err := queries.ListTeamleadPeriodPayoutTransfers(ctx, db.ListTeamleadPeriodPayoutTransfersParams{
		TeamID:          record.TeamID,
		DateFrom:        timestamptzValueFromTime(dateFrom),
		DateToExclusive: timestamptzValueFromTime(dateToExclusive),
	})
	if err != nil {
		return Run{}, err
	}

	result, err := buildTeamleadPeriodOutboundReconciliation(
		teamleadScopeOrdersFromPeriodDB(teamleadRows),
		teamleadPeriodPayoutOrdersFromDB(payoutRows),
		teamleadPeriodPayoutTransfersFromDB(transferRows),
	)
	if err != nil {
		return Run{}, err
	}
	summary := result.summary
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

	if err := createReconciliationItems(ctx, queries, run.ID, result.items); err != nil {
		return Run{}, err
	}

	status := teamleadReconciliationStatus(summary, len(result.items))

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

	teamleadRows, err := queries.ListTeamleadCurrentReconciliationScopeItems(ctx, db.ListTeamleadCurrentReconciliationScopeItemsParams{
		TeamID:        record.TeamID,
		Direction:     record.Direction,
		ImportBatchID: latestImport.ID,
	})
	if err != nil {
		return Run{}, err
	}

	teamleadOrders := teamleadScopeOrdersFromCurrentDB(teamleadRows)
	innerIDs := teamleadScopeOrderInnerIDs(teamleadOrders)
	traderOrders := []teamleadScopeOrderRow{}
	if len(innerIDs) > 0 {
		traderRows, err := queries.ListTraderReconciliationScopeItemsByInnerIDs(ctx, db.ListTraderReconciliationScopeItemsByInnerIDsParams{
			TeamID:    record.TeamID,
			Direction: record.Direction,
			InnerIds:  innerIDs,
		})
		if err != nil {
			return Run{}, err
		}
		traderOrders = teamleadScopeOrdersFromTraderInnerIDsDB(traderRows)
	}

	result, err := buildTeamleadCurrentReconciliation(teamleadOrders, traderOrders)
	if err != nil {
		return Run{}, err
	}
	summary := result.summary
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

	if err := createReconciliationItems(ctx, queries, run.ID, result.items); err != nil {
		return Run{}, err
	}

	status := teamleadReconciliationStatus(summary, len(result.items))

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

func createReconciliationItems(ctx context.Context, queries *db.Queries, runID int64, items []traderReconciliationItemRecord) error {
	for _, item := range items {
		if _, err := queries.CreateReconciliationItem(ctx, db.CreateReconciliationItemParams{
			RunID:             runID,
			IssueType:         item.issueType,
			ExternalOrderID:   int8Value(item.externalOrderID),
			ExternalInnerID:   textValue(item.externalInnerID),
			TeamleadValueJson: jsonbValue(item.teamleadValueJSON),
			TraderValueJson:   jsonbValue(item.traderValueJSON),
			Message:           textValue(&item.message),
		}); err != nil {
			return err
		}
	}
	return nil
}

func traderInboundRequisitesFromDB(rows []db.ListTraderInboundReconciliationRequisitesRow, bankNames map[string]string) []traderInboundRequisiteRow {
	items := make([]traderInboundRequisiteRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderInboundRequisiteRow{
			shiftRequisiteID: row.ShiftRequisiteID,
			requisiteID:      row.RequisiteID,
			phone:            row.Phone,
			bankName:         bankNames[row.BankCode],
			actualAmount:     row.ActualAmountMinor,
		})
	}
	return items
}

func traderInboundReviewRequisitesFromDB(rows []db.ListTraderInboundReconciliationReviewRequisitesRow) []traderInboundReviewRequisiteRow {
	items := make([]traderInboundReviewRequisiteRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderInboundReviewRequisiteRow{
			shiftRequisiteID: row.ShiftRequisiteID,
			requisiteID:      row.RequisiteID,
			phone:            row.Phone,
			actualAmount:     row.ActualAmountMinor,
		})
	}
	return items
}

func traderInboundScopeItemsFromDB(rows []db.ListTraderInboundReconciliationScopeItemsRow) []traderInboundScopeItem {
	items := make([]traderInboundScopeItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderInboundScopeItem{
			csvRequisite:     row.CsvRequisite,
			amountMinor:      row.AmountMinor,
			normalizedStatus: row.NormalizedStatus,
		})
	}
	return items
}

func traderOutboundSourceRequisitesFromDB(rows []db.ListTraderOutboundReconciliationSourceRequisitesRow, bankNames map[string]string) []traderOutboundSourceRequisiteRow {
	items := make([]traderOutboundSourceRequisiteRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderOutboundSourceRequisiteRow{
			shiftRequisiteID:            row.ShiftRequisiteID,
			requisiteID:                 row.RequisiteID,
			phone:                       row.Phone,
			bankName:                    bankNames[row.BankCode],
			closedOutboundTurnoverMinor: row.ClosedOutboundTurnoverMinor,
		})
	}
	return items
}

func traderOutboundScopeItemsFromDB(rows []db.ListTraderOutboundReconciliationScopeItemsRow) []traderOutboundScopeItem {
	items := make([]traderOutboundScopeItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderOutboundScopeItem{
			id:               row.ID,
			externalOrderID:  row.ExternalOrderID,
			externalInnerID:  row.ExternalInnerID,
			amountMinor:      row.AmountMinor,
			normalizedStatus: row.NormalizedStatus,
			createdAt:        row.CreatedAt.Time,
		})
	}
	return items
}

func traderOutboundPayoutOrdersFromDB(rows []db.ListTraderOutboundReconciliationPayoutOrdersRow) []traderOutboundPayoutOrderRow {
	items := make([]traderOutboundPayoutOrderRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderOutboundPayoutOrderRow{
			id:                   row.ID,
			destinationBank:      row.DestinationBank,
			destinationRequisite: row.DestinationRequisite,
			amountMinor:          row.AmountMinor,
			createdAt:            row.CreatedAt.Time,
		})
	}
	return items
}

func traderOutboundTransfersFromDB(rows []db.ListTraderOutboundReconciliationTransfersRow) []traderOutboundTransferRow {
	items := make([]traderOutboundTransferRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, traderOutboundTransferRow{
			manualPayoutOrderID:    row.ManualPayoutOrderID,
			sourceShiftRequisiteID: row.SourceShiftRequisiteID,
			amountMinor:            row.AmountMinor,
		})
	}
	return items
}

func teamleadPeriodDateRange(ctx context.Context, queries *db.Queries, teamID int64, accountingPeriodID int64) (time.Time, time.Time, error) {
	period, err := queries.GetReconciliationAccountingPeriod(ctx, db.GetReconciliationAccountingPeriodParams{
		TeamID:             teamID,
		AccountingPeriodID: accountingPeriodID,
	})
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	dateFrom := dateToTime(period.DateFrom)
	dateToExclusive := dateToTime(period.DateTo).AddDate(0, 0, 1)
	return dateFrom, dateToExclusive, nil
}

func teamleadScopeOrdersFromPeriodDB(rows []db.ListTeamleadPeriodReconciliationScopeItemsRow) []teamleadScopeOrderRow {
	items := make([]teamleadScopeOrderRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, teamleadScopeOrderRow{
			id:                row.ID,
			externalOrderID:   row.ExternalOrderID,
			externalInnerID:   row.ExternalInnerID,
			workerName:        row.WorkerName,
			traderID:          int8Ptr(row.TraderID),
			requisiteRaw:      textPtr(row.RequisiteRaw),
			requisitePhone:    textPtr(row.RequisitePhone),
			amountMinor:       row.AmountMinor,
			normalizedStatus:  row.NormalizedStatus,
			rawStatus:         row.RawStatus,
			createdAtExternal: row.CreatedAtExternal.Time,
			createdAt:         row.CreatedAt.Time,
		})
	}
	return items
}

func teamleadScopeOrdersFromCurrentDB(rows []db.ListTeamleadCurrentReconciliationScopeItemsRow) []teamleadScopeOrderRow {
	items := make([]teamleadScopeOrderRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, teamleadScopeOrderRow{
			id:                row.ID,
			externalOrderID:   row.ExternalOrderID,
			externalInnerID:   row.ExternalInnerID,
			workerName:        row.WorkerName,
			traderID:          int8Ptr(row.TraderID),
			requisiteRaw:      textPtr(row.RequisiteRaw),
			requisitePhone:    textPtr(row.RequisitePhone),
			amountMinor:       row.AmountMinor,
			normalizedStatus:  row.NormalizedStatus,
			rawStatus:         row.RawStatus,
			createdAtExternal: row.CreatedAtExternal.Time,
			createdAt:         row.CreatedAt.Time,
		})
	}
	return items
}

func teamleadScopeOrdersFromTraderInnerIDsDB(rows []db.ListTraderReconciliationScopeItemsByInnerIDsRow) []teamleadScopeOrderRow {
	items := make([]teamleadScopeOrderRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, teamleadScopeOrderRow{
			id:                row.ID,
			externalOrderID:   row.ExternalOrderID,
			externalInnerID:   row.ExternalInnerID,
			workerName:        row.WorkerName,
			traderID:          int8Ptr(row.TraderID),
			traderLogin:       textPtr(row.TraderLogin),
			requisiteRaw:      textPtr(row.RequisiteRaw),
			requisitePhone:    textPtr(row.RequisitePhone),
			amountMinor:       row.AmountMinor,
			normalizedStatus:  row.NormalizedStatus,
			rawStatus:         row.RawStatus,
			createdAtExternal: row.CreatedAtExternal.Time,
			createdAt:         row.CreatedAt.Time,
		})
	}
	return items
}

func teamleadPeriodInboundShiftRequisitesFromDB(rows []db.ListTeamleadPeriodInboundShiftRequisitesRow) []teamleadPeriodInboundShiftRequisiteRow {
	items := make([]teamleadPeriodInboundShiftRequisiteRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, teamleadPeriodInboundShiftRequisiteRow{
			shiftRequisiteID:     row.ShiftRequisiteID,
			requisiteID:          row.RequisiteID,
			requisitePhone:       row.RequisitePhone,
			bankCode:             row.BankCode,
			traderID:             row.TraderID,
			traderLogin:          row.TraderLogin,
			amountMinor:          row.AmountMinor,
			shiftRequisiteStatus: row.ShiftRequisiteStatus,
		})
	}
	return items
}

func teamleadPeriodPayoutOrdersFromDB(rows []db.ListTeamleadPeriodPayoutOrdersRow) []teamleadPeriodPayoutOrderRow {
	items := make([]teamleadPeriodPayoutOrderRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, teamleadPeriodPayoutOrderRow{
			id:                   row.ID,
			destinationBank:      row.DestinationBank,
			destinationRequisite: row.DestinationRequisite,
			amountMinor:          row.AmountMinor,
			traderID:             row.TraderID,
			traderLogin:          row.TraderLogin,
			createdAt:            row.CreatedAt.Time,
		})
	}
	return items
}

func teamleadPeriodPayoutTransfersFromDB(rows []db.ListTeamleadPeriodPayoutTransfersRow) []teamleadPeriodPayoutTransferRow {
	items := make([]teamleadPeriodPayoutTransferRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, teamleadPeriodPayoutTransferRow{
			manualPayoutOrderID: row.ManualPayoutOrderID,
			amountMinor:         row.AmountMinor,
		})
	}
	return items
}

func teamleadScopeOrderInnerIDs(rows []teamleadScopeOrderRow) []string {
	orders := latestTeamleadScopeOrders(rows)
	innerIDs := make([]string, 0, len(orders))
	seen := make(map[string]bool, len(orders))
	for _, row := range orders {
		if seen[row.externalInnerID] {
			continue
		}
		seen[row.externalInnerID] = true
		innerIDs = append(innerIDs, row.externalInnerID)
	}
	return innerIDs
}

func timestamptzValueFromTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func reconciliationBankNamesByCode(rows []db.Bank) map[string]string {
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.Code] = row.Name
	}
	return names
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

type teamleadImportRow struct {
	ID             int64
	RowNumber      int64
	RawPayloadJSON []byte
}

type teamleadExternalOrderApplyRecord struct {
	RunID       int64
	TeamID      int64
	BatchID     int64
	Direction   string
	Row         imports.ParsedOrderRow
	TraderID    *int64
	RequisiteID *int64
	TLStatus    string
}

func loadTeamleadImportRows(ctx context.Context, tx pgx.Tx, batchID int64) ([]teamleadImportRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, row_number, raw_payload_json
		FROM import_rows
		WHERE import_batch_id = $1
		ORDER BY row_number
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]teamleadImportRow, 0)
	for rows.Next() {
		var item teamleadImportRow
		if err := rows.Scan(&item.ID, &item.RowNumber, &item.RawPayloadJSON); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func parsedOrderRowFromImportRow(row teamleadImportRow) (imports.ParsedOrderRow, error) {
	var raw map[string]string
	if err := json.Unmarshal(row.RawPayloadJSON, &raw); err != nil {
		return imports.ParsedOrderRow{}, err
	}
	parsed, parseErrors := imports.ParseRawOrderRow(int(row.RowNumber), raw, imports.ParseOptions{ColumnSet: imports.ColumnSetTeamlead})
	if len(parseErrors) > 0 {
		return imports.ParsedOrderRow{}, fmt.Errorf("row %d: %s", row.RowNumber, parseErrors[0].Message)
	}
	return parsed, nil
}

func selectTeamleadExternalOrderSnapshot(ctx context.Context, tx pgx.Tx, teamID int64, direction string, innerID string) (*TeamleadExternalOrderSnapshot, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, direction, external_inner_id, worker_name, trader_id, requisite_id, amount_minor, raw_status, normalized_status, created_at_external
		FROM external_orders
		WHERE team_id = $1
		  AND direction = $2
		  AND external_inner_id = $3
		FOR UPDATE
	`, teamID, direction, innerID)

	var snapshot TeamleadExternalOrderSnapshot
	var traderID pgtype.Int8
	var requisiteID pgtype.Int8
	var createdAt pgtype.Timestamptz
	if err := row.Scan(
		&snapshot.ID,
		&snapshot.Direction,
		&snapshot.ExternalInnerID,
		&snapshot.WorkerName,
		&traderID,
		&requisiteID,
		&snapshot.AmountMinor,
		&snapshot.RawStatus,
		&snapshot.NormalizedStatus,
		&createdAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	snapshot.TraderID = int8Ptr(traderID)
	snapshot.RequisiteID = int8Ptr(requisiteID)
	if createdAt.Valid {
		snapshot.CreatedAtExternal = createdAt.Time
	}
	return &snapshot, nil
}

func teamleadExternalOrderChanged(existing TeamleadExternalOrderSnapshot, row imports.ParsedOrderRow, traderID *int64, requisiteID *int64) bool {
	if existing.AmountMinor != row.AmountMinor || existing.NormalizedStatus != row.NormalizedStatus || existing.RawStatus != row.RawStatus {
		return true
	}
	if existing.TraderID == nil || traderID == nil || *existing.TraderID != *traderID {
		return true
	}
	if existing.RequisiteID == nil || requisiteID == nil || *existing.RequisiteID != *requisiteID {
		return true
	}
	return !sameExternalDate(existing.CreatedAtExternal, row.CreatedAtExternal)
}

func upsertTeamleadExternalOrder(ctx context.Context, tx pgx.Tx, record teamleadExternalOrderApplyRecord) (int64, error) {
	row := record.Row
	var orderID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO external_orders (
			team_id, direction, external_id, external_inner_id, external_foreign_id, worker_name,
			trader_id, requisite_raw, requisite_phone, requisite_external_id, requisite_id,
			device_name, method_type, method_name, amount_minor, currency, course, course_worker,
			worker_amount, worker_profit, raw_status, normalized_status, created_at_external,
			closed_at_external, updated_at_external, old_amount_minor, had_dispute, receipt,
			order_comment, ordered, counted, initials, last_import_batch_id,
			tl_reconciliation_status, last_teamlead_reconciliation_id, tl_reconciled_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17::numeric, $18::numeric,
			$19::numeric, $20::numeric, $21, $22, $23,
			$24, $25, $26, $27, $28,
			$29, $30, $31, $32, $33,
			$34, $35, now()
		)
		ON CONFLICT (team_id, direction, external_inner_id)
		DO UPDATE SET
			external_id = EXCLUDED.external_id,
			external_foreign_id = EXCLUDED.external_foreign_id,
			worker_name = EXCLUDED.worker_name,
			trader_id = EXCLUDED.trader_id,
			requisite_raw = EXCLUDED.requisite_raw,
			requisite_phone = EXCLUDED.requisite_phone,
			requisite_external_id = EXCLUDED.requisite_external_id,
			requisite_id = EXCLUDED.requisite_id,
			device_name = EXCLUDED.device_name,
			method_type = EXCLUDED.method_type,
			method_name = EXCLUDED.method_name,
			amount_minor = EXCLUDED.amount_minor,
			currency = EXCLUDED.currency,
			course = EXCLUDED.course,
			course_worker = EXCLUDED.course_worker,
			worker_amount = EXCLUDED.worker_amount,
			worker_profit = EXCLUDED.worker_profit,
			raw_status = EXCLUDED.raw_status,
			normalized_status = EXCLUDED.normalized_status,
			created_at_external = EXCLUDED.created_at_external,
			closed_at_external = EXCLUDED.closed_at_external,
			updated_at_external = EXCLUDED.updated_at_external,
			old_amount_minor = EXCLUDED.old_amount_minor,
			had_dispute = EXCLUDED.had_dispute,
			receipt = EXCLUDED.receipt,
			order_comment = EXCLUDED.order_comment,
			ordered = EXCLUDED.ordered,
			counted = EXCLUDED.counted,
			initials = EXCLUDED.initials,
			last_import_batch_id = EXCLUDED.last_import_batch_id,
			tl_reconciliation_status = EXCLUDED.tl_reconciliation_status,
			last_teamlead_reconciliation_id = EXCLUDED.last_teamlead_reconciliation_id,
			tl_reconciled_at = EXCLUDED.tl_reconciled_at,
			updated_at = now()
		RETURNING id
	`,
		record.TeamID,
		record.Direction,
		row.ExternalID,
		row.ExternalInnerID,
		nullableString(row.ExternalForeignID),
		row.WorkerName,
		nullableInt64(record.TraderID),
		nullableString(row.RequisiteRaw),
		nullableString(row.RequisitePhone),
		nullableString(row.RequisiteExternalID),
		nullableInt64(record.RequisiteID),
		nullableString(row.DeviceName),
		nullableString(row.MethodType),
		nullableString(row.MethodName),
		row.AmountMinor,
		row.Currency,
		nullableString(row.Course),
		nullableString(row.CourseWorker),
		nullableString(row.WorkerAmount),
		nullableString(row.WorkerProfit),
		row.RawStatus,
		row.NormalizedStatus,
		row.CreatedAtExternal,
		nullableTime(row.ClosedAtExternal),
		nullableTime(row.UpdatedAtExternal),
		nullableInt64(row.OldAmountMinor),
		nullableBool(row.HadDispute),
		nullableString(row.Receipt),
		nullableString(row.OrderComment),
		nullableBool(row.Ordered),
		nullableBool(row.Counted),
		nullableString(row.Initials),
		record.BatchID,
		record.TLStatus,
		record.RunID,
	).Scan(&orderID)
	return orderID, err
}

func markMissingTeamleadOrdersDiscrepant(ctx context.Context, tx pgx.Tx, run TeamleadRun, direction string, seenInnerIDs []string) ([]int64, error) {
	dateToExclusive := run.DateTo.AddDate(0, 0, 1)
	rows, err := tx.Query(ctx, `
		UPDATE external_orders
		SET tl_reconciliation_status = $5,
		    last_teamlead_reconciliation_id = $6,
		    tl_reconciled_at = now(),
		    updated_at = now()
		WHERE team_id = $1
		  AND direction = $2
		  AND created_at_external >= $3::timestamptz
		  AND created_at_external < $4::timestamptz
		  AND NOT (external_inner_id = ANY($7::text[]))
		RETURNING id
	`, run.TeamID, direction, run.DateFrom, dateToExclusive, TLStatusDiscrepancy, run.ID, seenInnerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func markAffectedShiftTLStatus(ctx context.Context, tx pgx.Tx, teamID int64, runID int64, orderIDs []int64, status string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		WITH affected AS (
			SELECT DISTINCT shift_id
			FROM order_scope_items
			WHERE team_id = $1
			  AND scope_type = 'trader_shift'
			  AND is_active = TRUE
			  AND shift_id IS NOT NULL
			  AND external_order_id = ANY($2::bigint[])
		)
		UPDATE trader_shifts ts
		SET tl_reconciliation_status = $3,
		    last_teamlead_reconciliation_id = $4,
		    tl_reconciled_at = now(),
		    updated_at = now()
		FROM affected
		WHERE ts.team_id = $1
		  AND ts.id = affected.shift_id
	`, teamID, orderIDs, status, runID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		WITH affected AS (
			SELECT DISTINCT osi.shift_id, eo.requisite_id
			FROM order_scope_items osi
			JOIN external_orders eo ON eo.id = osi.external_order_id
			WHERE osi.team_id = $1
			  AND osi.scope_type = 'trader_shift'
			  AND osi.is_active = TRUE
			  AND osi.shift_id IS NOT NULL
			  AND eo.requisite_id IS NOT NULL
			  AND osi.external_order_id = ANY($2::bigint[])
		)
		UPDATE shift_requisites sr
		SET tl_reconciliation_status = $3,
		    last_teamlead_reconciliation_id = $4,
		    tl_reconciled_at = now(),
		    updated_at = now()
		FROM affected
		WHERE sr.team_id = $1
		  AND sr.shift_id = affected.shift_id
		  AND sr.requisite_id = affected.requisite_id
	`, teamID, orderIDs, status, runID)
	return err
}

func fromDBTeamleadTraderMatches(rows []db.ListTeamleadReconciliationTradersRow) []TeamleadTraderMatch {
	result := make([]TeamleadTraderMatch, 0, len(rows))
	for _, row := range rows {
		result = append(result, TeamleadTraderMatch{
			TraderID:           row.TraderID,
			ExternalWorkerName: row.ExternalWorkerName,
		})
	}
	return result
}

func fromDBTeamleadRequisiteMatches(rows []db.ListTeamleadReconciliationRequisitesRow) []TeamleadRequisiteMatch {
	result := make([]TeamleadRequisiteMatch, 0, len(rows))
	for _, row := range rows {
		result = append(result, TeamleadRequisiteMatch{
			ID:                   row.ID,
			BankCode:             row.BankCode,
			Phone:                row.Phone,
			CardNumber:           textPtr(row.CardNumber),
			NormalizedPhone:      textString(row.NormalizedPhone),
			NormalizedCardNumber: textString(row.NormalizedCardNumber),
		})
	}
	return result
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

func insertTeamleadImportRow(ctx context.Context, queries *db.Queries, batchID int64, parsedRow imports.ParsedOrderRow) (db.ImportRow, error) {
	rawPayload, err := json.Marshal(parsedRow.RawPayload)
	if err != nil {
		return db.ImportRow{}, err
	}
	return queries.InsertImportRow(ctx, db.InsertImportRowParams{
		ImportBatchID:   batchID,
		RowNumber:       int64(parsedRow.RowNumber),
		ExternalID:      textValue(&parsedRow.ExternalID),
		ExternalInnerID: textValue(&parsedRow.ExternalInnerID),
		RawPayloadJson:  rawPayload,
	})
}

func fromDBTeamleadRun(row db.TeamleadReconciliation) TeamleadRun {
	return TeamleadRun{
		ID:                    row.ID,
		TeamID:                row.TeamID,
		DateFrom:              dateToTime(row.DateFrom),
		DateTo:                dateToTime(row.DateTo),
		Status:                row.Status,
		CreatedBy:             row.CreatedBy,
		ConfirmedBy:           int8Ptr(row.ConfirmedBy),
		RejectedBy:            int8Ptr(row.RejectedBy),
		InboundImportBatchID:  int8Ptr(row.InboundImportBatchID),
		OutboundImportBatchID: int8Ptr(row.OutboundImportBatchID),
		Comment:               textPtr(row.Comment),
		MismatchCount:         row.MismatchCount,
		ConflictCount:         row.ConflictCount,
		BlockedCount:          row.BlockedCount,
		PipelineJSON:          row.PipelineJson,
		InboundSummaryJSON:    row.InboundSummaryJson,
		OutboundSummaryJSON:   row.OutboundSummaryJson,
		PreviewJSON:           row.PreviewJson,
		ApplyResultJSON:       row.ApplyResultJson,
		ErrorMessage:          textPtr(row.ErrorMessage),
		CreatedAt:             row.CreatedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
		AnalyzedAt:            timePtr(row.AnalyzedAt),
		ConfirmedAt:           timePtr(row.ConfirmedAt),
		RejectedAt:            timePtr(row.RejectedAt),
		ApplyQueuedAt:         timePtr(row.ApplyQueuedAt),
		AppliedAt:             timePtr(row.AppliedAt),
	}
}

func fromDBTeamleadItem(row db.TeamleadReconciliationItem) TeamleadItem {
	return TeamleadItem{
		ID:                       row.ID,
		TeamleadReconciliationID: row.TeamleadReconciliationID,
		TeamID:                   row.TeamID,
		Direction:                row.Direction,
		Stage:                    row.Stage,
		IssueType:                row.IssueType,
		Severity:                 row.Severity,
		ExternalOrderID:          int8Ptr(row.ExternalOrderID),
		ExternalInnerID:          textPtr(row.ExternalInnerID),
		TraderID:                 int8Ptr(row.TraderID),
		RequisiteID:              int8Ptr(row.RequisiteID),
		ShiftID:                  int8Ptr(row.ShiftID),
		BeforeJSON:               row.BeforeJson,
		AfterJSON:                row.AfterJson,
		Message:                  textPtr(row.Message),
		IsBlocking:               row.IsBlocking,
		AppliedAt:                timePtr(row.AppliedAt),
		CreatedAt:                row.CreatedAt.Time,
	}
}

func fromDBExternalOrderSnapshots(rows []db.ListTeamleadReconciliationExternalOrdersRow) []TeamleadExternalOrderSnapshot {
	result := make([]TeamleadExternalOrderSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, TeamleadExternalOrderSnapshot{
			ID:                row.ID,
			Direction:         row.Direction,
			ExternalInnerID:   row.ExternalInnerID,
			WorkerName:        row.WorkerName,
			TraderID:          int8Ptr(row.TraderID),
			RequisiteID:       int8Ptr(row.RequisiteID),
			AmountMinor:       row.AmountMinor,
			RawStatus:         row.RawStatus,
			NormalizedStatus:  row.NormalizedStatus,
			CreatedAtExternal: row.CreatedAtExternal.Time,
		})
	}
	return result
}

func fromDBExternalOrderPeriodSnapshots(rows []db.ListTeamleadReconciliationExternalOrdersInPeriodRow) []TeamleadExternalOrderSnapshot {
	result := make([]TeamleadExternalOrderSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, TeamleadExternalOrderSnapshot{
			ID:                row.ID,
			Direction:         row.Direction,
			ExternalInnerID:   row.ExternalInnerID,
			WorkerName:        row.WorkerName,
			TraderID:          int8Ptr(row.TraderID),
			RequisiteID:       int8Ptr(row.RequisiteID),
			AmountMinor:       row.AmountMinor,
			RawStatus:         row.RawStatus,
			NormalizedStatus:  row.NormalizedStatus,
			CreatedAtExternal: row.CreatedAtExternal.Time,
		})
	}
	return result
}

func dateValue(value time.Time) pgtype.Date {
	if value.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: value, Valid: true}
}

func dateToTime(value pgtype.Date) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func jsonbValue(value json.RawMessage) []byte {
	if len(value) == 0 {
		return nil
	}
	return []byte(value)
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
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

func optionalText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
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
