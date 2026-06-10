package orders

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) ListTraderOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	shiftID, err := r.currentShiftID(ctx, scope)
	if err != nil {
		return ListResult{}, err
	}

	rows, err := r.queries.ListTraderOrders(ctx, db.ListTraderOrdersParams{
		TeamID:     scope.TeamID,
		ShiftID:    int8Value(&shiftID),
		Direction:  scope.Direction,
		DateFrom:   dateValue(filters.DateFrom),
		DateTo:     dateValue(filters.DateTo),
		WorkerName: textValue(filters.WorkerName),
		Requisite:  textValue(filters.Requisite),
		MethodType: textValue(filters.MethodType),
		Status:     textValue(filters.Status),
		AmountFrom: int8Value(filters.AmountFrom),
		AmountTo:   int8Value(filters.AmountTo),
		Sort:       filters.Sort,
		PageOffset: pageOffset(filters),
		PageSize:   int32(filters.PageSize),
	})
	if err != nil {
		return ListResult{}, err
	}

	return listResultFromTraderRows(rows, filters), nil
}

func (r *Repository) ListTeamleadOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	rows, err := r.queries.ListTeamleadOrders(ctx, db.ListTeamleadOrdersParams{
		TeamID:     scope.TeamID,
		Direction:  scope.Direction,
		DateFrom:   dateValue(filters.DateFrom),
		DateTo:     dateValue(filters.DateTo),
		TraderID:   int8Value(filters.TraderID),
		WorkerName: textValue(filters.WorkerName),
		Requisite:  textValue(filters.Requisite),
		MethodType: textValue(filters.MethodType),
		Status:     textValue(filters.Status),
		AmountFrom: int8Value(filters.AmountFrom),
		AmountTo:   int8Value(filters.AmountTo),
		Sort:       filters.Sort,
		PageOffset: pageOffset(filters),
		PageSize:   int32(filters.PageSize),
	})
	if err != nil {
		return ListResult{}, err
	}

	return listResultFromTeamleadRows(rows, filters), nil
}

func (r *Repository) TraderDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	shiftID, err := r.currentShiftID(ctx, scope)
	if err != nil {
		return Dashboard{}, err
	}

	summary, err := r.queries.TraderOrdersSummary(ctx, db.TraderOrdersSummaryParams{
		TeamID:    scope.TeamID,
		ShiftID:   int8Value(&shiftID),
		Direction: scope.Direction,
		DateFrom:  dateValue(filters.DateFrom),
		DateTo:    dateValue(filters.DateTo),
	})
	if err != nil {
		return Dashboard{}, err
	}

	breakdownRows, err := r.queries.TraderStatusBreakdown(ctx, db.TraderStatusBreakdownParams{
		TeamID:    scope.TeamID,
		ShiftID:   int8Value(&shiftID),
		Direction: scope.Direction,
		DateFrom:  dateValue(filters.DateFrom),
		DateTo:    dateValue(filters.DateTo),
	})
	if err != nil {
		return Dashboard{}, err
	}

	importRows, err := r.queries.TraderRecentImports(ctx, db.TraderRecentImportsParams{
		TeamID:     scope.TeamID,
		ShiftID:    int8Value(&shiftID),
		Direction:  scope.Direction,
		LimitCount: 10,
	})
	if err != nil {
		return Dashboard{}, err
	}

	breakdown := traderBreakdownFromRows(breakdownRows)
	return Dashboard{
		Summary: Summary{
			TotalAmountMinor:   summary.TotalAmountMinor,
			TotalCount:         summary.TotalCount,
			SuccessAmountMinor: summary.SuccessAmountMinor,
			SuccessCount:       summary.SuccessCount,
			FailedAmountMinor:  summary.FailedAmountMinor,
			FailedCount:        summary.FailedCount,
			UnknownAmountMinor: summary.UnknownAmountMinor,
			UnknownCount:       summary.UnknownCount,
		},
		StatusBreakdown: breakdown,
		UnknownStatuses: unknownStatuses(breakdown),
		RecentImports:   importsFromRows(importRows),
	}, nil
}

