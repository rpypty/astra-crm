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

	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

func TestImportHandlerTraderInboundUsesCurrentShift(t *testing.T) {
	importService := &fakeImportService{
		result: imports.ApplyResult{
			Batch: imports.ImportBatch{
				ID:        100,
				Status:    imports.BatchStatusApplied,
				CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			},
			RowsCount:        1,
			CreatedOrders:    1,
			ActiveScopeItems: 1,
			Parse: imports.ParseResult{
				RawStatusCounts: map[string]int{
					"hand_success": 1,
				},
				NormalizedStatusCounts: map[string]int{
					imports.NormalizedStatusSuccess: 1,
				},
			},
		},
	}
	shiftService := &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	}
	handler := NewImportHandler(importService, shiftService)

	response := httptest.NewRecorder()
	request := multipartRequest(t, "/api/v1/trader/inbound/import", nil, "orders.csv", "id|innerId|amount|currency|status|createdAt|workerName\n1|in-1|10.0|RUB|hand_success|28.05.2026 11:21:35|Bliss_OP2")
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.TraderInbound(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if importService.params.Scope.Type != imports.ScopeTypeTraderShift {
		t.Fatalf("scope type = %q, want trader_shift", importService.params.Scope.Type)
	}
	if importService.params.Scope.Direction != imports.DirectionInbound {
		t.Fatalf("direction = %q, want inbound", importService.params.Scope.Direction)
	}
	if importService.params.Scope.ShiftID == nil || *importService.params.Scope.ShiftID != 10 {
		t.Fatalf("shift id = %v, want 10", importService.params.Scope.ShiftID)
	}
	if importService.params.ParseOptions.ColumnSet != imports.ColumnSetTrader {
		t.Fatalf("column set = %q, want trader", importService.params.ParseOptions.ColumnSet)
	}
}

func TestImportHandlerTraderImportRequiresCurrentShift(t *testing.T) {
	handler := NewImportHandler(&fakeImportService{}, &fakeImportShiftService{})

	response := httptest.NewRecorder()
	request := multipartRequest(t, "/api/v1/trader/inbound/import", nil, "orders.csv", "id|innerId|amount|currency|status|createdAt|workerName\n")
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.TraderInbound(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if !strings.Contains(response.Body.String(), "CURRENT_SHIFT_REQUIRED") {
		t.Fatalf("response does not include domain code: %s", response.Body.String())
	}
}

func TestImportHandlerTeamleadImportRequiresPeriodID(t *testing.T) {
	handler := NewImportHandler(&fakeImportService{}, nil)

	response := httptest.NewRecorder()
	request := multipartRequest(t, "/api/v1/teamlead/inbound/import", nil, "orders.csv", "id|innerId|amount|currency|status|createdAt|workerName\n")
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.TeamleadInbound(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "accountingPeriodId") {
		t.Fatalf("response does not mention accountingPeriodId: %s", response.Body.String())
	}
}

func TestImportHandlerTeamleadOutboundUsesPeriodScope(t *testing.T) {
	importService := &fakeImportService{
		result: imports.ApplyResult{
			Batch: imports.ImportBatch{
				ID:        100,
				Status:    imports.BatchStatusApplied,
				CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			},
			RowsCount:        1,
			UpdatedOrders:    1,
			ActiveScopeItems: 1,
			Parse: imports.ParseResult{
				Rows: []imports.ParsedOrderRow{
					{
						RawStatus:        "manual_review",
						NormalizedStatus: imports.NormalizedStatusUnknown,
					},
				},
				RawStatusCounts: map[string]int{
					"manual_review": 1,
				},
				NormalizedStatusCounts: map[string]int{
					imports.NormalizedStatusUnknown: 1,
				},
			},
		},
	}
	handler := NewImportHandler(importService, nil)

	response := httptest.NewRecorder()
	request := multipartRequest(t, "/api/v1/teamlead/outbound/import", map[string]string{
		"accountingPeriodId": "55",
	}, "orders.csv", "id|innerId|amount|currency|status|createdAt|workerName\n1|in-1|10.0|RUB|manual_review|28.05.2026 11:21:35|Bliss_OP2")
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     1,
		TeamID: 2,
		Role:   users.RoleTeamlead,
		Status: users.StatusActive,
	}))

	handler.TeamleadOutbound(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if importService.params.Scope.Type != imports.ScopeTypeTeamleadPeriod {
		t.Fatalf("scope type = %q, want teamlead_period", importService.params.Scope.Type)
	}
	if importService.params.Scope.Direction != imports.DirectionOutbound {
		t.Fatalf("direction = %q, want outbound", importService.params.Scope.Direction)
	}
	if importService.params.Scope.AccountingPeriodID == nil || *importService.params.Scope.AccountingPeriodID != 55 {
		t.Fatalf("period id = %v, want 55", importService.params.Scope.AccountingPeriodID)
	}
	if !strings.Contains(response.Body.String(), `"unknownStatuses":["manual_review"]`) {
		t.Fatalf("response does not include unknown status: %s", response.Body.String())
	}
}

func TestImportHandlerMapsCSVValidationError(t *testing.T) {
	handler := NewImportHandler(&fakeImportService{
		err: imports.ErrValidation,
		result: imports.ApplyResult{
			Parse: imports.ParseResult{
				Errors: []imports.ParseError{
					{
						Code:    imports.ParseCodeDuplicateInnerID,
						Field:   "innerId",
						InnerID: "dup-1",
						Rows:    []int{2, 3},
						Message: "duplicated innerId inside CSV",
					},
				},
			},
		},
	}, &fakeImportShiftService{
		shift: &shifts.Shift{
			ID:       10,
			TeamID:   2,
			TraderID: 3,
			Status:   shifts.StatusOpen,
		},
	})

	response := httptest.NewRecorder()
	request := multipartRequest(t, "/api/v1/trader/inbound/import", nil, "orders.csv", "id|innerId|amount|currency|status|createdAt|workerName\n")
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.TraderInbound(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), imports.ParseCodeDuplicateInnerID) {
		t.Fatalf("response does not include parse code: %s", response.Body.String())
	}
}

func multipartRequest(t *testing.T, target string, fields map[string]string, fileName string, body string) *http.Request {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	fileWriter, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte(body)); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, target, &buffer)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

type fakeImportService struct {
	params imports.ApplyCSVParams
	result imports.ApplyResult
	err    error
}

func (s *fakeImportService) ApplyCSV(ctx context.Context, params imports.ApplyCSVParams) (imports.ApplyResult, error) {
	s.params = params
	if s.err != nil {
		return s.result, s.err
	}
	return s.result, nil
}

type fakeImportShiftService struct {
	shift *shifts.Shift
	err   error
}

func (s *fakeImportShiftService) Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.shift, nil
}
