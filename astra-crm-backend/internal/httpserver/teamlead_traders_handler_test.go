package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func TestTeamleadTradersHandlerCreateReturnsPublicTrader(t *testing.T) {
	service := &fakeTeamleadTraderService{
		createResult: users.Trader{
			ID:                 10,
			TeamID:             2,
			Role:               users.RoleTrader,
			Login:              "trader_ivan",
			Status:             users.StatusActive,
			SalaryRateBps:      50,
			ExternalWorkerName: "Bliss_OP2",
			CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			UpdatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadTradersHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/traders", strings.NewReader(`{
		"login": "trader_ivan",
		"password": "temporary-password",
		"salaryRateBps": 50,
		"externalWorkerName": "Bliss_OP2"
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
	if service.createParams.ActorID != 1 || service.createParams.TeamID != 2 {
		t.Fatalf("create params actor/team = %d/%d, want 1/2", service.createParams.ActorID, service.createParams.TeamID)
	}
	if strings.Contains(response.Body.String(), "temporary-password") {
		t.Fatalf("password leaked to create response: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"externalWorkerName":"Bliss_OP2"`) {
		t.Fatalf("response does not contain public trader: %s", response.Body.String())
	}
}

func TestTeamleadTradersHandlerPatchValidatesNegativeSalaryRate(t *testing.T) {
	handler := NewTeamleadTradersHandler(&fakeTeamleadTraderService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/teamlead/traders/10", strings.NewReader(`{"salaryRateBps":-1}`))
	request = request.WithContext(withTraderID(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "10"))

	handler.Patch(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "salaryRateBps") {
		t.Fatalf("response does not mention salaryRateBps: %s", response.Body.String())
	}
}

func TestTeamleadTradersHandlerResetPasswordReturnsTemporaryPasswordOnce(t *testing.T) {
	service := &fakeTeamleadTraderService{
		resetResult: users.ResetTraderPasswordResult{
			Trader: users.Trader{
				ID:                 10,
				TeamID:             2,
				Role:               users.RoleTrader,
				Login:              "trader_ivan",
				Status:             users.StatusActive,
				SalaryRateBps:      50,
				ExternalWorkerName: "Bliss_OP2",
			},
			TemporaryPassword: "generated-temporary-password",
		},
	}
	handler := NewTeamleadTradersHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/traders/10/reset-password", nil)
	request = request.WithContext(withTraderID(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "10"))

	handler.ResetPassword(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.resetParams.TraderID != 10 {
		t.Fatalf("reset trader id = %d, want 10", service.resetParams.TraderID)
	}
	if !strings.Contains(response.Body.String(), `"temporaryPassword":"generated-temporary-password"`) {
		t.Fatalf("temporary password missing from reset response: %s", response.Body.String())
	}
}

func withTraderID(ctx context.Context, id string) context.Context {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("traderId", id)
	return context.WithValue(ctx, chi.RouteCtxKey, routeContext)
}

type fakeTeamleadTraderService struct {
	createParams CreateTraderParamsCapture
	createResult users.Trader
	createErr    error
	resetParams  users.ResetTraderPasswordParams
	resetResult  users.ResetTraderPasswordResult
}

type CreateTraderParamsCapture = users.CreateTraderParams

func (s *fakeTeamleadTraderService) Create(ctx context.Context, params users.CreateTraderParams) (users.Trader, error) {
	s.createParams = params
	if s.createErr != nil {
		return users.Trader{}, s.createErr
	}
	return s.createResult, nil
}

func (s *fakeTeamleadTraderService) List(ctx context.Context, teamID int64) ([]users.Trader, error) {
	return nil, nil
}

func (s *fakeTeamleadTraderService) Get(ctx context.Context, teamID int64, traderID int64) (users.Trader, error) {
	return users.Trader{}, errors.New("not implemented")
}

func (s *fakeTeamleadTraderService) Patch(ctx context.Context, params users.PatchTraderParams) (users.Trader, error) {
	return users.Trader{}, errors.New("not implemented")
}

func (s *fakeTeamleadTraderService) Delete(ctx context.Context, actorID int64, teamID int64, traderID int64) error {
	return errors.New("not implemented")
}

func (s *fakeTeamleadTraderService) ResetPassword(ctx context.Context, params users.ResetTraderPasswordParams) (users.ResetTraderPasswordResult, error) {
	s.resetParams = params
	return s.resetResult, nil
}
