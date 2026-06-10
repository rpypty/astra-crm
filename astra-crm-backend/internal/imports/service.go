package imports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

var ErrInvalidInput = errors.New("invalid import input")

type Store interface {
	ApplyImport(ctx context.Context, record ApplyImportRecord) (ApplyResult, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type ReconciliationHook interface {
	AfterImportApplied(ctx context.Context, result ApplyResult) error
}

type Service struct {
	store Store
	audit AuditService
	hook  ReconciliationHook
}

func NewService(store Store, auditService AuditService, hook ReconciliationHook) *Service {
	return &Service{
		store: store,
		audit: auditService,
		hook:  hook,
	}
}

type ApplyCSVParams struct {
	ActorID      int64
	TeamID       int64
	Scope        Scope
	FileName     string
	Reader       io.Reader
	ParseOptions ParseOptions
}

func (s *Service) ApplyCSV(ctx context.Context, params ApplyCSVParams) (ApplyResult, error) {
	scope := params.Scope
	if scope.Type == ScopeTypeTraderShift && scope.TraderID == nil && params.ActorID > 0 {
		scope.TraderID = &params.ActorID
	}

	if params.ActorID <= 0 || params.TeamID <= 0 || strings.TrimSpace(params.FileName) == "" || params.Reader == nil || !validScope(scope) {
		return ApplyResult{}, ErrInvalidInput
	}
	if s.store == nil {
		return ApplyResult{}, ErrRepositoryNotConfigured
	}

	payload, err := io.ReadAll(params.Reader)
	if err != nil {
		return ApplyResult{}, err
	}

	parseResult, err := ParseCSV(bytes.NewReader(payload), params.ParseOptions)
	if err != nil {
		return ApplyResult{Parse: parseResult}, err
	}

	hash := sha256.Sum256(payload)
	result, err := s.store.ApplyImport(ctx, ApplyImportRecord{
		TeamID:     params.TeamID,
		UploadedBy: params.ActorID,
		Scope:      scope,
		FileName:   strings.TrimSpace(params.FileName),
		FileHash:   hex.EncodeToString(hash[:]),
		Rows:       parseResult.Rows,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	result.Parse = parseResult

	if s.hook != nil {
		if err := s.hook.AfterImportApplied(ctx, result); err != nil {
			return ApplyResult{}, err
		}
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionImportApplied,
		EntityType: "import_batch",
		EntityID:   strconv.FormatInt(result.Batch.ID, 10),
		After: map[string]any{
			"batchId":                result.Batch.ID,
			"scopeType":              result.Batch.ScopeType,
			"direction":              result.Batch.Direction,
			"rowsCount":              result.RowsCount,
			"createdOrders":          result.CreatedOrders,
			"updatedOrders":          result.UpdatedOrders,
			"deactivatedScopeItems":  result.DeactivatedScopeItems,
			"activeScopeItems":       result.ActiveScopeItems,
			"supersededBatches":      result.SupersededBatches,
			"rawStatusCounts":        result.Parse.RawStatusCounts,
			"normalizedStatusCounts": result.Parse.NormalizedStatusCounts,
		},
	}); err != nil {
		return ApplyResult{}, err
	}

	return result, nil
}

func validScope(scope Scope) bool {
	if scope.Direction != DirectionInbound && scope.Direction != DirectionOutbound {
		return false
	}

	switch scope.Type {
	case ScopeTypeTraderShift:
		return scope.ShiftID != nil && *scope.ShiftID > 0 && scope.AccountingPeriodID == nil
	case ScopeTypeTeamleadPeriod:
		return scope.AccountingPeriodID != nil && *scope.AccountingPeriodID > 0 && scope.ShiftID == nil
	default:
		return false
	}
}

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}