func (r *Repository) TeamleadDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	summary, err := r.queries.TeamleadOrdersSummary(ctx, db.TeamleadOrdersSummaryParams{
		TeamID:    scope.TeamID,
		Direction: scope.Direction,
		DateFrom:  dateValue(filters.DateFrom),
		DateTo:    dateValue(filters.DateTo),
		TraderID:  int8Value(filters.TraderID),
	})
	if err != nil {
		return Dashboard{}, err
	}

	breakdownRows, err := r.queries.TeamleadStatusBreakdown(ctx, db.TeamleadStatusBreakdownParams{
		TeamID:    scope.TeamID,
		Direction: scope.Direction,
		DateFrom:  dateValue(filters.DateFrom),
		DateTo:    dateValue(filters.DateTo),
		TraderID:  int8Value(filters.TraderID),
	})
	if err != nil {
		return Dashboard{}, err
	}

	importRows, err := r.queries.TeamleadRecentImports(ctx, db.TeamleadRecentImportsParams{
		TeamID:     scope.TeamID,
		Direction:  scope.Direction,
		LimitCount: 10,
	})
	if err != nil {
		return Dashboard{}, err
	}

	breakdown := teamleadBreakdownFromRows(breakdownRows)
	return Dashboard{
		Summary: Summary{
			TotalAmountMinor:   summary.TotalAmountMinor,
			TotalCount:         summary.TotalCount,
			SuccessAmountMinor: summary.SuccessAmountMinor,
			SuccessCount:       summary.SuccessCount,
			FailedAmountMinor:  summary.FailedAmountMinor,
			FailedCount:        summary.FailedCount,
			UnknownAmountMinor: summary.UnknownAmountMinor,
			UnknownCount:       summary.UnknownCount,
		},
		StatusBreakdown: breakdown,
		UnknownStatuses: unknownStatuses(breakdown),
		RecentImports:   importsFromRows(importRows),
	}, nil
}

