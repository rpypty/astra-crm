package payouts

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

var ErrInvalidInput = errors.New("invalid payout input")

type Store interface {
	ListOrders(ctx context.Context, teamID int64, traderID int64) ([]Order, error)
	GetOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, error)
	CreateOrder(ctx context.Context, params CreateOrderRecord) (Order, error)
	UpdateOrder(ctx context.Context, params UpdateOrderRecord) (Order, error)
	CancelOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, error)
	AddTransfer(ctx context.Context, params AddTransferRecord) (Transfer, error)
	DeleteTransfer(ctx context.Context, teamID int64, traderID int64, payoutID int64, transferID int64) (Transfer, error)
	ListTransfers(ctx context.Context, teamID int64, traderID int64, payoutID int64) ([]Transfer, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type MutationHook interface {
	AfterManualPayoutChanged(ctx context.Context, teamID int64, traderID int64, shiftID int64) error
}

type Service struct {
	store Store
	audit AuditService
	hook  MutationHook
}

func NewService(store Store, auditService AuditService, hooks ...MutationHook) *Service {
	service := &Service{
		store: store,
		audit: auditService,
	}
	if len(hooks) > 0 {
		service.hook = hooks[0]
	}

	return service
}

type CreateOrderParams struct {
	ActorID              int64
	TeamID               int64
	TraderID             int64
	DestinationBank      string
	DestinationRequisite string
	AmountMinor          int64
}

type PatchOrderParams struct {
	ActorID              int64
	TeamID               int64
	TraderID             int64
	PayoutID             int64
	DestinationBank      *string
	DestinationRequisite *string
	AmountMinor          *int64
}

type AddTransferParams struct {
	ActorID                int64
	TeamID                 int64
	TraderID               int64
	PayoutID               int64
	SourceShiftRequisiteID int64
	AmountMinor            int64
	Comment                *string
}

func (s *Service) ListOrders(ctx context.Context, teamID int64, traderID int64) ([]Order, error) {
	return s.store.ListOrders(ctx, teamID, traderID)
}

func (s *Service) GetOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, []Transfer, error) {
	order, err := s.store.GetOrder(ctx, teamID, traderID, payoutID)
	if err != nil {
		return Order{}, nil, err
	}

	transfers, err := s.store.ListTransfers(ctx, teamID, traderID, payoutID)
	if err != nil {
		return Order{}, nil, err
	}

	return order, transfers, nil
}

func (s *Service) CreateOrder(ctx context.Context, params CreateOrderParams) (Order, error) {
	destinationBank := strings.TrimSpace(params.DestinationBank)
	destinationRequisite := strings.TrimSpace(params.DestinationRequisite)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || destinationBank == "" || destinationRequisite == "" || params.AmountMinor <= 0 {
		return Order{}, ErrInvalidInput
	}

	order, err := s.store.CreateOrder(ctx, CreateOrderRecord{
		TeamID:               params.TeamID,
		TraderID:             params.TraderID,
		DestinationBank:      destinationBank,
		DestinationRequisite: destinationRequisite,
		AmountMinor:          params.AmountMinor,
	})
	if err != nil {
		return Order{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionManualPayoutCreated,
		EntityType: "manual_payout_order",
		EntityID:   strconv.FormatInt(order.ID, 10),
		After:      PublicOrderFromDomain(order),
	}); err != nil {
		return Order{}, err
	}

	if err := s.afterManualPayoutChanged(ctx, order.TeamID, order.TraderID, order.ShiftID); err != nil {
		return Order{}, err
	}

	return order, nil
}

