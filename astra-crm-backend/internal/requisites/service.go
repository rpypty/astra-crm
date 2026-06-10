package requisites

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

var (
	ErrInvalidInput   = errors.New("invalid requisite input")
	ErrInactiveTrader = errors.New("trader is not active")
)

type Store interface {
	Create(ctx context.Context, params CreateRecord) (Requisite, error)
	GetDetails(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error)
	ListDetails(ctx context.Context, teamID int64) ([]RequisiteDetails, error)
	Update(ctx context.Context, params UpdateRecord) (Requisite, error)
	Assign(ctx context.Context, params AssignRecord) (Assignment, error)
	Unassign(ctx context.Context, teamID int64, requisiteID int64) (Assignment, error)
	AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64) ([]Assignment, error)
}

type TraderReader interface {
	GetTraderByID(ctx context.Context, teamID int64, traderID int64) (users.Trader, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type Service struct {
	store   Store
	traders TraderReader
	audit   AuditService
}

func NewService(store Store, traders TraderReader, auditService AuditService) *Service {
	return &Service{
		store:   store,
		traders: traders,
		audit:   auditService,
	}
}

type CreateParams struct {
	ActorID          int64
	TeamID           int64
	Phone            string
	MethodType       string
	Proxy            *string
	AssignedTraderID *int64
	AssignComment    *string
}

type PatchParams struct {
	ActorID     int64
	TeamID      int64
	RequisiteID int64
	Phone       *string
	MethodType  *string
	Proxy       *string
	Status      *string
}

type AssignParams struct {
	ActorID     int64
	TeamID      int64
	RequisiteID int64
	TraderID    int64
	Comment     *string
}

func (s *Service) Create(ctx context.Context, params CreateParams) (RequisiteDetails, error) {
	phone := strings.TrimSpace(params.Phone)
	methodType := strings.TrimSpace(params.MethodType)
	proxy := cleanOptionalString(params.Proxy)
	if params.ActorID <= 0 || params.TeamID <= 0 || phone == "" || methodType == "" {
		return RequisiteDetails{}, ErrInvalidInput
	}
	if params.AssignedTraderID != nil {
		trader, err := s.traders.GetTraderByID(ctx, params.TeamID, *params.AssignedTraderID)
		if err != nil {
			return RequisiteDetails{}, err
		}
		if trader.Status != users.StatusActive {
			return RequisiteDetails{}, ErrInactiveTrader
		}
	}

	created, err := s.store.Create(ctx, CreateRecord{
		TeamID:     params.TeamID,
		Phone:      phone,
		MethodType: methodType,
		Proxy:      proxy,
		CreatedBy:  params.ActorID,
	})
	if err != nil {
		return RequisiteDetails{}, err
	}

	details := RequisiteDetails{Requisite: created}
	if params.AssignedTraderID != nil {
		assignment, err := s.store.Assign(ctx, AssignRecord{
			TeamID:      params.TeamID,
			RequisiteID: created.ID,
			TraderID:    *params.AssignedTraderID,
			AssignedBy:  params.ActorID,
			Comment:     cleanOptionalString(params.AssignComment),
		})
		if err != nil {
			return RequisiteDetails{}, err
		}
		details.ActiveAssignmentID = &assignment.ID
		details.AssignedTraderID = &assignment.TraderID
		if err := s.writeAudit(ctx, audit.Event{
			TeamID:     params.TeamID,
			ActorID:    params.ActorID,
			Action:     audit.ActionRequisiteAssigned,
			EntityType: "requisite",
			EntityID:   strconv.FormatInt(created.ID, 10),
			After:      PublicAssignment(assignment),
			Comment:    cleanOptionalString(params.AssignComment),
		}); err != nil {
			return RequisiteDetails{}, err
		}
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionRequisiteCreated,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(created.ID, 10),
		After:      PublicRequisiteFromDetails(details),
	}); err != nil {
		return RequisiteDetails{}, err
	}

	return details, nil
}

func (s *Service) List(ctx context.Context, teamID int64) ([]RequisiteDetails, error) {
	return s.store.ListDetails(ctx, teamID)
}

func (s *Service) Get(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error) {
	return s.store.GetDetails(ctx, teamID, requisiteID)
}

func (s *Service) Patch(ctx context.Context, params PatchParams) (RequisiteDetails, error) {
	current, err := s.store.GetDetails(ctx, params.TeamID, params.RequisiteID)
	if err != nil {
		return RequisiteDetails{}, err
	}

	next := current.Requisite
	if params.Phone != nil {
		phone := strings.TrimSpace(*params.Phone)
		if phone == "" {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.Phone = phone
	}
	if params.MethodType != nil {
		methodType := strings.TrimSpace(*params.MethodType)
		if methodType == "" {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.MethodType = methodType
	}
	if params.Proxy != nil {
		next.Proxy = cleanOptionalString(params.Proxy)
	}
	if params.Status != nil {
		status := strings.TrimSpace(*params.Status)
		if !validStatus(status) {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.Status = status
	}

	updated, err := s.store.Update(ctx, UpdateRecord{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
		Phone:       next.Phone,
		MethodType:  next.MethodType,
		Proxy:       next.Proxy,
		Status:      next.Status,
	})
	if err != nil {
		return RequisiteDetails{}, err
	}

	details := current
	details.Requisite = updated
	action := audit.ActionRequisiteUpdated
	if updated.Status == StatusArchived {
		action = audit.ActionRequisiteArchived
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     action,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		Before:     PublicRequisiteFromDetails(current),
		After:      PublicRequisiteFromDetails(details),
	}); err != nil {
		return RequisiteDetails{}, err
	}

	return details, nil
}

func (s *Service) Delete(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error {
	status := StatusArchived
	_, err := s.Patch(ctx, PatchParams{
		ActorID:     actorID,
		TeamID:      teamID,
		RequisiteID: requisiteID,
		Status:      &status,
	})
	return err
}

func (s *Service) Assign(ctx context.Context, params AssignParams) (Assignment, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.RequisiteID <= 0 || params.TraderID <= 0 {
		return Assignment{}, ErrInvalidInput
	}

	requisite, err := s.store.GetDetails(ctx, params.TeamID, params.RequisiteID)
	if err != nil {
		return Assignment{}, err
	}
	if requisite.Status != StatusActive {
		return Assignment{}, ErrInvalidInput
	}

	trader, err := s.traders.GetTraderByID(ctx, params.TeamID, params.TraderID)
	if err != nil {
		return Assignment{}, err
	}
	if trader.Status != users.StatusActive {
		return Assignment{}, ErrInactiveTrader
	}

	comment := cleanOptionalString(params.Comment)
	assignment, err := s.store.Assign(ctx, AssignRecord{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
		TraderID:    params.TraderID,
		AssignedBy:  params.ActorID,
		Comment:     comment,
	})
	if err != nil {
		return Assignment{}, err
	}

	action := audit.ActionRequisiteAssigned
	if assignment.WasReassign {
		action = audit.ActionRequisiteReassigned
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     action,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(params.RequisiteID, 10),
		After:      PublicAssignment(assignment),
		Comment:    comment,
	}); err != nil {
		return Assignment{}, err
	}

	return assignment, nil
}

func (s *Service) Unassign(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error {
	assignment, err := s.store.Unassign(ctx, teamID, requisiteID)
	if err != nil {
		return err
	}

	return s.writeAudit(ctx, audit.Event{
		TeamID:     teamID,
		ActorID:    actorID,
		Action:     audit.ActionRequisiteUnassigned,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(requisiteID, 10),
		After:      PublicAssignment(assignment),
	})
}

func (s *Service) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64) ([]Assignment, error) {
	return s.store.AssignmentHistory(ctx, teamID, requisiteID)
}

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func validStatus(status string) bool {
	switch status {
	case StatusActive, StatusDisabled, StatusArchived:
		return true
	default:
		return false
	}
}
