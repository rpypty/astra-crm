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

	importRowsJSON, err := batchImportRowsJSON(record.Rows)
	if err != nil {
		return ApplyResult{}, err
	}
	externalOrderRowsJSON, err := batchExternalOrderRowsJSON(record.Rows)
	if err != nil {
		return ApplyResult{}, err
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

	importRows, err := queries.BatchInsertImportRows(ctx, db.BatchInsertImportRowsParams{
		ImportBatchID: batch.ID,
		RowsJson:      importRowsJSON,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if len(importRows) != len(record.Rows) {
		return ApplyResult{}, fmt.Errorf("batch insert import rows: inserted %d rows, want %d", len(importRows), len(record.Rows))
	}

	externalOrders, err := queries.BatchUpsertExternalOrders(ctx, db.BatchUpsertExternalOrdersParams{
		TeamID:            record.TeamID,
		Direction:         record.Scope.Direction,
		TraderID:          int8Value(record.Scope.TraderID),
		LastImportBatchID: pgtype.Int8{Int64: batch.ID, Valid: true},
		RowsJson:          externalOrderRowsJSON,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if len(externalOrders) != len(record.Rows) {
		return ApplyResult{}, fmt.Errorf("batch upsert external orders: upserted %d rows, want %d", len(externalOrders), len(record.Rows))
	}

	createdOrders := int64(0)
	updatedOrders := int64(0)
	for _, externalOrder := range externalOrders {
		if externalOrder.Inserted {
			createdOrders++
		} else {
			updatedOrders++
		}
	}

	deactivated, err := deactivateScopeItems(ctx, queries, record)
	if err != nil {
		return ApplyResult{}, err
	}

	superseded, err := supersedeImportBatches(ctx, queries, record, batch.ID)
	if err != nil {
		return ApplyResult{}, err
	}

	activeScopeItems, err := createScopeItemsBatch(ctx, queries, record, batch.ID)
	if err != nil {
		return ApplyResult{}, err
	}
	if len(activeScopeItems) != len(record.Rows) {
		return ApplyResult{}, fmt.Errorf("batch create scope items: created %d rows, want %d", len(activeScopeItems), len(record.Rows))
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
		ActiveScopeItems:      int64(len(activeScopeItems)),
		SupersededBatches:     int64(len(superseded)),
	}, nil
}

type batchImportRow struct {
	RowNumber       int64             `json:"row_number"`
	ExternalID      string            `json:"external_id"`
	ExternalInnerID string            `json:"external_inner_id"`
	RawPayloadJSON  map[string]string `json:"raw_payload_json"`
}

type batchExternalOrderRow struct {
	ExternalID          string     `json:"external_id"`
	ExternalInnerID     string     `json:"external_inner_id"`
	ExternalForeignID   *string    `json:"external_foreign_id"`
	WorkerName          string     `json:"worker_name"`
	RequisiteRaw        *string    `json:"requisite_raw"`
	RequisitePhone      *string    `json:"requisite_phone"`
	RequisiteExternalID *string    `json:"requisite_external_id"`
	DeviceName          *string    `json:"device_name"`
	MethodType          *string    `json:"method_type"`
	MethodName          *string    `json:"method_name"`
	AmountMinor         int64      `json:"amount_minor"`
	Currency            string     `json:"currency"`
	Course              *string    `json:"course"`
	CourseWorker        *string    `json:"course_worker"`
	WorkerAmount        *string    `json:"worker_amount"`
	WorkerProfit        *string    `json:"worker_profit"`
	RawStatus           string     `json:"raw_status"`
	NormalizedStatus    string     `json:"normalized_status"`
	CreatedAtExternal   time.Time  `json:"created_at_external"`
	ClosedAtExternal    *time.Time `json:"closed_at_external"`
	UpdatedAtExternal   *time.Time `json:"updated_at_external"`
	OldAmountMinor      *int64     `json:"old_amount_minor"`
	HadDispute          *bool      `json:"had_dispute"`
	Receipt             *string    `json:"receipt"`
	OrderComment        *string    `json:"order_comment"`
	Ordered             *bool      `json:"ordered"`
	Counted             *bool      `json:"counted"`
	Initials            *string    `json:"initials"`
}

func batchImportRowsJSON(rows []ParsedOrderRow) ([]byte, error) {
	payload := make([]batchImportRow, 0, len(rows))
	for _, row := range rows {
		payload = append(payload, batchImportRow{
			RowNumber:       int64(row.RowNumber),
			ExternalID:      row.ExternalID,
			ExternalInnerID: row.ExternalInnerID,
			RawPayloadJSON:  row.RawPayload,
		})
	}
	return json.Marshal(payload)
}

func batchExternalOrderRowsJSON(rows []ParsedOrderRow) ([]byte, error) {
	payload := make([]batchExternalOrderRow, 0, len(rows))
	for _, row := range rows {
		if _, err := numericValue(row.Course); err != nil {
			return nil, fmt.Errorf("course: %w", err)
		}
		if _, err := numericValue(row.CourseWorker); err != nil {
			return nil, fmt.Errorf("courseWorker: %w", err)
		}
		if _, err := numericValue(row.WorkerAmount); err != nil {
			return nil, fmt.Errorf("workerAmount: %w", err)
		}
		if _, err := numericValue(row.WorkerProfit); err != nil {
			return nil, fmt.Errorf("workerProfit: %w", err)
		}

		payload = append(payload, batchExternalOrderRow{
			ExternalID:          row.ExternalID,
			ExternalInnerID:     row.ExternalInnerID,
			ExternalForeignID:   row.ExternalForeignID,
			WorkerName:          row.WorkerName,
			RequisiteRaw:        row.RequisiteRaw,
			RequisitePhone:      row.RequisitePhone,
			RequisiteExternalID: row.RequisiteExternalID,
			DeviceName:          row.DeviceName,
			MethodType:          row.MethodType,
			MethodName:          row.MethodName,
			AmountMinor:         row.AmountMinor,
			Currency:            row.Currency,
			Course:              row.Course,
			CourseWorker:        row.CourseWorker,
			WorkerAmount:        row.WorkerAmount,
			WorkerProfit:        row.WorkerProfit,
			RawStatus:           row.RawStatus,
			NormalizedStatus:    row.NormalizedStatus,
			CreatedAtExternal:   row.CreatedAtExternal,
			ClosedAtExternal:    row.ClosedAtExternal,
			UpdatedAtExternal:   row.UpdatedAtExternal,
			OldAmountMinor:      row.OldAmountMinor,
			HadDispute:          row.HadDispute,
			Receipt:             row.Receipt,
			OrderComment:        row.OrderComment,
			Ordered:             row.Ordered,
			Counted:             row.Counted,
			Initials:            row.Initials,
		})
	}
	return json.Marshal(payload)
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
			UploadedBy:         record.UploadedBy,
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
			UploadedBy:         record.UploadedBy,
			AccountingPeriodID: int8Value(record.Scope.AccountingPeriodID),
			Direction:          record.Scope.Direction,
		})
	default:
		return nil, fmt.Errorf("unsupported scope type: %s", record.Scope.Type)
	}
}

func createScopeItemsBatch(ctx context.Context, queries *db.Queries, record ApplyImportRecord, batchID int64) ([]int64, error) {
	switch record.Scope.Type {
	case ScopeTypeTraderShift:
		return queries.CreateTraderShiftScopeItemsBatch(ctx, db.CreateTraderShiftScopeItemsBatchParams{
			TeamID:        record.TeamID,
			Direction:     record.Scope.Direction,
			ShiftID:       int8Value(record.Scope.ShiftID),
			ImportBatchID: batchID,
		})
	case ScopeTypeTeamleadPeriod:
		return queries.CreateTeamleadPeriodScopeItemsBatch(ctx, db.CreateTeamleadPeriodScopeItemsBatchParams{
			TeamID:             record.TeamID,
			Direction:          record.Scope.Direction,
			AccountingPeriodID: int8Value(record.Scope.AccountingPeriodID),
			ImportBatchID:      batchID,
		})
	default:
		return nil, fmt.Errorf("unsupported scope type: %s", record.Scope.Type)
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
