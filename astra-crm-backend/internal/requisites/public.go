package requisites

import (
	"encoding/json"
	"time"
)

type PublicRequisite struct {
	ID                  int64      `json:"id"`
	TeamID              int64      `json:"teamId"`
	Phone               string     `json:"phone"`
	MethodType          string     `json:"methodType"`
	BankCode            string     `json:"bankCode"`
	BankName            string     `json:"bankName"`
	Proxy               *string    `json:"proxy,omitempty"`
	EmployeeComment     *string    `json:"employeeComment,omitempty"`
	HolderName          *string    `json:"holderName,omitempty"`
	CardNumber          *string    `json:"cardNumber,omitempty"`
	DetailsFilledAt     *time.Time `json:"detailsFilledAt,omitempty"`
	DetailsFilledBy     *int64     `json:"detailsFilledBy,omitempty"`
	Status              string     `json:"status"`
	AssignedTraderID    *int64     `json:"assignedTraderId,omitempty"`
	AssignedTraderLogin *string    `json:"assignedTraderLogin,omitempty"`
	AssignmentStatus    *string    `json:"assignmentStatus,omitempty"`
	AssignedForDate     *time.Time `json:"assignedForDate,omitempty"`
	TargetTurnoverMinor int64      `json:"targetTurnoverMinor"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type PublicRequisiteAssignment struct {
	ID                  int64      `json:"id"`
	TeamID              int64      `json:"teamId"`
	RequisiteID         int64      `json:"requisiteId"`
	TraderID            int64      `json:"traderId"`
	AssignedBy          int64      `json:"assignedBy"`
	AssignedAt          time.Time  `json:"assignedAt"`
	UnassignedAt        *time.Time `json:"unassignedAt,omitempty"`
	Comment             *string    `json:"comment,omitempty"`
	Status              string     `json:"status"`
	AssignedForDate     time.Time  `json:"assignedForDate"`
	TargetTurnoverMinor int64      `json:"targetTurnoverMinor"`
	StartedAt           *time.Time `json:"startedAt,omitempty"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
	CancelledAt         *time.Time `json:"cancelledAt,omitempty"`
	ShiftRequisiteID    *int64     `json:"shiftRequisiteId,omitempty"`
	WasReassign         bool       `json:"wasReassign"`
}

type PublicAssignmentWorkRow struct {
	AssignmentID          int64      `json:"assignmentId"`
	TeamID                int64      `json:"teamId"`
	RequisiteID           int64      `json:"requisiteId"`
	Phone                 string     `json:"phone"`
	BankCode              string     `json:"bankCode"`
	BankName              string     `json:"bankName"`
	Proxy                 *string    `json:"proxy,omitempty"`
	TraderID              int64      `json:"traderId"`
	TraderLogin           string     `json:"traderLogin"`
	Status                string     `json:"status"`
	AssignedForDate       time.Time  `json:"assignedForDate"`
	TargetTurnoverMinor   int64      `json:"targetTurnoverMinor"`
	InboundTurnoverMinor  int64      `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor int64      `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor   int64      `json:"closingBalanceMinor"`
	CardNumber            *string    `json:"cardNumber,omitempty"`
	HolderName            *string    `json:"holderName,omitempty"`
	TakenAt               *time.Time `json:"takenAt,omitempty"`
	ReleasedAt            *time.Time `json:"releasedAt,omitempty"`
	Comment               *string    `json:"comment,omitempty"`
	AssignedAt            time.Time  `json:"assignedAt"`
	StartedAt             *time.Time `json:"startedAt,omitempty"`
	CompletedAt           *time.Time `json:"completedAt,omitempty"`
	UpdatedAt             time.Time  `json:"updatedAt"`
	ShiftRequisiteID      *int64     `json:"shiftRequisiteId,omitempty"`
}

