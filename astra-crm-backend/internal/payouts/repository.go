package payouts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var (
	ErrOrderNotFound       = errors.New("payout order not found")
	ErrTransferNotFound    = errors.New("payout transfer not found")
	ErrNoCurrentShift      = errors.New("current shift not found")
	ErrTransferRejected    = errors.New("payout transfer rejected")
	ErrOrderUpdateRejected = errors.New("payout order update rejected")
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) ListOrders(ctx context.Context, teamID int64, traderID int64) ([]Order, error) {
	rows, err := r.queries.ListPayoutOrdersForCurrentShift(ctx, db.ListPayoutOrdersForCurrentShiftParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Order, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListOrderRow(row))
	}

	return items, nil
}

func (r *Repository) GetOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, error) {
	row, err := r.queries.GetPayoutOrderForTrader(ctx, db.GetPayoutOrderForTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
		PayoutID: payoutID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrOrderNotFound
	}
	if err != nil {
		return Order{}, err
	}

	return fromGetOrderRow(row), nil
}

func (r *Repository) CreateOrder(ctx context.Context, params CreateOrderRecord) (Order, error) {
	row, err := r.queries.CreatePayoutOrder(ctx, db.CreatePayoutOrderParams{
		TeamID:               params.TeamID,
		TraderID:             params.TraderID,
		DestinationBank:      params.DestinationBank,
		DestinationRequisite: params.DestinationRequisite,
		AmountMinor:          params.AmountMinor,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNoCurrentShift
	}
	if err != nil {
		return Order{}, err
	}

	return fromDBOrder(row), nil
}

func (r *Repository) UpdateOrder(ctx context.Context, params UpdateOrderRecord) (Order, error) {
	row, err := r.queries.UpdatePayoutOrder(ctx, db.UpdatePayoutOrderParams{
		TeamID:               params.TeamID,
		TraderID:             params.TraderID,
		PayoutID:             params.PayoutID,
		DestinationBank:      params.DestinationBank,
		DestinationRequisite: params.DestinationRequisite,
		AmountMinor:          params.AmountMinor,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrOrderUpdateRejected
	}
	if err != nil {
		return Order{}, err
	}

	return fromUpdateOrderRow(row), nil
}

func (r *Repository) CancelOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (Order, error) {
	row, err := r.queries.CancelPayoutOrder(ctx, db.CancelPayoutOrderParams{
		TeamID:   teamID,
		TraderID: traderID,
		PayoutID: payoutID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrOrderNotFound
	}
	if err != nil {
		return Order{}, err
	}

	return fromDBOrder(row), nil
}

func (r *Repository) AddTransfer(ctx context.Context, params AddTransferRecord) (Transfer, error) {
	row, err := r.queries.AddPayoutTransfer(ctx, db.AddPayoutTransferParams{
		TeamID:                 params.TeamID,
		TraderID:               params.TraderID,
		PayoutID:               params.PayoutID,
		SourceShiftRequisiteID: params.SourceShiftRequisiteID,
		AmountMinor:            params.AmountMinor,
		CreatedBy:              params.CreatedBy,
		Comment:                textValue(params.Comment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Transfer{}, ErrTransferRejected
	}
	if err != nil {
		return Transfer{}, err
	}

	return fromAddTransferRow(row), nil
}

func (r *Repository) DeleteTransfer(ctx context.Context, teamID int64, traderID int64, payoutID int64, transferID int64) (Transfer, error) {
	row, err := r.queries.DeletePayoutTransfer(ctx, db.DeletePayoutTransferParams{
		TeamID:     teamID,
		TraderID:   traderID,
		PayoutID:   payoutID,
		TransferID: transferID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Transfer{}, ErrTransferNotFound
	}
	if err != nil {
		return Transfer{}, err
	}

	return fromDeleteTransferRow(row), nil
}

func (r *Repository) ListTransfers(ctx context.Context, teamID int64, traderID int64, payoutID int64) ([]Transfer, error) {
	rows, err := r.queries.ListPayoutTransfers(ctx, db.ListPayoutTransfersParams{
		TeamID:   teamID,
		TraderID: traderID,
		PayoutID: payoutID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Transfer, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBTransfer(row))
	}

	return items, nil
}

type CreateOrderRecord struct {
	TeamID               int64
	TraderID             int64
	DestinationBank      string
	DestinationRequisite string
	AmountMinor          int64
}

type UpdateOrderRecord struct {
	TeamID               int64
	TraderID             int64
	PayoutID             int64
	DestinationBank      string
	DestinationRequisite string
	AmountMinor          int64
}

type AddTransferRecord struct {
	TeamID                 int64
	TraderID               int64
	PayoutID               int64
	SourceShiftRequisiteID int64
	AmountMinor            int64
	CreatedBy              int64
	Comment                *string
}

func fromListOrderRow(row db.ListPayoutOrdersForCurrentShiftRow) Order {
	return Order{
		ID:                   row.ID,
		TeamID:               row.TeamID,
		ShiftID:              row.ShiftID,
		TraderID:             row.TraderID,
		DestinationBank:      row.DestinationBank,
		DestinationRequisite: row.DestinationRequisite,
		AmountMinor:          row.AmountMinor,
		PaidAmountMinor:      row.PaidAmountMinor,
		RemainingAmountMinor: row.RemainingAmountMinor,
		Status:               row.Status,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
		DeletedAt:            timePtr(row.DeletedAt),
	}
}

func fromGetOrderRow(row db.GetPayoutOrderForTraderRow) Order {
	return Order{
		ID:                   row.ID,
		TeamID:               row.TeamID,
		ShiftID:              row.ShiftID,
		TraderID:             row.TraderID,
		DestinationBank:      row.DestinationBank,
		DestinationRequisite: row.DestinationRequisite,
		AmountMinor:          row.AmountMinor,
		PaidAmountMinor:      row.PaidAmountMinor,
		RemainingAmountMinor: row.RemainingAmountMinor,
		Status:               row.Status,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
		DeletedAt:            timePtr(row.DeletedAt),
	}
}

func fromUpdateOrderRow(row db.UpdatePayoutOrderRow) Order {
	return Order{
		ID:                   row.ID,
		TeamID:               row.TeamID,
		ShiftID:              row.ShiftID,
		TraderID:             row.TraderID,
		DestinationBank:      row.DestinationBank,
		DestinationRequisite: row.DestinationRequisite,
		AmountMinor:          row.AmountMinor,
		PaidAmountMinor:      row.PaidAmountMinor,
		RemainingAmountMinor: row.RemainingAmountMinor,
		Status:               row.Status,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
		DeletedAt:            timePtr(row.DeletedAt),
	}
}

func fromDBOrder(row db.ManualPayoutOrder) Order {
	return Order{
		ID:                   row.ID,
		TeamID:               row.TeamID,
		ShiftID:              row.ShiftID,
		TraderID:             row.TraderID,
		DestinationBank:      row.DestinationBank,
		DestinationRequisite: row.DestinationRequisite,
		AmountMinor:          row.AmountMinor,
		PaidAmountMinor:      0,
		RemainingAmountMinor: row.AmountMinor,
		Status:               row.Status,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
		DeletedAt:            timePtr(row.DeletedAt),
	}
}

func fromDBTransfer(row db.ManualPayoutTransfer) Transfer {
	return Transfer{
		ID:                     row.ID,
		TeamID:                 row.TeamID,
		ManualPayoutOrderID:    row.ManualPayoutOrderID,
		ShiftID:                row.ShiftID,
		TraderID:               row.TraderID,
		SourceShiftRequisiteID: row.SourceShiftRequisiteID,
		SourceRequisiteID:      row.SourceRequisiteID,
		AmountMinor:            row.AmountMinor,
		CreatedBy:              row.CreatedBy,
		CreatedAt:              row.CreatedAt.Time,
		Comment:                textPtr(row.Comment),
	}
}

func fromAddTransferRow(row db.AddPayoutTransferRow) Transfer {
	return Transfer{
		ID:                     row.ID,
		TeamID:                 row.TeamID,
		ManualPayoutOrderID:    row.ManualPayoutOrderID,
		ShiftID:                row.ShiftID,
		TraderID:               row.TraderID,
		SourceShiftRequisiteID: row.SourceShiftRequisiteID,
		SourceRequisiteID:      row.SourceRequisiteID,
		AmountMinor:            row.AmountMinor,
		CreatedBy:              row.CreatedBy,
		CreatedAt:              row.CreatedAt.Time,
		Comment:                textPtr(row.Comment),
	}
}

func fromDeleteTransferRow(row db.DeletePayoutTransferRow) Transfer {
	return Transfer{
		ID:                     row.ID,
		TeamID:                 row.TeamID,
		ManualPayoutOrderID:    row.ManualPayoutOrderID,
		ShiftID:                row.ShiftID,
		TraderID:               row.TraderID,
		SourceShiftRequisiteID: row.SourceShiftRequisiteID,
		SourceRequisiteID:      row.SourceRequisiteID,
		AmountMinor:            row.AmountMinor,
		CreatedBy:              row.CreatedBy,
		CreatedAt:              row.CreatedAt.Time,
		Comment:                textPtr(row.Comment),
	}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func textValue(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
