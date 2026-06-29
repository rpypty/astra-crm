package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func TestTeamleadReconciliationHandlerLatestInboundReturnsRun(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		latestRun: reconciliation.Run{
			ID:                 70,
			TeamID:             2,
			Type:               reconciliation.TypeTeamleadPeriodInbound,
			ScopeType:          "teamlead_period",
			AccountingPeriodID: reconciliationTestInt64Ptr(55),
			Status:             reconciliation.StatusMatched,
			CreatedAt:          time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/inbound/reconciliation/latest", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.LatestInbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.latestTeamID != 2 {
		t.Fatalf("latest team id = %d, want 2", service.latestTeamID)
	}
	if !strings.Contains(response.Body.String(), `"type":"teamlead_period_inbound"`) {
		t.Fatalf("response does not include teamlead period inbound type: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerLatestOutboundReturnsRun(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		latestOutboundRun: reconciliation.Run{
			ID:                 72,
			TeamID:             2,
			Type:               reconciliation.TypeTeamleadPeriodOutbound,
			ScopeType:          "teamlead_period",
			AccountingPeriodID: reconciliationTestInt64Ptr(55),
			Status:             reconciliation.StatusMatched,
			CreatedAt:          time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/outbound/reconciliation/latest", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.LatestOutbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.latestOutboundTeamID != 2 {
		t.Fatalf("latest outbound team id = %d, want 2", service.latestOutboundTeamID)
	}
	if !strings.Contains(response.Body.String(), `"type":"teamlead_period_outbound"`) {
		t.Fatalf("response does not include teamlead period outbound type: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerPeriodInboundUsesPeriodID(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		periodRun: reconciliation.Run{
			ID:                 71,
			TeamID:             2,
			Type:               reconciliation.TypeTeamleadPeriodInbound,
			ScopeType:          "teamlead_period",
			AccountingPeriodID: reconciliationTestInt64Ptr(55),
			Status:             reconciliation.StatusMismatch,
			CreatedAt:          time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/periods/55/reconciliation/inbound", nil)
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "periodId", "55"))

	handler.PeriodInbound(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.periodTeamID != 2 || service.periodID != 55 {
		t.Fatalf("period call = team:%d period:%d, want 2/55", service.periodTeamID, service.periodID)
	}
}

func TestTeamleadReconciliationHandlerPeriodItemsReturnsItems(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		items: []reconciliation.Item{
			{
				ID:                  80,
				ReconciliationRunID: 71,
				IssueType:           "missing_in_trader_import",
				ExternalInnerID:     reconciliationTestStringPtr("inner-1"),
				TeamleadValueJSON:   []byte(`{"amountMinor":1000}`),
				Message:             reconciliationTestStringPtr("missing"),
				CreatedAt:           time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/periods/55/reconciliation/items", nil)
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "periodId", "55"))

	handler.PeriodItems(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.itemsTeamID != 2 || service.itemsPeriodID != 55 {
		t.Fatalf("items call = team:%d period:%d, want 2/55", service.itemsTeamID, service.itemsPeriodID)
	}
	if !strings.Contains(response.Body.String(), `"issueType":"missing_in_trader_import"`) {
		t.Fatalf("response does not include issue type: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerShiftInboundItemsReturnsItems(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		shiftItems: []reconciliation.Item{
			{
				ID:                90,
				IssueType:         "total_mismatch",
				TeamleadValueJSON: []byte(`{"amountMinor":700000}`),
				TraderValueJSON:   []byte(`{"amountMinor":750000}`),
				Message:           reconciliationTestStringPtr("invoice total mismatch"),
				CreatedAt:         time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/shifts/33/reconciliation/inbound/items", nil)
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "shiftId", "33"))

	handler.ShiftInboundItems(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.shiftItemsTeamID != 2 || service.shiftItemsShiftID != 33 {
		t.Fatalf("shift items call = team:%d shift:%d, want 2/33", service.shiftItemsTeamID, service.shiftItemsShiftID)
	}
	if !strings.Contains(response.Body.String(), `"issueType":"total_mismatch"`) {
		t.Fatalf("response does not include issue type: %s", response.Body.String())
	}
}

type fakeTeamleadReconciliationService struct {
	latestTeamID          int64
	latestRun             reconciliation.Run
	latestErr             error
	latestOutboundTeamID  int64
	latestOutboundRun     reconciliation.Run
	latestOutboundErr     error
	startTeamID           int64
	startDirection        string
	startRun              reconciliation.Run
	startErr              error
	historyRecord         reconciliation.ListTeamleadCurrentRunsParams
	historyRuns           []reconciliation.Run
	historyErr            error
	currentRunRecord      reconciliation.GetTeamleadCurrentRunParams
	currentRun            reconciliation.Run
	currentRunErr         error
	currentRunItemsRecord reconciliation.GetTeamleadCurrentRunParams
	acceptRecord          reconciliation.AcceptTeamleadCurrentParams
	acceptedRun           reconciliation.Run
	acceptErr             error

	periodTeamID         int64
	periodID             int64
	periodRun            reconciliation.Run
	periodErr            error
	periodOutboundTeamID int64
	periodOutboundID     int64
	periodOutboundRun    reconciliation.Run
	periodOutboundErr    error

	itemsTeamID           int64
	itemsPeriodID         int64
	items                 []reconciliation.Item
	itemsErr              error
	outboundItemsTeamID   int64
	outboundItemsPeriodID int64

	shiftTeamID              int64
	shiftID                  int64
	shiftRun                 reconciliation.Run
	shiftErr                 error
	shiftOutboundTeamID      int64
	shiftOutboundID          int64
	shiftOutboundRun         reconciliation.Run
	shiftOutboundErr         error
	shiftItemsTeamID         int64
	shiftItemsShiftID        int64
	shiftItems               []reconciliation.Item
	shiftItemsErr            error
	shiftOutboundItemsTeamID int64
	shiftOutboundItemsID     int64
}

func (s *fakeTeamleadReconciliationService) LatestTeamleadInbound(ctx context.Context, teamID int64, actorID int64) (reconciliation.Run, error) {
	s.latestTeamID = teamID
	if s.latestErr != nil {
		return reconciliation.Run{}, s.latestErr
	}
	return s.latestRun, nil
}

func (s *fakeTeamleadReconciliationService) LatestTeamleadOutbound(ctx context.Context, teamID int64, actorID int64) (reconciliation.Run, error) {
	s.latestOutboundTeamID = teamID
	if s.latestOutboundErr != nil {
		return reconciliation.Run{}, s.latestOutboundErr
	}
	return s.latestOutboundRun, nil
}

func (s *fakeTeamleadReconciliationService) RecalculateTeamleadCurrent(ctx context.Context, params reconciliation.RecalculateTeamleadCurrentParams) (reconciliation.Run, error) {
	s.startTeamID = params.TeamID
	s.startDirection = params.Direction
	if s.startErr != nil {
		return reconciliation.Run{}, s.startErr
	}
	return s.startRun, nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadInboundItems(ctx context.Context, teamID int64, actorID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.itemsTeamID = teamID
	if s.itemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.itemsErr
	}
	return pagination.FromSlice(s.items, page), nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadOutboundItems(ctx context.Context, teamID int64, actorID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.outboundItemsTeamID = teamID
	if s.itemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.itemsErr
	}
	return pagination.FromSlice(s.items, page), nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadCurrentRuns(ctx context.Context, params reconciliation.ListTeamleadCurrentRunsParams) (pagination.Result[reconciliation.Run], error) {
	s.historyRecord = params
	if s.historyErr != nil {
		return pagination.Result[reconciliation.Run]{}, s.historyErr
	}
	return pagination.FromSlice(s.historyRuns, params.Page), nil
}

func (s *fakeTeamleadReconciliationService) GetTeamleadCurrentRun(ctx context.Context, params reconciliation.GetTeamleadCurrentRunParams) (reconciliation.Run, error) {
	s.currentRunRecord = params
	if s.currentRunErr != nil {
		return reconciliation.Run{}, s.currentRunErr
	}
	return s.currentRun, nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadCurrentItems(ctx context.Context, params reconciliation.GetTeamleadCurrentRunParams, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.currentRunItemsRecord = params
	if s.itemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.itemsErr
	}
	return pagination.FromSlice(s.items, page), nil
}

func (s *fakeTeamleadReconciliationService) AcceptTeamleadCurrent(ctx context.Context, params reconciliation.AcceptTeamleadCurrentParams) (reconciliation.Run, error) {
	s.acceptRecord = params
	if s.acceptErr != nil {
		return reconciliation.Run{}, s.acceptErr
	}
	return s.acceptedRun, nil
}

func (s *fakeTeamleadReconciliationService) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error) {
	s.periodTeamID = teamID
	s.periodID = accountingPeriodID
	if s.periodErr != nil {
		return reconciliation.Run{}, s.periodErr
	}
	return s.periodRun, nil
}

func (s *fakeTeamleadReconciliationService) LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error) {
	s.periodOutboundTeamID = teamID
	s.periodOutboundID = accountingPeriodID
	if s.periodOutboundErr != nil {
		return reconciliation.Run{}, s.periodOutboundErr
	}
	return s.periodOutboundRun, nil
}

func (s *fakeTeamleadReconciliationService) LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (reconciliation.Run, error) {
	s.shiftTeamID = teamID
	s.shiftID = shiftID
	if s.shiftErr != nil {
		return reconciliation.Run{}, s.shiftErr
	}
	return s.shiftRun, nil
}

func (s *fakeTeamleadReconciliationService) LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (reconciliation.Run, error) {
	s.shiftOutboundTeamID = teamID
	s.shiftOutboundID = shiftID
	if s.shiftOutboundErr != nil {
		return reconciliation.Run{}, s.shiftOutboundErr
	}
	return s.shiftOutboundRun, nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.itemsTeamID = teamID
	s.itemsPeriodID = accountingPeriodID
	if s.itemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.itemsErr
	}
	return pagination.FromSlice(s.items, page), nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadPeriodOutboundItems(ctx context.Context, teamID int64, accountingPeriodID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.outboundItemsTeamID = teamID
	s.outboundItemsPeriodID = accountingPeriodID
	if s.itemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.itemsErr
	}
	return pagination.FromSlice(s.items, page), nil
}

func (s *fakeTeamleadReconciliationService) ListTraderInboundItemsByShift(ctx context.Context, teamID int64, shiftID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.shiftItemsTeamID = teamID
	s.shiftItemsShiftID = shiftID
	if s.shiftItemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.shiftItemsErr
	}
	return pagination.FromSlice(s.shiftItems, page), nil
}

func (s *fakeTeamleadReconciliationService) ListTraderOutboundItemsByShift(ctx context.Context, teamID int64, shiftID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error) {
	s.shiftOutboundItemsTeamID = teamID
	s.shiftOutboundItemsID = shiftID
	if s.shiftItemsErr != nil {
		return pagination.Result[reconciliation.Item]{}, s.shiftItemsErr
	}
	return pagination.FromSlice(s.shiftItems, page), nil
}

func withTeamleadReconciliationRouteParam(ctx context.Context, key string, value string) context.Context {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, routeContext)
}

func reconciliationTestInt64Ptr(value int64) *int64 {
	return &value
}

func reconciliationTestStringPtr(value string) *string {
	return &value
}
