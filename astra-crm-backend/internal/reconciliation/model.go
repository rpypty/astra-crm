package reconciliation

import (
	"encoding/json"
	"time"
)

const (
	TypeTraderShiftInbound    = "trader_shift_inbound"
	TypeTraderShiftOutbound   = "trader_shift_outbound"
	TypeTeamleadPeriodInbound = "teamlead_period_inbound"

	StatusMatched             = "matched"
	StatusMismatch            = "mismatch"
	StatusAcceptedWithComment = "accepted_with_comment"
)

type Run struct {
	ID                  int64
	TeamID              int64
	Type                string
	ScopeType           string
	ShiftID             *int64
	AccountingPeriodID  *int64
	TraderID            *int64
	ImportBatchID       *int64
	ExpectedAmountMinor int64
	ActualAmountMinor   int64
	DiffAmountMinor     int64
	SuccessAmountMinor  int64
	SuccessCount        int64
	FailedAmountMinor   int64
	FailedCount         int64
	TotalAmountMinor    int64
	TotalCount          int64
	Status              string
	Comment             *string
	ConfirmedBy         *int64
	ConfirmedAt         *time.Time
	CreatedAt           time.Time
}

type Item struct {
	ID                  int64
	ReconciliationRunID int64
	IssueType           string
	ExternalOrderID     *int64
	ExternalInnerID     *string
	TeamleadValueJSON   json.RawMessage
	TraderValueJSON     json.RawMessage
	Message             *string
	CreatedAt           time.Time
}

type RecalculateTraderInboundRecord struct {
	TeamID        int64
	TraderID      int64
	ShiftID       int64
	ImportBatchID *int64
}

type RecalculateTraderOutboundRecord struct {
	TeamID        int64
	TraderID      int64
	ShiftID       int64
	ImportBatchID *int64
}

type RecalculateTeamleadPeriodInboundRecord struct {
	TeamID             int64
	AccountingPeriodID int64
	ImportBatchID      *int64
}

type AcceptTraderInboundRecord struct {
	RunID    int64
	TeamID   int64
	TraderID int64
	ActorID  int64
	Comment  string
}

type AcceptTraderOutboundRecord struct {
	RunID    int64
	TeamID   int64
	TraderID int64
	ActorID  int64
	Comment  string
}

type TeamleadInboundPeriodScope struct {
	AccountingPeriodID int64
	ImportBatchID      int64
}
