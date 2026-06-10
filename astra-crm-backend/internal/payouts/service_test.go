package payouts

import (
	"context"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

func TestServiceCreateOrderAuditsMutation(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	order, err := service.CreateOrder(context.Background(), CreateOrderParams{
		ActorID:              1,
		TeamID:               2,
		TraderID:             3,
		DestinationBank:      "Tinkoff",
		DestinationRequisite: "79990000000",
		AmountMinor:          100000,
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	if order.AmountMinor != 100000 {
		t.Fatalf("order amount = %d, want 100000", order.AmountMinor)
	}
	if store.createRecord.DestinationBank != "Tinkoff" {
		t.Fatalf("destination bank = %q, want Tinkoff", store.createRecord.DestinationBank)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionManualPayoutCreated {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionManualPayoutCreated)
	}
}

func TestServiceCreateOrderRejectsInvalidAmount(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.CreateOrder(context.Background(), CreateOrderParams{
		ActorID:              1,
		TeamID:               2,
		TraderID:             3,
		DestinationBank:      "Tinkoff",
		DestinationRequisite: "79990000000",
		AmountMinor:          0,
	})
	if err != ErrInvalidInput {
		t.Fatalf("CreateOrder() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceAddTransferAuditsMutation(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	comment := "частичная выплата"
	transfer, err := service.AddTransfer(context.Background(), AddTransferParams{
		ActorID:                1,
		TeamID:                 2,
		TraderID:               3,
		PayoutID:               40,
		SourceShiftRequisiteID: 20,
		AmountMinor:            50000,
		Comment:                &comment,
	})
	if err != nil {
		t.Fatalf("AddTransfer() error = %v", err)
	}

	if transfer.AmountMinor != 50000 {
		t.Fatalf("transfer amount = %d, want 50000", transfer.AmountMinor)
	}
	if store.transferRecord.SourceShiftRequisiteID != 20 {
		t.Fatalf("source shift requisite id = %d, want 20", store.transferRecord.SourceShiftRequisiteID)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionManualPayoutTransferAdded {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionManualPayoutTransferAdded)
	}
}

func TestServiceAddTransferRunsMutationHook(t *testing.T) {
	hook := &fakeMutationHook{}
	service := NewService(&fakeStore{}, nil, hook)

	_, err := service.AddTransfer(context.Background(), AddTransferParams{
		ActorID:                1,
		TeamID:                 2,
		TraderID:               3,
		PayoutID:               40,
		SourceShiftRequisiteID: 20,
		AmountMinor:            50000,
	})
	if err != nil {
		t.Fatalf("AddTransfer() error = %v", err)
	}

	if !hook.called || hook.teamID != 2 || hook.traderID != 3 || hook.shiftID != 10 {
		t.Fatalf("hook call = called:%v team:%d trader:%d shift:%d, want true/2/3/10", hook.called, hook.teamID, hook.traderID, hook.shiftID)
	}
}

func TestServicePatchOrderRejectsAmountBelowTransferredSum(t *testing.T) {
	service := NewService(&fakeStore{
		updateErr: ErrOrderUpdateRejected,
	}, nil)

	amount := int64(10)
	_, err := service.PatchOrder(context.Background(), PatchOrderParams{
		ActorID:     1,
		TeamID:      2,
		TraderID:    3,
		PayoutID:    40,
		AmountMinor: &amount,
	})
	if err != ErrOrderUpdateRejected {
		t.Fatalf("PatchOrder() error = %v, want ErrOrderUpdateRejected", err)
	}
}

type fakeStore struct {
	createRecord   CreateOrderRecord
	updateRecord   UpdateOrderRecord
	updateErr      error
	transferRecord AddTransferRecord
}

func (s *fakeStore) ListOrders(ctx context.Context, teamID int64, traderID int64) ([]Order, error) {
	return nil, nil
}

func (s *fakeStore) GetOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, error) {
	return Order{
		ID:                   payoutID,
		TeamID:               teamID,
		ShiftID:              10,
		TraderID:             traderID,
		DestinationBank:      "Tinkoff",
		DestinationRequisite: "79990000000",
		AmountMinor:          100000,
		PaidAmountMinor:      50000,
		RemainingAmountMinor: 50000,
		Status:               StatusInProgress,
		CreatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) CreateOrder(ctx context.Context, params CreateOrderRecord) (Order, error) {
	s.createRecord = params
	return Order{
		ID:                   40,
		TeamID:               params.TeamID,
		ShiftID:              10,
		TraderID:             params.TraderID,
		DestinationBank:      params.DestinationBank,
		DestinationRequisite: params.DestinationRequisite,
		AmountMinor:          params.AmountMinor,
		RemainingAmountMinor: params.AmountMinor,
		Status:               StatusInProgress,
		CreatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) UpdateOrder(ctx context.Context, params UpdateOrderRecord) (Order, error) {
	s.updateRecord = params
	if s.updateErr != nil {
		return Order{}, s.updateErr
	}
	return Order{
		ID:                   params.PayoutID,
		TeamID:               params.TeamID,
		ShiftID:              10,
		TraderID:             params.TraderID,
		DestinationBank:      params.DestinationBank,
		DestinationRequisite: params.DestinationRequisite,
		AmountMinor:          params.AmountMinor,
		RemainingAmountMinor: params.AmountMinor,
		Status:               StatusInProgress,
		CreatedAt:            time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) CancelOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, error) {
	return Order{ID: payoutID, TeamID: teamID, ShiftID: 10, TraderID: traderID, Status: StatusCancelled}, nil
}

func (s *fakeStore) AddTransfer(ctx context.Context, params AddTransferRecord) (Transfer, error) {
	s.transferRecord = params
	return Transfer{
		ID:                     50,
		TeamID:                 params.TeamID,
		ManualPayoutOrderID:    params.PayoutID,
		ShiftID:                10,
		TraderID:               params.TraderID,
		SourceShiftRequisiteID: params.SourceShiftRequisiteID,
		SourceRequisiteID:      4,
		AmountMinor:            params.AmountMinor,
		CreatedBy:              params.CreatedBy,
		CreatedAt:              time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
		Comment:                params.Comment,
	}, nil
}

func (s *fakeStore) DeleteTransfer(ctx context.Context, teamID int64, traderID int64, payoutID int64, transferID int64) (Transfer, error) {
	return Transfer{ID: transferID, TeamID: teamID, ShiftID: 10, TraderID: traderID, ManualPayoutOrderID: payoutID}, nil
}

func (s *fakeStore) ListTransfers(ctx context.Context, teamID int64, traderID int64, payoutID int64) ([]Transfer, error) {
	return nil, nil
}

type fakeAuditService struct {
	events []audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.events = append(s.events, event)
	return nil
}

type fakeMutationHook struct {
	called   bool
	teamID   int64
	traderID int64
	shiftID  int64
}

func (h *fakeMutationHook) AfterManualPayoutChanged(ctx context.Context, teamID int64, traderID int64, shiftID int64) error {
	h.called = true
	h.teamID = teamID
	h.traderID = traderID
	h.shiftID = shiftID
	return nil
}
