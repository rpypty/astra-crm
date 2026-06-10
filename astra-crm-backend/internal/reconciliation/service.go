package reconciliation

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
)

var ErrInvalidInput = errors.New("invalid reconciliation input")

type Store interface {
	RecalculateTraderInbound(ctx context.Context, record RecalculateTraderInboundRecord) (Run, error)
	LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error)
	AcceptTraderInbound(ctx context.Context, record AcceptTraderInboundRecord) (Run, error)
	RecalculateTraderOutbound(ctx context.Context, record RecalculateTraderOutboundRecord) (Run, error)
	LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error)
	AcceptTraderOutbound(ctx context.Context, record AcceptTraderOutboundRecord) (Run, error)
	RecalculateTeamleadPeriodInbound(ctx context.Context, record RecalculateTeamleadPeriodInboundRecord) (Run, error)
	LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error)
	LatestTeamleadInbound(ctx context.Context, teamID int64) (Run, error)
	ListItems(ctx context.Context, runID int64) ([]Item, error)
	ListActiveTeamleadInboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadInboundPeriodScope, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type Service struct {
	store Store
	audit AuditService
}

func NewService(store Store, auditService AuditService) *Service {
	return &Service{
		store: store,
		audit: auditService,
	}
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

func (s *Service) LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTraderInbound(ctx, teamID, traderID, shiftID)
}

func (s *Service) LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTraderOutbound(ctx, teamID, traderID, shiftID)
}

func (s *Service) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	if teamID <= 0 || accountingPeriodID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTeamleadPeriodInbound(ctx, teamID, accountingPeriodID)
}

func (s *Service) LatestTeamleadInbound(ctx context.Context, teamID int64) (Run, error) {
	if teamID <= 0 {
		return Run{}, ErrInvalidInput
	}

	return s.store.LatestTeamleadInbound(ctx, teamID)
}

func (s *Service) ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64) ([]Item, error) {
	run, err := s.LatestTeamleadPeriodInbound(ctx, teamID, accountingPeriodID)
	if err != nil {
		return nil, err
	}

	return s.store.ListItems(ctx, run.ID)
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
		return err
	default:
		return nil
	}
}

func (s *Service) afterTeamleadPeriodImportApplied(ctx context.Context, result imports.ApplyResult) error {
	if result.Batch.Direction != imports.DirectionInbound || result.Batch.AccountingPeriodID == nil {
		return nil
	}

	_, err := s.RecalculateTeamleadPeriodInbound(ctx, RecalculateTeamleadPeriodInboundParams{
		TeamID:             result.Batch.TeamID,
		AccountingPeriodID: *result.Batch.AccountingPeriodID,
		ImportBatchID:      &result.Batch.ID,
	})
	return err
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
