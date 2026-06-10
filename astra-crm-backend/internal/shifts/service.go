package shifts

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

var (
	ErrInvalidInput         = errors.New("invalid shift input")
	ErrRequisiteNotAssigned = errors.New("requisite is not assigned to trader")
	ErrCloseBlocked         = errors.New("shift close is blocked")
)

type Store interface {
	CurrentShift(ctx context.Context, teamID int64, traderID int64) (Shift, error)
	CreateShift(ctx context.Context, teamID int64, traderID int64) (Shift, error)
	ActiveAssignment(ctx context.Context, teamID int64, traderID int64, requisiteID int64) (int64, error)
	AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error)
	CreateShiftRequisite(ctx context.Context, params CreateShiftRequisiteRecord) (ShiftRequisite, error)
	ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]ShiftRequisite, error)
	UpdateShiftRequisiteDetails(ctx context.Context, params UpdateShiftRequisiteDetailsRecord) (ShiftRequisite, error)
	CreateTurnoverEntry(ctx context.Context, params CreateTurnoverEntryRecord) (TurnoverEntry, error)
	LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]TurnoverEntry, error)
	TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]TurnoverEntry, error)
	CurrentShiftChecklist(ctx context.Context, teamID int64, traderID int64) (CloseChecklist, error)
	CloseCurrentShift(ctx context.Context, params CloseShiftRecord) (Shift, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type TurnoverHook interface {
	AfterTurnoverCreated(ctx context.Context, entry TurnoverEntry) error
}

type Service struct {
	store        Store
	audit        AuditService
	turnoverHook TurnoverHook
}

func NewService(store Store, auditService AuditService, turnoverHooks ...TurnoverHook) *Service {
	service := &Service{
		store: store,
		audit: auditService,
	}
	if len(turnoverHooks) > 0 {
		service.turnoverHook = turnoverHooks[0]
	}

	return service
}

type TakeRequisiteParams struct {
	ActorID     int64
	TeamID      int64
	TraderID    int64
	RequisiteID int64
	CardNumber  string
	HolderName  string
}

type TakeRequisiteResult struct {
	Shift          Shift
	ShiftRequisite ShiftRequisite
	ShiftCreated   bool
}

type UpdateShiftRequisiteParams struct {
	ActorID          int64
	TeamID           int64
	TraderID         int64
	ShiftRequisiteID int64
	CardNumber       string
	HolderName       string
}

type CreateTurnoverParams struct {
	ActorID          int64
	TeamID           int64
	TraderID         int64
	ShiftRequisiteID int64
	AmountMinor      int64
	Comment          *string
}

type CloseShiftParams struct {
	ActorID      int64
	TeamID       int64
	TraderID     int64
	CloseComment *string
}

func (s *Service) Current(ctx context.Context, teamID int64, traderID int64) (*Shift, error) {
	shift, err := s.store.CurrentShift(ctx, teamID, traderID)
	if errors.Is(err, ErrCurrentShiftNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &shift, nil
}

func (s *Service) AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]AssignedRequisite, error) {
	return s.store.AssignedRequisites(ctx, teamID, traderID)
}

func (s *Service) ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]ShiftRequisite, error) {
	return s.store.ShiftRequisites(ctx, teamID, traderID)
}

func (s *Service) LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]TurnoverEntry, error) {
	return s.store.LatestTurnovers(ctx, teamID, traderID)
}

func (s *Service) TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]TurnoverEntry, error) {
	if teamID <= 0 || traderID <= 0 || shiftRequisiteID <= 0 {
		return nil, ErrInvalidInput
	}

	return s.store.TurnoversByShiftRequisite(ctx, teamID, traderID, shiftRequisiteID)
}

