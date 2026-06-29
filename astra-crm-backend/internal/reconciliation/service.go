package reconciliation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
)

var (
	ErrInvalidInput = errors.New("invalid reconciliation input")
	ErrInvalidState = errors.New("invalid reconciliation state")
)

type Store interface {
	RecalculateTraderInbound(ctx context.Context, record RecalculateTraderInboundRecord) (Run, error)
	LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error)
	LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error)
	GetTraderInboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error)
	AcceptTraderInbound(ctx context.Context, record AcceptTraderInboundRecord) (Run, error)
	RecalculateTraderOutbound(ctx context.Context, record RecalculateTraderOutboundRecord) (Run, error)
	LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error)
	LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error)
	GetTraderOutboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error)
	AcceptTraderOutbound(ctx context.Context, record AcceptTraderOutboundRecord) (Run, error)
	RecalculateTeamleadPeriodInbound(ctx context.Context, record RecalculateTeamleadPeriodInboundRecord) (Run, error)
	RecalculateTeamleadPeriodOutbound(ctx context.Context, record RecalculateTeamleadPeriodOutboundRecord) (Run, error)
	RecalculateTeamleadCurrent(ctx context.Context, record RecalculateTeamleadCurrentRecord) (Run, error)
	LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error)
	LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error)
	LatestTeamleadInbound(ctx context.Context, teamID int64, actorID int64) (Run, error)
	LatestTeamleadOutbound(ctx context.Context, teamID int64, actorID int64) (Run, error)
	ListTeamleadCurrentRuns(ctx context.Context, teamID int64, actorID int64, direction string, page pagination.Params) (pagination.Result[Run], error)
	GetTeamleadCurrentRun(ctx context.Context, teamID int64, actorID int64, direction string, runID int64) (Run, error)
	AcceptTeamleadCurrent(ctx context.Context, record AcceptTeamleadCurrentRecord) (Run, error)
	ListItems(ctx context.Context, runID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error)
	ListActiveTeamleadInboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadInboundPeriodScope, error)
	ListActiveTeamleadOutboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadOutboundPeriodScope, error)
	CreateTeamleadAnalysis(ctx context.Context, record CreateTeamleadAnalysisRecord) (TeamleadRun, error)
	GetTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error)
	ListTeamleadReconciliations(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[TeamleadRun], error)
	ListTeamleadReconciliationItems(ctx context.Context, teamID int64, runID int64, filters TeamleadItemFilters, page pagination.Params) (pagination.Result[TeamleadItem], error)
	QueueTeamleadReconciliationApply(ctx context.Context, record QueueTeamleadApplyRecord) (TeamleadRun, error)
	RejectTeamleadReconciliation(ctx context.Context, record RejectTeamleadReconciliationRecord) (TeamleadRun, error)
	ApplyQueuedTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error)
	ListTeamleadReconciliationTraders(ctx context.Context, teamID int64) ([]TeamleadTraderMatch, error)
	ListTeamleadReconciliationRequisites(ctx context.Context, teamID int64) ([]TeamleadRequisiteMatch, error)
	ListTeamleadReconciliationExternalOrders(ctx context.Context, teamID int64, direction string, innerIDs []string) ([]TeamleadExternalOrderSnapshot, error)
	ListTeamleadReconciliationExternalOrdersInPeriod(ctx context.Context, teamID int64, direction string, dateFrom time.Time, dateTo time.Time) ([]TeamleadExternalOrderSnapshot, error)
	CalculateTeamleadV2InboundTurnover(ctx context.Context, teamID int64, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error)
	CalculateTeamleadV2OutboundTransfers(ctx context.Context, teamID int64, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type TeamleadApplyScheduler interface {
	EnqueueTeamleadApply(teamID int64, runID int64)
}

type Service struct {
	store     Store
	audit     AuditService
	scheduler TeamleadApplyScheduler
}

func NewService(store Store, auditService AuditService, schedulers ...TeamleadApplyScheduler) *Service {
	service := &Service{
		store: store,
		audit: auditService,
	}
	if len(schedulers) > 0 {
		service.scheduler = schedulers[0]
	} else {
		service.scheduler = service
	}
	return service
}

type RecalculateTraderInboundParams struct {
	TeamID        int64
	TraderID      int64
	ShiftID       int64
	ImportBatchID *int64
}

type RecalculateTraderOutboundParams struct {
	TeamID        int64
	TraderID      int64
	ShiftID       int64
	ImportBatchID *int64
}

type RecalculateTeamleadPeriodInboundParams struct {
	TeamID             int64
	AccountingPeriodID int64
	ImportBatchID      *int64
}

type RecalculateTeamleadPeriodOutboundParams struct {
	TeamID             int64
	AccountingPeriodID int64
	ImportBatchID      *int64
}

type RecalculateTeamleadCurrentParams struct {
	TeamID    int64
	ActorID   int64
	Direction string
}

type AcceptTraderInboundParams struct {
	ActorID  int64
	TeamID   int64
	TraderID int64
	RunID    int64
	Comment  string
}

type AcceptTraderOutboundParams struct {
	ActorID  int64
	TeamID   int64
	TraderID int64
	RunID    int64
	Comment  string
}

type AcceptTeamleadCurrentParams struct {
	ActorID   int64
	TeamID    int64
	Direction string
	RunID     int64
	Comment   string
}

type ListTeamleadCurrentRunsParams struct {
	TeamID    int64
	ActorID   int64
	Direction string
	Page      pagination.Params
}

type GetTeamleadCurrentRunParams struct {
	TeamID    int64
	ActorID   int64
	Direction string
	RunID     int64
}

type TeamleadCSVValidationError struct {
	Direction string
	Parse     imports.ParseResult
	Err       error
}

func (e *TeamleadCSVValidationError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s CSV validation failed", e.Direction)
}

func (e *TeamleadCSVValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (s *Service) CreateTeamleadReconciliation(ctx context.Context, params CreateTeamleadReconciliationParams) (TeamleadRun, error) {
	dateFrom := normalizeDate(params.DateFrom)
	dateTo := normalizeDate(params.DateTo)
	if params.ActorID <= 0 || params.TeamID <= 0 || dateFrom.IsZero() || dateTo.IsZero() || dateTo.Before(dateFrom) || (params.Inbound == nil && params.Outbound == nil) {
		return TeamleadRun{}, ErrInvalidInput
	}
	if s.store == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}

	traders, err := s.store.ListTeamleadReconciliationTraders(ctx, params.TeamID)
	if err != nil {
		return TeamleadRun{}, err
	}
	requisites, err := s.store.ListTeamleadReconciliationRequisites(ctx, params.TeamID)
	if err != nil {
		return TeamleadRun{}, err
	}
	lookups := newTeamleadAnalysisLookups(traders, requisites)

	var directions []TeamleadDirectionAnalysisRecord
	var allItems []TeamleadItemRecord
	if params.Inbound != nil {
		analysis, items, err := s.analyzeTeamleadDirection(ctx, params.TeamID, imports.DirectionInbound, *params.Inbound, dateFrom, dateTo, lookups)
		if err != nil {
			return TeamleadRun{}, err
		}
		directions = append(directions, analysis)
		allItems = append(allItems, items...)
	}
	if params.Outbound != nil {
		analysis, items, err := s.analyzeTeamleadDirection(ctx, params.TeamID, imports.DirectionOutbound, *params.Outbound, dateFrom, dateTo, lookups)
		if err != nil {
			return TeamleadRun{}, err
		}
		directions = append(directions, analysis)
		allItems = append(allItems, items...)
	}

	status, mismatchCount, conflictCount, blockedCount := teamleadRunStatusFromItems(allItems)
	pipelineJSON, err := marshalJSON(teamleadPipelineSummary(directions, allItems))
	if err != nil {
		return TeamleadRun{}, err
	}
	inboundSummary, outboundSummary, preview := teamleadSummaryJSON(directions)
	inboundSummaryJSON, err := marshalJSON(inboundSummary)
	if err != nil {
		return TeamleadRun{}, err
	}
	outboundSummaryJSON, err := marshalJSON(outboundSummary)
	if err != nil {
		return TeamleadRun{}, err
	}
	previewJSON, err := marshalJSON(preview)
	if err != nil {
		return TeamleadRun{}, err
	}

	run, err := s.store.CreateTeamleadAnalysis(ctx, CreateTeamleadAnalysisRecord{
		TeamID:          params.TeamID,
		ActorID:         params.ActorID,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
		Status:          status,
		MismatchCount:   mismatchCount,
		ConflictCount:   conflictCount,
		BlockedCount:    blockedCount,
		PipelineJSON:    pipelineJSON,
		InboundSummary:  inboundSummaryJSON,
		OutboundSummary: outboundSummaryJSON,
		PreviewJSON:     previewJSON,
		Directions:      directions,
		Items:           allItems,
	})
	if err != nil {
		return TeamleadRun{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionReconciliationCreated,
		EntityType: "teamlead_reconciliation",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicTeamleadRunFromDomain(run),
	}); err != nil {
		return TeamleadRun{}, err
	}

	return run, nil
}

func (s *Service) ConfirmTeamleadReconciliation(ctx context.Context, params ConfirmTeamleadReconciliationParams) (TeamleadRun, error) {
	comment := strings.TrimSpace(params.Comment)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.RunID <= 0 {
		return TeamleadRun{}, ErrInvalidInput
	}
	if s.store == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}

	current, err := s.store.GetTeamleadReconciliation(ctx, params.TeamID, params.RunID)
	if err != nil {
		return TeamleadRun{}, err
	}
	if !teamleadRunCanBeConfirmed(current.Status) {
		return TeamleadRun{}, ErrInvalidState
	}
	if (current.MismatchCount > 0 || current.ConflictCount > 0 || current.BlockedCount > 0) && comment == "" {
		return TeamleadRun{}, ErrInvalidInput
	}

	var commentPtr *string
	if comment != "" {
		commentPtr = &comment
	}
	run, err := s.store.QueueTeamleadReconciliationApply(ctx, QueueTeamleadApplyRecord{
		RunID:   params.RunID,
		TeamID:  params.TeamID,
		ActorID: params.ActorID,
		Comment: commentPtr,
	})
	if err != nil {
		return TeamleadRun{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionReconciliationConfirmed,
		EntityType: "teamlead_reconciliation",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicTeamleadRunFromDomain(run),
		Comment:    commentPtr,
	}); err != nil {
		return TeamleadRun{}, err
	}

	if s.scheduler != nil {
		s.scheduler.EnqueueTeamleadApply(params.TeamID, run.ID)
	}

	return run, nil
}

