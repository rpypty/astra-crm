package shifts

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/pagination"
)

var (
	ErrInvalidInput             = errors.New("invalid shift input")
	ErrRequisiteNotAssigned     = errors.New("requisite is not assigned to trader")
	ErrCloseBlocked             = errors.New("shift close is blocked")
	ErrInternalTransferNotFound = errors.New("internal transfer not found")
)

type Store interface {
	CurrentShift(ctx context.Context, teamID int64, traderID int64) (Shift, error)
	CreateShift(ctx context.Context, teamID int64, traderID int64) (Shift, error)
	ShiftHistory(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[Shift], error)
	TeamShiftHistory(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[Shift], error)
	ShiftReport(ctx context.Context, teamID int64, traderID int64, shiftID int64) (ShiftReportDetails, error)
	TeamShiftReport(ctx context.Context, teamID int64, shiftID int64) (ShiftReportDetails, error)
	ActiveAssignment(ctx context.Context, teamID int64, traderID int64, requisiteID int64) (int64, error)
	AssignedRequisites(ctx context.Context, teamID int64, traderID int64, filters AssignedRequisitesFilters, page pagination.Params) (pagination.Result[AssignedRequisite], error)
	FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error)
	HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error)
	AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error)
	AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error)
	CreateShiftRequisite(ctx context.Context, params CreateShiftRequisiteRecord) (ShiftRequisite, error)
	ShiftRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[ShiftRequisite], error)
	UpdateShiftRequisiteDetails(ctx context.Context, params UpdateShiftRequisiteDetailsRecord) (ShiftRequisite, error)
	CloseShiftRequisite(ctx context.Context, params CloseShiftRequisiteRecord) (ShiftRequisite, error)
	CorrectClosedShiftRequisiteTurnovers(ctx context.Context, params CorrectShiftRequisiteTurnoversRecord) (ShiftRequisite, error)
	ReturnShiftRequisiteToWork(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) (ShiftRequisite, error)
	GetShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) (ShiftRequisite, error)
	CreateTurnoverEntry(ctx context.Context, params CreateTurnoverEntryRecord) (TurnoverEntry, error)
	LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]TurnoverEntry, error)
	TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64, page pagination.Params) (pagination.Result[TurnoverEntry], error)
	InternalTransfersByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64, page pagination.Params) (pagination.Result[InternalTransfer], error)
	CreateInternalTransfer(ctx context.Context, params CreateInternalTransferRecord) (InternalTransfer, error)
	CancelInternalTransfer(ctx context.Context, params CancelInternalTransferRecord) (InternalTransfer, error)
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

type CloseShiftRequisiteParams struct {
	ActorID               int64
	TeamID                int64
	TraderID              int64
	ShiftRequisiteID      int64
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
	Blocked               bool
	Comment               *string
	ReleasedAt            *time.Time
}

type CorrectShiftRequisiteParams struct {
	ActorID               int64
	TeamID                int64
	TraderID              int64
	ShiftRequisiteID      int64
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
	Comment               string
}

type ReturnShiftRequisiteParams struct {
	ActorID          int64
	TeamID           int64
	TraderID         int64
	ShiftRequisiteID int64
}

type CloseShiftParams struct {
	ActorID      int64
	TeamID       int64
	TraderID     int64
	CloseComment *string
}

type AssignedRequisitesFilters struct {
	Statuses  []string
	Search    string
	ExcludeID *int64
}

type CreateInternalTransferParams struct {
	ActorID                     int64
	TeamID                      int64
	TraderID                    int64
	SourceShiftRequisiteID      int64
	DestinationShiftRequisiteID int64
	AmountMinor                 int64
	Comment                     *string
}

type CancelInternalTransferParams struct {
	ActorID    int64
	TeamID     int64
	TraderID   int64
	TransferID int64
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

func (s *Service) ShiftHistory(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[Shift], error) {
	if teamID <= 0 || traderID <= 0 {
		return pagination.Result[Shift]{}, ErrInvalidInput
	}

	return s.store.ShiftHistory(ctx, teamID, traderID, page)
}

func (s *Service) TeamShiftHistory(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[Shift], error) {
	if teamID <= 0 {
		return pagination.Result[Shift]{}, ErrInvalidInput
	}

	return s.store.TeamShiftHistory(ctx, teamID, page)
}

func (s *Service) ShiftReport(ctx context.Context, teamID int64, traderID int64, shiftID int64) (ShiftReportDetails, error) {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return ShiftReportDetails{}, ErrInvalidInput
	}

	return s.store.ShiftReport(ctx, teamID, traderID, shiftID)
}

func (s *Service) TeamShiftReport(ctx context.Context, teamID int64, shiftID int64) (ShiftReportDetails, error) {
	if teamID <= 0 || shiftID <= 0 {
		return ShiftReportDetails{}, ErrInvalidInput
	}

	return s.store.TeamShiftReport(ctx, teamID, shiftID)
}

