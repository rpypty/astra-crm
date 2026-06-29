package httpserver

import (
	"bytes"
	"context"
	"mime/multipart"
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

func TestTeamleadReconciliationHandlerCreateParsesPeriodAndInboundFile(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		createRun: reconciliation.TeamleadRun{
			ID:        500,
			TeamID:    2,
			DateFrom:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
			DateTo:    time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
			Status:    reconciliation.TeamleadRunStatusMismatch,
			CreatedBy: 1,
			CreatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)
	body, contentType := teamleadReconciliationMultipart(t, map[string]string{
		"dateFrom": "2026-06-10",
		"dateTo":   "2026-06-12",
	}, map[string]teamleadReconciliationFile{
		"inboundFile": {
			name:    "inbound.csv",
			content: "id|innerId|amount|currency|status|createdAt|workerName\n1|in-1|10.00|RUB|hand_success|10.06.2026 12:00:00|Bliss_OP1\n",
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/reconciliations", body)
	request.Header.Set("Content-Type", contentType)
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
	if service.createRecord.ActorID != 1 || service.createRecord.TeamID != 2 {
		t.Fatalf("create actor/team = %d/%d, want 1/2", service.createRecord.ActorID, service.createRecord.TeamID)
	}
	if service.createRecord.DateFrom.Format("2006-01-02") != "2026-06-10" || service.createRecord.DateTo.Format("2006-01-02") != "2026-06-12" {
		t.Fatalf("create period = %s/%s, want 2026-06-10/2026-06-12", service.createRecord.DateFrom, service.createRecord.DateTo)
	}
	if service.createRecord.Inbound == nil || service.createRecord.Inbound.FileName != "inbound.csv" {
		t.Fatalf("inbound input = %+v, want inbound.csv", service.createRecord.Inbound)
	}
	if service.createRecord.Outbound != nil {
		t.Fatalf("outbound input = %+v, want nil", service.createRecord.Outbound)
	}
	if !strings.Contains(response.Body.String(), `"status":"mismatch"`) {
		t.Fatalf("response does not include created run status: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerCreateRequiresCSVFile(t *testing.T) {
	service := &fakeTeamleadReconciliationService{}
	handler := NewTeamleadReconciliationHandler(service)
	body, contentType := teamleadReconciliationMultipart(t, map[string]string{
		"dateFrom": "2026-06-10",
		"dateTo":   "2026-06-12",
	}, nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/reconciliations", body)
	request.Header.Set("Content-Type", contentType)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if service.createRecord.TeamID != 0 {
		t.Fatalf("create was called with %+v, want no service call", service.createRecord)
	}
	if !strings.Contains(response.Body.String(), `"file"`) {
		t.Fatalf("response does not include file validation error: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerConfirmQueuesRun(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		confirmedTeamleadRun: reconciliation.TeamleadRun{
			ID:        500,
			TeamID:    2,
			Status:    reconciliation.TeamleadRunStatusApplyQueued,
			CreatedBy: 1,
			CreatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/reconciliations/500/confirm", strings.NewReader(`{"comment":"accepted"}`))
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "runId", "500"))

	handler.Confirm(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.confirmRecord.TeamID != 2 || service.confirmRecord.ActorID != 1 || service.confirmRecord.RunID != 500 {
		t.Fatalf("confirm record = %+v, want team 2 actor 1 run 500", service.confirmRecord)
	}
	if service.confirmRecord.Comment != "accepted" {
		t.Fatalf("confirm comment = %q, want accepted", service.confirmRecord.Comment)
	}
	if !strings.Contains(response.Body.String(), `"status":"apply_queued"`) {
		t.Fatalf("response does not include apply_queued: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerRejectRun(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		rejectedTeamleadRun: reconciliation.TeamleadRun{
			ID:        500,
			TeamID:    2,
			Status:    reconciliation.TeamleadRunStatusRejected,
			CreatedBy: 1,
			CreatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teamlead/reconciliations/500/reject", strings.NewReader(`{"comment":"bad export"}`))
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "runId", "500"))

	handler.Reject(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.rejectRecord.TeamID != 2 || service.rejectRecord.ActorID != 1 || service.rejectRecord.RunID != 500 {
		t.Fatalf("reject record = %+v, want team 2 actor 1 run 500", service.rejectRecord)
	}
	if service.rejectRecord.Comment != "bad export" {
		t.Fatalf("reject comment = %q, want bad export", service.rejectRecord.Comment)
	}
	if !strings.Contains(response.Body.String(), `"status":"rejected"`) {
		t.Fatalf("response does not include rejected: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerListV2Runs(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		teamleadRuns: []reconciliation.TeamleadRun{
			{
				ID:        500,
				TeamID:    2,
				DateFrom:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
				DateTo:    time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
				Status:    reconciliation.TeamleadRunStatusApplied,
				CreatedBy: 1,
				CreatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/reconciliations?page=1&pageSize=20", nil)
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.listTeamleadRecord.TeamID != 2 || service.listTeamleadRecord.Page.PageSize != 20 {
		t.Fatalf("list record = %+v, want team 2 pageSize 20", service.listTeamleadRecord)
	}
	if !strings.Contains(response.Body.String(), `"status":"applied"`) {
		t.Fatalf("response does not include applied run: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerGetV2Run(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		teamleadRun: reconciliation.TeamleadRun{
			ID:        500,
			TeamID:    2,
			Status:    reconciliation.TeamleadRunStatusMismatch,
			CreatedBy: 1,
			CreatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/reconciliations/500", nil)
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "runId", "500"))

	handler.Get(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.getTeamleadRecord.TeamID != 2 || service.getTeamleadRecord.RunID != 500 {
		t.Fatalf("get record = %+v, want team 2 run 500", service.getTeamleadRecord)
	}
	if !strings.Contains(response.Body.String(), `"id":500`) {
		t.Fatalf("response does not include run id: %s", response.Body.String())
	}
}

func TestTeamleadReconciliationHandlerV2ItemsPassesFilters(t *testing.T) {
	service := &fakeTeamleadReconciliationService{
		teamleadItems: []reconciliation.TeamleadItem{
			{
				ID:                       900,
				TeamleadReconciliationID: 500,
				TeamID:                   2,
				Direction:                "inbound",
				Stage:                    reconciliation.TeamleadItemStageTransactionCheck,
				IssueType:                "missing_in_crm",
				Severity:                 reconciliation.TeamleadItemSeverityWarning,
				CreatedAt:                time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewTeamleadReconciliationHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teamlead/reconciliations/500/items?direction=inbound&stage=transaction_check&issueType=missing_in_crm&severity=warning&traderId=3&requisiteId=20&onlyMismatch=true", nil)
	request = request.WithContext(withTeamleadReconciliationRouteParam(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}), "runId", "500"))

	handler.Items(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.teamleadItemsRecord.RunID != 500 || service.teamleadItemsFilters.Direction != "inbound" || service.teamleadItemsFilters.Stage != reconciliation.TeamleadItemStageTransactionCheck {
		t.Fatalf("items record/filter = %+v/%+v, want run 500 inbound transaction_check", service.teamleadItemsRecord, service.teamleadItemsFilters)
	}
	if service.teamleadItemsFilters.TraderID == nil || *service.teamleadItemsFilters.TraderID != 3 {
		t.Fatalf("trader filter = %v, want 3", service.teamleadItemsFilters.TraderID)
	}
	if service.teamleadItemsFilters.RequisiteID == nil || *service.teamleadItemsFilters.RequisiteID != 20 {
		t.Fatalf("requisite filter = %v, want 20", service.teamleadItemsFilters.RequisiteID)
	}
	if !service.teamleadItemsFilters.OnlyMismatch {
		t.Fatal("onlyMismatch filter = false, want true")
	}
	if !strings.Contains(response.Body.String(), `"issueType":"missing_in_crm"`) {
		t.Fatalf("response does not include issue: %s", response.Body.String())
	}
}

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
	createRecord          reconciliation.CreateTeamleadReconciliationParams
	createRun             reconciliation.TeamleadRun
	createErr             error
	confirmRecord         reconciliation.ConfirmTeamleadReconciliationParams
	confirmedTeamleadRun  reconciliation.TeamleadRun
	confirmErr            error
	rejectRecord          reconciliation.RejectTeamleadReconciliationParams
	rejectedTeamleadRun   reconciliation.TeamleadRun
	rejectErr             error
	listTeamleadRecord    reconciliation.ListTeamleadReconciliationsParams
	teamleadRuns          []reconciliation.TeamleadRun
	listTeamleadErr       error
	getTeamleadRecord     reconciliation.GetTeamleadReconciliationParams
	teamleadRun           reconciliation.TeamleadRun
	getTeamleadErr        error
	teamleadItemsRecord   reconciliation.GetTeamleadReconciliationParams
	teamleadItemsFilters  reconciliation.TeamleadItemFilters
	teamleadItems         []reconciliation.TeamleadItem
	teamleadItemsErr      error
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

func (s *fakeTeamleadReconciliationService) CreateTeamleadReconciliation(ctx context.Context, params reconciliation.CreateTeamleadReconciliationParams) (reconciliation.TeamleadRun, error) {
	s.createRecord = params
	if s.createErr != nil {
		return reconciliation.TeamleadRun{}, s.createErr
	}
	return s.createRun, nil
}

func (s *fakeTeamleadReconciliationService) ConfirmTeamleadReconciliation(ctx context.Context, params reconciliation.ConfirmTeamleadReconciliationParams) (reconciliation.TeamleadRun, error) {
	s.confirmRecord = params
	if s.confirmErr != nil {
		return reconciliation.TeamleadRun{}, s.confirmErr
	}
	return s.confirmedTeamleadRun, nil
}

func (s *fakeTeamleadReconciliationService) RejectTeamleadReconciliation(ctx context.Context, params reconciliation.RejectTeamleadReconciliationParams) (reconciliation.TeamleadRun, error) {
	s.rejectRecord = params
	if s.rejectErr != nil {
		return reconciliation.TeamleadRun{}, s.rejectErr
	}
	return s.rejectedTeamleadRun, nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadReconciliations(ctx context.Context, params reconciliation.ListTeamleadReconciliationsParams) (pagination.Result[reconciliation.TeamleadRun], error) {
	s.listTeamleadRecord = params
	if s.listTeamleadErr != nil {
		return pagination.Result[reconciliation.TeamleadRun]{}, s.listTeamleadErr
	}
	return pagination.FromSlice(s.teamleadRuns, params.Page), nil
}

func (s *fakeTeamleadReconciliationService) GetTeamleadReconciliation(ctx context.Context, params reconciliation.GetTeamleadReconciliationParams) (reconciliation.TeamleadRun, error) {
	s.getTeamleadRecord = params
	if s.getTeamleadErr != nil {
		return reconciliation.TeamleadRun{}, s.getTeamleadErr
	}
	return s.teamleadRun, nil
}

func (s *fakeTeamleadReconciliationService) ListTeamleadReconciliationItems(ctx context.Context, params reconciliation.GetTeamleadReconciliationParams, filters reconciliation.TeamleadItemFilters, page pagination.Params) (pagination.Result[reconciliation.TeamleadItem], error) {
	s.teamleadItemsRecord = params
	s.teamleadItemsFilters = filters
	if s.teamleadItemsErr != nil {
		return pagination.Result[reconciliation.TeamleadItem]{}, s.teamleadItemsErr
	}
	return pagination.FromSlice(s.teamleadItems, page), nil
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

type teamleadReconciliationFile struct {
	name    string
	content string
}

func teamleadReconciliationMultipart(t *testing.T, fields map[string]string, files map[string]teamleadReconciliationFile) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write multipart field: %v", err)
		}
	}
	for field, file := range files {
		part, err := writer.CreateFormFile(field, file.name)
		if err != nil {
			t.Fatalf("create multipart file: %v", err)
		}
		if _, err := part.Write([]byte(file.content)); err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}
