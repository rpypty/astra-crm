package requisites

import (
	"context"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestServiceCreateWithAssignedTraderCreatesAssignmentAndAudits(t *testing.T) {
	store := &fakeStore{}
	traderID := int64(30)
	service := NewService(store, &fakeTraderReader{
		trader: users.Trader{
			ID:     traderID,
			TeamID: 2,
			Status: users.StatusActive,
		},
	}, &fakeAuditService{})

	requisite, err := service.Create(context.Background(), CreateParams{
		ActorID:          1,
		TeamID:           2,
		Phone:            "+79991234567",
		MethodType:       "sbp",
		AssignedTraderID: &traderID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if requisite.AssignedTraderID == nil || *requisite.AssignedTraderID != traderID {
		t.Fatalf("assigned trader id = %v, want %d", requisite.AssignedTraderID, traderID)
	}
	if store.assigned.TraderID != traderID {
		t.Fatalf("assigned trader id in store = %d, want %d", store.assigned.TraderID, traderID)
	}
}

func TestServiceAssignRejectsDisabledTrader(t *testing.T) {
	service := NewService(&fakeStore{
		details: RequisiteDetails{
			Requisite: Requisite{
				ID:     10,
				TeamID: 2,
				Status: StatusActive,
			},
		},
	}, &fakeTraderReader{
		trader: users.Trader{
			ID:     30,
			TeamID: 2,
			Status: users.StatusDisabled,
		},
	}, nil)

	_, err := service.Assign(context.Background(), AssignParams{
		ActorID:     1,
		TeamID:      2,
		RequisiteID: 10,
		TraderID:    30,
	})
	if err != ErrInactiveTrader {
		t.Fatalf("Assign() error = %v, want ErrInactiveTrader", err)
	}
}

func TestServiceAssignAuditsReassignAction(t *testing.T) {
	store := &fakeStore{
		details: RequisiteDetails{
			Requisite: Requisite{
				ID:     10,
				TeamID: 2,
				Status: StatusActive,
			},
		},
		assignWasReassign: true,
	}
	auditService := &fakeAuditService{}
	service := NewService(store, &fakeTraderReader{
		trader: users.Trader{
			ID:     30,
			TeamID: 2,
			Status: users.StatusActive,
		},
	}, auditService)

	_, err := service.Assign(context.Background(), AssignParams{
		ActorID:     1,
		TeamID:      2,
		RequisiteID: 10,
		TraderID:    30,
	})
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}
	if auditService.events[len(auditService.events)-1].Action != audit.ActionRequisiteReassigned {
		t.Fatalf("audit action = %q, want %q", auditService.events[len(auditService.events)-1].Action, audit.ActionRequisiteReassigned)
	}
}

type fakeStore struct {
	created           CreateRecord
	updated           UpdateRecord
	assigned          AssignRecord
	details           RequisiteDetails
	assignWasReassign bool
}

func (s *fakeStore) Create(ctx context.Context, params CreateRecord) (Requisite, error) {
	s.created = params
	return Requisite{
		ID:         10,
		TeamID:     params.TeamID,
		Phone:      params.Phone,
		MethodType: params.MethodType,
		Proxy:      params.Proxy,
		Status:     StatusActive,
		CreatedBy:  params.CreatedBy,
		CreatedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) GetDetails(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error) {
	if s.details.ID != 0 {
		return s.details, nil
	}
	return RequisiteDetails{
		Requisite: Requisite{
			ID:     requisiteID,
			TeamID: teamID,
			Status: StatusActive,
		},
	}, nil
}

func (s *fakeStore) ListDetails(ctx context.Context, teamID int64) ([]RequisiteDetails, error) {
	return nil, nil
}

func (s *fakeStore) Update(ctx context.Context, params UpdateRecord) (Requisite, error) {
	s.updated = params
	return Requisite{
		ID:         params.RequisiteID,
		TeamID:     params.TeamID,
		Phone:      params.Phone,
		MethodType: params.MethodType,
		Proxy:      params.Proxy,
		Status:     params.Status,
	}, nil
}

func (s *fakeStore) Assign(ctx context.Context, params AssignRecord) (Assignment, error) {
	s.assigned = params
	return Assignment{
		ID:          100,
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
		TraderID:    params.TraderID,
		AssignedBy:  params.AssignedBy,
		AssignedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		WasReassign: s.assignWasReassign,
	}, nil
}

func (s *fakeStore) Unassign(ctx context.Context, teamID int64, requisiteID int64) (Assignment, error) {
	return Assignment{}, nil
}

func (s *fakeStore) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64) ([]Assignment, error) {
	return nil, nil
}

type fakeTraderReader struct {
	trader users.Trader
	err    error
}

func (r *fakeTraderReader) GetTraderByID(ctx context.Context, teamID int64, traderID int64) (users.Trader, error) {
	if r.err != nil {
		return users.Trader{}, r.err
	}
	return r.trader, nil
}

type fakeAuditService struct {
	events []audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.events = append(s.events, event)
	return nil
}
