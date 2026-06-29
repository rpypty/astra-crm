package reconciliation

import (
	"encoding/json"
	"time"
)

type PublicRun struct {
	ID                  int64      `json:"id"`
	TeamID              int64      `json:"teamId"`
	Type                string     `json:"type"`
	ScopeType           string     `json:"scopeType"`
	ShiftID             *int64     `json:"shiftId,omitempty"`
	AccountingPeriodID  *int64     `json:"accountingPeriodId,omitempty"`
	TraderID            *int64     `json:"traderId,omitempty"`
	ImportBatchID       *int64     `json:"importBatchId,omitempty"`
	ExpectedAmountMinor int64      `json:"expectedAmountMinor"`
	ActualAmountMinor   int64      `json:"actualAmountMinor"`
	DiffAmountMinor     int64      `json:"diffAmountMinor"`
	SuccessAmountMinor  int64      `json:"successAmountMinor"`
	SuccessCount        int64      `json:"successCount"`
	FailedAmountMinor   int64      `json:"failedAmountMinor"`
	FailedCount         int64      `json:"failedCount"`
	TotalAmountMinor    int64      `json:"totalAmountMinor"`
	TotalCount          int64      `json:"totalCount"`
	Status              string     `json:"status"`
	Comment             *string    `json:"comment,omitempty"`
	ConfirmedBy         *int64     `json:"confirmedBy,omitempty"`
	ConfirmedAt         *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
}

func PublicRunFromDomain(run Run) PublicRun {
	return PublicRun{
		ID:                  run.ID,
		TeamID:              run.TeamID,
		Type:                run.Type,
		ScopeType:           run.ScopeType,
		ShiftID:             run.ShiftID,
		AccountingPeriodID:  run.AccountingPeriodID,
		TraderID:            run.TraderID,
		ImportBatchID:       run.ImportBatchID,
		ExpectedAmountMinor: run.ExpectedAmountMinor,
		ActualAmountMinor:   run.ActualAmountMinor,
		DiffAmountMinor:     run.DiffAmountMinor,
		SuccessAmountMinor:  run.SuccessAmountMinor,
		SuccessCount:        run.SuccessCount,
		FailedAmountMinor:   run.FailedAmountMinor,
		FailedCount:         run.FailedCount,
		TotalAmountMinor:    run.TotalAmountMinor,
		TotalCount:          run.TotalCount,
		Status:              run.Status,
		Comment:             run.Comment,
		ConfirmedBy:         run.ConfirmedBy,
		ConfirmedAt:         run.ConfirmedAt,
		CreatedAt:           run.CreatedAt,
	}
}

func PublicRunsFromDomain(runs []Run) []PublicRun {
	publicRuns := make([]PublicRun, 0, len(runs))
	for _, run := range runs {
		publicRuns = append(publicRuns, PublicRunFromDomain(run))
	}

	return publicRuns
}

type PublicItem struct {
	ID                  int64           `json:"id"`
	ReconciliationRunID int64           `json:"reconciliationRunId"`
	IssueType           string          `json:"issueType"`
	ExternalOrderID     *int64          `json:"externalOrderId,omitempty"`
	ExternalInnerID     *string         `json:"externalInnerId,omitempty"`
	TeamleadValue       json.RawMessage `json:"teamleadValue,omitempty"`
	TraderValue         json.RawMessage `json:"traderValue,omitempty"`
	Message             *string         `json:"message,omitempty"`
	CreatedAt           time.Time       `json:"createdAt"`
}

func PublicItemFromDomain(item Item) PublicItem {
	return PublicItem{
		ID:                  item.ID,
		ReconciliationRunID: item.ReconciliationRunID,
		IssueType:           item.IssueType,
		ExternalOrderID:     item.ExternalOrderID,
		ExternalInnerID:     item.ExternalInnerID,
		TeamleadValue:       nullableRawJSON(item.TeamleadValueJSON),
		TraderValue:         nullableRawJSON(item.TraderValueJSON),
		Message:             item.Message,
		CreatedAt:           item.CreatedAt,
	}
}

func PublicItemsFromDomain(items []Item) []PublicItem {
	publicItems := make([]PublicItem, 0, len(items))
	for _, item := range items {
		publicItems = append(publicItems, PublicItemFromDomain(item))
	}

	return publicItems
}

