package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/orders"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestOrderReadHandlerTraderOrdersPassesScopeAndFilters(t *testing.T) {
	createdAt := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	service := &fakeOrderReadService{
		listResult: orders.ListResult{
			Items: []orders.Order{
				{
					ScopeItemID:       10,
					ExternalOrderID:   20,
					ExternalID:        "ext-1",
					ExternalInnerID:   "in-1",
					WorkerName:        "Bliss_OP2",
					AmountMinor:       15000,
					Currency:          "RUB",
					RawStatus:         "hand_success",
					NormalizedStatus:  "success",
					CreatedAtExternal: createdAt,
					ImportBatchID:     30,
				},
			},
			Page:     2,
			PageSize: 25,
			Total:    51,
		},
	}
	handler := NewOrderReadHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/inbound/orders?dateFrom=2026-06-01&dateTo=2026-06-09&traderId=99&workerName=Bliss&requisite=7900&methodType=sbp&status=success&amountFrom=100&amountTo=500&page=2&pageSize=25&sort=amount_desc", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.TraderInboundOrders(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.traderListTeamID != 2 || service.traderListTraderID != 3 || service.traderListDirection != orders.DirectionInbound {
		t.Fatalf("scope = team:%d trader:%d direction:%q, want 2/3/inbound", service.traderListTeamID, service.traderListTraderID, service.traderListDirection)
	}
	assertOrderFilterString(t, service.traderListFilters.WorkerName, "Bliss", "workerName")
	assertOrderFilterString(t, service.traderListFilters.Requisite, "7900", "requisite")
	assertOrderFilterString(t, service.traderListFilters.MethodType, "sbp", "methodType")
	assertOrderFilterString(t, service.traderListFilters.Status, "success", "status")
	assertOrderFilterInt64(t, service.traderListFilters.TraderID, 99, "traderId")
	assertOrderFilterInt64(t, service.traderListFilters.AmountFrom, 100, "amountFrom")
	assertOrderFilterInt64(t, service.traderListFilters.AmountTo, 500, "amountTo")
	if service.traderListFilters.Page != 2 || service.traderListFilters.PageSize != 25 || service.traderListFilters.Sort != orders.SortAmountDesc {
		t.Fatalf("pagination/sort = %d/%d/%q, want 2/25/amount_desc", service.traderListFilters.Page, service.traderListFilters.PageSize, service.traderListFilters.Sort)
	}
	if service.traderListFilters.DateFrom == nil || service.traderListFilters.DateFrom.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("dateFrom = %v, want 2026-06-01", service.traderListFilters.DateFrom)
	}
	if service.traderListFilters.DateTo == nil || service.traderListFilters.DateTo.Format("2006-01-02") != "2026-06-09" {
		t.Fatalf("dateTo = %v, want 2026-06-09", service.traderListFilters.DateTo)
	}
	if !strings.Contains(response.Body.String(), `"externalInnerId":"in-1"`) || !strings.Contains(response.Body.String(), `"total":51`) {
		t.Fatalf("response does not include order list payload: %s", response.Body.String())
	}
}

func TestOrderReadHandlerRejectsInvalidFilter(t *testing.T) {
	service := &fakeOrderReadService{}
	handler := NewOrderReadHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trader/inbound/orders?dateFrom=09-06-2026", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.TraderInboundOrders(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if service.traderListCalled {
		t.Fatal("service should not be called for invalid filters")
	}
	if !strings.Contains(response.Body.String(), "dateFrom") {
		t.Fatalf("response does not mention dateFrom: %s", response.Body.String())
	}
}