func (s *Service) AssignedRequisites(ctx context.Context, teamID int64, traderID int64, filters AssignedRequisitesFilters, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	return s.store.AssignedRequisites(ctx, teamID, traderID, normalizeAssignedRequisitesFilters(filters), page)
}

func (s *Service) FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	return s.store.FutureAssignedRequisites(ctx, teamID, traderID, page)
}

func (s *Service) HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	return s.store.HistoricalAssignedRequisites(ctx, teamID, traderID, page)
}

func (s *Service) AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	if teamID <= 0 || traderID <= 0 || shiftID <= 0 {
		return pagination.Result[AssignedRequisite]{}, ErrInvalidInput
	}

	return s.store.AssignedRequisitesByShift(ctx, teamID, traderID, shiftID, page)
}

func (s *Service) AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64, page pagination.Params) (pagination.Result[AssignedRequisite], error) {
	if teamID <= 0 || shiftID <= 0 {
		return pagination.Result[AssignedRequisite]{}, ErrInvalidInput
	}

	return s.store.AssignedRequisitesByTeamShift(ctx, teamID, shiftID, page)
}

func (s *Service) ShiftRequisites(ctx context.Context, teamID int64, traderID int64, page pagination.Params) (pagination.Result[ShiftRequisite], error) {
	return s.store.ShiftRequisites(ctx, teamID, traderID, page)
}

func (s *Service) LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]TurnoverEntry, error) {
	return s.store.LatestTurnovers(ctx, teamID, traderID)
}

func (s *Service) TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64, page pagination.Params) (pagination.Result[TurnoverEntry], error) {
	if teamID <= 0 || traderID <= 0 || shiftRequisiteID <= 0 {
		return pagination.Result[TurnoverEntry]{}, ErrInvalidInput
	}

	return s.store.TurnoversByShiftRequisite(ctx, teamID, traderID, shiftRequisiteID, page)
}

func (s *Service) InternalTransfersByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64, page pagination.Params) (pagination.Result[InternalTransfer], error) {
	if teamID <= 0 || traderID <= 0 || shiftRequisiteID <= 0 {
		return pagination.Result[InternalTransfer]{}, ErrInvalidInput
	}

	return s.store.InternalTransfersByShiftRequisite(ctx, teamID, traderID, shiftRequisiteID, page)
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

func (s *Service) CreateInternalTransfer(ctx context.Context, params CreateInternalTransferParams) (InternalTransfer, error) {
	if params.ActorID <= 0 ||
		params.TeamID <= 0 ||
		params.TraderID <= 0 ||
		params.SourceShiftRequisiteID <= 0 ||
		params.DestinationShiftRequisiteID <= 0 ||
		params.SourceShiftRequisiteID == params.DestinationShiftRequisiteID ||
		params.AmountMinor <= 0 {
		return InternalTransfer{}, ErrInvalidInput
	}

	comment := cleanOptionalString(params.Comment)
	transfer, err := s.store.CreateInternalTransfer(ctx, CreateInternalTransferRecord{
		TeamID:                      params.TeamID,
		TraderID:                    params.TraderID,
		SourceShiftRequisiteID:      params.SourceShiftRequisiteID,
		DestinationShiftRequisiteID: params.DestinationShiftRequisiteID,
		AmountMinor:                 params.AmountMinor,
		CreatedBy:                   params.ActorID,
		Comment:                     comment,
	})
	if err != nil {
		return InternalTransfer{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftInternalTransferCreated,
		EntityType: "shift_requisite_internal_transfer",
		EntityID:   strconv.FormatInt(transfer.ID, 10),
		After:      PublicInternalTransferFromDomain(transfer),
		Comment:    comment,
	}); err != nil {
		return InternalTransfer{}, err
	}

	return transfer, nil
}

func (s *Service) CancelInternalTransfer(ctx context.Context, params CancelInternalTransferParams) (InternalTransfer, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.TransferID <= 0 {
		return InternalTransfer{}, ErrInvalidInput
	}

	transfer, err := s.store.CancelInternalTransfer(ctx, CancelInternalTransferRecord{
		TeamID:      params.TeamID,
		TraderID:    params.TraderID,
		TransferID:  params.TransferID,
		CancelledBy: params.ActorID,
	})
	if err != nil {
		return InternalTransfer{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftInternalTransferCancelled,
		EntityType: "shift_requisite_internal_transfer",
		EntityID:   strconv.FormatInt(transfer.ID, 10),
		After:      PublicInternalTransferFromDomain(transfer),
	}); err != nil {
		return InternalTransfer{}, err
	}

	return transfer, nil
}

func (s *Service) CloseShiftRequisite(ctx context.Context, params CloseShiftRequisiteParams) (ShiftRequisite, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftRequisiteID <= 0 || params.InboundTurnoverMinor < 0 || params.OutboundTurnoverMinor < 0 || params.ClosingBalanceMinor < 0 {
		return ShiftRequisite{}, ErrInvalidInput
	}

	closed, err := s.store.CloseShiftRequisite(ctx, CloseShiftRequisiteRecord{
		TeamID:                params.TeamID,
		TraderID:              params.TraderID,
		ShiftRequisiteID:      params.ShiftRequisiteID,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		Blocked:               params.Blocked,
		CreatedBy:             params.ActorID,
		ReleasedAt:            params.ReleasedAt,
	})
	if err != nil {
		return ShiftRequisite{}, err
	}

	comment := cleanOptionalString(params.Comment)
	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftRequisiteClosed,
		EntityType: "shift_requisite",
		EntityID:   strconv.FormatInt(closed.ID, 10),
		After:      PublicShiftRequisiteFromDomain(closed),
		Comment:    comment,
	}); err != nil {
		return ShiftRequisite{}, err
	}

	return closed, nil
}

