package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var ErrRepositoryNotConfigured = errors.New("imports repository is not configured")

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	db txBeginner
}

func NewRepository(db txBeginner) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ApplyImport(ctx context.Context, record ApplyImportRecord) (ApplyResult, error) {
	if r.db == nil {
		return ApplyResult{}, ErrRepositoryNotConfigured
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := db.New(tx)
	batch, err := queries.CreateImportBatch(ctx, db.CreateImportBatchParams{
		TeamID:             record.TeamID,
		UploadedBy:         record.UploadedBy,
		ScopeType:          record.Scope.Type,
		Direction:          record.Scope.Direction,
		ShiftID:            int8Value(record.Scope.ShiftID),
		AccountingPeriodID: int8Value(record.Scope.AccountingPeriodID),
		TraderID:           int8Value(record.Scope.TraderID),
		FileName:           record.FileName,
		FileHash:           record.FileHash,
		RowsCount:          int64(len(record.Rows)),
	})
	if err != nil {
		return ApplyResult{}, err
	}

	type scopeLink struct {
		importRowID     int64
		externalOrderID int64
	}
	scopeLinks := make([]scopeLink, 0, len(record.Rows))
	createdOrders := int64(0)
	updatedOrders := int64(0)
	for _, parsedRow := range record.Rows {
		importRow, err := insertImportRow(ctx, queries, batch.ID, parsedRow)
		if err != nil {
			return ApplyResult{}, err
		}

		externalOrder, err := upsertExternalOrder(ctx, queries, record, batch.ID, parsedRow)
		if err != nil {
			return ApplyResult{}, err
		}
		if externalOrder.Inserted {
			createdOrders++
		} else {
			updatedOrders++
		}

		scopeLinks = append(scopeLinks, scopeLink{
			importRowID:     importRow.ID,
			externalOrderID: externalOrder.ID,
		})
	}

	deactivated, err := deactivateScopeItems(ctx, queries, record)
	if err != nil {
		return ApplyResult{}, err
	}

	superseded, err := supersedeImportBatches(ctx, queries, record, batch.ID)
	if err != nil {
		return ApplyResult{}, err
	}

	for _, link := range scopeLinks {
		if err := createScopeItem(ctx, queries, record, batch.ID, link.importRowID, link.externalOrderID); err != nil {
			return ApplyResult{}, err
		}
	}

	appliedBatch, err := queries.MarkImportBatchApplied(ctx, batch.ID)
	if err != nil {
		return ApplyResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ApplyResult{}, err
	}
	committed = true

	return ApplyResult{
		Batch:                 fromDBImportBatch(appliedBatch),
		RowsCount:             int64(len(record.Rows)),
		CreatedOrders:         createdOrders,
		UpdatedOrders:         updatedOrders,
		DeactivatedScopeItems: int64(len(deactivated)),
		ActiveScopeItems:      int64(len(scopeLinks)),
		SupersededBatches:     int64(len(superseded)),
	}, nil
}

func insertImportRow(ctx context.Context, queries *db.Queries, batchID int64, parsedRow ParsedOrderRow) (db.ImportRow, error) {
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

func upsertExternalOrder(ctx context.Context, queries *db.Queries, record ApplyImportRecord, batchID int64, parsedRow ParsedOrderRow) (db.UpsertExternalOrderRow, error) {
	params, err := upsertExternalOrderParams(record, batchID, parsedRow)
	if err != nil {
		return db.UpsertExternalOrderRow{}, err
	}

	return queries.UpsertExternalOrder(ctx, params)
}

func upsertExternalOrderParams(record ApplyImportRecord, batchID int64, parsedRow ParsedOrderRow) (db.UpsertExternalOrderParams, error) {
	course, err := numericValue(parsedRow.Course)
	if err != nil {
		return db.UpsertExternalOrderParams{}, fmt.Errorf("course: %w", err)
	}
	courseWorker, err := numericValue(parsedRow.CourseWorker)
	if err != nil {
		return db.UpsertExternalOrderParams{}, fmt.Errorf("courseWorker: %w", err)
	}
	workerAmount, err := numericValue(parsedRow.WorkerAmount)
	if err != nil {
		return db.UpsertExternalOrderParams{}, fmt.Errorf("workerAmount: %w", err)
	}
	workerProfit, err := numericValue(parsedRow.WorkerProfit)
	if err != nil {
		return db.UpsertExternalOrderParams{}, fmt.Errorf("workerProfit: %w", err)
	}

	return db.UpsertExternalOrderParams{
		TeamID:              record.TeamID,
		Direction:           record.Scope.Direction,
		ExternalID:          parsedRow.ExternalID,
		ExternalInnerID:     parsedRow.ExternalInnerID,
		ExternalForeignID:   textValue(parsedRow.ExternalForeignID),
		WorkerName:          parsedRow.WorkerName,
		TraderID:            int8Value(record.Scope.TraderID),
		RequisiteRaw:        textValue(parsedRow.RequisiteRaw),
		RequisitePhone:      textValue(parsedRow.RequisitePhone),
		RequisiteExternalID: textValue(parsedRow.RequisiteExternalID),
		RequisiteID:         pgtype.Int8{},
		DeviceName:          textValue(parsedRow.DeviceName),
		MethodType:          textValue(parsedRow.MethodType),
		MethodName:          textValue(parsedRow.MethodName),
		AmountMinor:         parsedRow.AmountMinor,
		Currency:            parsedRow.Currency,
		Course:              course,
		CourseWorker:        courseWorker,
		WorkerAmount:        workerAmount,
		WorkerProfit:        workerProfit,
		RawStatus:           parsedRow.RawStatus,
		NormalizedStatus:    parsedRow.NormalizedStatus,
		CreatedAtExternal:   timeValue(parsedRow.CreatedAtExternal),
		ClosedAtExternal:    timePtrValue(parsedRow.ClosedAtExternal),
		UpdatedAtExternal:   timePtrValue(parsedRow.UpdatedAtExternal),
		OldAmountMinor:      int8Value(parsedRow.OldAmountMinor),
		HadDispute:          boolValue(parsedRow.HadDispute),
		Receipt:             textValue(parsedRow.Receipt),
		OrderComment:        textValue(parsedRow.OrderComment),
		Ordered:             boolValue(parsedRow.Ordered),
		Counted:             boolValue(parsedRow.Counted),
		Initials:            textValue(parsedRow.Initials),
		LastImportBatchID:   pgtype.Int8{Int64: batchID, Valid: true},
	}, nil
}

func deactivateScopeItems(ctx context.Context, queries *db.Queries, record ApplyImportRecord) ([]int64, error) {
	switch record.Scope.Type {
	case ScopeTypeTraderShift:
		return queries.DeactivateTraderShiftScopeItems(ctx, db.DeactivateTraderShiftScopeItemsParams{
			TeamID:    record.TeamID,
			ShiftID:   int8Value(record.Scope.ShiftID),
			Direction: record.Scope.Direction,
		})
	case ScopeTypeTeamleadPeriod:
		return queries.DeactivateTeamleadPeriodScopeItems(ctx, db.DeactivateTeamleadPeriodScopeItemsParams{
			TeamID:             record.TeamID,
			AccountingPeriodID: int8Value(record.Scope.AccountingPeriodID),
			Direction:          record.Scope.Direction,
		})
	default:
		return nil, fmt.Errorf("unsupported scope type: %s", record.Scope.Type)
	}
}

func supersedeImportBatches(ctx context.Context, queries *db.Queries, record ApplyImportRecord, batchID int64) ([]int64, error) {
	newBatchID := pgtype.Int8{Int64: batchID, Valid: true}
	switch record.Scope.Type {
	case ScopeTypeTraderShift:
		return queries.SupersedeTraderShiftImportBatches(ctx, db.SupersedeTraderShiftImportBatchesParams{
			NewBatchID: newBatchID,
			TeamID:     record.TeamID,
			ShiftID:    int8Value(record.Scope.ShiftID),
			Direction:  record.Scope.Direction,
		})
	case ScopeTypeTeamleadPeriod:
		return queries.SupersedeTeamleadPeriodImportBatches(ctx, db.SupersedeTeamleadPeriodImportBatchesParams{
			NewBatchID:         newBatchID,
			TeamID:             record.TeamID,
			AccountingPeriodID: int8Value(record.Scope.AccountingPeriodID),
			Direction:          record.Scope.Direction,
		})
	default:
		return nil, fmt.Errorf("unsupported scope type: %s", record.Scope.Type)
	}
}

func createScopeItem(ctx context.Context, queries *db.Queries, record ApplyImportRecord, batchID int64, importRowID int64, externalOrderID int64) error {
	switch record.Scope.Type {
	case ScopeTypeTraderShift:
		_, err := queries.CreateTraderShiftScopeItem(ctx, db.CreateTraderShiftScopeItemParams{
			TeamID:          record.TeamID,
			Direction:       record.Scope.Direction,
			ShiftID:         int8Value(record.Scope.ShiftID),
			ImportBatchID:   batchID,
			ImportRowID:     importRowID,
			ExternalOrderID: externalOrderID,
		})
		return err
	case ScopeTypeTeamleadPeriod:
		_, err := queries.CreateTeamleadPeriodScopeItem(ctx, db.CreateTeamleadPeriodScopeItemParams{
			TeamID:             record.TeamID,
			Direction:          record.Scope.Direction,
			AccountingPeriodID: int8Value(record.Scope.AccountingPeriodID),
			ImportBatchID:      batchID,
			ImportRowID:        importRowID,
			ExternalOrderID:    externalOrderID,
		})
		return err
	default:
		return fmt.Errorf("unsupported scope type: %s", record.Scope.Type)
	}
}

func fromDBImportBatch(row db.ImportBatch) ImportBatch {
	return ImportBatch{
		ID:                  row.ID,
		TeamID:              row.TeamID,
		UploadedBy:          row.UploadedBy,
		ScopeType:           row.ScopeType,
		Direction:           row.Direction,
		ShiftID:             int8Ptr(row.ShiftID),
		AccountingPeriodID:  int8Ptr(row.AccountingPeriodID),
		TraderID:            int8Ptr(row.TraderID),
		FileName:            row.FileName,
		FileHash:            row.FileHash,
		RowsCount:           row.RowsCount,
		Status:              row.Status,
		SupersededByBatchID: int8Ptr(row.SupersededByBatchID),
		ErrorMessage:        textPtr(row.ErrorMessage),
		CreatedAt:           row.CreatedAt.Time,
		AppliedAt:           timePtr(row.AppliedAt),
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

func boolValue(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{}
	}

	return pgtype.Bool{Bool: *value, Valid: true}
}

func timeValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timePtrValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

func numericValue(value *string) (pgtype.Numeric, error) {
	if value == nil {
		return pgtype.Numeric{}, nil
	}

	var numeric pgtype.Numeric
	if err := numeric.Scan(*value); err != nil {
		return pgtype.Numeric{}, err
	}

	return numeric, nil
}
