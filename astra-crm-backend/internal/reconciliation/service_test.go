package reconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
)

func TestServiceAfterImportAppliedRecalculatesTraderInbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionInbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if store.recalculateRecord.TeamID != 2 || store.recalculateRecord.TraderID != 3 || store.recalculateRecord.ShiftID != 10 {
		t.Fatalf("recalculate record = %+v, want team/trader/shift 2/3/10", store.recalculateRecord)
	}
	if store.recalculateRecord.ImportBatchID == nil || *store.recalculateRecord.ImportBatchID != 100 {
		t.Fatalf("import batch id = %v, want 100", store.recalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterImportAppliedRecalculatesActiveTeamleadPeriodsAfterTraderInbound(t *testing.T) {
	store := &fakeStore{
		activeTeamleadScopes: []TeamleadInboundPeriodScope{
			{AccountingPeriodID: 55, ImportBatchID: 500},
			{AccountingPeriodID: 56, ImportBatchID: 501},
		},
	}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionInbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if len(store.teamleadPeriodRecalculateRecords) != 2 {
		t.Fatalf("teamlead period recalculate count = %d, want 2", len(store.teamleadPeriodRecalculateRecords))
	}
	if store.teamleadPeriodRecalculateRecords[0].AccountingPeriodID != 55 || store.teamleadPeriodRecalculateRecords[1].AccountingPeriodID != 56 {
		t.Fatalf("teamlead period recalculate records = %+v, want periods 55 and 56", store.teamleadPeriodRecalculateRecords)
	}
}

func TestServiceAfterImportAppliedRecalculatesTraderOutbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionOutbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}
	if store.outboundRecalculateRecord.TeamID != 2 || store.outboundRecalculateRecord.TraderID != 3 || store.outboundRecalculateRecord.ShiftID != 10 {
		t.Fatalf("outbound recalculate record = %+v, want team/trader/shift 2/3/10", store.outboundRecalculateRecord)
	}
	if store.outboundRecalculateRecord.ImportBatchID == nil || *store.outboundRecalculateRecord.ImportBatchID != 100 {
		t.Fatalf("outbound import batch id = %v, want 100", store.outboundRecalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterImportAppliedRecalculatesTeamleadPeriodInbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	periodID := int64(55)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:                 500,
			TeamID:             2,
			ScopeType:          imports.ScopeTypeTeamleadPeriod,
			Direction:          imports.DirectionInbound,
			AccountingPeriodID: &periodID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if len(store.teamleadPeriodRecalculateRecords) != 1 {
		t.Fatalf("teamlead period recalculate count = %d, want 1", len(store.teamleadPeriodRecalculateRecords))
	}
	record := store.teamleadPeriodRecalculateRecords[0]
	if record.TeamID != 2 || record.AccountingPeriodID != 55 {
		t.Fatalf("teamlead period recalculate record = %+v, want team/period 2/55", record)
	}
	if record.ImportBatchID == nil || *record.ImportBatchID != 500 {
		t.Fatalf("teamlead import batch id = %v, want 500", record.ImportBatchID)
	}
}

func TestServiceAfterTurnoverCreatedRerunsExistingInboundRun(t *testing.T) {
	importBatchID := int64(100)
	store := &fakeStore{
		latestRun: Run{
			ID:            50,
			TeamID:        2,
			ShiftID:       int64Ptr(10),
			TraderID:      int64Ptr(3),
			ImportBatchID: &importBatchID,
			Status:        StatusMismatch,
			CreatedAt:     time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	service := NewService(store, nil)

	err := service.AfterTurnoverCreated(context.Background(), shifts.TurnoverEntry{
		ID:       30,
		TeamID:   2,
		ShiftID:  10,
		TraderID: 3,
	})
	if err != nil {
		t.Fatalf("AfterTurnoverCreated() error = %v", err)
	}
	if !store.latestCalled || !store.recalculateCalled {
		t.Fatalf("latest/recalculate called = %v/%v, want true/true", store.latestCalled, store.recalculateCalled)
	}
	if store.recalculateRecord.ImportBatchID == nil || *store.recalculateRecord.ImportBatchID != 100 {
		t.Fatalf("import batch id = %v, want 100", store.recalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterTurnoverCreatedIgnoresMissingInboundRun(t *testing.T) {
	store := &fakeStore{latestErr: ErrRunNotFound}
	service := NewService(store, nil)

	err := service.AfterTurnoverCreated(context.Background(), shifts.TurnoverEntry{
		ID:       30,
		TeamID:   2,
		ShiftID:  10,
		TraderID: 3,
	})
	if err != nil {
		t.Fatalf("AfterTurnoverCreated() error = %v", err)
	}
	if store.recalculateCalled {
		t.Fatal("recalculate was called without existing inbound run")
	}
}

func TestServiceAfterManualPayoutChangedRerunsExistingOutboundRun(t *testing.T) {
	importBatchID := int64(200)
	store := &fakeStore{
		latestOutboundRun: Run{
			ID:            70,
			TeamID:        2,
			ShiftID:       int64Ptr(10),
			TraderID:      int64Ptr(3),
			ImportBatchID: &importBatchID,
			Status:        StatusMismatch,
			CreatedAt:     time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	service := NewService(store, nil)

	err := service.AfterManualPayoutChanged(context.Background(), 2, 3, 10)
	if err != nil {
		t.Fatalf("AfterManualPayoutChanged() error = %v", err)
	}
	if !store.latestOutboundCalled || !store.outboundRecalculateCalled {
		t.Fatalf("latest outbound/recalculate outbound called = %v/%v, want true/true", store.latestOutboundCalled, store.outboundRecalculateCalled)
	}
	if store.outboundRecalculateRecord.ImportBatchID == nil || *store.outboundRecalculateRecord.ImportBatchID != 200 {
		t.Fatalf("outbound import batch id = %v, want 200", store.outboundRecalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterManualPayoutChangedIgnoresMissingOutboundRun(t *testing.T) {
	store := &fakeStore{latestOutboundErr: ErrRunNotFound}
	service := NewService(store, nil)

	err := service.AfterManualPayoutChanged(context.Background(), 2, 3, 10)
	if err != nil {
		t.Fatalf("AfterManualPayoutChanged() error = %v", err)
	}
	if store.outboundRecalculateCalled {
		t.Fatal("outbound recalculate was called without existing outbound run")
	}
}

func TestServiceAcceptTraderInboundRequiresComment(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.AcceptTraderInbound(context.Background(), AcceptTraderInboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    50,
		Comment:  " ",
	})
	if err != ErrInvalidInput {
		t.Fatalf("AcceptTraderInbound() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceAcceptTraderOutboundRequiresComment(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.AcceptTraderOutbound(context.Background(), AcceptTraderOutboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    50,
		Comment:  " ",
	})
	if err != ErrInvalidInput {
		t.Fatalf("AcceptTraderOutbound() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceAcceptTraderInboundAuditsMutation(t *testing.T) {
	store := &fakeStore{
		acceptedRun: Run{
			ID:        50,
			TeamID:    2,
			Type:      TypeTraderShiftInbound,
			ScopeType: imports.ScopeTypeTraderShift,
			ShiftID:   int64Ptr(10),
			TraderID:  int64Ptr(3),
			Status:    StatusAcceptedWithComment,
			Comment:   stringPtr("accepted"),
			CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	run, err := service.AcceptTraderInbound(context.Background(), AcceptTraderInboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    50,
		Comment:  " accepted ",
	})
	if err != nil {
		t.Fatalf("AcceptTraderInbound() error = %v", err)
	}
	if run.Status != StatusAcceptedWithComment {
		t.Fatalf("run status = %q, want accepted_with_comment", run.Status)
	}
	if store.acceptRecord.Comment != "accepted" {
		t.Fatalf("stored comment = %q, want trimmed comment", store.acceptRecord.Comment)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionReconciliationAcceptedWithComment {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionReconciliationAcceptedWithComment)
	}
}

func TestServiceAcceptTraderOutboundAuditsMutation(t *testing.T) {
	store := &fakeStore{
		acceptedOutboundRun: Run{
			ID:        60,
			TeamID:    2,
			Type:      TypeTraderShiftOutbound,
			ScopeType: imports.ScopeTypeTraderShift,
			ShiftID:   int64Ptr(10),
			TraderID:  int64Ptr(3),
			Status:    StatusAcceptedWithComment,
			Comment:   stringPtr("accepted"),
			CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	run, err := service.AcceptTraderOutbound(context.Background(), AcceptTraderOutboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    60,
		Comment:  " accepted ",
	})
	if err != nil {
		t.Fatalf("AcceptTraderOutbound() error = %v", err)
	}
	if run.Status != StatusAcceptedWithComment {
		t.Fatalf("run status = %q, want accepted_with_comment", run.Status)
	}
	if store.acceptOutboundRecord.Comment != "accepted" {
		t.Fatalf("stored outbound comment = %q, want trimmed comment", store.acceptOutboundRecord.Comment)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionReconciliationAcceptedWithComment {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionReconciliationAcceptedWithComment)
	}
}

type fakeStore struct {
	recalculateCalled                bool
	recalculateRecord                RecalculateTraderInboundRecord
	outboundRecalculateCalled        bool
	outboundRecalculateRecord        RecalculateTraderOutboundRecord
	teamleadPeriodRecalculateRecords []RecalculateTeamleadPeriodInboundRecord
	latestCalled                     bool
	latestRun                        Run
	latestErr                        error
	latestOutboundCalled             bool
	latestOutboundRun                Run
	latestOutboundErr                error
	acceptRecord                     AcceptTraderInboundRecord
	acceptedRun                      Run
	acceptOutboundRecord             AcceptTraderOutboundRecord
	acceptedOutboundRun              Run
	activeTeamleadScopes             []TeamleadInboundPeriodScope
	items                            []Item
}

func (s *fakeStore) RecalculateTraderInbound(ctx context.Context, record RecalculateTraderInboundRecord) (Run, error) {
	s.recalculateCalled = true
	s.recalculateRecord = record
	return Run{
		ID:        51,
		TeamID:    record.TeamID,
		Type:      TypeTraderShiftInbound,
		ScopeType: imports.ScopeTypeTraderShift,
		ShiftID:   &record.ShiftID,
		TraderID:  &record.TraderID,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) RecalculateTraderOutbound(ctx context.Context, record RecalculateTraderOutboundRecord) (Run, error) {
	s.outboundRecalculateCalled = true
	s.outboundRecalculateRecord = record
	return Run{
		ID:        61,
		TeamID:    record.TeamID,
		Type:      TypeTraderShiftOutbound,
		ScopeType: imports.ScopeTypeTraderShift,
		ShiftID:   &record.ShiftID,
		TraderID:  &record.TraderID,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) RecalculateTeamleadPeriodInbound(ctx context.Context, record RecalculateTeamleadPeriodInboundRecord) (Run, error) {
	s.teamleadPeriodRecalculateRecords = append(s.teamleadPeriodRecalculateRecords, record)
	return Run{
		ID:                 71,
		TeamID:             record.TeamID,
		Type:               TypeTeamleadPeriodInbound,
		ScopeType:          imports.ScopeTypeTeamleadPeriod,
		AccountingPeriodID: &record.AccountingPeriodID,
		ImportBatchID:      record.ImportBatchID,
		Status:             StatusMatched,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	s.latestCalled = true
	if s.latestErr != nil {
		return Run{}, s.latestErr
	}
	return s.latestRun, nil
}

func (s *fakeStore) LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	s.latestOutboundCalled = true
	if s.latestOutboundErr != nil {
		return Run{}, s.latestOutboundErr
	}
	return s.latestOutboundRun, nil
}

func (s *fakeStore) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	return Run{
		ID:                 71,
		TeamID:             teamID,
		Type:               TypeTeamleadPeriodInbound,
		ScopeType:          imports.ScopeTypeTeamleadPeriod,
		AccountingPeriodID: &accountingPeriodID,
		Status:             StatusMatched,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTeamleadInbound(ctx context.Context, teamID int64) (Run, error) {
	return Run{
		ID:        71,
		TeamID:    teamID,
		Type:      TypeTeamleadPeriodInbound,
		ScopeType: imports.ScopeTypeTeamleadPeriod,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) ListItems(ctx context.Context, runID int64) ([]Item, error) {
	return s.items, nil
}

func (s *fakeStore) ListActiveTeamleadInboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadInboundPeriodScope, error) {
	return s.activeTeamleadScopes, nil
}

func (s *fakeStore) AcceptTraderInbound(ctx context.Context, record AcceptTraderInboundRecord) (Run, error) {
	s.acceptRecord = record
	return s.acceptedRun, nil
}

func (s *fakeStore) AcceptTraderOutbound(ctx context.Context, record AcceptTraderOutboundRecord) (Run, error) {
	s.acceptOutboundRecord = record
	return s.acceptedOutboundRun, nil
}

type fakeAuditService struct {
	events []audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.events = append(s.events, event)
	return nil
}

func int64Ptr(value int64) *int64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