func (s *Service) CorrectShiftRequisite(ctx context.Context, params CorrectShiftRequisiteParams) (ShiftRequisite, error) {
	comment := strings.TrimSpace(params.Comment)
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftRequisiteID <= 0 || params.InboundTurnoverMinor < 0 || params.OutboundTurnoverMinor < 0 || params.ClosingBalanceMinor < 0 || comment == "" {
		return ShiftRequisite{}, ErrInvalidInput
	}

	updated, err := s.store.CorrectClosedShiftRequisiteTurnovers(ctx, CorrectShiftRequisiteTurnoversRecord{
		TeamID:                params.TeamID,
		TraderID:              params.TraderID,
		ShiftRequisiteID:      params.ShiftRequisiteID,
		InboundTurnoverMinor:  params.InboundTurnoverMinor,
		OutboundTurnoverMinor: params.OutboundTurnoverMinor,
		ClosingBalanceMinor:   params.ClosingBalanceMinor,
		CreatedBy:             params.ActorID,
		Comment:               comment,
	})
	if err != nil {
		return ShiftRequisite{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftRequisiteCorrected,
		EntityType: "shift_requisite",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		After:      PublicShiftRequisiteFromDomain(updated),
		Comment:    &comment,
	}); err != nil {
		return ShiftRequisite{}, err
	}

	if s.turnoverHook != nil {
		if err := s.turnoverHook.AfterTurnoverCreated(ctx, TurnoverEntry{
			TeamID:           updated.TeamID,
			ShiftID:          updated.ShiftID,
			ShiftRequisiteID: updated.ID,
			RequisiteID:      updated.RequisiteID,
			TraderID:         updated.TraderID,
			AmountMinor:      updated.InboundTurnoverMinor,
			CreatedBy:        params.ActorID,
			Comment:          &comment,
		}); err != nil {
			return ShiftRequisite{}, err
		}
	}

	refreshed, err := s.store.GetShiftRequisite(ctx, params.TeamID, params.TraderID, params.ShiftRequisiteID)
	if err != nil {
		return ShiftRequisite{}, err
	}

	return refreshed, nil
}

func (s *Service) ReturnShiftRequisiteToWork(ctx context.Context, params ReturnShiftRequisiteParams) (ShiftRequisite, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.TraderID <= 0 || params.ShiftRequisiteID <= 0 {
		return ShiftRequisite{}, ErrInvalidInput
	}

	updated, err := s.store.ReturnShiftRequisiteToWork(ctx, params.TeamID, params.TraderID, params.ShiftRequisiteID)
	if err != nil {
		return ShiftRequisite{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionShiftRequisiteReturnedToWork,
		EntityType: "shift_requisite",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		After:      PublicShiftRequisiteFromDomain(updated),
	}); err != nil {
		return ShiftRequisite{}, err
	}

	return updated, nil
}

func (s *Service) CloseChecklist(ctx context.Context, teamID int64, traderID int64) (CloseChecklist, error) {
	checklist, err := s.store.CurrentShiftChecklist(ctx, teamID, traderID)
	if err != nil {
		return CloseChecklist{}, err
	}

	checklist.AllRequisitesClosed = checklist.OpenRequisiteCount == 0
	checklist.CanClose = checklist.InboundImported &&
		checklist.InboundOk &&
		checklist.OutboundImported &&
		checklist.OutboundOk &&
		checklist.AllRequisitesClosed &&
		checklist.AllPayoutsFullyPaid

	return checklist, nil
}

func (s *Service) CloseCurrent(ctx context.Context, params CloseShiftParams) (Shift, error) {
	checklist, err := s.CloseChecklist(ctx, params.TeamID, params.TraderID)
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

func normalizeAssignedRequisitesFilters(filters AssignedRequisitesFilters) AssignedRequisitesFilters {
	statuses := make([]string, 0, len(filters.Statuses))
	seen := map[string]bool{}
	for _, status := range filters.Statuses {
		status = strings.TrimSpace(status)
		if status == "" || status == "all" || seen[status] {
			continue
		}
		seen[status] = true
		statuses = append(statuses, status)
	}

	return AssignedRequisitesFilters{
		Statuses:  statuses,
		Search:    strings.TrimSpace(filters.Search),
		ExcludeID: filters.ExcludeID,
	}
}

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}
