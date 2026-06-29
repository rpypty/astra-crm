package orders

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) ListTraderOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	if filters.ConfirmedOnly {
		if scope.TraderID == nil {
			return ListResult{}, ErrInvalidInput
		}
		return r.traderConfirmedOrders(ctx, scope, filters)
	}

	shiftID, err := r.currentShiftID(ctx, scope)
	if err != nil {
		return ListResult{}, err
	}

	createdFrom, createdToExclusive := createdRangeValues(filters)
	countParams := db.CountTraderOrdersParams{
		TeamID:             scope.TeamID,
		ShiftID:            int8Value(&shiftID),
		Direction:          scope.Direction,
		CreatedFrom:        createdFrom,
		CreatedToExclusive: createdToExclusive,
		WorkerName:         textValue(filters.WorkerName),
		Requisite:          textValue(filters.Requisite),
		MethodType:         textValue(filters.MethodType),
		Status:             textValue(filters.Status),
		AmountFrom:         int8Value(filters.AmountFrom),
		AmountTo:           int8Value(filters.AmountTo),
	}

	var (
		total int64
		rows  []orderQueryRow
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		total, err = r.queries.CountTraderOrders(groupCtx, countParams)
		return err
	})
	g.Go(func() error {
		var err error
		rows, err = r.listTraderOrderRows(groupCtx, countParams, filters)
		return err
	})
	if err := g.Wait(); err != nil {
		return ListResult{}, err
	}

	return listResultFromRows(rows, filters, total), nil
}

func (r *Repository) ListTeamleadOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	createdFrom, createdToExclusive := createdRangeValues(filters)
	countParams := db.CountTeamleadOrdersParams{
		TeamID:             scope.TeamID,
		Direction:          scope.Direction,
		CreatedFrom:        createdFrom,
		CreatedToExclusive: createdToExclusive,
		TraderID:           int8Value(filters.TraderID),
		TraderIds:          filters.TraderIDs,
		WorkerName:         textValue(filters.WorkerName),
		Requisite:          textValue(filters.Requisite),
		MethodType:         textValue(filters.MethodType),
		Status:             textValue(filters.Status),
		AmountFrom:         int8Value(filters.AmountFrom),
		AmountTo:           int8Value(filters.AmountTo),
		ConfirmedOnly:      filters.ConfirmedOnly,
	}

	var (
		total int64
		rows  []orderQueryRow
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		total, err = r.queries.CountTeamleadOrders(groupCtx, countParams)
		return err
	})
	g.Go(func() error {
		var err error
		rows, err = r.listTeamleadOrderRows(groupCtx, countParams, filters)
		return err
	})
	if err := g.Wait(); err != nil {
		return ListResult{}, err
	}

	return listResultFromRows(rows, filters, total), nil
}

func (r *Repository) traderConfirmedOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	createdFrom, createdToExclusive := createdRangeValues(filters)
	countParams := db.CountTeamleadOrdersParams{
		TeamID:             scope.TeamID,
		Direction:          scope.Direction,
		CreatedFrom:        createdFrom,
		CreatedToExclusive: createdToExclusive,
		TraderID:           int8Value(scope.TraderID),
		WorkerName:         textValue(filters.WorkerName),
		Requisite:          textValue(filters.Requisite),
		MethodType:         textValue(filters.MethodType),
		Status:             textValue(filters.Status),
		AmountFrom:         int8Value(filters.AmountFrom),
		AmountTo:           int8Value(filters.AmountTo),
		ConfirmedOnly:      true,
	}

	var (
		total int64
		rows  []orderQueryRow
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		total, err = r.queries.CountTeamleadOrders(groupCtx, countParams)
		return err
	})
	g.Go(func() error {
		var err error
		rows, err = r.listTeamleadOrderRows(groupCtx, countParams, filters)
		return err
	})
	if err := g.Wait(); err != nil {
		return ListResult{}, err
	}

	return listResultFromRows(rows, filters, total), nil
}

