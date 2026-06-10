package payouts

import "time"

type PublicOrder struct {
	ID                   int64      `json:"id"`
	TeamID               int64      `json:"teamId"`
	ShiftID              int64      `json:"shiftId"`
	TraderID             int64      `json:"traderId"`
	DestinationBank      string     `json:"destinationBank"`
	DestinationRequisite string     `json:"destinationRequisite"`
	AmountMinor          int64      `json:"amountMinor"`
	PaidAmountMinor      int64      `json:"paidAmountMinor"`
	RemainingAmountMinor int64      `json:"remainingAmountMinor"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	DeletedAt            *time.Time `json:"deletedAt,omitempty"`
}

type PublicTransfer struct {
	ID                     int64     `json:"id"`
	TeamID                 int64     `json:"teamId"`
	ManualPayoutOrderID    int64     `json:"manualPayoutOrderId"`
	ShiftID                int64     `json:"shiftId"`
	TraderID               int64     `json:"traderId"`
	SourceShiftRequisiteID int64     `json:"sourceShiftRequisiteId"`
	SourceRequisiteID      int64     `json:"sourceRequisiteId"`
	AmountMinor            int64     `json:"amountMinor"`
	CreatedBy              int64     `json:"createdBy"`
	CreatedAt              time.Time `json:"createdAt"`
	Comment                *string   `json:"comment,omitempty"`
}

func PublicOrderFromDomain(order Order) PublicOrder {
	return PublicOrder{
		ID:                   order.ID,
		TeamID:               order.TeamID,
		ShiftID:              order.ShiftID,
		TraderID:             order.TraderID,
		DestinationBank:      order.DestinationBank,
		DestinationRequisite: order.DestinationRequisite,
		AmountMinor:          order.AmountMinor,
		PaidAmountMinor:      order.PaidAmountMinor,
		RemainingAmountMinor: order.RemainingAmountMinor,
		Status:               order.Status,
		CreatedAt:            order.CreatedAt,
		UpdatedAt:            order.UpdatedAt,
		DeletedAt:            order.DeletedAt,
	}
}

func PublicOrders(items []Order) []PublicOrder {
	result := make([]PublicOrder, 0, len(items))
	for _, item := range items {
		result = append(result, PublicOrderFromDomain(item))
	}

	return result
}

func PublicTransferFromDomain(transfer Transfer) PublicTransfer {
	return PublicTransfer{
		ID:                     transfer.ID,
		TeamID:                 transfer.TeamID,
		ManualPayoutOrderID:    transfer.ManualPayoutOrderID,
		ShiftID:                transfer.ShiftID,
		TraderID:               transfer.TraderID,
		SourceShiftRequisiteID: transfer.SourceShiftRequisiteID,
		SourceRequisiteID:      transfer.SourceRequisiteID,
		AmountMinor:            transfer.AmountMinor,
		CreatedBy:              transfer.CreatedBy,
		CreatedAt:              transfer.CreatedAt,
		Comment:                transfer.Comment,
	}
}

func PublicTransfers(items []Transfer) []PublicTransfer {
	result := make([]PublicTransfer, 0, len(items))
	for _, item := range items {
		result = append(result, PublicTransferFromDomain(item))
	}

	return result
}