func (s *Service) RejectTeamleadReconciliation(ctx context.Context, params RejectTeamleadReconciliationParams) (TeamleadRun, error) {
	comment := strings.TrimSpace(params.Comment)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.RunID <= 0 || comment == "" {
		return TeamleadRun{}, ErrInvalidInput
	}
	if s.store == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}

	current, err := s.store.GetTeamleadReconciliation(ctx, params.TeamID, params.RunID)
	if err != nil {
		return TeamleadRun{}, err
	}
	if !teamleadRunCanBeRejected(current.Status) {
		return TeamleadRun{}, ErrInvalidState
	}

	run, err := s.store.RejectTeamleadReconciliation(ctx, RejectTeamleadReconciliationRecord{
		RunID:   params.RunID,
		TeamID:  params.TeamID,
		ActorID: params.ActorID,
		Comment: comment,
	})
	if err != nil {
		return TeamleadRun{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionReconciliationRejected,
		EntityType: "teamlead_reconciliation",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicTeamleadRunFromDomain(run),
		Comment:    &comment,
	}); err != nil {
		return TeamleadRun{}, err
	}

	return run, nil
}

func (s *Service) ApplyQueuedTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error) {
	if teamID <= 0 || runID <= 0 {
		return TeamleadRun{}, ErrInvalidInput
	}
	if s.store == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}

	run, err := s.store.ApplyQueuedTeamleadReconciliation(ctx, teamID, runID)
	if err != nil {
		return TeamleadRun{}, err
	}

	action := audit.ActionReconciliationApplied
	if run.Status == TeamleadRunStatusApplyFailed {
		action = audit.ActionReconciliationApplied
	}
	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     teamID,
		Action:     action,
		EntityType: "teamlead_reconciliation",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicTeamleadRunFromDomain(run),
	}); err != nil {
		return TeamleadRun{}, err
	}

	return run, nil
}