func (r *Repository) currentShiftID(ctx context.Context, scope Scope) (int64, error) {
	if scope.TraderID == nil {
		return 0, ErrInvalidInput
	}

	shiftID, err := r.queries.GetCurrentShiftIDForReadModel(ctx, db.GetCurrentShiftIDForReadModelParams{
		TeamID:   scope.TeamID,
		TraderID: *scope.TraderID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoCurrentShift
	}
	if err != nil {
		return 0, err
	}

	return shiftID, nil
}

func listResultFromTraderRows(rows []db.ListTraderOrdersRow, filters Filters) ListResult {
	items := make([]Order, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		if row.TotalCount > total {
			total = row.TotalCount
		}
		items = append(items, orderFromTraderRow(row))
	}

	return ListResult{
		Items:    items,
		Page:     filters.Page,
		PageSize: filters.PageSize,
		Total:    total,
	}
}

func listResultFromTeamleadRows(rows []db.ListTeamleadOrdersRow, filters Filters) ListResult {
	items := make([]Order, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		if row.TotalCount > total {
			total = row.TotalCount
		}
		items = append(items, orderFromTeamleadRow(row))
	}

	return ListResult{
		Items:    items,
		Page:     filters.Page,
		PageSize: filters.PageSize,
		Total:    total,
	}
}

func orderFromTraderRow(row db.ListTraderOrdersRow) Order {
	return Order{
		ScopeItemID:       row.ScopeItemID,
		ExternalOrderID:   row.ExternalOrderID,
		ExternalID:        row.ExternalID,
		ExternalInnerID:   row.ExternalInnerID,
		WorkerName:        row.WorkerName,
		TraderID:          int8Ptr(row.TraderID),
		TraderLogin:       textPtr(row.TraderLogin),
		RequisiteRaw:      textPtr(row.RequisiteRaw),
		RequisitePhone:    textPtr(row.RequisitePhone),
		MethodType:        textPtr(row.MethodType),
		MethodName:        textPtr(row.MethodName),
		AmountMinor:       row.AmountMinor,
		Currency:          row.Currency,
		RawStatus:         row.RawStatus,
		NormalizedStatus:  row.NormalizedStatus,
		CreatedAtExternal: row.CreatedAtExternal.Time,
		ImportBatchID:     row.ImportBatchID,
	}
}

func orderFromTeamleadRow(row db.ListTeamleadOrdersRow) Order {
	return Order{
		ScopeItemID:       row.ScopeItemID,
		ExternalOrderID:   row.ExternalOrderID,
		ExternalID:        row.ExternalID,
		ExternalInnerID:   row.ExternalInnerID,
		WorkerName:        row.WorkerName,
		TraderID:          int8Ptr(row.TraderID),
		TraderLogin:       textPtr(row.TraderLogin),
		RequisiteRaw:      textPtr(row.RequisiteRaw),
		RequisitePhone:    textPtr(row.RequisitePhone),
		MethodType:        textPtr(row.MethodType),
		MethodName:        textPtr(row.MethodName),
		AmountMinor:       row.AmountMinor,
		Currency:          row.Currency,
		RawStatus:         row.RawStatus,
		NormalizedStatus:  row.NormalizedStatus,
		CreatedAtExternal: row.CreatedAtExternal.Time,
		ImportBatchID:     row.ImportBatchID,
	}
}

func traderBreakdownFromRows(rows []db.TraderStatusBreakdownRow) []StatusBreakdownItem {
	items := make([]StatusBreakdownItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, StatusBreakdownItem{
			RawStatus:        row.RawStatus,
			NormalizedStatus: row.NormalizedStatus,
			AmountMinor:      row.AmountMinor,
			Count:            row.Count,
		})
	}

	return items
}

func teamleadBreakdownFromRows(rows []db.TeamleadStatusBreakdownRow) []StatusBreakdownItem {
	items := make([]StatusBreakdownItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, StatusBreakdownItem{
			RawStatus:        row.RawStatus,
			NormalizedStatus: row.NormalizedStatus,
			AmountMinor:      row.AmountMinor,
			Count:            row.Count,
		})
	}

	return items
}

func importsFromRows(rows []db.ImportBatch) []ImportHistoryItem {
	items := make([]ImportHistoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ImportHistoryItem{
			ID:                  row.ID,
			TeamID:              row.TeamID,
			UploadedBy:          row.UploadedBy,
			ScopeType:           row.ScopeType,
			Direction:           row.Direction,
			ShiftID:             int8Ptr(row.ShiftID),
			AccountingPeriodID:  int8Ptr(row.AccountingPeriodID),
			TraderID:            int8Ptr(row.TraderID),
			FileName:            row.FileName,
			RowsCount:           row.RowsCount,
			Status:              row.Status,
			SupersededByBatchID: int8Ptr(row.SupersededByBatchID),
			ErrorMessage:        textPtr(row.ErrorMessage),
			CreatedAt:           row.CreatedAt.Time,
			AppliedAt:           timePtr(row.AppliedAt),
		})
	}

	return items
}

func unknownStatuses(items []StatusBreakdownItem) []string {
	seen := map[string]bool{}
	statuses := make([]string, 0)
	for _, item := range items {
		if item.NormalizedStatus != "unknown" || seen[item.RawStatus] {
			continue
		}
		seen[item.RawStatus] = true
		statuses = append(statuses, item.RawStatus)
	}

	return statuses
}

func pageOffset(filters Filters) int32 {
	return int32((filters.Page - 1) * filters.PageSize)
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

func dateValue(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}

	return pgtype.Date{Time: *value, Valid: true}
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
