package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/go-chi/chi/v5"
)

type TraderShiftService interface {
	Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error)
	AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error)
	ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.ShiftRequisite, error)
	TakeRequisite(ctx context.Context, params shifts.TakeRequisiteParams) (shifts.TakeRequisiteResult, error)
	UpdateShiftRequisite(ctx context.Context, params shifts.UpdateShiftRequisiteParams) (shifts.ShiftRequisite, error)
	LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]shifts.TurnoverEntry, error)
	TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]shifts.TurnoverEntry, error)
	CreateTurnover(ctx context.Context, params shifts.CreateTurnoverParams) (shifts.TurnoverEntry, error)
	CloseChecklist(ctx context.Context, teamID int64, traderID int64) (shifts.CloseChecklist, error)
	CloseCurrent(ctx context.Context, params shifts.CloseShiftParams) (shifts.Shift, error)
}

type TraderShiftHandler struct {
	service TraderShiftService
}

func NewTraderShiftHandler(service TraderShiftService) *TraderShiftHandler {
	return &TraderShiftHandler{service: service}
}

type currentShiftResponse struct {
	Shift *shifts.PublicShift `json:"shift"`
}

type assignedRequisitesResponse struct {
	Items []shifts.PublicAssignedRequisite `json:"items"`
}

type shiftRequisitesResponse struct {
	Items []shifts.PublicShiftRequisite `json:"items"`
}

type takeRequisiteResponse struct {
	Shift          shifts.PublicShift          `json:"shift"`
	ShiftRequisite shifts.PublicShiftRequisite `json:"shiftRequisite"`
	ShiftCreated   bool                        `json:"shiftCreated"`
}

type shiftRequisiteResponse struct {
	ShiftRequisite shifts.PublicShiftRequisite `json:"shiftRequisite"`
}

type turnoversResponse struct {
	Items []shifts.PublicTurnoverEntry `json:"items"`
}

type turnoverResponse struct {
	Turnover shifts.PublicTurnoverEntry `json:"turnover"`
}

type closeChecklistResponse struct {
	Checklist shifts.PublicCloseChecklist `json:"checklist"`
}

type closeShiftResponse struct {
	Shift shifts.PublicShift `json:"shift"`
}

type takeRequisiteRequest struct {
	CardNumber string `json:"cardNumber"`
	HolderName string `json:"holderName"`
}

type updateShiftRequisiteRequest struct {
	CardNumber string `json:"cardNumber"`
	HolderName string `json:"holderName"`
}

type createTurnoverRequest struct {
	ShiftRequisiteID int64   `json:"shiftRequisiteId"`
	AmountMinor      int64   `json:"amountMinor"`
	Comment          *string `json:"comment"`
}

type closeShiftRequest struct {
	CloseComment *string `json:"closeComment"`
}

func (h *TraderShiftHandler) Current(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shift, err := h.service.Current(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}
	if shift == nil {
		WriteJSON(w, http.StatusOK, currentShiftResponse{})
		return
	}

	publicShift := shifts.PublicShiftFromDomain(*shift)
	WriteJSON(w, http.StatusOK, currentShiftResponse{
		Shift: &publicShift,
	})
}

func (h *TraderShiftHandler) AssignedRequisites(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.AssignedRequisites(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignedRequisitesResponse{
		Items: shifts.PublicAssignedRequisites(items),
	})
}

