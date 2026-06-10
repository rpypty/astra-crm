package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/payouts"
	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func TestTraderPayoutHandlerCreateValidatesAmount(t *testing.T) {
	handler := NewTraderPayoutHandler(&fakeTraderPayoutService{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/payouts", strings.NewReader(`{"destinationBank":"Tinkoff","destinationRequisite":"79990000000","amountMinor":0}`))
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "amountMinor") {
		t.Fatalf("response does not mention amountMinor: %s", response.Body.String())
	}
}

func TestTraderPayoutHandlerCreatePassesActorScope(t *testing.T) {
	service := &fakeTraderPayoutService{
		order: payouts.Order{
			ID:                   40,
			TeamID:               2,
			ShiftID:              10,
			TraderID:             3,
			DestinationBank:      "Tinkoff",
			DestinationRequisite: "79990000000",
			AmountMinor:          100000,
			RemainingAmountMinor: 100000,
			Status:               payouts.StatusInProgress,
			CreatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderPayoutHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/payouts", strings.NewReader(`{"destinationBank":"Tinkoff","destinationRequisite":"79990000000","amountMinor":100000}`))
	request = request.WithContext(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}))

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.createParams.ActorID != 3 || service.createParams.TeamID != 2 || service.createParams.TraderID != 3 {
		t.Fatalf("create scope = actor %d team %d trader %d, want 3/2/3", service.createParams.ActorID, service.createParams.TeamID, service.createParams.TraderID)
	}
}

func TestTraderPayoutHandlerAddTransferMapsRejectedTransfer(t *testing.T) {
	handler := NewTraderPayoutHandler(&fakeTraderPayoutService{
		addTransferErr: payouts.ErrTransferRejected,
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/payouts/40/transfers", strings.NewReader(`{"sourceShiftRequisiteId":20,"amountMinor":100000}`))
	request = request.WithContext(withPayoutRouteParams(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), map[string]string{
		"payoutId": "40",
	}))

	handler.AddTransfer(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if !strings.Contains(response.Body.String(), "PAYOUT_TRANSFER_REJECTED") {
		t.Fatalf("response does not include domain code: %s", response.Body.String())
	}
}

func TestTraderPayoutHandlerAddTransferPassesRouteAndBodyIDs(t *testing.T) {
	service := &fakeTraderPayoutService{
		transfer: payouts.Transfer{
			ID:                     50,
			TeamID:                 2,
			ManualPayoutOrderID:    40,
			ShiftID:                10,
			TraderID:               3,
			SourceShiftRequisiteID: 20,
			SourceRequisiteID:      4,
			AmountMinor:            50000,
			CreatedBy:              3,
			CreatedAt:              time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
		},
	}
	handler := NewTraderPayoutHandler(service)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trader/payouts/40/transfers", strings.NewReader(`{"sourceShiftRequisiteId":20,"amountMinor":50000}`))
	request = request.WithContext(withPayoutRouteParams(ContextWithCurrentUser(request.Context(), users.User{
		ID:     3,
		TeamID: 2,
		Role:   users.RoleTrader,
		Status: users.StatusActive,
	}), map[string]string{
		"payoutId": "40",
	}))

	handler.AddTransfer(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.addTransferParams.PayoutID != 40 || service.addTransferParams.SourceShiftRequisiteID != 20 {
		t.Fatalf("transfer ids = payout %d source %d, want 40/20", service.addTransferParams.PayoutID, service.addTransferParams.SourceShiftRequisiteID)
	}
}

func withPayoutRouteParams(ctx context.Context, values map[string]string) context.Context {
	routeContext := chi.NewRouteContext()
	for key, value := range values {
		routeContext.URLParams.Add(key, value)
	}
	return context.WithValue(ctx, chi.RouteCtxKey, routeContext)
}

type fakeTraderPayoutService struct {
	createParams      payouts.CreateOrderParams
	addTransferParams payouts.AddTransferParams
	order             payouts.Order
	transfer          payouts.Transfer
	addTransferErr    error
}

func (s *fakeTraderPayoutService) ListOrders(ctx context.Context, teamID int64, traderID int64) ([]payouts.Order, error) {
	return nil, nil
}

func (s *fakeTraderPayoutService) GetOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (payouts.Order, []payouts.Transfer, error) {
	return s.order, nil, nil
}

func (s *fakeTraderPayoutService) CreateOrder(ctx context.Context, params payouts.CreateOrderParams) (payouts.Order, error) {
	s.createParams = params
	return s.order, nil
}

func (s *fakeTraderPayoutService) PatchOrder(ctx context.Context, params payouts.PatchOrderParams) (payouts.Order, error) {
	return s.order, nil
}

func (s *fakeTraderPayoutService) CancelOrder(ctx context.Context, actorID int64, teamID int64, traderID int64, payoutID int64) error {
	return nil
}

func (s *fakeTraderPayoutService) AddTransfer(ctx context.Context, params payouts.AddTransferParams) (payouts.Transfer, error) {
	s.addTransferParams = params
	if s.addTransferErr != nil {
		return payouts.Transfer{}, s.addTransferErr
	}
	return s.transfer, nil
}

func (s *fakeTraderPayoutService) DeleteTransfer(ctx context.Context, actorID int64, teamID int64, traderID int64, payoutID int64, transferID int64) error {
	return nil
}
