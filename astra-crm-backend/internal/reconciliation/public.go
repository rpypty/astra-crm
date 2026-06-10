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

func nullableRawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}

	return value
}