func (s *Service) ListTeamleadReconciliations(ctx context.Context, params ListTeamleadReconciliationsParams) (pagination.Result[TeamleadRun], error) {
	if params.TeamID <= 0 {
		return pagination.Result[TeamleadRun]{}, ErrInvalidInput
	}
	if s.store == nil {
		return pagination.Result[TeamleadRun]{}, ErrRepositoryNotConfigured
	}
	return s.store.ListTeamleadReconciliations(ctx, params.TeamID, params.Page)
}

func (s *Service) GetTeamleadReconciliation(ctx context.Context, params GetTeamleadReconciliationParams) (TeamleadRun, error) {
	if params.TeamID <= 0 || params.RunID <= 0 {
		return TeamleadRun{}, ErrInvalidInput
	}
	if s.store == nil {
		return TeamleadRun{}, ErrRepositoryNotConfigured
	}
	return s.store.GetTeamleadReconciliation(ctx, params.TeamID, params.RunID)
}

func (s *Service) ListTeamleadReconciliationItems(ctx context.Context, params GetTeamleadReconciliationParams, filters TeamleadItemFilters, page pagination.Params) (pagination.Result[TeamleadItem], error) {
	if params.TeamID <= 0 || params.RunID <= 0 {
		return pagination.Result[TeamleadItem]{}, ErrInvalidInput
	}
	if s.store == nil {
		return pagination.Result[TeamleadItem]{}, ErrRepositoryNotConfigured
	}
	return s.store.ListTeamleadReconciliationItems(ctx, params.TeamID, params.RunID, filters, page)
}

func (s *Service) EnqueueTeamleadApply(teamID int64, runID int64) {
	go func() {
		_, _ = s.ApplyQueuedTeamleadReconciliation(context.Background(), teamID, runID)
	}()
}

func (s *Service) analyzeTeamleadDirection(ctx context.Context, teamID int64, direction string, input TeamleadCSVInput, dateFrom time.Time, dateTo time.Time, lookups teamleadAnalysisLookups) (TeamleadDirectionAnalysisRecord, []TeamleadItemRecord, error) {
	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" || len(input.Payload) == 0 {
		return TeamleadDirectionAnalysisRecord{}, nil, ErrInvalidInput
	}

	parse, err := imports.ParseCSV(bytes.NewReader(input.Payload), imports.ParseOptions{ColumnSet: imports.ColumnSetTeamlead})
	if err != nil {
		return TeamleadDirectionAnalysisRecord{}, nil, &TeamleadCSVValidationError{Direction: direction, Parse: parse, Err: err}
	}

	filtered := filterRowsByPeriod(parse.Rows, dateFrom, dateTo)
	hash := sha256.Sum256(input.Payload)
	summary := summarizeTLRows(direction, parse.Rows, filtered)
	items := make([]TeamleadItemRecord, 0)

	for _, row := range filtered {
		traderID := lookups.matchTrader(row.WorkerName)
		if traderID == nil {
			items = append(items, teamleadItem(direction, TeamleadItemStageMatching, "unmatched_trader", TeamleadItemSeverityBlocker, nil, &row.ExternalInnerID, nil, nil, map[string]any{
				"workerName": row.WorkerName,
				"rowNumber":  row.RowNumber,
			}, "CSV workerName is not mapped to an active trader", true))
		}

		requisiteMatch := lookups.matchRequisite(row)
		if requisiteMatch.issueType != "" {
			items = append(items, teamleadItem(direction, TeamleadItemStageMatching, requisiteMatch.issueType, requisiteMatch.severity, nil, &row.ExternalInnerID, nil, requisiteMatch.requisiteID, map[string]any{
				"methodName":     optionalStringValue(row.MethodName),
				"requisite":      optionalStringValue(row.RequisiteRaw),
				"requisitePhone": optionalStringValue(row.RequisitePhone),
				"rowNumber":      row.RowNumber,
			}, requisiteMatch.message, requisiteMatch.blocking))
		}
	}

	turnover, err := s.teamleadTurnoverSnapshot(ctx, teamID, direction, dateFrom, dateTo)
	if err != nil {
		return TeamleadDirectionAnalysisRecord{}, nil, err
	}
	summary.CRMAmountMinor = turnover.AmountMinor
	summary.CRMCount = turnover.Count
	summary.DiffAmountMinor = turnover.AmountMinor - summary.SuccessAmountMinor
	if summary.DiffAmountMinor != 0 {
		items = append(items, teamleadItem(direction, TeamleadItemStageTurnoverCheck, "turnover_mismatch", TeamleadItemSeverityWarning, nil, nil, nil, nil, map[string]any{
			"tlSuccessAmountMinor": summary.SuccessAmountMinor,
			"crmAmountMinor":       turnover.AmountMinor,
			"diffAmountMinor":      summary.DiffAmountMinor,
		}, "TL CSV success total differs from CRM turnover total", false))
	}

	transactionItems, createCount, updateCount, unchangedCount, err := s.teamleadTransactionDiff(ctx, teamID, direction, filtered, dateFrom, dateTo, lookups)
	if err != nil {
		return TeamleadDirectionAnalysisRecord{}, nil, err
	}
	items = append(items, transactionItems...)
	summary.CreateCount = createCount
	summary.UpdateCount = updateCount
	summary.UnchangedCount = unchangedCount
	for _, item := range items {
		if item.Direction == direction && item.IsBlocking {
			summary.BlockedCount++
		}
	}
	items = append(items, teamleadItem(direction, TeamleadItemStagePreview, "preview_summary", TeamleadItemSeverityInfo, nil, nil, nil, nil, map[string]any{
		"createCount":    summary.CreateCount,
		"updateCount":    summary.UpdateCount,
		"unchangedCount": summary.UnchangedCount,
		"blockedCount":   summary.BlockedCount,
	}, "Preview changes calculated", false))

	return TeamleadDirectionAnalysisRecord{
		Direction: direction,
		FileName:  fileName,
		FileHash:  hex.EncodeToString(hash[:]),
		Rows:      parse.Rows,
		Filtered:  filtered,
		Summary:   summary,
	}, items, nil
}

func (s *Service) teamleadTurnoverSnapshot(ctx context.Context, teamID int64, direction string, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error) {
	switch direction {
	case imports.DirectionInbound:
		return s.store.CalculateTeamleadV2InboundTurnover(ctx, teamID, dateFrom, dateTo)
	case imports.DirectionOutbound:
		return s.store.CalculateTeamleadV2OutboundTransfers(ctx, teamID, dateFrom, dateTo)
	default:
		return TeamleadTurnoverSnapshot{}, ErrInvalidInput
	}
}