type PublicTeamleadRun struct {
	ID                    int64           `json:"id"`
	TeamID                int64           `json:"teamId"`
	DateFrom              string          `json:"dateFrom"`
	DateTo                string          `json:"dateTo"`
	Status                string          `json:"status"`
	CreatedBy             int64           `json:"createdBy"`
	ConfirmedBy           *int64          `json:"confirmedBy,omitempty"`
	RejectedBy            *int64          `json:"rejectedBy,omitempty"`
	InboundImportBatchID  *int64          `json:"inboundImportBatchId,omitempty"`
	OutboundImportBatchID *int64          `json:"outboundImportBatchId,omitempty"`
	Comment               *string         `json:"comment,omitempty"`
	MismatchCount         int64           `json:"mismatchCount"`
	ConflictCount         int64           `json:"conflictCount"`
	BlockedCount          int64           `json:"blockedCount"`
	Pipeline              json.RawMessage `json:"pipeline,omitempty"`
	InboundSummary        json.RawMessage `json:"inboundSummary,omitempty"`
	OutboundSummary       json.RawMessage `json:"outboundSummary,omitempty"`
	Preview               json.RawMessage `json:"preview,omitempty"`
	ApplyResult           json.RawMessage `json:"applyResult,omitempty"`
	ErrorMessage          *string         `json:"errorMessage,omitempty"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
	AnalyzedAt            *time.Time      `json:"analyzedAt,omitempty"`
	ConfirmedAt           *time.Time      `json:"confirmedAt,omitempty"`
	RejectedAt            *time.Time      `json:"rejectedAt,omitempty"`
	ApplyQueuedAt         *time.Time      `json:"applyQueuedAt,omitempty"`
	AppliedAt             *time.Time      `json:"appliedAt,omitempty"`
}

func PublicTeamleadRunFromDomain(run TeamleadRun) PublicTeamleadRun {
	return PublicTeamleadRun{
		ID:                    run.ID,
		TeamID:                run.TeamID,
		DateFrom:              formatDate(run.DateFrom),
		DateTo:                formatDate(run.DateTo),
		Status:                run.Status,
		CreatedBy:             run.CreatedBy,
		ConfirmedBy:           run.ConfirmedBy,
		RejectedBy:            run.RejectedBy,
		InboundImportBatchID:  run.InboundImportBatchID,
		OutboundImportBatchID: run.OutboundImportBatchID,
		Comment:               run.Comment,
		MismatchCount:         run.MismatchCount,
		ConflictCount:         run.ConflictCount,
		BlockedCount:          run.BlockedCount,
		Pipeline:              nullableRawJSON(run.PipelineJSON),
		InboundSummary:        nullableRawJSON(run.InboundSummaryJSON),
		OutboundSummary:       nullableRawJSON(run.OutboundSummaryJSON),
		Preview:               nullableRawJSON(run.PreviewJSON),
		ApplyResult:           nullableRawJSON(run.ApplyResultJSON),
		ErrorMessage:          run.ErrorMessage,
		CreatedAt:             run.CreatedAt,
		UpdatedAt:             run.UpdatedAt,
		AnalyzedAt:            run.AnalyzedAt,
		ConfirmedAt:           run.ConfirmedAt,
		RejectedAt:            run.RejectedAt,
		ApplyQueuedAt:         run.ApplyQueuedAt,
		AppliedAt:             run.AppliedAt,
	}
}

type PublicTeamleadItem struct {
	ID                       int64           `json:"id"`
	TeamleadReconciliationID int64           `json:"teamleadReconciliationId"`
	TeamID                   int64           `json:"teamId"`
	Direction                string          `json:"direction"`
	Stage                    string          `json:"stage"`
	IssueType                string          `json:"issueType"`
	Severity                 string          `json:"severity"`
	ExternalOrderID          *int64          `json:"externalOrderId,omitempty"`
	ExternalInnerID          *string         `json:"externalInnerId,omitempty"`
	TraderID                 *int64          `json:"traderId,omitempty"`
	RequisiteID              *int64          `json:"requisiteId,omitempty"`
	ShiftID                  *int64          `json:"shiftId,omitempty"`
	Before                   json.RawMessage `json:"before,omitempty"`
	After                    json.RawMessage `json:"after,omitempty"`
	Message                  *string         `json:"message,omitempty"`
	IsBlocking               bool            `json:"isBlocking"`
	AppliedAt                *time.Time      `json:"appliedAt,omitempty"`
	CreatedAt                time.Time       `json:"createdAt"`
}

func PublicTeamleadItemFromDomain(item TeamleadItem) PublicTeamleadItem {
	return PublicTeamleadItem{
		ID:                       item.ID,
		TeamleadReconciliationID: item.TeamleadReconciliationID,
		TeamID:                   item.TeamID,
		Direction:                item.Direction,
		Stage:                    item.Stage,
		IssueType:                item.IssueType,
		Severity:                 item.Severity,
		ExternalOrderID:          item.ExternalOrderID,
		ExternalInnerID:          item.ExternalInnerID,
		TraderID:                 item.TraderID,
		RequisiteID:              item.RequisiteID,
		ShiftID:                  item.ShiftID,
		Before:                   nullableRawJSON(item.BeforeJSON),
		After:                    nullableRawJSON(item.AfterJSON),
		Message:                  item.Message,
		IsBlocking:               item.IsBlocking,
		AppliedAt:                item.AppliedAt,
		CreatedAt:                item.CreatedAt,
	}
}

func PublicTeamleadItemsFromDomain(items []TeamleadItem) []PublicTeamleadItem {
	publicItems := make([]PublicTeamleadItem, 0, len(items))
	for _, item := range items {
		publicItems = append(publicItems, PublicTeamleadItemFromDomain(item))
	}
	return publicItems
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func nullableRawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}

	return value
}