func (r *Repository) TraderDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	if filters.ConfirmedOnly {
		if scope.TraderID == nil {
			return Dashboard{}, ErrInvalidInput
		}
		return r.traderConfirmedDashboard(ctx, scope, filters)
	}

	shiftID, err := r.currentShiftID(ctx, scope)
	if err != nil {
		return Dashboard{}, err
	}

	createdFrom, createdToExclusive := createdRangeValues(filters)
	var (
		summary             db.TraderOrdersSummaryRow
		blockedBalanceMinor int64
		breakdownRows       []db.TraderStatusBreakdownRow
		importRows          []db.ImportBatch
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		summary, err = r.queries.TraderOrdersSummary(groupCtx, db.TraderOrdersSummaryParams{
			TeamID:             scope.TeamID,
			ShiftID:            int8Value(&shiftID),
			Direction:          scope.Direction,
			CreatedFrom:        createdFrom,
			CreatedToExclusive: createdToExclusive,
		})
		return err
	})
	g.Go(func() error {
		var err error
		blockedBalanceMinor, err = r.queries.TraderBlockedBalanceSummary(groupCtx, db.TraderBlockedBalanceSummaryParams{
			TeamID:   scope.TeamID,
			TraderID: *scope.TraderID,
		})
		return err
	})
	g.Go(func() error {
		var err error
		breakdownRows, err = r.queries.TraderStatusBreakdown(groupCtx, db.TraderStatusBreakdownParams{
			TeamID:             scope.TeamID,
			ShiftID:            int8Value(&shiftID),
			Direction:          scope.Direction,
			CreatedFrom:        createdFrom,
			CreatedToExclusive: createdToExclusive,
		})
		return err
	})
	g.Go(func() error {
		var err error
		importRows, err = r.queries.TraderRecentImports(groupCtx, db.TraderRecentImportsParams{
			TeamID:     scope.TeamID,
			ShiftID:    int8Value(&shiftID),
			Direction:  scope.Direction,
			LimitCount: 10,
		})
		return err
	})
	if err := g.Wait(); err != nil {
		return Dashboard{}, err
	}

	breakdown := traderBreakdownFromRows(breakdownRows)
	return Dashboard{
		Summary: Summary{
			TotalAmountMinor:    summary.TotalAmountMinor,
			TotalCount:          summary.TotalCount,
			SuccessAmountMinor:  summary.SuccessAmountMinor,
			SuccessCount:        summary.SuccessCount,
			FailedAmountMinor:   summary.FailedAmountMinor,
			FailedCount:         summary.FailedCount,
			UnknownAmountMinor:  summary.UnknownAmountMinor,
			UnknownCount:        summary.UnknownCount,
			BlockedBalanceMinor: blockedBalanceMinor,
		},
		StatusBreakdown: breakdown,
		UnknownStatuses: unknownStatuses(breakdown),
		RecentImports:   importsFromRows(importRows),
	}, nil
}

func (r *Repository) TeamleadDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	createdFrom, createdToExclusive := createdRangeValues(filters)
	var (
		summary             db.TeamleadOrdersSummaryRow
		blockedBalanceMinor int64
		breakdownRows       []db.TeamleadStatusBreakdownRow
		importRows          []db.ImportBatch
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		summary, err = r.queries.TeamleadOrdersSummary(groupCtx, db.TeamleadOrdersSummaryParams{
			TeamID:             scope.TeamID,
			Direction:          scope.Direction,
			ConfirmedOnly:      filters.ConfirmedOnly,
			CreatedFrom:        createdFrom,
			CreatedToExclusive: createdToExclusive,
			TraderID:           int8Value(filters.TraderID),
			TraderIds:          filters.TraderIDs,
		})
		return err
	})
	g.Go(func() error {
		var err error
		blockedBalanceMinor, err = r.queries.TeamleadBlockedBalanceSummary(groupCtx, db.TeamleadBlockedBalanceSummaryParams{
			TeamID:    scope.TeamID,
			TraderID:  int8Value(filters.TraderID),
			TraderIds: filters.TraderIDs,
		})
		return err
	})
	g.Go(func() error {
		var err error
		breakdownRows, err = r.queries.TeamleadStatusBreakdown(groupCtx, db.TeamleadStatusBreakdownParams{
			TeamID:             scope.TeamID,
			Direction:          scope.Direction,
			ConfirmedOnly:      filters.ConfirmedOnly,
			CreatedFrom:        createdFrom,
			CreatedToExclusive: createdToExclusive,
			TraderID:           int8Value(filters.TraderID),
			TraderIds:          filters.TraderIDs,
		})
		return err
	})
	g.Go(func() error {
		var err error
		importRows, err = r.queries.TeamleadRecentImports(groupCtx, db.TeamleadRecentImportsParams{
			TeamID:     scope.TeamID,
			Direction:  scope.Direction,
			UploadedBy: *scope.ActorID,
			LimitCount: 10,
		})
		return err
	})
	if err := g.Wait(); err != nil {
		return Dashboard{}, err
	}

	breakdown := teamleadBreakdownFromRows(breakdownRows)
	return Dashboard{
		Summary: Summary{
			TotalAmountMinor:    summary.TotalAmountMinor,
			TotalCount:          summary.TotalCount,
			SuccessAmountMinor:  summary.SuccessAmountMinor,
			SuccessCount:        summary.SuccessCount,
			FailedAmountMinor:   summary.FailedAmountMinor,
			FailedCount:         summary.FailedCount,
			UnknownAmountMinor:  summary.UnknownAmountMinor,
			UnknownCount:        summary.UnknownCount,
			BlockedBalanceMinor: blockedBalanceMinor,
		},
		StatusBreakdown: breakdown,
		UnknownStatuses: unknownStatuses(breakdown),
		RecentImports:   importsFromRows(importRows),
	}, nil
}