func (s *Service) teamleadTransactionDiff(ctx context.Context, teamID int64, direction string, rows []imports.ParsedOrderRow, dateFrom time.Time, dateTo time.Time, lookups teamleadAnalysisLookups) ([]TeamleadItemRecord, int64, int64, int64, error) {
	innerIDs := make([]string, 0, len(rows))
	seenInnerID := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := seenInnerID[row.ExternalInnerID]; exists {
			continue
		}
		seenInnerID[row.ExternalInnerID] = struct{}{}
		innerIDs = append(innerIDs, row.ExternalInnerID)
	}

	existingByInnerID := map[string]TeamleadExternalOrderSnapshot{}
	if len(innerIDs) > 0 {
		existing, err := s.store.ListTeamleadReconciliationExternalOrders(ctx, teamID, direction, innerIDs)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		for _, item := range existing {
			existingByInnerID[item.ExternalInnerID] = item
		}
	}
	periodExisting, err := s.store.ListTeamleadReconciliationExternalOrdersInPeriod(ctx, teamID, direction, dateFrom, dateTo)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	items := make([]TeamleadItemRecord, 0)
	createCount := int64(0)
	updateCount := int64(0)
	unchangedCount := int64(0)
	for _, row := range rows {
		traderID := lookups.matchTrader(row.WorkerName)
		requisiteID := lookups.matchRequisite(row).requisiteID
		after := teamleadOrderJSON(row, traderID, requisiteID)
		existing, exists := existingByInnerID[row.ExternalInnerID]
		if !exists {
			createCount++
			items = append(items, teamleadItem(direction, TeamleadItemStageTransactionCheck, "missing_in_crm", TeamleadItemSeverityWarning, nil, &row.ExternalInnerID, traderID, requisiteID, after, "TL CSV order is missing in CRM transactions", false))
			continue
		}

		changed := false
		before := externalOrderJSON(existing)
		if existing.AmountMinor != row.AmountMinor {
			changed = true
			items = append(items, teamleadChangeItem(direction, "amount_changed", existing, row, traderID, requisiteID, before, after))
		}
		if existing.NormalizedStatus != row.NormalizedStatus {
			changed = true
			items = append(items, teamleadChangeItem(direction, "status_changed", existing, row, traderID, requisiteID, before, after))
		}
		if traderID != nil && (existing.TraderID == nil || *existing.TraderID != *traderID) {
			changed = true
			items = append(items, teamleadChangeItem(direction, "trader_changed", existing, row, traderID, requisiteID, before, after))
		}
		if requisiteID != nil && (existing.RequisiteID == nil || *existing.RequisiteID != *requisiteID) {
			changed = true
			items = append(items, teamleadChangeItem(direction, "requisite_changed", existing, row, traderID, requisiteID, before, after))
		}
		if !sameExternalDate(existing.CreatedAtExternal, row.CreatedAtExternal) {
			changed = true
			items = append(items, teamleadChangeItem(direction, "date_changed", existing, row, traderID, requisiteID, before, after))
		}
		if changed {
			updateCount++
		} else {
			unchangedCount++
		}
	}

	for _, existing := range periodExisting {
		if _, ok := seenInnerID[existing.ExternalInnerID]; ok {
			continue
		}
		innerID := existing.ExternalInnerID
		items = append(items, TeamleadItemRecord{
			Direction:       direction,
			Stage:           TeamleadItemStageTransactionCheck,
			IssueType:       "missing_in_tl",
			Severity:        TeamleadItemSeverityWarning,
			ExternalOrderID: &existing.ID,
			ExternalInnerID: &innerID,
			TraderID:        existing.TraderID,
			RequisiteID:     existing.RequisiteID,
			BeforeJSON:      mustRawJSON(externalOrderJSON(existing)),
			Message:         rawStringPtr("CRM transaction is missing in TL CSV for selected period"),
		})
	}

	return items, createCount, updateCount, unchangedCount, nil
}

type teamleadAnalysisLookups struct {
	traderByWorker map[string]int64
	requisites     []TeamleadRequisiteMatch
}

type requisiteMatchResult struct {
	requisiteID *int64
	issueType   string
	severity    string
	message     string
	blocking    bool
}

func newTeamleadAnalysisLookups(traders []TeamleadTraderMatch, requisites []TeamleadRequisiteMatch) teamleadAnalysisLookups {
	traderByWorker := make(map[string]int64, len(traders))
	for _, trader := range traders {
		traderByWorker[strings.ToLower(strings.TrimSpace(trader.ExternalWorkerName))] = trader.TraderID
	}
	return teamleadAnalysisLookups{traderByWorker: traderByWorker, requisites: requisites}
}

func (l teamleadAnalysisLookups) matchTrader(workerName string) *int64 {
	traderID, ok := l.traderByWorker[strings.ToLower(strings.TrimSpace(workerName))]
	if !ok {
		return nil
	}
	return &traderID
}

func (l teamleadAnalysisLookups) matchRequisite(row imports.ParsedOrderRow) requisiteMatchResult {
	bankCode := normalizeBankCodeFromCSV(row.MethodName)
	if bankCode == "" {
		return requisiteMatchResult{issueType: "unmatched_requisite", severity: TeamleadItemSeverityBlocker, message: "CSV bank cannot be mapped to bank_code", blocking: true}
	}
	phone := normalizePhoneKey(firstPresentString(row.RequisitePhone, row.RequisiteRaw))
	card := normalizeCardKey(row.RequisiteRaw)

	if phone != "" && card != "" {
		for _, requisite := range l.requisites {
			if requisite.BankCode == bankCode && requisite.NormalizedPhone == phone && requisite.NormalizedCardNumber == card {
				id := requisite.ID
				return requisiteMatchResult{requisiteID: &id}
			}
		}
		for _, requisite := range l.requisites {
			if requisite.NormalizedCardNumber == card && (requisite.BankCode != bankCode || requisite.NormalizedPhone != phone) {
				id := requisite.ID
				return requisiteMatchResult{requisiteID: &id, issueType: "conflict_requisite", severity: TeamleadItemSeverityBlocker, message: "CSV card points to another bank or phone", blocking: true}
			}
		}
		return requisiteMatchResult{issueType: "unmatched_requisite", severity: TeamleadItemSeverityBlocker, message: "CSV bank, phone and card do not match any requisite", blocking: true}
	}

	if card != "" {
		var matches []TeamleadRequisiteMatch
		for _, requisite := range l.requisites {
			if requisite.BankCode == bankCode && requisite.NormalizedCardNumber == card {
				matches = append(matches, requisite)
			}
		}
		if len(matches) == 1 {
			id := matches[0].ID
			return requisiteMatchResult{requisiteID: &id}
		}
		if len(matches) > 1 {
			return requisiteMatchResult{issueType: "ambiguous_requisite", severity: TeamleadItemSeverityBlocker, message: "CSV card matches several requisites", blocking: true}
		}
	}

	if phone != "" {
		var matches []TeamleadRequisiteMatch
		for _, requisite := range l.requisites {
			if requisite.BankCode == bankCode && requisite.NormalizedPhone == phone {
				matches = append(matches, requisite)
			}
		}
		if len(matches) == 1 {
			id := matches[0].ID
			return requisiteMatchResult{requisiteID: &id}
		}
		if len(matches) > 1 {
			return requisiteMatchResult{issueType: "ambiguous_requisite", severity: TeamleadItemSeverityBlocker, message: "CSV phone matches several requisite cards", blocking: true}
		}
	}

	return requisiteMatchResult{issueType: "unmatched_requisite", severity: TeamleadItemSeverityBlocker, message: "CSV requisite does not match CRM requisite identity", blocking: true}
}

