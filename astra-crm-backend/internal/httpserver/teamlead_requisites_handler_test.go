package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/requisites"
	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func TestTeamleadRequisitesHandlerCreateReturnsBaseRequisiteOnly(t *testing.T) {
	proxy := "192.168.1.1:8080"
	service := &fakeTeamleadRequisiteService{
		createResult: requisites.RequisiteDetails{
			Requisite: requisites.Requisite{
				ID:         10,
				TeamID:     2,
				Phone:      "+79991234567",
				MethodType: "sbp",
				Proxy:      &proxy,
				Status:     requisites.StatusActive,
				CreatedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewTeamleadRequisitesHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/requisites", strings.NewReader(`{
		"phone": "+79991234567",
		"methodType": "sbp",
		"proxy": "192.168.1.1:8080"
	}`))
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "card") || strings.Contains(response.Body.String(), "holder") {
		t.Fatalf("daily card/holder fields leaked into base requisite response: %s", response.Body.String())
	}
	if service.createParams.ActorID != 1 || service.createParams.TeamID != 2 {
		t.Fatalf("create actor/team = %d/%d, want 1/2", service.createParams.ActorID, service.createParams.TeamID)
	}
}

func TestTeamleadRequisitesHandlerAssignValidatesTraderID(t *testing.T) {
	handler := NewTeamleadRequisitesHandler(&fakeTeamleadRequisiteService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/requisites/10/assign", strings.NewReader(`{"traderId":0}`))
	request = request.WithContext(withRequisiteID(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "10"))

	handler.Assign(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "traderId") {
		t.Fatalf("response does not mention traderId: %s", response.Body.String())
	}
}

func TestTeamleadRequisitesHandlerAssignmentHistoryReturnsItems(t *testing.T) {
	service := &fakeTeamleadRequisiteService{
		history: []requisites.Assignment{
			{
				ID:          100,
				TeamID:      2,
				RequisiteID: 10,
				TraderID:    30,
				AssignedBy:  1,
				AssignedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewTeamleadRequisitesHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/requisites/10/assignment-history", nil)
	request = request.WithContext(withRequisiteID(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "10"))

	handler.AssignmentHistory(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"requisiteId":10`) {
		t.Fatalf("history response does not contain assignment: %s", response.Body.String())
	}
}

func withRequisiteID(ctx context.Context, id string) context.Context {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("requisiteId", id)
	return context.WithValue(ctx, chi.RouteCtxKey, routeContext)
}

type fakeTeamleadRequisiteService struct {
	createParams requisites.CreateParams
	createResult requisites.RequisiteDetails
	history      []requisites.Assignment
}

func (s *fakeTeamleadRequisiteService) Create(ctx context.Context, params requisites.CreateParams) (requisites.RequisiteDetails, error) {
	s.createParams = params
	return s.createResult, nil
}

func (s *fakeTeamleadRequisiteService) List(ctx context.Context, teamID int64) ([]requisites.RequisiteDetails, error) {
	return nil, nil
}

func (s *fakeTeamleadRequisiteService) Get(ctx context.Context, teamID int64, requisiteID int64) (requisites.RequisiteDetails, error) {
	return requisites.RequisiteDetails{}, nil
}

func (s *fakeTeamleadRequisiteService) Patch(ctx context.Context, params requisites.PatchParams) (requisites.RequisiteDetails, error) {
	return requisites.RequisiteDetails{}, nil
}

func (s *fakeTeamleadRequisiteService) Delete(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error {
	return nil
}

func (s *fakeTeamleadRequisiteService) Assign(ctx context.Context, params requisites.AssignParams) (requisites.Assignment, error) {
	return requisites.Assignment{}, nil
}

func (s *fakeTeamleadRequisiteService) Unassign(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error {
	return nil
}

func (s *fakeTeamleadRequisiteService) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64) ([]requisites.Assignment, error) {
	return s.history, nil
}
