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
	ID                   int64      `json:"id"`
	TeamID               int64      `json:"teamId"`
	Phone                string     `json:"phone"`
	MethodType           string     `json:"methodType"`
	Proxy                *string    `json:"proxy,omitempty"`
	Status               string     `json:"status"`
	AssignmentID         int64      `json:"assignmentId"`
	ShiftRequisiteID     *int64     `json:"shiftRequisiteId,omitempty"`
	CardNumber           *string    `json:"cardNumber,omitempty"`
	HolderName           *string    `json:"holderName,omitempty"`
	ShiftRequisiteStatus *string    `json:"shiftRequisiteStatus,omitempty"`
	TakenAt              *time.Time `json:"takenAt,omitempty"`
	InWork               bool       `json:"inWork"`
}

type PublicShiftRequisite struct {
	ID           int64      `json:"id"`
	TeamID       int64      `json:"teamId"`
	ShiftID      int64      `json:"shiftId"`
	TraderID     int64      `json:"traderId"`
	RequisiteID  int64      `json:"requisiteId"`
	AssignmentID *int64     `json:"assignmentId,omitempty"`
	CardNumber   string     `json:"cardNumber"`
	HolderName   string     `json:"holderName"`
	TakenAt      time.Time  `json:"takenAt"`
	ReleasedAt   *time.Time `json:"releasedAt,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
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
	Shift                 PublicShift `json:"shift"`
	InboundImported       bool        `json:"inboundImported"`
	InboundOk             bool        `json:"inboundOk"`
	OutboundImported      bool        `json:"outboundImported"`
	OutboundOk            bool        `json:"outboundOk"`
	AllPayoutsFullyPaid   bool        `json:"allPayoutsFullyPaid"`
	UnpaidPayoutCount     int64       `json:"unpaidPayoutCount"`
	CanClose              bool        `json:"canClose"`
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

func PublicAssignedRequisiteFromDomain(item AssignedRequisite) PublicAssignedRequisite {
	return PublicAssignedRequisite{
		ID:                   item.ID,
		TeamID:               item.TeamID,
		Phone:                item.Phone,
		MethodType:           item.MethodType,
		Proxy:                item.Proxy,
		Status:               item.Status,
		AssignmentID:         item.AssignmentID,
		ShiftRequisiteID:     item.ShiftRequisiteID,
		CardNumber:           item.CardNumber,
		HolderName:           item.HolderName,
		ShiftRequisiteStatus: item.ShiftRequisiteStatus,
		TakenAt:              item.TakenAt,
		InWork:               item.ShiftRequisiteID != nil,
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
		ID:           item.ID,
		TeamID:       item.TeamID,
		ShiftID:      item.ShiftID,
		TraderID:     item.TraderID,
		RequisiteID:  item.RequisiteID,
		AssignmentID: item.AssignmentID,
		CardNumber:   item.CardNumber,
		HolderName:   item.HolderName,
		TakenAt:      item.TakenAt,
		ReleasedAt:   item.ReleasedAt,
		Status:       item.Status,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
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
		AllPayoutsFullyPaid: checklist.AllPayoutsFullyPaid,
		UnpaidPayoutCount:   checklist.UnpaidPayoutCount,
		CanClose:            checklist.CanClose,
	}
}