func filterRowsByPeriod(rows []imports.ParsedOrderRow, dateFrom time.Time, dateTo time.Time) []imports.ParsedOrderRow {
	filtered := make([]imports.ParsedOrderRow, 0, len(rows))
	fromKey := dateKey(dateFrom)
	toKey := dateKey(dateTo)
	for _, row := range rows {
		rowKey := dateKey(row.CreatedAtExternal)
		if rowKey < fromKey || rowKey > toKey {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func dateKey(value time.Time) int {
	year, month, day := value.Date()
	return year*10000 + int(month)*100 + day
}

func summarizeTLRows(direction string, rows []imports.ParsedOrderRow, filtered []imports.ParsedOrderRow) TeamleadDirectionSummary {
	summary := TeamleadDirectionSummary{
		Direction:    direction,
		RowsTotal:    int64(len(rows)),
		RowsInPeriod: int64(len(filtered)),
	}
	for _, row := range filtered {
		summary.TotalCount++
		summary.TotalAmountMinor += row.AmountMinor
		switch row.NormalizedStatus {
		case imports.NormalizedStatusSuccess, imports.NormalizedStatusCorrected:
			summary.SuccessCount++
			summary.SuccessAmountMinor += row.AmountMinor
		case imports.NormalizedStatusFailed, imports.NormalizedStatusCancelled:
			summary.FailedCount++
			summary.FailedAmountMinor += row.AmountMinor
		}
	}
	return summary
}

func teamleadRunStatusFromItems(items []TeamleadItemRecord) (string, int64, int64, int64) {
	mismatchCount := int64(0)
	conflictCount := int64(0)
	blockedCount := int64(0)
	for _, item := range items {
		if item.Severity == TeamleadItemSeverityWarning || item.Severity == TeamleadItemSeverityError || item.Severity == TeamleadItemSeverityBlocker {
			mismatchCount++
		}
		if strings.Contains(item.IssueType, "conflict") || item.Severity == TeamleadItemSeverityError {
			conflictCount++
		}
		if item.IsBlocking || item.Severity == TeamleadItemSeverityBlocker {
			blockedCount++
		}
	}
	if mismatchCount > 0 {
		return TeamleadRunStatusMismatch, mismatchCount, conflictCount, blockedCount
	}
	return TeamleadRunStatusMatched, 0, 0, 0
}

func teamleadRunCanBeConfirmed(status string) bool {
	switch status {
	case TeamleadRunStatusMatched, TeamleadRunStatusMismatch, TeamleadRunStatusApplyFailed:
		return true
	default:
		return false
	}
}

func teamleadRunCanBeRejected(status string) bool {
	switch status {
	case TeamleadRunStatusMatched, TeamleadRunStatusMismatch, TeamleadRunStatusApplyFailed:
		return true
	default:
		return false
	}
}

func teamleadPipelineSummary(directions []TeamleadDirectionAnalysisRecord, items []TeamleadItemRecord) []map[string]any {
	stages := []string{
		TeamleadItemStageNormalization,
		TeamleadItemStageMatching,
		TeamleadItemStageTurnoverCheck,
		TeamleadItemStageTransactionCheck,
		TeamleadItemStagePreview,
	}
	result := make([]map[string]any, 0, len(stages))
	for _, stage := range stages {
		stageIssues := int64(0)
		for _, item := range items {
			if item.Stage == stage && item.Severity != TeamleadItemSeverityInfo {
				stageIssues++
			}
		}
		result = append(result, map[string]any{
			"stage":           stage,
			"status":          map[bool]string{true: "mismatch", false: "matched"}[stageIssues > 0],
			"issuesCount":     stageIssues,
			"directionsCount": len(directions),
		})
	}
	return result
}

func teamleadSummaryJSON(directions []TeamleadDirectionAnalysisRecord) (map[string]any, map[string]any, map[string]any) {
	inbound := map[string]any{}
	outbound := map[string]any{}
	preview := map[string]any{}
	for _, direction := range directions {
		summary := direction.Summary
		value := map[string]any{
			"rowsTotal":          summary.RowsTotal,
			"rowsInPeriod":       summary.RowsInPeriod,
			"successAmountMinor": summary.SuccessAmountMinor,
			"successCount":       summary.SuccessCount,
			"failedAmountMinor":  summary.FailedAmountMinor,
			"failedCount":        summary.FailedCount,
			"totalAmountMinor":   summary.TotalAmountMinor,
			"totalCount":         summary.TotalCount,
			"crmAmountMinor":     summary.CRMAmountMinor,
			"crmCount":           summary.CRMCount,
			"diffAmountMinor":    summary.DiffAmountMinor,
		}
		preview[direction.Direction] = map[string]any{
			"createCount":    summary.CreateCount,
			"updateCount":    summary.UpdateCount,
			"unchangedCount": summary.UnchangedCount,
			"blockedCount":   summary.BlockedCount,
		}
		if direction.Direction == imports.DirectionInbound {
			inbound = value
		} else {
			outbound = value
		}
	}
	return inbound, outbound, preview
}

func teamleadItem(direction string, stage string, issueType string, severity string, externalOrderID *int64, externalInnerID *string, traderID *int64, requisiteID *int64, after any, message string, blocking bool) TeamleadItemRecord {
	return TeamleadItemRecord{
		Direction:       direction,
		Stage:           stage,
		IssueType:       issueType,
		Severity:        severity,
		ExternalOrderID: externalOrderID,
		ExternalInnerID: externalInnerID,
		TraderID:        traderID,
		RequisiteID:     requisiteID,
		AfterJSON:       mustRawJSON(after),
		Message:         rawStringPtr(message),
		IsBlocking:      blocking,
	}
}

func teamleadChangeItem(direction string, issueType string, existing TeamleadExternalOrderSnapshot, row imports.ParsedOrderRow, traderID *int64, requisiteID *int64, before any, after any) TeamleadItemRecord {
	innerID := row.ExternalInnerID
	return TeamleadItemRecord{
		Direction:       direction,
		Stage:           TeamleadItemStageTransactionCheck,
		IssueType:       issueType,
		Severity:        TeamleadItemSeverityWarning,
		ExternalOrderID: &existing.ID,
		ExternalInnerID: &innerID,
		TraderID:        traderID,
		RequisiteID:     requisiteID,
		BeforeJSON:      mustRawJSON(before),
		AfterJSON:       mustRawJSON(after),
		Message:         rawStringPtr("CRM transaction differs from TL CSV"),
	}
}

func teamleadOrderJSON(row imports.ParsedOrderRow, traderID *int64, requisiteID *int64) map[string]any {
	return map[string]any{
		"externalInnerId":   row.ExternalInnerID,
		"workerName":        row.WorkerName,
		"traderId":          traderID,
		"requisiteId":       requisiteID,
		"amountMinor":       row.AmountMinor,
		"rawStatus":         row.RawStatus,
		"normalizedStatus":  row.NormalizedStatus,
		"createdAtExternal": row.CreatedAtExternal,
	}
}

func externalOrderJSON(order TeamleadExternalOrderSnapshot) map[string]any {
	return map[string]any{
		"externalOrderId":   order.ID,
		"externalInnerId":   order.ExternalInnerID,
		"workerName":        order.WorkerName,
		"traderId":          order.TraderID,
		"requisiteId":       order.RequisiteID,
		"amountMinor":       order.AmountMinor,
		"rawStatus":         order.RawStatus,
		"normalizedStatus":  order.NormalizedStatus,
		"createdAtExternal": order.CreatedAtExternal,
	}
}

func sameExternalDate(left time.Time, right time.Time) bool {
	return left.Equal(right)
}

func normalizeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func marshalJSON(value any) (json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func mustRawJSON(value any) json.RawMessage {
	payload, err := marshalJSON(value)
	if err != nil {
		return nil
	}
	return payload
}

func firstPresentString(values ...*string) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		if trimmed := strings.TrimSpace(*value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizePhoneKey(value string) string {
	digits := digitsOnly(value)
	if len(digits) == 10 {
		return "7" + digits
	}
	if len(digits) == 11 && strings.HasPrefix(digits, "8") {
		return "7" + digits[1:]
	}
	if len(digits) == 11 && strings.HasPrefix(digits, "7") {
		return digits
	}
	return ""
}

func normalizeCardKey(value *string) string {
	if value == nil {
		return ""
	}
	digits := digitsOnly(*value)
	if len(digits) < 8 {
		return ""
	}
	return digits
}

func normalizeBankCodeFromCSV(value *string) string {
	if value == nil {
		return ""
	}
	normalized := strings.ToLower(strings.TrimSpace(*value))
	normalized = strings.ReplaceAll(normalized, "ё", "е")
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", "банк", "")
	normalized = replacer.Replace(normalized)
	switch {
	case strings.Contains(normalized, "сбер"):
		return "sber"
	case strings.Contains(normalized, "тинь") || strings.Contains(normalized, "тбанк") || strings.Contains(normalized, "tbank"):
		return "tbank"
	case strings.Contains(normalized, "альфа") || strings.Contains(normalized, "alfa"):
		return "alfa"
	case strings.Contains(normalized, "втб") || strings.Contains(normalized, "vtb"):
		return "vtb"
	case strings.Contains(normalized, "райф") || strings.Contains(normalized, "raif"):
		return "raif"
	case strings.Contains(normalized, "ozon") || strings.Contains(normalized, "озон"):
		return "ozon"
	case strings.Contains(normalized, "газпром"):
		return "gazprombank"
	case strings.Contains(normalized, "мкб"):
		return "mkb"
	case strings.Contains(normalized, "россельхоз") || strings.Contains(normalized, "рсхб"):
		return "rshb"
	default:
		return ""
	}
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if unicode.IsDigit(char) {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func rawStringPtr(value string) *string {
	return &value
}

func (s *Service) RecalculateTraderInbound(ctx context.Context, params RecalculateTraderInboundParams) (Run, error) {
	if params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.RecalculateTraderInbound(ctx, RecalculateTraderInboundRecord{
		TeamID:        params.TeamID,
		TraderID:      params.TraderID,
		ShiftID:       params.ShiftID,
		ImportBatchID: params.ImportBatchID,
	})
}

func (s *Service) RecalculateTraderOutbound(ctx context.Context, params RecalculateTraderOutboundParams) (Run, error) {
	if params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.RecalculateTraderOutbound(ctx, RecalculateTraderOutboundRecord{
		TeamID:        params.TeamID,
		TraderID:      params.TraderID,
		ShiftID:       params.ShiftID,
		ImportBatchID: params.ImportBatchID,
	})
}

func (s *Service) RecalculateTeamleadPeriodInbound(ctx context.Context, params RecalculateTeamleadPeriodInboundParams) (Run, error) {
	if params.TeamID <= 0 || params.AccountingPeriodID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.RecalculateTeamleadPeriodInbound(ctx, RecalculateTeamleadPeriodInboundRecord{
		TeamID:             params.TeamID,
		AccountingPeriodID: params.AccountingPeriodID,
		ImportBatchID:      params.ImportBatchID,
	})
}

func (s *Service) RecalculateTeamleadPeriodOutbound(ctx context.Context, params RecalculateTeamleadPeriodOutboundParams) (Run, error) {
	if params.TeamID <= 0 || params.AccountingPeriodID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.RecalculateTeamleadPeriodOutbound(ctx, RecalculateTeamleadPeriodOutboundRecord{
		TeamID:             params.TeamID,
		AccountingPeriodID: params.AccountingPeriodID,
		ImportBatchID:      params.ImportBatchID,
	})
}

func (s *Service) RecalculateTeamleadCurrent(ctx context.Context, params RecalculateTeamleadCurrentParams) (Run, error) {
	if params.TeamID <= 0 || params.ActorID <= 0 || !isSupportedDirection(params.Direction) {
		return Run{}, ErrInvalidInput
	}

	return s.store.RecalculateTeamleadCurrent(ctx, RecalculateTeamleadCurrentRecord{
		TeamID:    params.TeamID,
		ActorID:   params.ActorID,
		Direction: params.Direction,
	})
}

func (s *Service) LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTraderInbound(ctx, teamID, traderID, shiftID)
}

func (s *Service) LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error) {
	if teamID <= 0 || shiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTraderInboundByShift(ctx, teamID, shiftID)
}

func (s *Service) LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTraderOutbound(ctx, teamID, traderID, shiftID)
}

func (s *Service) LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error) {
	if teamID <= 0 || shiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTraderOutboundByShift(ctx, teamID, shiftID)
}

func (s *Service) GetTraderInboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error) {
	if teamID <= 0 || traderID <= 0 || runID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.GetTraderInboundRun(ctx, teamID, traderID, runID)
}

func (s *Service) GetTraderOutboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error) {
	if teamID <= 0 || traderID <= 0 || runID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.GetTraderOutboundRun(ctx, teamID, traderID, runID)
}

func (s *Service) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	if teamID <= 0 || accountingPeriodID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTeamleadPeriodInbound(ctx, teamID, accountingPeriodID)
}

