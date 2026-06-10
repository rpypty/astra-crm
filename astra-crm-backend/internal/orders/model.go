package orders

import (
	"errors"
	"time"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"

	SortCreatedAtDesc = "created_at_desc"
	SortCreatedAtAsc  = "created_at_asc"
	SortAmountAsc     = "amount_asc"
	SortAmountDesc    = "amount_desc"

	DefaultPage     = int64(1)
	DefaultPageSize = int64(50)
	MaxPageSize     = int64(200)
)

var (
	ErrInvalidInput   = errors.New("invalid orders input")
	ErrNoCurrentShift = errors.New("current shift not found")
)

type Filters struct {
	DateFrom   *time.Time
	DateTo     *time.Time
	TraderID   *int64
	WorkerName *string
	Requisite  *string
	MethodType *string
	Status     *string
	AmountFrom *int64
	AmountTo   *int64
	Page       int64
	PageSize   int64
	Sort       string
}

type Scope struct {
	TeamID    int64
	TraderID  *int64
	Direction string
}

type ListResult struct {
	Items    []Order
	Page     int64
	PageSize int64
	Total    int64
}

type Order struct {
	ScopeItemID       int64
	ExternalOrderID   int64
	ExternalID        string
	ExternalInnerID   string
	WorkerName        string
	TraderID          *int64
	TraderLogin       *string
	RequisiteRaw      *string
	RequisitePhone    *string
	MethodType        *string
	MethodName        *string
	AmountMinor       int64
	Currency          string
	RawStatus         string
	NormalizedStatus  string
	CreatedAtExternal time.Time
	ImportBatchID     int64
}

type Dashboard struct {
	Summary         Summary
	StatusBreakdown []StatusBreakdownItem
	UnknownStatuses []string
	RecentImports   []ImportHistoryItem
}

type Summary struct {
	TotalAmountMinor   int64
	TotalCount         int64
	SuccessAmountMinor int64
	SuccessCount       int64
	FailedAmountMinor  int64
	FailedCount        int64
	UnknownAmountMinor int64
	UnknownCount       int64
}

type StatusBreakdownItem struct {
	RawStatus        string
	NormalizedStatus string
	AmountMinor      int64
	Count            int64
}

type ImportHistoryItem struct {
	ID                  int64
	TeamID              int64
	UploadedBy          int64
	ScopeType           string
	Direction           string
	ShiftID             *int64
	AccountingPeriodID  *int64
	TraderID            *int64
	FileName            string
	RowsCount           int64
	Status              string
	SupersededByBatchID *int64
	ErrorMessage        *string
	CreatedAt           time.Time
	AppliedAt           *time.Time
}
