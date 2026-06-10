package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type fakeTeamleadReconciliationService struct {
	latestTeamID int64
	latestRun    reconciliation.Run
	latestErr    error

	periodTeamID int64
	periodID     int64
	periodRun    reconciliation.Run
	periodErr    error

	itemsTeamID   int64
	itemsPeriodID int64
	items         []reconciliation.Item
	itemsErr      error
}

func (s *fakeTeamleadReconciliationService) LatestTeamleadInbound(ctx context.Context, teamID int64) (reconciliation.Run, error) {
	s.latestTeamID = teamID
	if s.latestErr != nil {
		return reconciliation.Run{}, s.latestErr
	}
	return s.latestRun, nil
}

func (s *fakeTeamleadReconciliationService) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error) {
	s.periodTeamID = teamID
	s.periodID = accountingPeriodID
	if s.periodErr != nil {
		return reconciliation.Run{}, s.periodErr
	}
	return s.periodRun, nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64) ([]reconciliation.Item, error) {
	s.itemsTeamID = teamID
	s.itemsPeriodID = accountingPeriodID
	if s.itemsErr != nil {
		return nil, s.itemsErr
	}
	return s.items, nil
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