func TestOrderReadHandlerTeamleadDashboardReturnsWarningsAndImportHistory(t *testing.T) {
	appliedAt := time.Date(2026, 6, 8, 10, 5, 0, 0, time.UTC)
	service := &fakeOrderReadService{
		dashboard: orders.Dashboard{
			Summary: orders.Summary{
				TotalAmountMinor:   30000,
				TotalCount:         3,
				SuccessAmountMinor: 10000,
				SuccessCount:       1,
				UnknownAmountMinor: 20000,
				UnknownCount:       2,
			},
			StatusBreakdown: []orders.StatusBreakdownItem{
				{
					RawStatus:        "manual_review",
					NormalizedStatus: "unknown",
					AmountMinor:      20000,
					Count:            2,
				},
			},
			UnknownStatuses: []string{"manual_review"},
			RecentImports: []orders.ImportHistoryItem{
				{
					ID:                 50,
					TeamID:             2,
					UploadedBy:         1,
					ScopeType:          "teamlead_period",
					Direction:          orders.DirectionOutbound,
					AccountingPeriodID: orderInt64Ptr(70),
					FileName:           "orders.csv",
					RowsCount:          3,
					Status:             "applied",
					CreatedAt:          appliedAt,
					AppliedAt:          &appliedAt,
				},
			},
		},
	}
	handler := NewOrderReadHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/outbound/dashboard?status=unknown", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.TeamleadOutboundDashboard(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.teamleadDashboardTeamID != 2 || service.teamleadDashboardDirection != orders.DirectionOutbound {
		t.Fatalf("scope = team:%d direction:%q, want 2/outbound", service.teamleadDashboardTeamID, service.teamleadDashboardDirection)
	}
	assertOrderFilterString(t, service.teamleadDashboardFilters.Status, "unknown", "status")
	if !strings.Contains(response.Body.String(), `"unknownStatuses":["manual_review"]`) {
		t.Fatalf("response does not include unknown statuses: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"fileName":"orders.csv"`) || !strings.Contains(response.Body.String(), `"totalAmountMinor":30000`) {
		t.Fatalf("response does not include dashboard payload: %s", response.Body.String())
	}
}

type fakeOrderReadService struct {
	listResult orders.ListResult
	dashboard  orders.Dashboard
	err        error

	traderListCalled    bool
	traderListTeamID    int64
	traderListTraderID  int64
	traderListDirection string
	traderListFilters   orders.Filters

	teamleadListTeamID    int64
	teamleadListDirection string
	teamleadListFilters   orders.Filters

	traderDashboardTeamID    int64
	traderDashboardTraderID  int64
	traderDashboardDirection string
	traderDashboardFilters   orders.Filters

	teamleadDashboardTeamID    int64
	teamleadDashboardDirection string
	teamleadDashboardFilters   orders.Filters
}

func (s *fakeOrderReadService) ListTraderOrders(ctx context.Context, teamID int64, traderID int64, direction string, filters orders.Filters) (orders.ListResult, error) {
	s.traderListCalled = true
	s.traderListTeamID = teamID
	s.traderListTraderID = traderID
	s.traderListDirection = direction
	s.traderListFilters = filters
	if s.err != nil {
		return orders.ListResult{}, s.err
	}
	return s.listResult, nil
}

func (s *fakeOrderReadService) ListTeamleadOrders(ctx context.Context, teamID int64, direction string, filters orders.Filters) (orders.ListResult, error) {
	s.teamleadListTeamID = teamID
	s.teamleadListDirection = direction
	s.teamleadListFilters = filters
	if s.err != nil {
		return orders.ListResult{}, s.err
	}
	return s.listResult, nil
}

func (s *fakeOrderReadService) TraderDashboard(ctx context.Context, teamID int64, traderID int64, direction string, filters orders.Filters) (orders.Dashboard, error) {
	s.traderDashboardTeamID = teamID
	s.traderDashboardTraderID = traderID
	s.traderDashboardDirection = direction
	s.traderDashboardFilters = filters
	if s.err != nil {
		return orders.Dashboard{}, s.err
	}
	return s.dashboard, nil
}

func (s *fakeOrderReadService) TeamleadDashboard(ctx context.Context, teamID int64, direction string, filters orders.Filters) (orders.Dashboard, error) {
	s.teamleadDashboardTeamID = teamID
	s.teamleadDashboardDirection = direction
	s.teamleadDashboardFilters = filters
	if s.err != nil {
		return orders.Dashboard{}, s.err
	}
	return s.dashboard, nil
}

func assertOrderFilterString(t *testing.T, value *string, want string, field string) {
	t.Helper()

	if value == nil || *value != want {
		t.Fatalf("%s = %v, want %q", field, value, want)
	}
}

func assertOrderFilterInt64(t *testing.T, value *int64, want int64, field string) {
	t.Helper()

	if value == nil || *value != want {
		t.Fatalf("%s = %v, want %d", field, value, want)
	}
}

func orderInt64Ptr(value int64) *int64 {
	return &value
}
