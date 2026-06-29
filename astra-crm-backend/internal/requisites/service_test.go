package requisites

import (
	"context"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/pagination"
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
		BankCode:         "sber",
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

func TestServiceCreatePlanAllowsPastDateForTestBackfill(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, &fakeTraderReader{
		trader: users.Trader{
			ID:     30,
			TeamID: 2,
			Status: users.StatusActive,
		},
	}, nil)

	pastDate := time.Now().AddDate(0, 0, -3)
	_, err := service.CreatePlan(context.Background(), PlanParams{
		ActorID:             1,
		TeamID:              2,
		RequisiteID:         10,
		TraderID:            30,
		AssignedForDate:     pastDate,
		TargetTurnoverMinor: 100000,
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if !store.createdPlan.AssignedForDate.Equal(normalizeDate(pastDate)) {
		t.Fatalf("assigned for date = %v, want %v", store.createdPlan.AssignedForDate, normalizeDate(pastDate))
	}
}

func TestServiceListFiltersByBackendParams(t *testing.T) {
	traderID := int64(84)
	store := &fakeStore{
		listItems: []RequisiteDetails{
			{
				Requisite: Requisite{
					ID:       1,
					Phone:    "79021002004",
					BankCode: "sber",
					BankName: "Сбер",
					Status:   StatusActive,
				},
				AssignedTraderID: &traderID,
			},
			{
				Requisite: Requisite{
					ID:       2,
					Phone:    "+7 (903) 748-12-97",
					BankCode: "ozon",
					BankName: "Ozon Банк",
					Status:   StatusActive,
				},
			},
			{
				Requisite: Requisite{
					ID:       3,
					Phone:    "79001002000",
					BankCode: "sber",
					BankName: "Сбер",
					Status:   StatusArchived,
				},
			},
		},
	}
	service := NewService(store, nil, nil)

	items, err := service.List(context.Background(), 2, ListParams{
		Search: "1002",
		Status: StatusActive,
	}, pagination.Params{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items.Items) != 1 || items.Items[0].ID != 1 {
		t.Fatalf("List() by phone digits returned ids %v, want [1]", requisiteIDs(items.Items))
	}

	items, err = service.List(context.Background(), 2, ListParams{
		Search: "1297",
	}, pagination.Params{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items.Items) != 1 || items.Items[0].ID != 2 {
		t.Fatalf("List() by formatted phone substring returned ids %v, want [2]", requisiteIDs(items.Items))
	}

	items, err = service.List(context.Background(), 2, ListParams{
		BankCode: "sber",
		TraderID: "84",
		Status:   StatusActive,
	}, pagination.Params{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items.Items) != 1 || items.Items[0].ID != 1 {
		t.Fatalf("List() by bank/trader/status returned ids %v, want [1]", requisiteIDs(items.Items))
	}

	items, err = service.List(context.Background(), 2, ListParams{
		TraderID: "unassigned",
		Status:   StatusActive,
	}, pagination.Params{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items.Items) != 1 || items.Items[0].ID != 2 {
		t.Fatalf("List() by unassigned returned ids %v, want [2]", requisiteIDs(items.Items))
	}
}

func requisiteIDs(items []RequisiteDetails) []int64 {
	result := make([]int64, 0, len(items))
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

type fakeStore struct {
	created           CreateRecord
	updated           UpdateRecord
	assigned          AssignRecord
	createdPlan       CreatePlanRecord
	details           RequisiteDetails
	listItems         []RequisiteDetails
	assignWasReassign bool
}

func (s *fakeStore) Create(ctx context.Context, params CreateRecord) (Requisite, error) {
	s.created = params
	return Requisite{
		ID:         10,
		TeamID:     params.TeamID,
		Phone:      params.Phone,
		MethodType: params.MethodType,
		BankCode:   params.BankCode,
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

func (s *fakeStore) ListDetails(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[RequisiteDetails], error) {
	params = normalizeListParams(params)
	items := make([]RequisiteDetails, 0, len(s.listItems))
	for _, item := range s.listItems {
		if requisiteMatchesListParams(item, params) {
			items = append(items, item)
		}
	}

	return pagination.FromSlice(items, page), nil
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

func (s *fakeStore) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64, page pagination.Params) (pagination.Result[Assignment], error) {
	return pagination.FromSlice([]Assignment{}, page), nil
}

func (s *fakeStore) CreatePlan(ctx context.Context, params CreatePlanRecord) (Assignment, error) {
	s.createdPlan = params
	return Assignment{
		ID:                  200,
		TeamID:              params.TeamID,
		RequisiteID:         params.RequisiteID,
		TraderID:            params.TraderID,
		AssignedBy:          params.AssignedBy,
		AssignedForDate:     params.AssignedForDate,
		TargetTurnoverMinor: params.TargetTurnoverMinor,
		AssignedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:              AssignmentStatusPlanned,
	}, nil
}

func (s *fakeStore) ListPlans(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[AssignmentWorkRow], error) {
	return pagination.FromSlice([]AssignmentWorkRow{}, page), nil
}

func (s *fakeStore) ListActivity(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[AssignmentWorkRow], error) {
	return pagination.FromSlice([]AssignmentWorkRow{}, page), nil
}

func (s *fakeStore) GetAssignment(ctx context.Context, teamID int64, assignmentID int64) (Assignment, error) {
	return Assignment{}, nil
}

func (s *fakeStore) UpdatePlan(ctx context.Context, params UpdatePlanRecord) (Assignment, error) {
	return Assignment{}, nil
}

func (s *fakeStore) CancelPlan(ctx context.Context, teamID int64, assignmentID int64) (Assignment, error) {
	return Assignment{}, nil
}

func (s *fakeStore) CreateAssignmentEvent(ctx context.Context, params AssignmentEventRecord) (AssignmentEvent, error) {
	return AssignmentEvent{}, nil
}

func (s *fakeStore) AssignmentEvents(ctx context.Context, teamID int64, assignmentID int64, page pagination.Params) (pagination.Result[AssignmentEvent], error) {
	return pagination.FromSlice([]AssignmentEvent{}, page), nil
}

func (s *fakeStore) Report(ctx context.Context, teamID int64, requisiteID int64) (RequisiteReport, error) {
	return RequisiteReport{}, nil
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