func (h *TraderShiftHandler) TakeRequisite(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := shiftRequisiteRouteID(w, r, "requisiteId", "Некорректный ID реквизита")
	if !ok {
		return
	}

	var request takeRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateTakeRequisiteRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	result, err := h.service.TakeRequisite(r.Context(), shifts.TakeRequisiteParams{
		ActorID:     actor.ID,
		TeamID:      actor.TeamID,
		TraderID:    actor.ID,
		RequisiteID: requisiteID,
		CardNumber:  request.CardNumber,
		HolderName:  request.HolderName,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, takeRequisiteResponse{
		Shift:          shifts.PublicShiftFromDomain(result.Shift),
		ShiftRequisite: shifts.PublicShiftRequisiteFromDomain(result.ShiftRequisite),
		ShiftCreated:   result.ShiftCreated,
	})
}

func (h *TraderShiftHandler) ShiftRequisites(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.ShiftRequisites(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftRequisitesResponse{
		Items: shifts.PublicShiftRequisites(items),
	})
}

func (h *TraderShiftHandler) UpdateShiftRequisite(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftRequisiteID, ok := shiftRequisiteRouteID(w, r, "shiftRequisiteId", "Некорректный ID shift requisite")
	if !ok {
		return
	}

	var request updateShiftRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateUpdateShiftRequisiteRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	item, err := h.service.UpdateShiftRequisite(r.Context(), shifts.UpdateShiftRequisiteParams{
		ActorID:          actor.ID,
		TeamID:           actor.TeamID,
		TraderID:         actor.ID,
		ShiftRequisiteID: shiftRequisiteID,
		CardNumber:       request.CardNumber,
		HolderName:       request.HolderName,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftRequisiteResponse{
		ShiftRequisite: shifts.PublicShiftRequisiteFromDomain(item),
	})
}

func (h *TraderShiftHandler) LatestTurnovers(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.LatestTurnovers(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, turnoversResponse{
		Items: shifts.PublicTurnoverEntries(items),
	})
}

func (h *TraderShiftHandler) CreateTurnover(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request createTurnoverRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateCreateTurnoverRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	entry, err := h.service.CreateTurnover(r.Context(), shifts.CreateTurnoverParams{
		ActorID:          actor.ID,
		TeamID:           actor.TeamID,
		TraderID:         actor.ID,
		ShiftRequisiteID: request.ShiftRequisiteID,
		AmountMinor:      request.AmountMinor,
		Comment:          request.Comment,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, turnoverResponse{
		Turnover: shifts.PublicTurnoverEntryFromDomain(entry),
	})
}

func (h *TraderShiftHandler) TurnoversByShiftRequisite(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftRequisiteID, ok := shiftRequisiteRouteID(w, r, "shiftRequisiteId", "Некорректный ID shift requisite")
	if !ok {
		return
	}

	items, err := h.service.TurnoversByShiftRequisite(r.Context(), actor.TeamID, actor.ID, shiftRequisiteID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, turnoversResponse{
		Items: shifts.PublicTurnoverEntries(items),
	})
}

func (h *TraderShiftHandler) CloseChecklist(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	checklist, err := h.service.CloseChecklist(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, closeChecklistResponse{
		Checklist: shifts.PublicCloseChecklistFromDomain(checklist),
	})
}

func (h *TraderShiftHandler) CloseCurrent(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request closeShiftRequest
	if r.Body != nil && r.ContentLength != 0 {
		if !decodeJSON(w, r, &request) {
			return
		}
	}

	shift, err := h.service.CloseCurrent(r.Context(), shifts.CloseShiftParams{
		ActorID:      actor.ID,
		TeamID:       actor.TeamID,
		TraderID:     actor.ID,
		CloseComment: request.CloseComment,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, closeShiftResponse{
		Shift: shifts.PublicShiftFromDomain(shift),
	})
}

func shiftRequisiteRouteID(w http.ResponseWriter, r *http.Request, param string, message string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			param: message,
		}))
		return 0, false
	}

	return id, true
}

func validateTakeRequisiteRequest(request takeRequisiteRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(request.CardNumber) == "" {
		fields["cardNumber"] = "Номер карты обязателен"
	}
	if strings.TrimSpace(request.HolderName) == "" {
		fields["holderName"] = "ФИО держателя обязательно"
	}

	return fields
}

func validateUpdateShiftRequisiteRequest(request updateShiftRequisiteRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(request.CardNumber) == "" {
		fields["cardNumber"] = "Номер карты обязателен"
	}
	if strings.TrimSpace(request.HolderName) == "" {
		fields["holderName"] = "ФИО держателя обязательно"
	}

	return fields
}

func validateCreateTurnoverRequest(request createTurnoverRequest) map[string]string {
	fields := map[string]string{}
	if request.ShiftRequisiteID <= 0 {
		fields["shiftRequisiteId"] = "Некорректный ID shift requisite"
	}
	if request.AmountMinor < 0 {
		fields["amountMinor"] = "Оборот не может быть отрицательным"
	}

	return fields
}

func mapShiftError(err error) error {
	switch {
	case errors.Is(err, shifts.ErrInvalidInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		})
	case errors.Is(err, shifts.ErrRequisiteNotAssigned):
		return DomainError("REQUISITE_NOT_ASSIGNED", "Реквизит не назначен текущему трейдеру")
	case errors.Is(err, shifts.ErrShiftRequisiteExists):
		return DomainError("REQUISITE_ALREADY_IN_WORK", "Реквизит уже взят в работу в текущей смене")
	case errors.Is(err, shifts.ErrShiftRequisiteNotFound):
		return NotFoundError()
	case errors.Is(err, shifts.ErrTurnoverTargetNotFound):
		return NotFoundError()
	case errors.Is(err, shifts.ErrCurrentShiftNotFound):
		return NotFoundError()
	case errors.Is(err, shifts.ErrCloseBlocked):
		return DomainError("SHIFT_CANNOT_BE_CLOSED", "Смену нельзя закрыть: checklist не выполнен")
	case errors.Is(err, shifts.ErrShiftCannotBeClosed):
		return DomainError("SHIFT_CANNOT_BE_CLOSED", "Смену нельзя закрыть: checklist не выполнен")
	default:
		return err
	}
}
