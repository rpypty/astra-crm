package shifts

import "time"

type PublicShift struct {
	ID                           int64      `json:"id"`
	TeamID                       int64      `json:"teamId"`
	TraderID                     int64      `json:"traderId"`
	StartedAt                    time.Time  `json:"startedAt"`
	EndedAt                      *time.Time `json:"endedAt,omitempty"`
	Status                       string     `json:"status"`
	InboundReconciliationStatus  string     `json:"inboundReconciliationStatus"`
	OutboundReconciliationStatus string     `json:"outboundReconciliationStatus"`
	CloseComment                 *string    `json:"closeComment,omitempty"`
	CreatedAt                    time.Time  `json:"createdAt"`
	UpdatedAt                    time.Time  `json:"updatedAt"`
	ClosedAt                     *time.Time `json:"closedAt,omitempty"`
}

type PublicAssignedRequisite struct {
	ID                    int64      `json:"id"`
	TeamID                int64      `json:"teamId"`
	Phone                 string     `json:"phone"`
	MethodType            string     `json:"methodType"`
	BankCode              string     `json:"bankCode"`
	BankName              string     `json:"bankName"`
	Proxy                 *string    `json:"proxy,omitempty"`
	EmployeeComment       *string    `json:"employeeComment,omitempty"`
	Status                string     `json:"status"`
	AssignmentID          int64      `json:"assignmentId"`
	AssignmentStatus      string     `json:"assignmentStatus"`
	AssignedForDate       time.Time  `json:"assignedForDate"`
	TargetTurnoverMinor   int64      `json:"targetTurnoverMinor"`
	ShiftRequisiteID      *int64     `json:"shiftRequisiteId,omitempty"`
	CardNumber            *string    `json:"cardNumber,omitempty"`
	HolderName            *string    `json:"holderName,omitempty"`
	ShiftRequisiteStatus  *string    `json:"shiftRequisiteStatus,omitempty"`
	TakenAt               *time.Time `json:"takenAt,omitempty"`
	ReleasedAt            *time.Time `json:"releasedAt,omitempty"`
	InboundTurnoverMinor  int64      `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor int64      `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor   int64      `json:"closingBalanceMinor"`
	InWork                bool       `json:"inWork"`
}

type PublicShiftRequisite struct {
	ID                    int64      `json:"id"`
	TeamID                int64      `json:"teamId"`
	ShiftID               int64      `json:"shiftId"`
	TraderID              int64      `json:"traderId"`
	RequisiteID           int64      `json:"requisiteId"`
	AssignmentID          *int64     `json:"assignmentId,omitempty"`
	CardNumber            string     `json:"cardNumber"`
	HolderName            string     `json:"holderName"`
	TakenAt               time.Time  `json:"takenAt"`
	ReleasedAt            *time.Time `json:"releasedAt,omitempty"`
	Status                string     `json:"status"`
	InboundTurnoverMinor  int64      `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor int64      `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor   int64      `json:"closingBalanceMinor"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type PublicTurnoverEntry struct {
	ID               int64     `json:"id"`
	TeamID           int64     `json:"teamId"`
	ShiftID          int64     `json:"shiftId"`
	ShiftRequisiteID int64     `json:"shiftRequisiteId"`
	RequisiteID      int64     `json:"requisiteId"`
	TraderID         int64     `json:"traderId"`
	AmountMinor      int64     `json:"amountMinor"`
	CreatedBy        int64     `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
	Comment          *string   `json:"comment,omitempty"`
}

type PublicCloseChecklist struct {
	Shift               PublicShift `json:"shift"`
	InboundImported     bool        `json:"inboundImported"`
	InboundOk           bool        `json:"inboundOk"`
	OutboundImported    bool        `json:"outboundImported"`
	OutboundOk          bool        `json:"outboundOk"`
	AllRequisitesClosed bool        `json:"allRequisitesClosed"`
	OpenRequisiteCount  int64       `json:"openRequisiteCount"`
	AllPayoutsFullyPaid bool        `json:"allPayoutsFullyPaid"`
	UnpaidPayoutCount   int64       `json:"unpaidPayoutCount"`
	CanClose            bool        `json:"canClose"`
}

type PublicShiftReportDetails struct {
	Shift    PublicShift                      `json:"shift"`
	Inbound  *PublicShiftReportReconciliation `json:"inbound,omitempty"`
	Outbound *PublicShiftReportReconciliation `json:"outbound,omitempty"`
	Rows     []PublicShiftReportRow           `json:"rows"`
}

