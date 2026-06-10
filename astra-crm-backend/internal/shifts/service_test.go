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
	currentErr            error
	currentShift          Shift
	createdShift          Shift
	assignmentID          int64
	assignmentErr         error
	createdShiftRequisite CreateShiftRequisiteRecord
	createdTurnover       CreateTurnoverEntryRecord
	checklist             CloseChecklist
	closeRecord           CloseShiftRecord
	closedShift           Shift
	closeErr              error
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

func (s *fakeStore) ActiveAssignment(ctx context.Context, teamID int64, traderID int64, requisiteID int64) (int64, error) {
	if s.assignmentErr != nil {
		return 0, s.assignmentErr
	}
	return s.assignmentID, nil
}

func (s *fakeStore) AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
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
