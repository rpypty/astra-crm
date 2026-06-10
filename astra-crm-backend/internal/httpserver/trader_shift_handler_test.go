package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func TestTraderShiftHandlerCurrentReturnsNullWhenNoShift(t *testing.T) {
	handler := NewTraderShiftHandler(&fakeTraderShiftService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/shift/current", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.Current(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"shift":null`) {
		t.Fatalf("response = %s, want shift null", response.Body.String())
	}
}

func TestTraderShiftHandlerTakeValidatesDailyDetails(t *testing.T) {
	handler := NewTraderShiftHandler(&fakeTraderShiftService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/requisites/4/take", strings.NewReader(`{"cardNumber":"","holderName":"Иванов Иван"}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "requisiteId", "4"))

	handler.TakeRequisite(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "cardNumber") {
		t.Fatalf("response does not mention cardNumber: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerTakeReturnsCreatedShift(t *testing.T) {
	service := &fakeTraderShiftService{
		takeResult: shifts.TakeRequisiteResult{
			Shift: shifts.Shift{
				ID:                           10,
				TeamID:                       2,
				TraderID:                     3,
				StartedAt:                    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
				Status:                       shifts.StatusOpen,
				InboundReconciliationStatus:  "not_started",
				OutboundReconciliationStatus: "not_started",
				CreatedAt:                    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
				UpdatedAt:                    time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			},
			ShiftRequisite: shifts.ShiftRequisite{
				ID:          20,
				TeamID:      2,
				ShiftID:     10,
				TraderID:    3,
				RequisiteID: 4,
				CardNumber:  "1234",
				HolderName:  "Иванов Иван",
				TakenAt:     time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
				Status:      shifts.RequisiteStatusActive,
				CreatedAt:   time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			},
			ShiftCreated: true,
		},
	}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/requisites/4/take", strings.NewReader(`{"cardNumber":"1234","holderName":"Иванов Иван"}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "requisiteId", "4"))

	handler.TakeRequisite(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.takeParams.TraderID != 3 || service.takeParams.TeamID != 2 {
		t.Fatalf("take params trader/team = %d/%d, want 3/2", service.takeParams.TraderID, service.takeParams.TeamID)
	}
	if !strings.Contains(response.Body.String(), `"shiftCreated":true`) {
		t.Fatalf("response does not report shiftCreated: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerCreateTurnoverValidatesAmount(t *testing.T) {
	handler := NewTraderShiftHandler(&fakeTraderShiftService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift/current/turnovers", strings.NewReader(`{"shiftRequisiteId":20,"amountMinor":-1}`))
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.CreateTurnover(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "amountMinor") {
		t.Fatalf("response does not mention amountMinor: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerCreateTurnoverPassesActorScope(t *testing.T) {
	service := &fakeTraderShiftService{
		turnover: shifts.TurnoverEntry{
			ID:               30,
			TeamID:           2,
			ShiftID:          10,
			ShiftRequisiteID: 20,
			RequisiteID:      4,
			TraderID:         3,
			AmountMinor:      150000,
			CreatedBy:        3,
			CreatedAt:        time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift/current/turnovers", strings.NewReader(`{"shiftRequisiteId":20,"amountMinor":150000}`))
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.CreateTurnover(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.turnoverParams.TeamID != 2 || service.turnoverParams.TraderID != 3 {
		t.Fatalf("turnover params team/trader = %d/%d, want 2/3", service.turnoverParams.TeamID, service.turnoverParams.TraderID)
	}
	if !strings.Contains(response.Body.String(), `"amountMinor":150000`) {
		t.Fatalf("response does not include turnover amount: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerCloseCurrentMapsBlockedClose(t *testing.T) {
	handler := NewTraderShiftHandler(&fakeTraderShiftService{
		closeErr: shifts.ErrCloseBlocked,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift/current/close", strings.NewReader(`{}`))
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.CloseCurrent(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if !strings.Contains(response.Body.String(), "SHIFT_CANNOT_BE_CLOSED") {
		t.Fatalf("response does not include domain code: %s", response.Body.String())
	}
}

func withShiftRouteParam(ctx context.Context, key string, value string) context.Context {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, routeContext)
}

type fakeTraderShiftService struct {
	takeParams     shifts.TakeRequisiteParams
	takeResult     shifts.TakeRequisiteResult
	turnoverParams shifts.CreateTurnoverParams
	turnover       shifts.TurnoverEntry
	closeErr       error
}

func (s *fakeTraderShiftService) Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.ShiftRequisite, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) TakeRequisite(ctx context.Context, params shifts.TakeRequisiteParams) (shifts.TakeRequisiteResult, error) {
	s.takeParams = params
	return s.takeResult, nil
}

func (s *fakeTraderShiftService) UpdateShiftRequisite(ctx context.Context, params shifts.UpdateShiftRequisiteParams) (shifts.ShiftRequisite, error) {
	return shifts.ShiftRequisite{}, nil
}

func (s *fakeTraderShiftService) LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]shifts.TurnoverEntry, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]shifts.TurnoverEntry, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) CreateTurnover(ctx context.Context, params shifts.CreateTurnoverParams) (shifts.TurnoverEntry, error) {
	s.turnoverParams = params
	return s.turnover, nil
}

func (s *fakeTraderShiftService) CloseChecklist(ctx context.Context, teamID int64, traderID int64) (shifts.CloseChecklist, error) {
	return shifts.CloseChecklist{}, nil
}

func (s *fakeTraderShiftService) CloseCurrent(ctx context.Context, params shifts.CloseShiftParams) (shifts.Shift, error) {
	if s.closeErr != nil {
		return shifts.Shift{}, s.closeErr
	}
	return shifts.Shift{}, nil
}