func (s *Service) PatchOrder(ctx context.Context, params PatchOrderParams) (Order, error) {
	current, err := s.store.GetOrder(ctx, params.TeamID, params.TraderID, params.PayoutID)
	if err != nil {
		return Order{}, err
	}

	next := current
	if params.DestinationBank != nil {
		destinationBank := strings.TrimSpace(*params.DestinationBank)
		if destinationBank == "" {
			return Order{}, ErrInvalidInput
		}
		next.DestinationBank = destinationBank
	}
	if params.DestinationRequisite != nil {
		destinationRequisite := strings.TrimSpace(*params.DestinationRequisite)
		if destinationRequisite == "" {
			return Order{}, ErrInvalidInput
		}
		next.DestinationRequisite = destinationRequisite
	}
	if params.AmountMinor != nil {
		if *params.AmountMinor <= 0 {
			return Order{}, ErrInvalidInput
		}
		next.AmountMinor = *params.AmountMinor
	}

	updated, err := s.store.UpdateOrder(ctx, UpdateOrderRecord{
		TeamID:               params.TeamID,
		TraderID:             params.TraderID,
		PayoutID:             params.PayoutID,
		DestinationBank:      next.DestinationBank,
		DestinationRequisite: next.DestinationRequisite,
		AmountMinor:          next.AmountMinor,
	})
	if err != nil {
		return Order{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionManualPayoutUpdated,
		EntityType: "manual_payout_order",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		Before:     PublicOrderFromDomain(current),
		After:      PublicOrderFromDomain(updated),
	}); err != nil {
		return Order{}, err
	}

	if err := s.afterManualPayoutChanged(ctx, updated.TeamID, updated.TraderID, updated.ShiftID); err != nil {
		return Order{}, err
	}

	return updated, nil
}

func (s *Service) CancelOrder(ctx context.Context, actorID int64, teamID int64, traderID int64, payoutID int64) error {
	order, err := s.store.CancelOrder(ctx, teamID, traderID, payoutID)
	if err != nil {
		return err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     teamID,
		ActorID:    actorID,
		Action:     audit.ActionManualPayoutCancelled,
		EntityType: "manual_payout_order",
		EntityID:   strconv.FormatInt(order.ID, 10),
		After:      PublicOrderFromDomain(order),
	}); err != nil {
		return err
	}

	return s.afterManualPayoutChanged(ctx, order.TeamID, order.TraderID, order.ShiftID)
}

func (s *Service) AddTransfer(ctx context.Context, params AddTransferParams) (Transfer, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.PayoutID <= 0 || params.SourceShiftRequisiteID <= 0 || params.AmountMinor <= 0 {
		return Transfer{}, ErrInvalidInput
	}

	comment := cleanOptionalString(params.Comment)
	transfer, err := s.store.AddTransfer(ctx, AddTransferRecord{
		TeamID:                 params.TeamID,
		TraderID:               params.TraderID,
		PayoutID:               params.PayoutID,
		SourceShiftRequisiteID: params.SourceShiftRequisiteID,
		AmountMinor:            params.AmountMinor,
		CreatedBy:              params.ActorID,
		Comment:                comment,
	})
	if err != nil {
		return Transfer{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionManualPayoutTransferAdded,
		EntityType: "manual_payout_transfer",
		EntityID:   strconv.FormatInt(transfer.ID, 10),
		After:      PublicTransferFromDomain(transfer),
		Comment:    comment,
	}); err != nil {
		return Transfer{}, err
	}

	if err := s.afterManualPayoutChanged(ctx, transfer.TeamID, transfer.TraderID, transfer.ShiftID); err != nil {
		return Transfer{}, err
	}

	return transfer, nil
}

func (s *Service) DeleteTransfer(ctx context.Context, actorID int64, teamID int64, traderID int64, payoutID int64, transferID int64) error {
	transfer, err := s.store.DeleteTransfer(ctx, teamID, traderID, payoutID, transferID)
	if err != nil {
		return err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     teamID,
		ActorID:    actorID,
		Action:     audit.ActionManualPayoutTransferDeleted,
		EntityType: "manual_payout_transfer",
		EntityID:   strconv.FormatInt(transfer.ID, 10),
		Before:     PublicTransferFromDomain(transfer),
	}); err != nil {
		return err
	}

	return s.afterManualPayoutChanged(ctx, transfer.TeamID, transfer.TraderID, transfer.ShiftID)
}

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}

func (s *Service) afterManualPayoutChanged(ctx context.Context, teamID int64, traderID int64, shiftID int64) error {
	if s.hook == nil {
		return nil
	}

	return s.hook.AfterManualPayoutChanged(ctx, teamID, traderID, shiftID)
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
