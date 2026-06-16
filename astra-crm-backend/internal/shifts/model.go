package shifts

import "time"

const (
	StatusOpen                  = "open"
	StatusClosing               = "closing"
	StatusClosed                = "closed"
	StatusClosedWithDiscrepancy = "closed_with_discrepancy"

	RequisiteStatusActive     = "active"
	RequisiteStatusReleased   = "released"
	RequisiteStatusWorked     = "worked"
	RequisiteStatusCorrection = "correction"
	RequisiteStatusBlocked    = "blocked"
)

type Shift struct {
	ID                           int64
	TeamID                       int64
	TraderID                     int64
	StartedAt                    time.Time
	EndedAt                      *time.Time
	Status                       string
	InboundReconciliationStatus  string
	OutboundReconciliationStatus string
	CloseComment                 *string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
	ClosedAt                     *time.Time
}

type AssignedRequisite struct {
	ID                    int64
	TeamID                int64
	Phone                 string
	MethodType            string
	BankCode              string
	BankName              string
	Proxy                 *string
	EmployeeComment       *string
	Status                string
	AssignmentID          int64
	AssignmentStatus      string
	AssignedForDate       time.Time
	TargetTurnoverMinor   int64
	ShiftRequisiteID      *int64
	CardNumber            *string
	HolderName            *string
	ShiftRequisiteStatus  *string
	TakenAt               *time.Time
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
}

type ShiftRequisite struct {
	ID                    int64
	TeamID                int64
	ShiftID               int64
	TraderID              int64
	RequisiteID           int64
	AssignmentID          *int64
	CardNumber            string
	HolderName            string
	TakenAt               time.Time
	ReleasedAt            *time.Time
	Status                string
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type TurnoverEntry struct {
	ID               int64
	TeamID           int64
	ShiftID          int64
	ShiftRequisiteID int64
	RequisiteID      int64
	TraderID         int64
	AmountMinor      int64
	CreatedBy        int64
	CreatedAt        time.Time
	Comment          *string
}

type CloseChecklist struct {
	Shift               Shift
	InboundImported     bool
	InboundOk           bool
	OutboundImported    bool
	OutboundOk          bool
	AllRequisitesClosed bool
	OpenRequisiteCount  int64
	AllPayoutsFullyPaid bool
	UnpaidPayoutCount   int64
	CanClose            bool
}
