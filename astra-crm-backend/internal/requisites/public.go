package requisites

import "time"

type PublicRequisite struct {
	ID                  int64     `json:"id"`
	TeamID              int64     `json:"teamId"`
	Phone               string    `json:"phone"`
	MethodType          string    `json:"methodType"`
	Proxy               *string   `json:"proxy,omitempty"`
	Status              string    `json:"status"`
	AssignedTraderID    *int64    `json:"assignedTraderId,omitempty"`
	AssignedTraderLogin *string   `json:"assignedTraderLogin,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type PublicRequisiteAssignment struct {
	ID           int64      `json:"id"`
	TeamID       int64      `json:"teamId"`
	RequisiteID  int64      `json:"requisiteId"`
	TraderID     int64      `json:"traderId"`
	AssignedBy   int64      `json:"assignedBy"`
	AssignedAt   time.Time  `json:"assignedAt"`
	UnassignedAt *time.Time `json:"unassignedAt,omitempty"`
	Comment      *string    `json:"comment,omitempty"`
	WasReassign  bool       `json:"wasReassign"`
}

func PublicRequisiteFromDetails(details RequisiteDetails) PublicRequisite {
	return PublicRequisite{
		ID:                  details.ID,
		TeamID:              details.TeamID,
		Phone:               details.Phone,
		MethodType:          details.MethodType,
		Proxy:               details.Proxy,
		Status:              details.Status,
		AssignedTraderID:    details.AssignedTraderID,
		AssignedTraderLogin: details.AssignedTraderLogin,
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
		ID:           assignment.ID,
		TeamID:       assignment.TeamID,
		RequisiteID:  assignment.RequisiteID,
		TraderID:     assignment.TraderID,
		AssignedBy:   assignment.AssignedBy,
		AssignedAt:   assignment.AssignedAt,
		UnassignedAt: assignment.UnassignedAt,
		Comment:      assignment.Comment,
		WasReassign:  assignment.WasReassign,
	}
}

func PublicAssignments(items []Assignment) []PublicRequisiteAssignment {
	result := make([]PublicRequisiteAssignment, 0, len(items))
	for _, item := range items {
		result = append(result, PublicAssignment(item))
	}

	return result
}