func (s *Service) LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	if teamID <= 0 || accountingPeriodID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTeamleadPeriodOutbound(ctx, teamID, accountingPeriodID)
}

func (s *Service) LatestTeamleadInbound(ctx context.Context, teamID int64, actorID int64) (Run, error) {
	if teamID <= 0 || actorID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTeamleadInbound(ctx, teamID, actorID)
}

func (s *Service) LatestTeamleadOutbound(ctx context.Context, teamID int64, actorID int64) (Run, error) {
	if teamID <= 0 || actorID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTeamleadOutbound(ctx, teamID, actorID)
}

func (s *Service) ListTeamleadCurrentRuns(ctx context.Context, params ListTeamleadCurrentRunsParams) (pagination.Result[Run], error) {
	if params.TeamID <= 0 || params.ActorID <= 0 || !isSupportedDirection(params.Direction) {
		return pagination.Result[Run]{}, ErrInvalidInput
	}

	return s.store.ListTeamleadCurrentRuns(ctx, params.TeamID, params.ActorID, params.Direction, params.Page)
}

func (s *Service) GetTeamleadCurrentRun(ctx context.Context, params GetTeamleadCurrentRunParams) (Run, error) {
	if params.TeamID <= 0 || params.ActorID <= 0 || params.RunID <= 0 || !isSupportedDirection(params.Direction) {
		return Run{}, ErrInvalidInput
	}

	return s.store.GetTeamleadCurrentRun(ctx, params.TeamID, params.ActorID, params.Direction, params.RunID)
}