type PublicAssignmentEvent struct {
	ID           int64     `json:"id"`
	TeamID       int64     `json:"teamId"`
	AssignmentID int64     `json:"assignmentId"`
	ActorID      int64     `json:"actorId"`
	Action       string    `json:"action"`
	BeforeJSON   JSONValue `json:"beforeJson,omitempty"`
	AfterJSON    JSONValue `json:"afterJson,omitempty"`
	Comment      *string   `json:"comment,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type JSONValue = json.RawMessage

func rawJSONValue(value []byte) JSONValue {
	if len(value) == 0 {
		return nil
	}
	return JSONValue(json.RawMessage(value))
}

func PublicRequisiteFromDetails(details RequisiteDetails) PublicRequisite {
	return PublicRequisite{
		ID:                  details.ID,
		TeamID:              details.TeamID,
		Phone:               details.Phone,
		MethodType:          details.MethodType,
		BankCode:            details.BankCode,
		BankName:            details.BankName,
		Proxy:               details.Proxy,
		EmployeeComment:     details.EmployeeComment,
		HolderName:          details.HolderName,
		CardNumber:          details.CardNumber,
		DetailsFilledAt:     details.DetailsFilledAt,
		DetailsFilledBy:     details.DetailsFilledBy,
		Status:              details.Status,
		AssignedTraderID:    details.AssignedTraderID,
		AssignedTraderLogin: details.AssignedTraderLogin,
		AssignmentStatus:    details.AssignmentStatus,
		AssignedForDate:     details.AssignedForDate,
		TargetTurnoverMinor: details.TargetTurnoverMinor,
		CreatedAt:           details.CreatedAt,
		UpdatedAt:           details.UpdatedAt,
	}
}

func PublicRequisites(items []RequisiteDetails) []PublicRequisite {
	result := make([]PublicRequisite, 0, len(items))
	for _, item := range items {
		result = append(result, PublicRequisiteFromDetails(item))
	}

	return result
}

func PublicAssignment(assignment Assignment) PublicRequisiteAssignment {
	return PublicRequisiteAssignment{
		ID:                  assignment.ID,
		TeamID:              assignment.TeamID,
		RequisiteID:         assignment.RequisiteID,
		TraderID:            assignment.TraderID,
		AssignedBy:          assignment.AssignedBy,
		AssignedAt:          assignment.AssignedAt,
		UnassignedAt:        assignment.UnassignedAt,
		Comment:             assignment.Comment,
		Status:              assignment.Status,
		AssignedForDate:     assignment.AssignedForDate,
		TargetTurnoverMinor: assignment.TargetTurnoverMinor,
		StartedAt:           assignment.StartedAt,
		CompletedAt:         assignment.CompletedAt,
		CancelledAt:         assignment.CancelledAt,
		ShiftRequisiteID:    assignment.ShiftRequisiteID,
		WasReassign:         assignment.WasReassign,
	}
}

func PublicAssignments(items []Assignment) []PublicRequisiteAssignment {
	result := make([]PublicRequisiteAssignment, 0, len(items))
	for _, item := range items {
		result = append(result, PublicAssignment(item))
	}

	return result
}

func PublicAssignmentWorkRowFromDomain(item AssignmentWorkRow) PublicAssignmentWorkRow {
	return PublicAssignmentWorkRow{
		AssignmentID:          item.AssignmentID,
		TeamID:                item.TeamID,
		RequisiteID:           item.RequisiteID,
		Phone:                 item.Phone,
		BankCode:              item.BankCode,
		BankName:              item.BankName,
		Proxy:                 item.Proxy,
		TraderID:              item.TraderID,
		TraderLogin:           item.TraderLogin,
		Status:                item.Status,
		AssignedForDate:       item.AssignedForDate,
		TargetTurnoverMinor:   item.TargetTurnoverMinor,
		InboundTurnoverMinor:  item.InboundTurnoverMinor,
		OutboundTurnoverMinor: item.OutboundTurnoverMinor,
		ClosingBalanceMinor:   item.ClosingBalanceMinor,
		CardNumber:            item.CardNumber,
		HolderName:            item.HolderName,
		TakenAt:               item.TakenAt,
		ReleasedAt:            item.ReleasedAt,
		Comment:               item.Comment,
		AssignedAt:            item.AssignedAt,
		StartedAt:             item.StartedAt,
		CompletedAt:           item.CompletedAt,
		UpdatedAt:             item.UpdatedAt,
		ShiftRequisiteID:      item.ShiftRequisiteID,
	}
}

func PublicAssignmentWorkRows(items []AssignmentWorkRow) []PublicAssignmentWorkRow {
	result := make([]PublicAssignmentWorkRow, 0, len(items))
	for _, item := range items {
		result = append(result, PublicAssignmentWorkRowFromDomain(item))
	}

	return result
}

func PublicAssignmentEventFromDomain(item AssignmentEvent) PublicAssignmentEvent {
	return PublicAssignmentEvent{
		ID:           item.ID,
		TeamID:       item.TeamID,
		AssignmentID: item.AssignmentID,
		ActorID:      item.ActorID,
		Action:       item.Action,
		BeforeJSON:   rawJSONValue(item.BeforeJSON),
		AfterJSON:    rawJSONValue(item.AfterJSON),
		Comment:      item.Comment,
		CreatedAt:    item.CreatedAt,
	}
}

func PublicAssignmentEvents(items []AssignmentEvent) []PublicAssignmentEvent {
	result := make([]PublicAssignmentEvent, 0, len(items))
	for _, item := range items {
		result = append(result, PublicAssignmentEventFromDomain(item))
	}

	return result
}