func (s *Service) TakeRequisite(ctx context.Context, params TakeRequisiteParams) (TakeRequisiteResult, error) {
	cardNumber := strings.TrimSpace(params.CardNumber)
	holderName := strings.TrimSpace(params.HolderName)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.RequisiteID <= 0 || cardNumber == "" || holderName == "" {
		return TakeRequisiteResult{}, ErrInvalidInput
	}

	assignmentID, err := s.store.ActiveAssignment(ctx, params.TeamID, params.TraderID, params.RequisiteID)
	if errors.Is(err, ErrActiveAssignmentNotFound) {
		return TakeRequisiteResult{}, ErrRequisiteNotAssigned
	}
	if err != nil {
		return TakeRequisiteResult{}, err
	}

	shift, err := s.store.CurrentShift(ctx, params.TeamID, params.TraderID)
	shiftCreated := false
	if errors.Is(err, ErrCurrentShiftNotFound) {
		shift, err = s.store.CreateShift(ctx, params.TeamID, params.TraderID)
		if err != nil {
			return TakeRequisiteResult{}, err
		}
		shiftCreated = true
		if err := s.writeAudit(ctx, audit.Event{
			TeamID:     params.TeamID,
			ActorID:    params.ActorID,
			Action:     audit.ActionShiftCreated,
			EntityType: "trader_shift",
			EntityID:   strconv.FormatInt(shift.ID, 10),
			After:      PublicShiftFromDomain(shift),
		}); err != nil {
			return TakeRequisiteResult{}, err
		}
	} else if err != nil {
		return TakeRequisiteResult{}, err
	}

	shiftRequisite, err := s.store.CreateShiftRequisite(ctx, CreateShiftRequisiteRecord{
		TeamID:       params.TeamID,
		ShiftID:      shift.ID,
		TraderID:     params.TraderID,
		RequisiteID:  params.RequisiteID,
		AssignmentID: assignmentID,
		CardNumber:   cardNumber,
		HolderName:   holderName,
	})
	if err != nil {
		return TakeRequisiteResult{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftRequisiteTaken,
		EntityType: "shift_requisite",
		EntityID:   strconv.FormatInt(shiftRequisite.ID, 10),
		After:      PublicShiftRequisiteFromDomain(shiftRequisite),
	}); err != nil {
		return TakeRequisiteResult{}, err
	}

	return TakeRequisiteResult{
		Shift:          shift,
		ShiftRequisite: shiftRequisite,
		ShiftCreated:   shiftCreated,
	}, nil
}

func (s *Service) UpdateShiftRequisite(ctx context.Context, params UpdateShiftRequisiteParams) (ShiftRequisite, error) {
	cardNumber := strings.TrimSpace(params.CardNumber)
	holderName := strings.TrimSpace(params.HolderName)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftRequisiteID <= 0 || cardNumber == "" || holderName == "" {
		return ShiftRequisite{}, ErrInvalidInput
	}

	updated, err := s.store.UpdateShiftRequisiteDetails(ctx, UpdateShiftRequisiteDetailsRecord{
		TeamID:           params.TeamID,
		TraderID:         params.TraderID,
		ShiftRequisiteID: params.ShiftRequisiteID,
		CardNumber:       cardNumber,
		HolderName:       holderName,
	})
	if err != nil {
		return ShiftRequisite{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftDailyDetailsUpdated,
		EntityType: "shift_requisite",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		After:      PublicShiftRequisiteFromDomain(updated),
	}); err != nil {
		return ShiftRequisite{}, err
	}

	return updated, nil
}

func (s *Service) CreateTurnover(ctx context.Context, params CreateTurnoverParams) (TurnoverEntry, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftRequisiteID <= 0 || params.AmountMinor < 0 {
		return TurnoverEntry{}, ErrInvalidInput
	}

	comment := cleanOptionalString(params.Comment)
	entry, err := s.store.CreateTurnoverEntry(ctx, CreateTurnoverEntryRecord{
		TeamID:           params.TeamID,
		TraderID:         params.TraderID,
		ShiftRequisiteID: params.ShiftRequisiteID,
		AmountMinor:      params.AmountMinor,
		CreatedBy:        params.ActorID,
		Comment:          comment,
	})
	if err != nil {
		return TurnoverEntry{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftTurnoverAdded,
		EntityType: "requisite_turnover_entry",
		EntityID:   strconv.FormatInt(entry.ID, 10),
		After:      PublicTurnoverEntryFromDomain(entry),
		Comment:    comment,
	}); err != nil {
		return TurnoverEntry{}, err
	}

	if s.turnoverHook != nil {
		if err := s.turnoverHook.AfterTurnoverCreated(ctx, entry); err != nil {
			return TurnoverEntry{}, err
		}
	}

	return entry, nil
}

func (s *Service) CloseChecklist(ctx context.Context, teamID int64, traderID int64) (CloseChecklist, error) {
	return s.store.CurrentShiftChecklist(ctx, teamID, traderID)
}

func (s *Service) CloseCurrent(ctx context.Context, params CloseShiftParams) (Shift, error) {
	checklist, err := s.store.CurrentShiftChecklist(ctx, params.TeamID, params.TraderID)
	if err != nil {
		return Shift{}, err
	}
	if !checklist.CanClose {
		return Shift{}, ErrCloseBlocked
	}

	closed, err := s.store.CloseCurrentShift(ctx, CloseShiftRecord{
		TeamID:       params.TeamID,
		TraderID:     params.TraderID,
		ShiftID:      checklist.Shift.ID,
		CloseComment: cleanOptionalString(params.CloseComment),
	})
	if err != nil {
		return Shift{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftClosed,
		EntityType: "trader_shift",
		EntityID:   strconv.FormatInt(closed.ID, 10),
		After:      PublicShiftFromDomain(closed),
		Comment:    cleanOptionalString(params.CloseComment),
	}); err != nil {
		return Shift{}, err
	}

	return closed, nil
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

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}