func (r *Repository) traderConfirmedDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	createdFrom, createdToExclusive := createdRangeValues(filters)
	var (
		summary             db.TeamleadOrdersSummaryRow
		blockedBalanceMinor int64
		breakdownRows       []db.TeamleadStatusBreakdownRow
	)
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		summary, err = r.queries.TeamleadOrdersSummary(groupCtx, db.TeamleadOrdersSummaryParams{
			TeamID:             scope.TeamID,
			Direction:          scope.Direction,
			ConfirmedOnly:      true,
			CreatedFrom:        createdFrom,
			CreatedToExclusive: createdToExclusive,
			TraderID:           int8Value(scope.TraderID),
		})
		return err
	})
	g.Go(func() error {
		var err error
		blockedBalanceMinor, err = r.queries.TraderBlockedBalanceSummary(groupCtx, db.TraderBlockedBalanceSummaryParams{
			TeamID:   scope.TeamID,
			TraderID: *scope.TraderID,
		})
		return err
	})
	g.Go(func() error {
		var err error
		breakdownRows, err = r.queries.TeamleadStatusBreakdown(groupCtx, db.TeamleadStatusBreakdownParams{
			TeamID:             scope.TeamID,
			Direction:          scope.Direction,
			ConfirmedOnly:      true,
			CreatedFrom:        createdFrom,
			CreatedToExclusive: createdToExclusive,
			TraderID:           int8Value(scope.TraderID),
		})
		return err
	})
	if err := g.Wait(); err != nil {
		return Dashboard{}, err
	}

	breakdown := teamleadBreakdownFromRows(breakdownRows)
	return Dashboard{
		Summary: Summary{
			TotalAmountMinor:    summary.TotalAmountMinor,
			TotalCount:          summary.TotalCount,
			SuccessAmountMinor:  summary.SuccessAmountMinor,
			SuccessCount:        summary.SuccessCount,
			FailedAmountMinor:   summary.FailedAmountMinor,
			FailedCount:         summary.FailedCount,
			UnknownAmountMinor:  summary.UnknownAmountMinor,
			UnknownCount:        summary.UnknownCount,
			BlockedBalanceMinor: blockedBalanceMinor,
		},
		StatusBreakdown: breakdown,
		UnknownStatuses: unknownStatuses(breakdown),
		RecentImports:   []ImportHistoryItem{},
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

func (r *Repository) listTraderOrderRows(ctx context.Context, params db.CountTraderOrdersParams, filters Filters) ([]orderQueryRow, error) {
	switch filters.Sort {
	case SortCreatedAtAsc:
		rows, err := r.queries.ListTraderOrdersCreatedAsc(ctx, db.ListTraderOrdersCreatedAscParams{
			TeamID:             params.TeamID,
			ShiftID:            params.ShiftID,
			Direction:          params.Direction,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTraderCreatedAsc), err
	case SortAmountAsc:
		rows, err := r.queries.ListTraderOrdersAmountAsc(ctx, db.ListTraderOrdersAmountAscParams{
			TeamID:             params.TeamID,
			ShiftID:            params.ShiftID,
			Direction:          params.Direction,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTraderAmountAsc), err
	case SortAmountDesc:
		rows, err := r.queries.ListTraderOrdersAmountDesc(ctx, db.ListTraderOrdersAmountDescParams{
			TeamID:             params.TeamID,
			ShiftID:            params.ShiftID,
			Direction:          params.Direction,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTraderAmountDesc), err
	default:
		rows, err := r.queries.ListTraderOrdersCreatedDesc(ctx, db.ListTraderOrdersCreatedDescParams{
			TeamID:             params.TeamID,
			ShiftID:            params.ShiftID,
			Direction:          params.Direction,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTraderCreatedDesc), err
	}
}

func (r *Repository) listTeamleadOrderRows(ctx context.Context, params db.CountTeamleadOrdersParams, filters Filters) ([]orderQueryRow, error) {
	switch filters.Sort {
	case SortCreatedAtAsc:
		rows, err := r.queries.ListTeamleadOrdersCreatedAsc(ctx, db.ListTeamleadOrdersCreatedAscParams{
			TeamID:             params.TeamID,
			Direction:          params.Direction,
			ConfirmedOnly:      params.ConfirmedOnly,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			TraderID:           params.TraderID,
			TraderIds:          params.TraderIds,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTeamleadCreatedAsc), err
	case SortAmountAsc:
		rows, err := r.queries.ListTeamleadOrdersAmountAsc(ctx, db.ListTeamleadOrdersAmountAscParams{
			TeamID:             params.TeamID,
			Direction:          params.Direction,
			ConfirmedOnly:      params.ConfirmedOnly,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			TraderID:           params.TraderID,
			TraderIds:          params.TraderIds,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTeamleadAmountAsc), err
	case SortAmountDesc:
		rows, err := r.queries.ListTeamleadOrdersAmountDesc(ctx, db.ListTeamleadOrdersAmountDescParams{
			TeamID:             params.TeamID,
			Direction:          params.Direction,
			ConfirmedOnly:      params.ConfirmedOnly,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			TraderID:           params.TraderID,
			TraderIds:          params.TraderIds,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTeamleadAmountDesc), err
	default:
		rows, err := r.queries.ListTeamleadOrdersCreatedDesc(ctx, db.ListTeamleadOrdersCreatedDescParams{
			TeamID:             params.TeamID,
			Direction:          params.Direction,
			ConfirmedOnly:      params.ConfirmedOnly,
			CreatedFrom:        params.CreatedFrom,
			CreatedToExclusive: params.CreatedToExclusive,
			TraderID:           params.TraderID,
			TraderIds:          params.TraderIds,
			WorkerName:         params.WorkerName,
			Requisite:          params.Requisite,
			MethodType:         params.MethodType,
			Status:             params.Status,
			AmountFrom:         params.AmountFrom,
			AmountTo:           params.AmountTo,
			PageOffset:         pageOffset(filters),
			PageSize:           int32(filters.PageSize),
		})
		return mapOrderRows(rows, orderQueryRowFromTeamleadCreatedDesc), err
	}
}

func listResultFromRows(rows []orderQueryRow, filters Filters, total int64) ListResult {
	items := make([]Order, 0, len(rows))
	for _, row := range rows {
		items = append(items, orderFromRow(row))
	}

	return ListResult{
		Items:    items,
		Page:     filters.Page,
		PageSize: filters.PageSize,
		Total:    total,
	}
}

type orderQueryRow struct {
	ScopeItemID            int64
	ExternalOrderID        int64
	ExternalID             string
	ExternalInnerID        string
	WorkerName             string
	TraderID               pgtype.Int8
	TraderLogin            pgtype.Text
	RequisiteRaw           pgtype.Text
	RequisitePhone         pgtype.Text
	MethodType             pgtype.Text
	MethodName             pgtype.Text
	AmountMinor            int64
	Currency               string
	RawStatus              string
	NormalizedStatus       string
	TlReconciliationStatus string
	CreatedAtExternal      pgtype.Timestamptz
	ImportBatchID          int64
}

func orderFromRow(row orderQueryRow) Order {
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
		TLStatus:          row.TlReconciliationStatus,
		CreatedAtExternal: row.CreatedAtExternal.Time,
		ImportBatchID:     row.ImportBatchID,
	}
}

func mapOrderRows[T any](rows []T, convert func(T) orderQueryRow) []orderQueryRow {
	items := make([]orderQueryRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, convert(row))
	}
	return items
}

func orderQueryRowFromTraderCreatedDesc(row db.ListTraderOrdersCreatedDescRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTraderCreatedAsc(row db.ListTraderOrdersCreatedAscRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTraderAmountAsc(row db.ListTraderOrdersAmountAscRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTraderAmountDesc(row db.ListTraderOrdersAmountDescRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTeamleadCreatedDesc(row db.ListTeamleadOrdersCreatedDescRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTeamleadCreatedAsc(row db.ListTeamleadOrdersCreatedAscRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTeamleadAmountAsc(row db.ListTeamleadOrdersAmountAscRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
	}
}

func orderQueryRowFromTeamleadAmountDesc(row db.ListTeamleadOrdersAmountDescRow) orderQueryRow {
	return orderQueryRow{
		ScopeItemID:            row.ScopeItemID,
		ExternalOrderID:        row.ExternalOrderID,
		ExternalID:             row.ExternalID,
		ExternalInnerID:        row.ExternalInnerID,
		WorkerName:             row.WorkerName,
		TraderID:               row.TraderID,
		TraderLogin:            row.TraderLogin,
		RequisiteRaw:           row.RequisiteRaw,
		RequisitePhone:         row.RequisitePhone,
		MethodType:             row.MethodType,
		MethodName:             row.MethodName,
		AmountMinor:            row.AmountMinor,
		Currency:               row.Currency,
		RawStatus:              row.RawStatus,
		NormalizedStatus:       row.NormalizedStatus,
		TlReconciliationStatus: row.TlReconciliationStatus,
		CreatedAtExternal:      row.CreatedAtExternal,
		ImportBatchID:          row.ImportBatchID,
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

func createdRangeValues(filters Filters) (pgtype.Timestamptz, pgtype.Timestamptz) {
	return timestamptzValue(filters.DateFrom), dateToExclusiveValue(filters.DateTo)
}

func timestamptzValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func dateToExclusiveValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: value.AddDate(0, 0, 1), Valid: true}
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
