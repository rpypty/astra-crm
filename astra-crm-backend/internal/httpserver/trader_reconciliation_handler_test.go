package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestTraderReconciliationHandlerLatestInboundReturnsRun(t *testing.T) {
	service := &fakeTraderReconciliationService{
		inboundRun: reconciliation.Run{
			ID:                  50,
			TeamID:              2,
			Type:                reconciliation.TypeTraderShiftInbound,
			ScopeType:           "trader_shift",
			ShiftID:             reconciliationInt64Ptr(10),
			TraderID:            reconciliationInt64Ptr(3),
			ExpectedAmountMinor: 1000,
			ActualAmountMinor:   1000,
			Status:              reconciliation.StatusMatched,
			CreatedAt:           time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderReconciliationHandler(service, &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/inbound/reconciliation/latest", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.LatestInbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.latestShiftID != 10 {
		t.Fatalf("latest shift id = %d, want 10", service.latestShiftID)
	}
	if !strings.Contains(response.Body.String(), `"status":"matched"`) {
		t.Fatalf("response does not include matched status: %s", response.Body.String())
	}
}

func TestTraderReconciliationHandlerLatestOutboundReturnsRun(t *testing.T) {
	service := &fakeTraderReconciliationService{
		outboundRun: reconciliation.Run{
			ID:                  60,
			TeamID:              2,
			Type:                reconciliation.TypeTraderShiftOutbound,
			ScopeType:           "trader_shift",
			ShiftID:             reconciliationInt64Ptr(10),
			TraderID:            reconciliationInt64Ptr(3),
			ExpectedAmountMinor: 2000,
			ActualAmountMinor:   2000,
			Status:              reconciliation.StatusMatched,
			CreatedAt:           time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderReconciliationHandler(service, &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/outbound/reconciliation/latest", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.LatestOutbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.latestOutboundShiftID != 10 {
		t.Fatalf("latest outbound shift id = %d, want 10", service.latestOutboundShiftID)
	}
	if !strings.Contains(response.Body.String(), `"type":"trader_shift_outbound"`) {
		t.Fatalf("response does not include outbound type: %s", response.Body.String())
	}
}

func TestTraderReconciliationHandlerAcceptInboundRequiresComment(t *testing.T) {
	handler := NewTraderReconciliationHandler(&fakeTraderReconciliationService{}, &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/inbound/reconciliation/50/accept", strings.NewReader(`{"comment":" "}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "runId", "50"))

	handler.AcceptInbound(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "comment") {
		t.Fatalf("response does not mention comment: %s", response.Body.String())
	}
}

func TestTraderReconciliationHandlerAcceptInboundPassesActorScope(t *testing.T) {
	service := &fakeTraderReconciliationService{
		inboundRun: reconciliation.Run{
			ID:        50,
			TeamID:    2,
			Type:      reconciliation.TypeTraderShiftInbound,
			ScopeType: "trader_shift",
			ShiftID:   reconciliationInt64Ptr(10),
			TraderID:  reconciliationInt64Ptr(3),
			Status:    reconciliation.StatusAcceptedWithComment,
			Comment:   reconciliationStringPtr("accepted"),
			CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderReconciliationHandler(service, &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/inbound/reconciliation/50/accept", strings.NewReader(`{"comment":"accepted"}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "runId", "50"))

	handler.AcceptInbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.acceptParams.ActorID != 3 || service.acceptParams.TeamID != 2 || service.acceptParams.TraderID != 3 || service.acceptParams.RunID != 50 {
		t.Fatalf("accept params = %+v, want actor/team/trader/run 3/2/3/50", service.acceptParams)
	}
}

func TestTraderReconciliationHandlerAcceptOutboundPassesActorScope(t *testing.T) {
	service := &fakeTraderReconciliationService{
		outboundRun: reconciliation.Run{
			ID:        60,
			TeamID:    2,
			Type:      reconciliation.TypeTraderShiftOutbound,
			ScopeType: "trader_shift",
			ShiftID:   reconciliationInt64Ptr(10),
			TraderID:  reconciliationInt64Ptr(3),
			Status:    reconciliation.StatusAcceptedWithComment,
			Comment:   reconciliationStringPtr("accepted"),
			CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderReconciliationHandler(service, &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/outbound/reconciliation/60/accept", strings.NewReader(`{"comment":"accepted"}`))
	request = request.WithContext(withShiftRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), "runId", "60"))

	handler.AcceptOutbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.acceptOutboundParams.ActorID != 3 || service.acceptOutboundParams.TeamID != 2 || service.acceptOutboundParams.TraderID != 3 || service.acceptOutboundParams.RunID != 60 {
		t.Fatalf("accept outbound params = %+v, want actor/team/trader/run 3/2/3/60", service.acceptOutboundParams)
	}
}

type fakeTraderReconciliationService struct {
	inboundRun            reconciliation.Run
	outboundRun           reconciliation.Run
	latestShiftID         int64
	latestOutboundShiftID int64
	latestErr             error
	latestOutboundErr     error
	acceptParams          reconciliation.AcceptTraderInboundParams
	acceptOutboundParams  reconciliation.AcceptTraderOutboundParams
	acceptErr             error
	acceptOutboundErr     error
}

func (s *fakeTraderReconciliationService) LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (reconciliation.Run, error) {
	s.latestShiftID = shiftID
	if s.latestErr != nil {
		return reconciliation.Run{}, s.latestErr
	}
	return s.inboundRun, nil
}

func (s *fakeTraderReconciliationService) AcceptTraderInbound(ctx context.Context, params reconciliation.AcceptTraderInboundParams) (reconciliation.Run, error) {
	s.acceptParams = params
	if s.acceptErr != nil {
		return reconciliation.Run{}, s.acceptErr
	}
	return s.inboundRun, nil
}

func (s *fakeTraderReconciliationService) LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (reconciliation.Run, error) {
	s.latestOutboundShiftID = shiftID
	if s.latestOutboundErr != nil {
		return reconciliation.Run{}, s.latestOutboundErr
	}
	return s.outboundRun, nil
}

func (s *fakeTraderReconciliationService) AcceptTraderOutbound(ctx context.Context, params reconciliation.AcceptTraderOutboundParams) (reconciliation.Run, error) {
	s.acceptOutboundParams = params
	if s.acceptOutboundErr != nil {
		return reconciliation.Run{}, s.acceptOutboundErr
	}
	return s.outboundRun, nil
}

func reconciliationInt64Ptr(value int64) *int64 {
	return &value
}

func reconciliationStringPtr(value string) *string {
	return &value
}
