package imports

import "time"

const (
	ScopeTypeTraderShift    = "trader_shift"
	ScopeTypeTeamleadPeriod = "teamlead_period"

	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"

	BatchStatusParsed     = "parsed"
	BatchStatusApplied    = "applied"
	BatchStatusReconciled = "reconciled"
	BatchStatusSuperseded = "superseded"
	BatchStatusFailed     = "failed"
)

type Scope struct {
	Type               string
	Direction          string
	ShiftID            *int64
	AccountingPeriodID *int64
	TraderID           *int64
}

type ImportBatch struct {
	ID                  int64
	TeamID              int64
	UploadedBy          int64
	ScopeType           string
	Direction           string
	ShiftID             *int64
	AccountingPeriodID  *int64
	TraderID            *int64
	FileName            string
	FileHash            string
	RowsCount           int64
	Status              string
	SupersededByBatchID *int64
	ErrorMessage        *string
	CreatedAt           time.Time
	AppliedAt           *time.Time
}

type ApplyImportRecord struct {
	TeamID     int64
	UploadedBy int64
	Scope      Scope
	FileName   string
	FileHash   string
	Rows       []ParsedOrderRow
}

type ApplyResult struct {
	Batch                 ImportBatch
	Parse                 ParseResult
	RowsCount             int64
	CreatedOrders         int64
	UpdatedOrders         int64
	DeactivatedScopeItems int64
	ActiveScopeItems      int64
	SupersededBatches     int64
}
