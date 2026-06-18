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

func TestTraderShiftHandlerAssignedRequisitesByShiftPassesShiftScope(t *testing.T) {
	service := &fakeTraderShiftService{
		shiftAssignedRequisites: []shifts.AssignedRequisite{
			{
				ID:                   4,
				TeamID:               2,
				Phone:                "+79991234567",
				MethodType:           "sbp",
				BankCode:             "sber",
				BankName:             "Сбер",
				Status:               "active",
				AssignmentID:         100,
				AssignmentStatus:     "worked",
				TargetTurnoverMinor:  0,
				ShiftRequisiteID:     int64TestPtr(20),
				ShiftRequisiteStatus: stringTestPtr(shifts.RequisiteStatusWorked),
			},
		},
	}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/shifts/10/requisites", nil)
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftId", "10"))

	handler.AssignedRequisitesByShift(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.shiftRequisitesByShiftID != 10 {
		t.Fatalf("shift id = %d, want 10", service.shiftRequisitesByShiftID)
	}
	if !strings.Contains(response.Body.String(), `"phone":"+79991234567"`) {
		t.Fatalf("response does not include requisite phone: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerTeamAssignedRequisitesByShiftPassesTeamScope(t *testing.T) {
	service := &fakeTraderShiftService{
		teamShiftAssignedRequisites: []shifts.AssignedRequisite{
			{
				ID:                   4,
				TeamID:               2,
				Phone:                "+79991234567",
				MethodType:           "sbp",
				BankCode:             "sber",
				BankName:             "Сбер",
				Status:               "active",
				AssignmentID:         100,
				AssignmentStatus:     "worked",
				TargetTurnoverMinor:  0,
				ShiftRequisiteID:     int64TestPtr(20),
				ShiftRequisiteStatus: stringTestPtr(shifts.RequisiteStatusWorked),
			},
		},
	}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/shifts/10/requisites", nil)
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "shiftId", "10"))

	handler.AssignedRequisitesByTeamShift(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.teamShiftRequisitesByShiftID != 10 {
		t.Fatalf("shift id = %d, want 10", service.teamShiftRequisitesByShiftID)
	}
	if !strings.Contains(response.Body.String(), `"phone":"+79991234567"`) {
		t.Fatalf("response does not include requisite phone: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerShiftReportPassesTraderScope(t *testing.T) {
	service := &fakeTraderShiftService{
		shiftReport: shifts.ShiftReportDetails{
			Shift: shifts.Shift{
				ID:       10,
				TeamID:   2,
				TraderID: 3,
				Status:   shifts.StatusClosed,
			},
			Rows: []shifts.ShiftReportRow{
				{
					RowKey:               "crm:20",
					ShiftRequisiteID:     int64TestPtr(20),
					RequisiteID:          int64TestPtr(4),
					Phone:                "+79991234567",
					BankCode:             "sber",
					BankName:             "Сбер",
					InboundTurnoverMinor: 150000,
					CSVInboundMinor:      100000,
					InboundDiffMinor:     50000,
					HasMismatch:          true,
				},
			},
		},
	}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/shifts/10/report", nil)
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftId", "10"))

	handler.ShiftReport(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.shiftReportTraderID != 3 || service.shiftReportShiftID != 10 {
		t.Fatalf("report call = trader:%d shift:%d, want 3/10", service.shiftReportTraderID, service.shiftReportShiftID)
	}
	if !strings.Contains(response.Body.String(), `"hasMismatch":true`) {
		t.Fatalf("response does not include mismatch row: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerTeamShiftReportPassesTeamScope(t *testing.T) {
	service := &fakeTraderShiftService{
		teamShiftReport: shifts.ShiftReportDetails{
			Shift: shifts.Shift{
				ID:       10,
				TeamID:   2,
				TraderID: 3,
				Status:   shifts.StatusClosedWithDiscrepancy,
			},
		},
	}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/shifts/10/report", nil)
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "shiftId", "10"))

	handler.TeamShiftReport(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.teamShiftReportShiftID != 10 {
		t.Fatalf("team report shift id = %d, want 10", service.teamShiftReportShiftID)
	}
	if !strings.Contains(response.Body.String(), `"status":"closed_with_discrepancy"`) {
		t.Fatalf("response does not include shift status: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerCloseShiftRequisitePassesClosingBalance(t *testing.T) {
	service := &fakeTraderShiftService{}
	handler := NewTraderShiftHandler(service)

	releasedAt := "2026-06-07T15:30:00Z"
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift-requisites/20/close", strings.NewReader(`{"inboundTurnoverMinor":150000,"outboundTurnoverMinor":25000,"closingBalanceMinor":7500,"releasedAt":"`+releasedAt+`","blocked":false}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftRequisiteId", "20"))

	handler.CloseShiftRequisite(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.closeRequisiteParams.ClosingBalanceMinor != 7500 {
		t.Fatalf("closing balance = %d, want 7500", service.closeRequisiteParams.ClosingBalanceMinor)
	}
	if service.closeRequisiteParams.ReleasedAt == nil || !service.closeRequisiteParams.ReleasedAt.Equal(time.Date(2026, 6, 7, 15, 30, 0, 0, time.UTC)) {
		t.Fatalf("released at = %v, want %s", service.closeRequisiteParams.ReleasedAt, releasedAt)
	}
}

func TestTraderShiftHandlerCloseShiftRequisiteValidatesClosingBalance(t *testing.T) {
	handler := NewTraderShiftHandler(&fakeTraderShiftService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift-requisites/20/close", strings.NewReader(`{"inboundTurnoverMinor":150000,"outboundTurnoverMinor":25000,"closingBalanceMinor":-1,"blocked":false}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftRequisiteId", "20"))

	handler.CloseShiftRequisite(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "closingBalanceMinor") {
		t.Fatalf("response does not mention closingBalanceMinor: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerCorrectShiftRequisitePassesCorrection(t *testing.T) {
	service := &fakeTraderShiftService{}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift-requisites/20/correction", strings.NewReader(`{"inboundTurnoverMinor":150000,"outboundTurnoverMinor":25000,"closingBalanceMinor":7500,"comment":"исправлено после сверки"}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftRequisiteId", "20"))

	handler.CorrectShiftRequisite(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.correctRequisiteParams.InboundTurnoverMinor != 150000 || service.correctRequisiteParams.OutboundTurnoverMinor != 25000 {
		t.Fatalf("corrected turnovers = %d/%d, want 150000/25000", service.correctRequisiteParams.InboundTurnoverMinor, service.correctRequisiteParams.OutboundTurnoverMinor)
	}
	if service.correctRequisiteParams.ClosingBalanceMinor != 7500 {
		t.Fatalf("closing balance = %d, want 7500", service.correctRequisiteParams.ClosingBalanceMinor)
	}
	if service.correctRequisiteParams.Comment != "исправлено после сверки" {
		t.Fatalf("comment = %q, want correction comment", service.correctRequisiteParams.Comment)
	}
	if !strings.Contains(response.Body.String(), `"status":"worked_verified"`) {
		t.Fatalf("response does not include refreshed status: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerCorrectShiftRequisiteRequiresComment(t *testing.T) {
	handler := NewTraderShiftHandler(&fakeTraderShiftService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift-requisites/20/correction", strings.NewReader(`{"inboundTurnoverMinor":150000,"outboundTurnoverMinor":25000,"closingBalanceMinor":7500,"comment":" "}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftRequisiteId", "20"))

	handler.CorrectShiftRequisite(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "comment") {
		t.Fatalf("response does not mention comment: %s", response.Body.String())
	}
}

func TestTraderShiftHandlerReturnShiftRequisiteToWorkPassesScope(t *testing.T) {
	service := &fakeTraderShiftService{}
	handler := NewTraderShiftHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/shift-requisites/20/return-to-work", strings.NewReader(`{}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "shiftRequisiteId", "20"))

	handler.ReturnShiftRequisiteToWork(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.returnRequisiteParams.TeamID != 2 || service.returnRequisiteParams.TraderID != 3 || service.returnRequisiteParams.ShiftRequisiteID != 20 {
		t.Fatalf("return params = team:%d trader:%d shiftRequisite:%d, want 2/3/20", service.returnRequisiteParams.TeamID, service.returnRequisiteParams.TraderID, service.returnRequisiteParams.ShiftRequisiteID)
	}
	if !strings.Contains(response.Body.String(), `"status":"active"`) {
		t.Fatalf("response does not include active status: %s", response.Body.String())
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
	takeParams                   shifts.TakeRequisiteParams
	takeResult                   shifts.TakeRequisiteResult
	turnoverParams               shifts.CreateTurnoverParams
	turnover                     shifts.TurnoverEntry
	closeRequisiteParams         shifts.CloseShiftRequisiteParams
	correctRequisiteParams       shifts.CorrectShiftRequisiteParams
	returnRequisiteParams        shifts.ReturnShiftRequisiteParams
	shiftRequisitesByShiftID     int64
	shiftAssignedRequisites      []shifts.AssignedRequisite
	teamShiftRequisitesByShiftID int64
	teamShiftAssignedRequisites  []shifts.AssignedRequisite
	shiftReportTraderID          int64
	shiftReportShiftID           int64
	shiftReport                  shifts.ShiftReportDetails
	teamShiftReportShiftID       int64
	teamShiftReport              shifts.ShiftReportDetails
	closeErr                     error
}

func (s *fakeTraderShiftService) Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) ShiftHistory(ctx context.Context, teamID int64, traderID int64, limit int32) ([]shifts.Shift, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) TeamShiftHistory(ctx context.Context, teamID int64, limit int32) ([]shifts.Shift, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) ShiftReport(ctx context.Context, teamID int64, traderID int64, shiftID int64) (shifts.ShiftReportDetails, error) {
	s.shiftReportTraderID = traderID
	s.shiftReportShiftID = shiftID
	return s.shiftReport, nil
}

func (s *fakeTraderShiftService) TeamShiftReport(ctx context.Context, teamID int64, shiftID int64) (shifts.ShiftReportDetails, error) {
	s.teamShiftReportShiftID = shiftID
	return s.teamShiftReport, nil
}

func (s *fakeTraderShiftService) AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error) {
	return nil, nil
}

func (s *fakeTraderShiftService) AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64) ([]shifts.AssignedRequisite, error) {
	s.shiftRequisitesByShiftID = shiftID
	return s.shiftAssignedRequisites, nil
}

func (s *fakeTraderShiftService) AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64) ([]shifts.AssignedRequisite, error) {
	s.teamShiftRequisitesByShiftID = shiftID
	return s.teamShiftAssignedRequisites, nil
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

func (s *fakeTraderShiftService) CloseShiftRequisite(ctx context.Context, params shifts.CloseShiftRequisiteParams) (shifts.ShiftRequisite, error) {
	s.closeRequisiteParams = params
	return shifts.ShiftRequisite{
		ID:                    params.ShiftRequisiteID,
		TeamID:                params.TeamID,
		ShiftID:               10,
		TraderID:              params.TraderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                shifts.RequisiteStatusWorked,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeTraderShiftService) CorrectShiftRequisite(ctx context.Context, params shifts.CorrectShiftRequisiteParams) (shifts.ShiftRequisite, error) {
	s.correctRequisiteParams = params
	return shifts.ShiftRequisite{
		ID:                    params.ShiftRequisiteID,
		TeamID:                params.TeamID,
		ShiftID:               10,
		TraderID:              params.TraderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                shifts.RequisiteStatusVerified,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeTraderShiftService) ReturnShiftRequisiteToWork(ctx context.Context, params shifts.ReturnShiftRequisiteParams) (shifts.ShiftRequisite, error) {
	s.returnRequisiteParams = params
	return shifts.ShiftRequisite{
		ID:                    params.ShiftRequisiteID,
		TeamID:                params.TeamID,
		ShiftID:               10,
		TraderID:              params.TraderID,
		RequisiteID:           4,
		CardNumber:            "1234567890123456",
		HolderName:            "Иванов Иван",
		TakenAt:               time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		Status:                shifts.RequisiteStatusActive,
		InboundTurnoverMinor:  150000,
		OutboundTurnoverMinor: 25000,
		ClosingBalanceMinor:   7500,
		CreatedAt:             time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
	}, nil
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

func int64TestPtr(value int64) *int64 {
	return &value
}

func stringTestPtr(value string) *string {
	return &value
}
