package payouts

import "time"

const (
	StatusDraft      = "draft"
	StatusInProgress = "in_progress"
	StatusPaid       = "paid"
	StatusCancelled  = "cancelled"
)

type Order struct {
	ID                   int64
	TeamID               int64
	ShiftID              int64
	TraderID             int64
	DestinationBank      string
	DestinationRequisite string
	AmountMinor          int64
	PaidAmountMinor      int64
	RemainingAmountMinor int64
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

type Transfer struct {
	ID                     int64
	TeamID                 int64
	ManualPayoutOrderID    int64
	ShiftID                int64
	TraderID               int64
	SourceShiftRequisiteID int64
	SourceRequisiteID      int64
	AmountMinor            int64
	CreatedBy              int64
	CreatedAt              time.Time
	Comment                *string
}

