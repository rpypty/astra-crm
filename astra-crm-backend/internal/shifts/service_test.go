package shifts

import (
	"context"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

func TestServiceTakeRequisiteCreatesShiftWhenMissing(t *testing.T) {
	store := &fakeStore{
		currentErr:   ErrCurrentShiftNotFound,
		assignmentID: 100,
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	result, err := service.TakeRequisite(context.Background(), TakeRequisiteParams{
		ActorID:     1,
		TeamID:      2,
		TraderID:    3,
		RequisiteID: 4,
		CardNumber:  "1234 5678 9012 3456",
		HolderName:  "Иванов Иван",
	})
	if err != nil {
		t.Fatalf("TakeRequisite() error = %v", err)
	}

	if !result.ShiftCreated {
		t.Fatal("shift was not marked as created")
	}
	if store.createdShift.TeamID != 2 || store.createdShift.TraderID != 3 {
		t.Fatalf("created shift team/trader = %d/%d, want 2/3", store.createdShift.TeamID, store.createdShift.TraderID)
	}
	if store.createdShiftRequisite.AssignmentID != 100 {
		t.Fatalf("assignment id = %d, want 100", store.createdShiftRequisite.AssignmentID)
	}
	if len(auditService.events) != 2 {
		t.Fatalf("audit events count = %d, want 2", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionShiftCreated {
		t.Fatalf("first audit action = %q, want %q", auditService.events[0].Action, audit.ActionShiftCreated)
	}
	if auditService.events[1].Action != audit.ActionShiftRequisiteTaken {
		t.Fatalf("second audit action = %q, want %q", auditService.events[1].Action, audit.ActionShiftRequisiteTaken)
	}
}

func TestServiceTakeRequisiteRejectsUnassignedRequisite(t *testing.T) {
	service := NewService(&fakeStore{
		assignmentErr: ErrActiveAssignmentNotFound,
	}, nil)

	_, err := service.TakeRequisite(context.Background(), TakeRequisiteParams{
		ActorID:     1,
		TeamID:      2,
		TraderID:    3,
		RequisiteID: 4,
		CardNumber:  "1234",
		HolderName:  "Иванов Иван",
	})
	if err != ErrRequisiteNotAssigned {
		t.Fatalf("TakeRequisite() error = %v, want ErrRequisiteNotAssigned", err)
	}
}

func TestServiceUpdateShiftRequisiteRejectsEmptyDailyDetails(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.UpdateShiftRequisite(context.Background(), UpdateShiftRequisiteParams{
		ActorID:          1,
		TeamID:           2,
		TraderID:         3,
		ShiftRequisiteID: 4,
		CardNumber:       "",
		HolderName:       "Иванов Иван",
	})
	if err != ErrInvalidInput {
		t.Fatalf("UpdateShiftRequisite() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceCreateTurnoverAuditsMutation(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	comment := "на конец дня"
	entry, err := service.CreateTurnover(context.Background(), CreateTurnoverParams{
		ActorID:          1,
		TeamID:           2,
		TraderID:         3,
		ShiftRequisiteID: 20,
		AmountMinor:      150000,
		Comment:          &comment,
	})
	if err != nil {
		t.Fatalf("CreateTurnover() error = %v", err)
	}

	if entry.AmountMinor != 150000 {
		t.Fatalf("turnover amount = %d, want 150000", entry.AmountMinor)
	}
	if store.createdTurnover.ShiftRequisiteID != 20 {
		t.Fatalf("shift requisite id = %d, want 20", store.createdTurnover.ShiftRequisiteID)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionShiftTurnoverAdded {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionShiftTurnoverAdded)
	}
}

func TestServiceCreateTurnoverRejectsNegativeAmount(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.CreateTurnover(context.Background(), CreateTurnoverParams{
		ActorID:          1,
		TeamID:           2,
		TraderID:         3,
		ShiftRequisiteID: 20,
		AmountMinor:      -1,
	})
	if err != ErrInvalidInput {
		t.Fatalf("CreateTurnover() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceCloseShiftRequisiteStoresFinalTurnoversAndAudits(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	comment := "карта отработала"
	item, err := service.CloseShiftRequisite(context.Background(), CloseShiftRequisiteParams{
		ActorID:               1,
		TeamID:                2,
		TraderID:              3,
		ShiftRequisiteID:      20,
		InboundTurnoverMinor:  150000,
		OutboundTurnoverMinor: 25000,
		ClosingBalanceMinor:   7500,
		Blocked:               true,
		Comment:               &comment,
	})
	if err != nil {
		t.Fatalf("CloseShiftRequisite() error = %v", err)
	}

	if item.Status != RequisiteStatusBlocked {
		t.Fatalf("status = %q, want %q", item.Status, RequisiteStatusBlocked)
	}
	if store.closedShiftRequisite.InboundTurnoverMinor != 150000 || store.closedShiftRequisite.OutboundTurnoverMinor != 25000 {
		t.Fatalf("closed turnovers = %d/%d, want 150000/25000", store.closedShiftRequisite.InboundTurnoverMinor, store.closedShiftRequisite.OutboundTurnoverMinor)
	}
	if store.closedShiftRequisite.ClosingBalanceMinor != 7500 {
		t.Fatalf("closing balance = %d, want 7500", store.closedShiftRequisite.ClosingBalanceMinor)
	}
	if !store.closedShiftRequisite.Blocked {
		t.Fatal("blocked flag was not passed to store")
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionShiftRequisiteClosed {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionShiftRequisiteClosed)
	}
}

func TestServiceCloseShiftRequisiteRejectsNegativeTurnover(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.CloseShiftRequisite(context.Background(), CloseShiftRequisiteParams{
		ActorID:              1,
		TeamID:               2,
		TraderID:             3,
		ShiftRequisiteID:     20,
		InboundTurnoverMinor: -1,
	})
	if err != ErrInvalidInput {
		t.Fatalf("CloseShiftRequisite() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceCloseShiftRequisiteRejectsNegativeClosingBalance(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.CloseShiftRequisite(context.Background(), CloseShiftRequisiteParams{
		ActorID:               1,
		TeamID:                2,
		TraderID:              3,
		ShiftRequisiteID:      20,
		InboundTurnoverMinor:  100,
		OutboundTurnoverMinor: 0,
		ClosingBalanceMinor:   -1,
	})
	if err != ErrInvalidInput {
		t.Fatalf("CloseShiftRequisite() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceCorrectShiftRequisiteStoresCorrectionAuditsAndRecalculates(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	hook := &fakeTurnoverHook{}
	service := NewService(store, auditService, hook)

	item, err := service.CorrectShiftRequisite(context.Background(), CorrectShiftRequisiteParams{
		ActorID:               1,
		TeamID:                2,
		TraderID:              3,
		ShiftRequisiteID:      20,
		InboundTurnoverMinor:  150000,
		OutboundTurnoverMinor: 25000,
		ClosingBalanceMinor:   7500,
		Comment:               "исправили после сверки",
	})
	if err != nil {
		t.Fatalf("CorrectShiftRequisite() error = %v", err)
	}

	if store.correctedShiftRequisite.InboundTurnoverMinor != 150000 || store.correctedShiftRequisite.OutboundTurnoverMinor != 25000 {
		t.Fatalf("corrected turnovers = %d/%d, want 150000/25000", store.correctedShiftRequisite.InboundTurnoverMinor, store.correctedShiftRequisite.OutboundTurnoverMinor)
	}
	if store.correctedShiftRequisite.ClosingBalanceMinor != 7500 {
		t.Fatalf("closing balance = %d, want 7500", store.correctedShiftRequisite.ClosingBalanceMinor)
	}
	if store.correctedShiftRequisite.Comment != "исправили после сверки" {
		t.Fatalf("comment = %q, want correction comment", store.correctedShiftRequisite.Comment)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionShiftRequisiteCorrected {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionShiftRequisiteCorrected)
	}
	if hook.entry.ShiftRequisiteID != 20 || hook.entry.AmountMinor != 150000 {
		t.Fatalf("hook entry shift requisite/amount = %d/%d, want 20/150000", hook.entry.ShiftRequisiteID, hook.entry.AmountMinor)
	}
	if item.Status != RequisiteStatusVerified {
		t.Fatalf("returned status = %q, want %q", item.Status, RequisiteStatusVerified)
	}
}

func TestServiceCorrectShiftRequisiteRejectsEmptyComment(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.CorrectShiftRequisite(context.Background(), CorrectShiftRequisiteParams{
		ActorID:              1,
		TeamID:               2,
		TraderID:             3,
		ShiftRequisiteID:     20,
		InboundTurnoverMinor: 100,
		Comment:              "  ",
	})
	if err != ErrInvalidInput {
		t.Fatalf("CorrectShiftRequisite() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceReturnShiftRequisiteToWorkAuditsMutation(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	item, err := service.ReturnShiftRequisiteToWork(context.Background(), ReturnShiftRequisiteParams{
		ActorID:          1,
		TeamID:           2,
		TraderID:         3,
		ShiftRequisiteID: 20,
	})
	if err != nil {
		t.Fatalf("ReturnShiftRequisiteToWork() error = %v", err)
	}

	if store.returnedShiftRequisiteID != 20 {
		t.Fatalf("returned shift requisite id = %d, want 20", store.returnedShiftRequisiteID)
	}
	if item.Status != RequisiteStatusActive {
		t.Fatalf("status = %q, want %q", item.Status, RequisiteStatusActive)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionShiftRequisiteReturnedToWork {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionShiftRequisiteReturnedToWork)
	}
}

func TestServiceCloseCurrentBlocksWhenChecklistIncomplete(t *testing.T) {
	service := NewService(&fakeStore{
		checklist: CloseChecklist{
			Shift: Shift{
				ID:       10,
				TeamID:   2,
				TraderID: 3,
				Status:   StatusOpen,
			},
			InboundImported:     true,
			InboundOk:           true,
			OutboundImported:    false,
			OutboundOk:          false,
			AllPayoutsFullyPaid: true,
			CanClose:            false,
		},
	}, nil)

	_, err := service.CloseCurrent(context.Background(), CloseShiftParams{
		ActorID:  1,
		TeamID:   2,
		TraderID: 3,
	})
	if err != ErrCloseBlocked {
		t.Fatalf("CloseCurrent() error = %v, want ErrCloseBlocked", err)
	}
}

func TestServiceCloseCurrentBlocksWhenPayoutsAreUnpaidEvenIfReconciled(t *testing.T) {
	service := NewService(&fakeStore{
		checklist: CloseChecklist{
			Shift: Shift{
				ID:                           10,
				TeamID:                       2,
				TraderID:                     3,
				Status:                       StatusOpen,
				InboundReconciliationStatus:  "matched",
				OutboundReconciliationStatus: "matched",
			},
			InboundImported:     true,
			InboundOk:           true,
			OutboundImported:    true,
			OutboundOk:          true,
			AllPayoutsFullyPaid: false,
			UnpaidPayoutCount:   1,
			CanClose:            false,
		},
	}, nil)

	_, err := service.CloseCurrent(context.Background(), CloseShiftParams{
		ActorID:  1,
		TeamID:   2,
		TraderID: 3,
	})
	if err != ErrCloseBlocked {
		t.Fatalf("CloseCurrent() error = %v, want ErrCloseBlocked", err)
	}
}

func TestServiceCloseCurrentBlocksWhenRequisitesAreOpen(t *testing.T) {
	service := NewService(&fakeStore{
		checklist: CloseChecklist{
			Shift: Shift{
				ID:                           10,
				TeamID:                       2,
				TraderID:                     3,
				Status:                       StatusOpen,
				InboundReconciliationStatus:  "matched",
				OutboundReconciliationStatus: "matched",
			},
			InboundImported:     true,
			InboundOk:           true,
			OutboundImported:    true,
			OutboundOk:          true,
			AllPayoutsFullyPaid: true,
			CanClose:            true,
		},
		shiftRequisites: []ShiftRequisite{
			{
				ID:       20,
				ShiftID:  10,
				Status:   RequisiteStatusActive,
				TeamID:   2,
				TraderID: 3,
			},
		},
	}, nil)

	_, err := service.CloseCurrent(context.Background(), CloseShiftParams{
		ActorID:  1,
		TeamID:   2,
		TraderID: 3,
	})
	if err != ErrCloseBlocked {
		t.Fatalf("CloseCurrent() error = %v, want ErrCloseBlocked", err)
	}
}

func TestServiceCloseCurrentAuditsClosedShift(t *testing.T) {
	store := &fakeStore{
		checklist: CloseChecklist{
			Shift: Shift{
				ID:       10,
				TeamID:   2,
				TraderID: 3,
				Status:   StatusOpen,
			},
			InboundImported:     true,
			InboundOk:           true,
			OutboundImported:    true,
			OutboundOk:          true,
			AllPayoutsFullyPaid: true,
			CanClose:            true,
		},
		closedShift: Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   StatusClosed,
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	shift, err := service.CloseCurrent(context.Background(), CloseShiftParams{
		ActorID:  1,
		TeamID:   2,
		TraderID: 3,
	})
	if err != nil {
		t.Fatalf("CloseCurrent() error = %v", err)
	}

	if shift.Status != StatusClosed {
		t.Fatalf("closed shift status = %q, want %q", shift.Status, StatusClosed)
	}
	if store.closeRecord.ShiftID != 10 {
		t.Fatalf("closed shift id = %d, want 10", store.closeRecord.ShiftID)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionShiftClosed {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionShiftClosed)
	}
}

type fakeStore struct {
	currentErr               error
	currentShift             Shift
	createdShift             Shift
	assignmentID             int64
	assignmentErr            error
	createdShiftRequisite    CreateShiftRequisiteRecord
	createdTurnover          CreateTurnoverEntryRecord
	closedShiftRequisite     CloseShiftRequisiteRecord
	correctedShiftRequisite  CorrectShiftRequisiteTurnoversRecord
	refreshedShiftRequisite  ShiftRequisite
	returnedShiftRequisiteID int64
	shiftRequisites          []ShiftRequisite
	checklist                CloseChecklist
	closeRecord              CloseShiftRecord
	closedShift              Shift
	closeErr                 error
}

func (s *fakeStore) CurrentShift(ctx context.Context, teamID int64, traderID int64) (Shift, error) {
	if s.currentErr != nil {
		return Shift{}, s.currentErr
	}
	if s.currentShift.ID != 0 {
		return s.currentShift, nil
	}
	return Shift{ID: 10, TeamID: teamID, TraderID: traderID, Status: StatusOpen}, nil
}

func (s *fakeStore) CreateShift(ctx context.Context, teamID int64, traderID int64) (Shift, error) {
	s.createdShift = Shift{
		ID:                           10,
		TeamID:                       teamID,
		TraderID:                     traderID,
		StartedAt:                    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                       StatusOpen,
		InboundReconciliationStatus:  "not_started",
		OutboundReconciliationStatus: "not_started",
		CreatedAt:                    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:                    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}
	return s.createdShift, nil
}

func (s *fakeStore) ShiftHistory(ctx context.Context, teamID int64, traderID int64, limit int32) ([]Shift, error) {
	return nil, nil
}

func (s *fakeStore) TeamShiftHistory(ctx context.Context, teamID int64, limit int32) ([]Shift, error) {
	return nil, nil
}

func (s *fakeStore) ShiftReport(ctx context.Context, teamID int64, traderID int64, shiftID int64) (ShiftReportDetails, error) {
	return ShiftReportDetails{}, nil
}

func (s *fakeStore) TeamShiftReport(ctx context.Context, teamID int64, shiftID int64) (ShiftReportDetails, error) {
	return ShiftReportDetails{}, nil
}

func (s *fakeStore) ActiveAssignment(ctx context.Context, teamID int64, traderID int64, requisiteID int64) (int64, error) {
	if s.assignmentErr != nil {
		return 0, s.assignmentErr
	}
	return s.assignmentID, nil
}

func (s *fakeStore) AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeStore) FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeStore) HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeStore) AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64) ([]AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeStore) AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64) ([]AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeStore) CreateShiftRequisite(ctx context.Context, params CreateShiftRequisiteRecord) (ShiftRequisite, error) {
	s.createdShiftRequisite = params
	return ShiftRequisite{
		ID:           20,
		TeamID:       params.TeamID,
		ShiftID:      params.ShiftID,
		TraderID:     params.TraderID,
		RequisiteID:  params.RequisiteID,
		AssignmentID: &params.AssignmentID,
		CardNumber:   params.CardNumber,
		HolderName:   params.HolderName,
		TakenAt:      time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:       RequisiteStatusActive,
		CreatedAt:    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]ShiftRequisite, error) {
	if s.shiftRequisites != nil {
		return s.shiftRequisites, nil
	}
	return nil, nil
}

func (s *fakeStore) UpdateShiftRequisiteDetails(ctx context.Context, params UpdateShiftRequisiteDetailsRecord) (ShiftRequisite, error) {
	return ShiftRequisite{}, nil
}

func (s *fakeStore) CreateTurnoverEntry(ctx context.Context, params CreateTurnoverEntryRecord) (TurnoverEntry, error) {
	s.createdTurnover = params
	return TurnoverEntry{
		ID:               30,
		TeamID:           params.TeamID,
		ShiftID:          10,
		ShiftRequisiteID: params.ShiftRequisiteID,
		RequisiteID:      4,
		TraderID:         params.TraderID,
		AmountMinor:      params.AmountMinor,
		CreatedBy:        params.CreatedBy,
		CreatedAt:        time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
		Comment:          params.Comment,
	}, nil
}

func (s *fakeStore) CloseShiftRequisite(ctx context.Context, params CloseShiftRequisiteRecord) (ShiftRequisite, error) {
	s.closedShiftRequisite = params
	status := RequisiteStatusWorked
	if params.Blocked {
		status = RequisiteStatusBlocked
	}
	return ShiftRequisite{
		ID:                    params.ShiftRequisiteID,
		TeamID:                params.TeamID,
		ShiftID:               10,
		TraderID:              params.TraderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                status,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) CorrectClosedShiftRequisiteTurnovers(ctx context.Context, params CorrectShiftRequisiteTurnoversRecord) (ShiftRequisite, error) {
	s.correctedShiftRequisite = params
	return ShiftRequisite{
		ID:                    params.ShiftRequisiteID,
		TeamID:                params.TeamID,
		ShiftID:               10,
		TraderID:              params.TraderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                RequisiteStatusWorked,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) GetShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) (ShiftRequisite, error) {
	if s.refreshedShiftRequisite.ID != 0 {
		return s.refreshedShiftRequisite, nil
	}
	return ShiftRequisite{
		ID:                    shiftRequisiteID,
		TeamID:                teamID,
		ShiftID:               10,
		TraderID:              traderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                RequisiteStatusVerified,
		InboundTurnoverMinor:  s.correctedShiftRequisite.InboundTurnoverMinor,
		OutboundTurnoverMinor: s.correctedShiftRequisite.OutboundTurnoverMinor,
		ClosingBalanceMinor:   s.correctedShiftRequisite.ClosingBalanceMinor,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) ReturnShiftRequisiteToWork(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) (ShiftRequisite, error) {
	s.returnedShiftRequisiteID = shiftRequisiteID
	return ShiftRequisite{
		ID:                    shiftRequisiteID,
		TeamID:                teamID,
		ShiftID:               10,
		TraderID:              traderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                RequisiteStatusActive,
		InboundTurnoverMinor:  150000,
		OutboundTurnoverMinor: 25000,
		ClosingBalanceMinor:   7500,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]TurnoverEntry, error) {
	return nil, nil
}

func (s *fakeStore) TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]TurnoverEntry, error) {
	return nil, nil
}

func (s *fakeStore) CurrentShiftChecklist(ctx context.Context, teamID int64, traderID int64) (CloseChecklist, error) {
	if s.checklist.Shift.ID != 0 {
		return s.checklist, nil
	}

	return CloseChecklist{
		Shift: Shift{
			ID:       10,
			TeamID:   teamID,
			TraderID: traderID,
			Status:   StatusOpen,
		},
		InboundImported:     true,
		InboundOk:           true,
		OutboundImported:    true,
		OutboundOk:          true,
		AllPayoutsFullyPaid: true,
		CanClose:            true,
	}, nil
}

func (s *fakeStore) CloseCurrentShift(ctx context.Context, params CloseShiftRecord) (Shift, error) {
	s.closeRecord = params
	if s.closeErr != nil {
		return Shift{}, s.closeErr
	}
	if s.closedShift.ID != 0 {
		return s.closedShift, nil
	}
	return Shift{
		ID:       params.ShiftID,
		TeamID:   params.TeamID,
		TraderID: params.TraderID,
		Status:   StatusClosed,
	}, nil
}

type fakeAuditService struct {
	events []audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.events = append(s.events, event)
	return nil
}

type fakeTurnoverHook struct {
	entry TurnoverEntry
}

func (h *fakeTurnoverHook) AfterTurnoverCreated(ctx context.Context, entry TurnoverEntry) error {
	h.entry = entry
	return nil
}