func (s *Service) ListTeamleadCurrentItems(ctx context.Context, params GetTeamleadCurrentRunParams, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.GetTeamleadCurrentRun(ctx, params)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTeamleadInboundItems(ctx context.Context, teamID int64, actorID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTeamleadInbound(ctx, teamID, actorID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTeamleadOutboundItems(ctx context.Context, teamID int64, actorID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTeamleadOutbound(ctx, teamID, actorID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTeamleadPeriodInbound(ctx, teamID, accountingPeriodID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTeamleadPeriodOutboundItems(ctx context.Context, teamID int64, accountingPeriodID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTeamleadPeriodOutbound(ctx, teamID, accountingPeriodID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTraderInboundItems(ctx context.Context, teamID int64, traderID int64, shiftID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTraderInbound(ctx, teamID, traderID, shiftID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTraderInboundItemsByShift(ctx context.Context, teamID int64, shiftID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTraderInboundByShift(ctx, teamID, shiftID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTraderOutboundItems(ctx context.Context, teamID int64, traderID int64, shiftID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTraderOutbound(ctx, teamID, traderID, shiftID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTraderOutboundItemsByShift(ctx context.Context, teamID int64, shiftID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.LatestTraderOutboundByShift(ctx, teamID, shiftID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTraderInboundRunItems(ctx context.Context, teamID int64, traderID int64, runID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.GetTraderInboundRun(ctx, teamID, traderID, runID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) ListTraderOutboundRunItems(ctx context.Context, teamID int64, traderID int64, runID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	run, err := s.GetTraderOutboundRun(ctx, teamID, traderID, runID)
	if err != nil {
		return pagination.Result[Item]{}, err
	}

	return s.store.ListItems(ctx, run.ID, normalizeItemFilters(filters), page)
}

func (s *Service) AcceptTraderInbound(ctx context.Context, params AcceptTraderInboundParams) (Run, error) {
	comment := strings.TrimSpace(params.Comment)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.RunID <= 0 || comment == "" {
		return Run{}, ErrInvalidInput
	}

	run, err := s.store.AcceptTraderInbound(ctx, AcceptTraderInboundRecord{
		RunID:    params.RunID,
		TeamID:   params.TeamID,
		TraderID: params.TraderID,
		ActorID:  params.ActorID,
		Comment:  comment,
	})
	if err != nil {
		return Run{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionReconciliationAcceptedWithComment,
		EntityType: "reconciliation_run",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicRunFromDomain(run),
		Comment:    &comment,
	}); err != nil {
		return Run{}, err
	}

	return run, nil
}

func (s *Service) AcceptTraderOutbound(ctx context.Context, params AcceptTraderOutboundParams) (Run, error) {
	comment := strings.TrimSpace(params.Comment)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.RunID <= 0 || comment == "" {
		return Run{}, ErrInvalidInput
	}

	run, err := s.store.AcceptTraderOutbound(ctx, AcceptTraderOutboundRecord{
		RunID:    params.RunID,
		TeamID:   params.TeamID,
		TraderID: params.TraderID,
		ActorID:  params.ActorID,
		Comment:  comment,
	})
	if err != nil {
		return Run{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionReconciliationAcceptedWithComment,
		EntityType: "reconciliation_run",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicRunFromDomain(run),
		Comment:    &comment,
	}); err != nil {
		return Run{}, err
	}

	return run, nil
}

func (s *Service) AcceptTeamleadCurrent(ctx context.Context, params AcceptTeamleadCurrentParams) (Run, error) {
	comment := strings.TrimSpace(params.Comment)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.RunID <= 0 || !isSupportedDirection(params.Direction) || comment == "" {
		return Run{}, ErrInvalidInput
	}

	run, err := s.store.AcceptTeamleadCurrent(ctx, AcceptTeamleadCurrentRecord{
		RunID:   params.RunID,
		TeamID:  params.TeamID,
		ActorID: params.ActorID,
		Comment: comment,
	})
	if err != nil {
		return Run{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionReconciliationAcceptedWithComment,
		EntityType: "reconciliation_run",
		EntityID:   strconv.FormatInt(run.ID, 10),
		After:      PublicRunFromDomain(run),
		Comment:    &comment,
	}); err != nil {
		return Run{}, err
	}

	return run, nil
}

func (s *Service) AfterImportApplied(ctx context.Context, result imports.ApplyResult) error {
	switch result.Batch.ScopeType {
	case imports.ScopeTypeTraderShift:
		return s.afterTraderShiftImportApplied(ctx, result)
	case imports.ScopeTypeTeamleadPeriod:
		return s.afterTeamleadPeriodImportApplied(ctx, result)
	default:
		return nil
	}
}

func (s *Service) afterTraderShiftImportApplied(ctx context.Context, result imports.ApplyResult) error {
	if result.Batch.ShiftID == nil || result.Batch.TraderID == nil {
		return nil
	}

	switch result.Batch.Direction {
	case imports.DirectionInbound:
		_, err := s.RecalculateTraderInbound(ctx, RecalculateTraderInboundParams{
			TeamID:        result.Batch.TeamID,
			TraderID:      *result.Batch.TraderID,
			ShiftID:       *result.Batch.ShiftID,
			ImportBatchID: &result.Batch.ID,
		})
		if err != nil {
			return err
		}

		return s.recalculateActiveTeamleadInboundPeriods(ctx, result.Batch.TeamID)
	case imports.DirectionOutbound:
		_, err := s.RecalculateTraderOutbound(ctx, RecalculateTraderOutboundParams{
			TeamID:        result.Batch.TeamID,
			TraderID:      *result.Batch.TraderID,
			ShiftID:       *result.Batch.ShiftID,
			ImportBatchID: &result.Batch.ID,
		})
		if err != nil {
			return err
		}

		return s.recalculateActiveTeamleadOutboundPeriods(ctx, result.Batch.TeamID)
	default:
		return nil
	}
}

func (s *Service) afterTeamleadPeriodImportApplied(ctx context.Context, result imports.ApplyResult) error {
	if result.Batch.AccountingPeriodID == nil {
		return nil
	}

	switch result.Batch.Direction {
	case imports.DirectionInbound:
		_, err := s.RecalculateTeamleadPeriodInbound(ctx, RecalculateTeamleadPeriodInboundParams{
			TeamID:             result.Batch.TeamID,
			AccountingPeriodID: *result.Batch.AccountingPeriodID,
			ImportBatchID:      &result.Batch.ID,
		})
		return err
	case imports.DirectionOutbound:
		_, err := s.RecalculateTeamleadPeriodOutbound(ctx, RecalculateTeamleadPeriodOutboundParams{
			TeamID:             result.Batch.TeamID,
			AccountingPeriodID: *result.Batch.AccountingPeriodID,
			ImportBatchID:      &result.Batch.ID,
		})
		return err
	default:
		return nil
	}
}

func isSupportedDirection(direction string) bool {
	return direction == imports.DirectionInbound || direction == imports.DirectionOutbound
}

func (s *Service) recalculateActiveTeamleadInboundPeriods(ctx context.Context, teamID int64) error {
	scopes, err := s.store.ListActiveTeamleadInboundPeriodScopes(ctx, teamID)
	if err != nil {
		return err
	}

	for _, scope := range scopes {
		importBatchID := scope.ImportBatchID
		if _, err := s.RecalculateTeamleadPeriodInbound(ctx, RecalculateTeamleadPeriodInboundParams{
			TeamID:             teamID,
			AccountingPeriodID: scope.AccountingPeriodID,
			ImportBatchID:      &importBatchID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) recalculateActiveTeamleadOutboundPeriods(ctx context.Context, teamID int64) error {
	scopes, err := s.store.ListActiveTeamleadOutboundPeriodScopes(ctx, teamID)
	if err != nil {
		return err
	}

	for _, scope := range scopes {
		importBatchID := scope.ImportBatchID
		if _, err := s.RecalculateTeamleadPeriodOutbound(ctx, RecalculateTeamleadPeriodOutboundParams{
			TeamID:             teamID,
			AccountingPeriodID: scope.AccountingPeriodID,
			ImportBatchID:      &importBatchID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) AfterManualPayoutChanged(ctx context.Context, teamID int64, traderID int64, shiftID int64) error {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return ErrInvalidInput
	}

	latest, err := s.LatestTraderOutbound(ctx, teamID, traderID, shiftID)
	if errors.Is(err, ErrRunNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = s.RecalculateTraderOutbound(ctx, RecalculateTraderOutboundParams{
		TeamID:        teamID,
		TraderID:      traderID,
		ShiftID:       shiftID,
		ImportBatchID: latest.ImportBatchID,
	})
	return err
}

func (s *Service) AfterTurnoverCreated(ctx context.Context, entry shifts.TurnoverEntry) error {
	latest, err := s.LatestTraderInbound(ctx, entry.TeamID, entry.TraderID, entry.ShiftID)
	if errors.Is(err, ErrRunNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = s.RecalculateTraderInbound(ctx, RecalculateTraderInboundParams{
		TeamID:        entry.TeamID,
		TraderID:      entry.TraderID,
		ShiftID:       entry.ShiftID,
		ImportBatchID: latest.ImportBatchID,
	})
	return err
}

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}

func normalizeItemFilters(filters ItemFilters) ItemFilters {
	status := strings.TrimSpace(filters.Status)
	return ItemFilters{
		Status:       status,
		OnlyMismatch: filters.OnlyMismatch || status == StatusMismatch,
	}
}