type PublicShiftReportReconciliation struct {
	ID                  int64     `json:"id"`
	Status              string    `json:"status"`
	ExpectedAmountMinor int64     `json:"expectedAmountMinor"`
	ActualAmountMinor   int64     `json:"actualAmountMinor"`
	DiffAmountMinor     int64     `json:"diffAmountMinor"`
	Comment             *string   `json:"comment,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

type PublicShiftReportRow struct {
	RowKey                string  `json:"rowKey"`
	ShiftRequisiteID      *int64  `json:"shiftRequisiteId,omitempty"`
	RequisiteID           *int64  `json:"requisiteId,omitempty"`
	Phone                 string  `json:"phone"`
	MethodType            string  `json:"methodType"`
	BankCode              string  `json:"bankCode"`
	BankName              string  `json:"bankName"`
	Proxy                 *string `json:"proxy,omitempty"`
	EmployeeComment       *string `json:"employeeComment,omitempty"`
	CardNumber            *string `json:"cardNumber,omitempty"`
	HolderName            *string `json:"holderName,omitempty"`
	Status                string  `json:"status"`
	InboundTurnoverMinor  int64   `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor int64   `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor   int64   `json:"closingBalanceMinor"`
	TargetTurnoverMinor   int64   `json:"targetTurnoverMinor"`
	CSVInboundMinor       int64   `json:"csvInboundMinor"`
	CSVOutboundMinor      int64   `json:"csvOutboundMinor"`
	InboundDiffMinor      int64   `json:"inboundDiffMinor"`
	OutboundDiffMinor     int64   `json:"outboundDiffMinor"`
	HasMismatch           bool    `json:"hasMismatch"`
	CSVOnly               bool    `json:"csvOnly"`
}

func PublicShiftFromDomain(shift Shift) PublicShift {
	return PublicShift{
		ID:                           shift.ID,
		TeamID:                       shift.TeamID,
		TraderID:                     shift.TraderID,
		StartedAt:                    shift.StartedAt,
		EndedAt:                      shift.EndedAt,
		Status:                       shift.Status,
		InboundReconciliationStatus:  shift.InboundReconciliationStatus,
		OutboundReconciliationStatus: shift.OutboundReconciliationStatus,
		CloseComment:                 shift.CloseComment,
		CreatedAt:                    shift.CreatedAt,
		UpdatedAt:                    shift.UpdatedAt,
		ClosedAt:                     shift.ClosedAt,
	}
}

func PublicShiftsFromDomain(items []Shift) []PublicShift {
	result := make([]PublicShift, 0, len(items))
	for _, item := range items {
		result = append(result, PublicShiftFromDomain(item))
	}

	return result
}

func PublicAssignedRequisiteFromDomain(item AssignedRequisite) PublicAssignedRequisite {
	return PublicAssignedRequisite{
		ID:                    item.ID,
		TeamID:                item.TeamID,
		Phone:                 item.Phone,
		MethodType:            item.MethodType,
		BankCode:              item.BankCode,
		BankName:              item.BankName,
		Proxy:                 item.Proxy,
		EmployeeComment:       item.EmployeeComment,
		Status:                item.Status,
		AssignmentID:          item.AssignmentID,
		AssignmentStatus:      item.AssignmentStatus,
		AssignedForDate:       item.AssignedForDate,
		TargetTurnoverMinor:   item.TargetTurnoverMinor,
		ShiftRequisiteID:      item.ShiftRequisiteID,
		CardNumber:            item.CardNumber,
		HolderName:            item.HolderName,
		ShiftRequisiteStatus:  item.ShiftRequisiteStatus,
		TakenAt:               item.TakenAt,
		ReleasedAt:            item.ReleasedAt,
		InboundTurnoverMinor:  item.InboundTurnoverMinor,
		OutboundTurnoverMinor: item.OutboundTurnoverMinor,
		ClosingBalanceMinor:   item.ClosingBalanceMinor,
		InWork:                item.ShiftRequisiteID != nil && item.ShiftRequisiteStatus != nil && *item.ShiftRequisiteStatus == RequisiteStatusActive,
	}
}

func PublicAssignedRequisites(items []AssignedRequisite) []PublicAssignedRequisite {
	result := make([]PublicAssignedRequisite, 0, len(items))
	for _, item := range items {
		result = append(result, PublicAssignedRequisiteFromDomain(item))
	}

	return result
}

func PublicShiftRequisiteFromDomain(item ShiftRequisite) PublicShiftRequisite {
	return PublicShiftRequisite{
		ID:                    item.ID,
		TeamID:                item.TeamID,
		ShiftID:               item.ShiftID,
		TraderID:              item.TraderID,
		RequisiteID:           item.RequisiteID,
		AssignmentID:          item.AssignmentID,
		CardNumber:            item.CardNumber,
		HolderName:            item.HolderName,
		TakenAt:               item.TakenAt,
		ReleasedAt:            item.ReleasedAt,
		Status:                item.Status,
		InboundTurnoverMinor:  item.InboundTurnoverMinor,
		OutboundTurnoverMinor: item.OutboundTurnoverMinor,
		ClosingBalanceMinor:   item.ClosingBalanceMinor,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func PublicShiftRequisites(items []ShiftRequisite) []PublicShiftRequisite {
	result := make([]PublicShiftRequisite, 0, len(items))
	for _, item := range items {
		result = append(result, PublicShiftRequisiteFromDomain(item))
	}

	return result
}

func PublicTurnoverEntryFromDomain(item TurnoverEntry) PublicTurnoverEntry {
	return PublicTurnoverEntry{
		ID:               item.ID,
		TeamID:           item.TeamID,
		ShiftID:          item.ShiftID,
		ShiftRequisiteID: item.ShiftRequisiteID,
		RequisiteID:      item.RequisiteID,
		TraderID:         item.TraderID,
		AmountMinor:      item.AmountMinor,
		CreatedBy:        item.CreatedBy,
		CreatedAt:        item.CreatedAt,
		Comment:          item.Comment,
	}
}

func PublicTurnoverEntries(items []TurnoverEntry) []PublicTurnoverEntry {
	result := make([]PublicTurnoverEntry, 0, len(items))
	for _, item := range items {
		result = append(result, PublicTurnoverEntryFromDomain(item))
	}

	return result
}

func PublicCloseChecklistFromDomain(checklist CloseChecklist) PublicCloseChecklist {
	return PublicCloseChecklist{
		Shift:               PublicShiftFromDomain(checklist.Shift),
		InboundImported:     checklist.InboundImported,
		InboundOk:           checklist.InboundOk,
		OutboundImported:    checklist.OutboundImported,
		OutboundOk:          checklist.OutboundOk,
		AllRequisitesClosed: checklist.AllRequisitesClosed,
		OpenRequisiteCount:  checklist.OpenRequisiteCount,
		AllPayoutsFullyPaid: checklist.AllPayoutsFullyPaid,
		UnpaidPayoutCount:   checklist.UnpaidPayoutCount,
		CanClose:            checklist.CanClose,
	}
}

func PublicShiftReportDetailsFromDomain(report ShiftReportDetails) PublicShiftReportDetails {
	return PublicShiftReportDetails{
		Shift:    PublicShiftFromDomain(report.Shift),
		Inbound:  PublicShiftReportReconciliationFromDomain(report.Inbound),
		Outbound: PublicShiftReportReconciliationFromDomain(report.Outbound),
		Rows:     PublicShiftReportRowsFromDomain(report.Rows),
	}
}

func PublicShiftReportReconciliationFromDomain(item *ShiftReportReconciliation) *PublicShiftReportReconciliation {
	if item == nil {
		return nil
	}

	return &PublicShiftReportReconciliation{
		ID:                  item.ID,
		Status:              item.Status,
		ExpectedAmountMinor: item.ExpectedAmountMinor,
		ActualAmountMinor:   item.ActualAmountMinor,
		DiffAmountMinor:     item.DiffAmountMinor,
		Comment:             item.Comment,
		CreatedAt:           item.CreatedAt,
	}
}

func PublicShiftReportRowsFromDomain(items []ShiftReportRow) []PublicShiftReportRow {
	result := make([]PublicShiftReportRow, 0, len(items))
	for _, item := range items {
		result = append(result, PublicShiftReportRowFromDomain(item))
	}

	return result
}

func PublicShiftReportRowFromDomain(item ShiftReportRow) PublicShiftReportRow {
	return PublicShiftReportRow{
		RowKey:                item.RowKey,
		ShiftRequisiteID:      item.ShiftRequisiteID,
		RequisiteID:           item.RequisiteID,
		Phone:                 item.Phone,
		MethodType:            item.MethodType,
		BankCode:              item.BankCode,
		BankName:              item.BankName,
		Proxy:                 item.Proxy,
		EmployeeComment:       item.EmployeeComment,
		CardNumber:            item.CardNumber,
		HolderName:            item.HolderName,
		Status:                item.Status,
		InboundTurnoverMinor:  item.InboundTurnoverMinor,
		OutboundTurnoverMinor: item.OutboundTurnoverMinor,
		ClosingBalanceMinor:   item.ClosingBalanceMinor,
		TargetTurnoverMinor:   item.TargetTurnoverMinor,
		CSVInboundMinor:       item.CSVInboundMinor,
		CSVOutboundMinor:      item.CSVOutboundMinor,
		InboundDiffMinor:      item.InboundDiffMinor,
		OutboundDiffMinor:     item.OutboundDiffMinor,
		HasMismatch:           item.HasMismatch,
		CSVOnly:               item.CSVOnly,
	}
}
