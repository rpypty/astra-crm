package reconciliation

import (
	"encoding/json"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/pagination"
)

const (
	TypeTraderShiftInbound     = "trader_shift_inbound"
	TypeTraderShiftOutbound    = "trader_shift_outbound"
	TypeTeamleadPeriodInbound  = "teamlead_period_inbound"
	TypeTeamleadPeriodOutbound = "teamlead_period_outbound"

	StatusMatched             = "matched"
	StatusMismatch            = "mismatch"
	StatusAcceptedWithComment = "accepted_with_comment"

	TeamleadRunStatusDraft       = "draft"
	TeamleadRunStatusAnalyzing   = "analyzing"
	TeamleadRunStatusMatched     = "matched"
	TeamleadRunStatusMismatch    = "mismatch"
	TeamleadRunStatusApplyQueued = "apply_queued"
	TeamleadRunStatusApplying    = "applying"
	TeamleadRunStatusApplied     = "applied"
	TeamleadRunStatusApplyFailed = "apply_failed"
	TeamleadRunStatusRejected    = "rejected"

	TLStatusNotChecked    = "not_checked"
	TLStatusConfirmedByTL = "confirmed_by_tl"
	TLStatusUpdatedByTL   = "updated_by_tl"
	TLStatusDiscrepancy   = "tl_discrepancy"
	TLStatusAccepted      = "tl_accepted"

	TeamleadItemStageNormalization    = "normalization"
	TeamleadItemStageMatching         = "matching"
	TeamleadItemStageTurnoverCheck    = "turnover_check"
	TeamleadItemStageTransactionCheck = "transaction_check"
	TeamleadItemStagePreview          = "preview"
	TeamleadItemStageApply            = "apply"

	TeamleadItemSeverityInfo    = "info"
	TeamleadItemSeverityWarning = "warning"
	TeamleadItemSeverityError   = "error"
	TeamleadItemSeverityBlocker = "blocker"
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

type TeamleadRun struct {
	ID                    int64
	TeamID                int64
	DateFrom              time.Time
	DateTo                time.Time
	Status                string
	CreatedBy             int64
	ConfirmedBy           *int64
	RejectedBy            *int64
	InboundImportBatchID  *int64
	OutboundImportBatchID *int64
	Comment               *string
	MismatchCount         int64
	ConflictCount         int64
	BlockedCount          int64
	PipelineJSON          json.RawMessage
	InboundSummaryJSON    json.RawMessage
	OutboundSummaryJSON   json.RawMessage
	PreviewJSON           json.RawMessage
	ApplyResultJSON       json.RawMessage
	ErrorMessage          *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AnalyzedAt            *time.Time
	ConfirmedAt           *time.Time
	RejectedAt            *time.Time
	ApplyQueuedAt         *time.Time
	AppliedAt             *time.Time
}

type TeamleadItem struct {
	ID                       int64
	TeamleadReconciliationID int64
	TeamID                   int64
	Direction                string
	Stage                    string
	IssueType                string
	Severity                 string
	ExternalOrderID          *int64
	ExternalInnerID          *string
	TraderID                 *int64
	RequisiteID              *int64
	ShiftID                  *int64
	BeforeJSON               json.RawMessage
	AfterJSON                json.RawMessage
	Message                  *string
	IsBlocking               bool
	AppliedAt                *time.Time
	CreatedAt                time.Time
}

type TeamleadCSVInput struct {
	FileName string
	Payload  []byte
}

type CreateTeamleadReconciliationParams struct {
	ActorID   int64
	TeamID    int64
	DateFrom  time.Time
	DateTo    time.Time
	TraderIDs []int64
	Inbound   *TeamleadCSVInput
	Outbound  *TeamleadCSVInput
}

type ConfirmTeamleadReconciliationParams struct {
	ActorID int64
	TeamID  int64
	RunID   int64
	Comment string
}

type RejectTeamleadReconciliationParams struct {
	ActorID int64
	TeamID  int64
	RunID   int64
	Comment string
}

type ListTeamleadReconciliationsParams struct {
	TeamID int64
	Page   pagination.Params
}

type GetTeamleadReconciliationParams struct {
	TeamID int64
	RunID  int64
}

type TeamleadDirectionSummary struct {
	Direction          string `json:"direction"`
	RowsTotal          int64  `json:"rowsTotal"`
	RowsInPeriod       int64  `json:"rowsInPeriod"`
	SuccessAmountMinor int64  `json:"successAmountMinor"`
	SuccessCount       int64  `json:"successCount"`
	FailedAmountMinor  int64  `json:"failedAmountMinor"`
	FailedCount        int64  `json:"failedCount"`
	TotalAmountMinor   int64  `json:"totalAmountMinor"`
	TotalCount         int64  `json:"totalCount"`
	CRMAmountMinor     int64  `json:"crmAmountMinor"`
	CRMCount           int64  `json:"crmCount"`
	DiffAmountMinor    int64  `json:"diffAmountMinor"`
	CreateCount        int64  `json:"createCount"`
	UpdateCount        int64  `json:"updateCount"`
	UnchangedCount     int64  `json:"unchangedCount"`
	ApplyRowsCount     int64  `json:"applyRowsCount"`
	BlockedCount       int64  `json:"blockedCount"`
}

type TeamleadDirectionAnalysisRecord struct {
	Direction string
	FileName  string
	FileHash  string
	Rows      []imports.ParsedOrderRow
	Filtered  []imports.ParsedOrderRow
	Summary   TeamleadDirectionSummary
}

type TeamleadItemRecord struct {
	Direction       string
	Stage           string
	IssueType       string
	Severity        string
	ExternalOrderID *int64
	ExternalInnerID *string
	TraderID        *int64
	RequisiteID     *int64
	ShiftID         *int64
	BeforeJSON      json.RawMessage
	AfterJSON       json.RawMessage
	Message         *string
	IsBlocking      bool
}

type TeamleadItemFilters struct {
	Direction    string
	Stage        string
	IssueType    string
	Severity     string
	TraderID     *int64
	RequisiteID  *int64
	OnlyMismatch bool
}

type CreateTeamleadAnalysisRecord struct {
	TeamID          int64
	ActorID         int64
	DateFrom        time.Time
	DateTo          time.Time
	Status          string
	MismatchCount   int64
	ConflictCount   int64
	BlockedCount    int64
	PipelineJSON    json.RawMessage
	InboundSummary  json.RawMessage
	OutboundSummary json.RawMessage
	PreviewJSON     json.RawMessage
	Directions      []TeamleadDirectionAnalysisRecord
	Items           []TeamleadItemRecord
}

type QueueTeamleadApplyRecord struct {
	RunID   int64
	TeamID  int64
	ActorID int64
	Comment *string
}

type RejectTeamleadReconciliationRecord struct {
	RunID   int64
	TeamID  int64
	ActorID int64
	Comment string
}

type TeamleadApplyResult struct {
	RunID             int64                          `json:"runId"`
	CreatedOrders     int64                          `json:"createdOrders"`
	UpdatedOrders     int64                          `json:"updatedOrders"`
	ConfirmedOrders   int64                          `json:"confirmedOrders"`
	DiscrepancyOrders int64                          `json:"discrepancyOrders"`
	Directions        []TeamleadDirectionApplyResult `json:"directions"`
}

type TeamleadDirectionApplyResult struct {
	Direction         string `json:"direction"`
	RowsApplied       int64  `json:"rowsApplied"`
	CreatedOrders     int64  `json:"createdOrders"`
	UpdatedOrders     int64  `json:"updatedOrders"`
	ConfirmedOrders   int64  `json:"confirmedOrders"`
	DiscrepancyOrders int64  `json:"discrepancyOrders"`
}

type TeamleadTraderMatch struct {
	TraderID           int64
	ExternalWorkerName string
}

type TeamleadRequisiteMatch struct {
	ID                   int64
	BankCode             string
	Phone                string
	CardNumber           *string
	NormalizedPhone      string
	NormalizedCardNumber string
}

type TeamleadBankAliasMatch struct {
	BankCode string
	BankName string
	CSVAlias *string
}

type TeamleadExternalOrderSnapshot struct {
	ID                int64
	Direction         string
	ExternalInnerID   string
	WorkerName        string
	TraderID          *int64
	RequisiteRaw      *string
	RequisitePhone    *string
	MethodType        *string
	MethodName        *string
	RequisiteID       *int64
	AmountMinor       int64
	RawStatus         string
	NormalizedStatus  string
	CreatedAtExternal time.Time
}

type TeamleadTurnoverSnapshot struct {
	AmountMinor int64
	Count       int64
}

type ItemFilters struct {
	Status       string
	OnlyMismatch bool
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

type RecalculateTeamleadPeriodOutboundRecord struct {
	TeamID             int64
	AccountingPeriodID int64
	ImportBatchID      *int64
}

type RecalculateTeamleadCurrentRecord struct {
	TeamID        int64
	ActorID       int64
	Direction     string
	ImportBatchID *int64
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

type AcceptTeamleadCurrentRecord struct {
	RunID   int64
	TeamID  int64
	ActorID int64
	Comment string
}

type TeamleadInboundPeriodScope struct {
	AccountingPeriodID int64
	ImportBatchID      int64
}

type TeamleadOutboundPeriodScope struct {
	AccountingPeriodID int64
	ImportBatchID      int64
}
