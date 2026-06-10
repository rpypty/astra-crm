package orders

import "time"

type PublicListResult struct {
	Items    []PublicOrder `json:"items"`
	Page     int64         `json:"page"`
	PageSize int64         `json:"pageSize"`
	Total    int64         `json:"total"`
}

type PublicOrder struct {
	ID                int64     `json:"id"`
	ExternalOrderID   int64     `json:"externalOrderId"`
	ExternalID        string    `json:"externalId"`
	ExternalInnerID   string    `json:"externalInnerId"`
	WorkerName        string    `json:"workerName"`
	TraderID          *int64    `json:"traderId,omitempty"`
	TraderLogin       *string   `json:"traderLogin,omitempty"`
	RequisiteRaw      *string   `json:"requisiteRaw,omitempty"`
	RequisitePhone    *string   `json:"requisitePhone,omitempty"`
	MethodType        *string   `json:"methodType,omitempty"`
	MethodName        *string   `json:"methodName,omitempty"`
	AmountMinor       int64     `json:"amountMinor"`
	Currency          string    `json:"currency"`
	RawStatus         string    `json:"rawStatus"`
	NormalizedStatus  string    `json:"normalizedStatus"`
	CreatedAtExternal time.Time `json:"createdAtExternal"`
	ImportBatchID     int64     `json:"importBatchId"`
}

type PublicDashboard struct {
	Summary         PublicSummary               `json:"summary"`
	StatusBreakdown []PublicStatusBreakdownItem `json:"statusBreakdown"`
	UnknownStatuses []string                    `json:"unknownStatuses"`
	RecentImports   []PublicImportHistoryItem   `json:"recentImports"`
}

type PublicSummary struct {
	TotalAmountMinor   int64 `json:"totalAmountMinor"`
	TotalCount         int64 `json:"totalCount"`
	SuccessAmountMinor int64 `json:"successAmountMinor"`
	SuccessCount       int64 `json:"successCount"`
	FailedAmountMinor  int64 `json:"failedAmountMinor"`
	FailedCount        int64 `json:"failedCount"`
	UnknownAmountMinor int64 `json:"unknownAmountMinor"`
	UnknownCount       int64 `json:"unknownCount"`
}

type PublicStatusBreakdownItem struct {
	RawStatus        string `json:"rawStatus"`
	NormalizedStatus string `json:"normalizedStatus"`
	AmountMinor      int64  `json:"amountMinor"`
	Count            int64  `json:"count"`
}

type PublicImportHistoryItem struct {
	ID                  int64      `json:"id"`
	TeamID              int64      `json:"teamId"`
	UploadedBy          int64      `json:"uploadedBy"`
	ScopeType           string     `json:"scopeType"`
	Direction           string     `json:"direction"`
	ShiftID             *int64     `json:"shiftId,omitempty"`
	AccountingPeriodID  *int64     `json:"accountingPeriodId,omitempty"`
	TraderID            *int64     `json:"traderId,omitempty"`
	FileName            string     `json:"fileName"`
	RowsCount           int64      `json:"rowsCount"`
	Status              string     `json:"status"`
	SupersededByBatchID *int64     `json:"supersededByBatchId,omitempty"`
	ErrorMessage        *string    `json:"errorMessage,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	AppliedAt           *time.Time `json:"appliedAt,omitempty"`
}

func PublicListFromDomain(result ListResult) PublicListResult {
	items := make([]PublicOrder, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, PublicOrderFromDomain(item))
	}

	return PublicListResult{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	}
}

func PublicOrderFromDomain(order Order) PublicOrder {
	return PublicOrder{
		ID:                order.ScopeItemID,
		ExternalOrderID:   order.ExternalOrderID,
		ExternalID:        order.ExternalID,
		ExternalInnerID:   order.ExternalInnerID,
		WorkerName:        order.WorkerName,
		TraderID:          order.TraderID,
		TraderLogin:       order.TraderLogin,
		RequisiteRaw:      order.RequisiteRaw,
		RequisitePhone:    order.RequisitePhone,
		MethodType:        order.MethodType,
		MethodName:        order.MethodName,
		AmountMinor:       order.AmountMinor,
		Currency:          order.Currency,
		RawStatus:         order.RawStatus,
		NormalizedStatus:  order.NormalizedStatus,
		CreatedAtExternal: order.CreatedAtExternal,
		ImportBatchID:     order.ImportBatchID,
	}
}

func PublicDashboardFromDomain(dashboard Dashboard) PublicDashboard {
	breakdown := make([]PublicStatusBreakdownItem, 0, len(dashboard.StatusBreakdown))
	for _, item := range dashboard.StatusBreakdown {
		breakdown = append(breakdown, PublicStatusBreakdownItem{
			RawStatus:        item.RawStatus,
			NormalizedStatus: item.NormalizedStatus,
			AmountMinor:      item.AmountMinor,
			Count:            item.Count,
		})
	}

	recentImports := make([]PublicImportHistoryItem, 0, len(dashboard.RecentImports))
	for _, item := range dashboard.RecentImports {
		recentImports = append(recentImports, PublicImportFromDomain(item))
	}

	return PublicDashboard{
		Summary: PublicSummary{
			TotalAmountMinor:   dashboard.Summary.TotalAmountMinor,
			TotalCount:         dashboard.Summary.TotalCount,
			SuccessAmountMinor: dashboard.Summary.SuccessAmountMinor,
			SuccessCount:       dashboard.Summary.SuccessCount,
			FailedAmountMinor:  dashboard.Summary.FailedAmountMinor,
			FailedCount:        dashboard.Summary.FailedCount,
			UnknownAmountMinor: dashboard.Summary.UnknownAmountMinor,
			UnknownCount:       dashboard.Summary.UnknownCount,
		},
		StatusBreakdown: breakdown,
		UnknownStatuses: dashboard.UnknownStatuses,
		RecentImports:   recentImports,
	}
}

func PublicImportFromDomain(item ImportHistoryItem) PublicImportHistoryItem {
	return PublicImportHistoryItem{
		ID:                  item.ID,
		TeamID:              item.TeamID,
		UploadedBy:          item.UploadedBy,
		ScopeType:           item.ScopeType,
		Direction:           item.Direction,
		ShiftID:             item.ShiftID,
		AccountingPeriodID:  item.AccountingPeriodID,
		TraderID:            item.TraderID,
		FileName:            item.FileName,
		RowsCount:           item.RowsCount,
		Status:              item.Status,
		SupersededByBatchID: item.SupersededByBatchID,
		ErrorMessage:        item.ErrorMessage,
		CreatedAt:           item.CreatedAt,
		AppliedAt:           item.AppliedAt,
	}
}
