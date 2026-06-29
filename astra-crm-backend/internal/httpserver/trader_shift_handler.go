package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/go-chi/chi/v5"
)

type TraderShiftService interface {
	Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error)
	ShiftHistory(ctx context.Context, teamID int64, traderID int64, limit int32) ([]shifts.Shift, error)
	TeamShiftHistory(ctx context.Context, teamID int64, limit int32) ([]shifts.Shift, error)
	ShiftReport(ctx context.Context, teamID int64, traderID int64, shiftID int64) (shifts.ShiftReportDetails, error)
	TeamShiftReport(ctx context.Context, teamID int64, shiftID int64) (shifts.ShiftReportDetails, error)
	AssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error)
	FutureAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error)
	HistoricalAssignedRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.AssignedRequisite, error)
	AssignedRequisitesByShift(ctx context.Context, teamID int64, traderID int64, shiftID int64) ([]shifts.AssignedRequisite, error)
	AssignedRequisitesByTeamShift(ctx context.Context, teamID int64, shiftID int64) ([]shifts.AssignedRequisite, error)
	ShiftRequisites(ctx context.Context, teamID int64, traderID int64) ([]shifts.ShiftRequisite, error)
	TakeRequisite(ctx context.Context, params shifts.TakeRequisiteParams) (shifts.TakeRequisiteResult, error)
	UpdateShiftRequisite(ctx context.Context, params shifts.UpdateShiftRequisiteParams) (shifts.ShiftRequisite, error)
	CloseShiftRequisite(ctx context.Context, params shifts.CloseShiftRequisiteParams) (shifts.ShiftRequisite, error)
	CorrectShiftRequisite(ctx context.Context, params shifts.CorrectShiftRequisiteParams) (shifts.ShiftRequisite, error)
	ReturnShiftRequisiteToWork(ctx context.Context, params shifts.ReturnShiftRequisiteParams) (shifts.ShiftRequisite, error)
	LatestTurnovers(ctx context.Context, teamID int64, traderID int64) ([]shifts.TurnoverEntry, error)
	TurnoversByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]shifts.TurnoverEntry, error)
	CreateTurnover(ctx context.Context, params shifts.CreateTurnoverParams) (shifts.TurnoverEntry, error)
	InternalTransfersByShiftRequisite(ctx context.Context, teamID int64, traderID int64, shiftRequisiteID int64) ([]shifts.InternalTransfer, error)
	CreateInternalTransfer(ctx context.Context, params shifts.CreateInternalTransferParams) (shifts.InternalTransfer, error)
	CancelInternalTransfer(ctx context.Context, params shifts.CancelInternalTransferParams) (shifts.InternalTransfer, error)
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

type shiftHistoryResponse struct {
	Items []shifts.PublicShift `json:"items"`
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

type internalTransfersResponse struct {
	Items []shifts.PublicInternalTransfer `json:"items"`
}

type internalTransferResponse struct {
	Transfer shifts.PublicInternalTransfer `json:"transfer"`
}

type closeChecklistResponse struct {
	Checklist shifts.PublicCloseChecklist `json:"checklist"`
}

type closeShiftResponse struct {
	Shift shifts.PublicShift `json:"shift"`
}

type shiftReportDetailsResponse struct {
	Report shifts.PublicShiftReportDetails `json:"report"`
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

type createInternalTransferRequest struct {
	SourceShiftRequisiteID      int64   `json:"sourceShiftRequisiteId"`
	DestinationShiftRequisiteID int64   `json:"destinationShiftRequisiteId"`
	AmountMinor                 int64   `json:"amountMinor"`
	Comment                     *string `json:"comment"`
}

type closeShiftRequisiteRequest struct {
	InboundTurnoverMinor  int64   `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor int64   `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor   int64   `json:"closingBalanceMinor"`
	Blocked               bool    `json:"blocked"`
	Comment               *string `json:"comment"`
	ReleasedAt            *string `json:"releasedAt"`
}

type correctShiftRequisiteRequest struct {
	InboundTurnoverMinor  int64  `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor int64  `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor   int64  `json:"closingBalanceMinor"`
	Comment               string `json:"comment"`
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

func (h *TraderShiftHandler) History(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.ShiftHistory(r.Context(), actor.TeamID, actor.ID, 30)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftHistoryResponse{
		Items: shifts.PublicShiftsFromDomain(items),
	})
}

func (h *TraderShiftHandler) TeamHistory(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.TeamShiftHistory(r.Context(), actor.TeamID, 100)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftHistoryResponse{
		Items: shifts.PublicShiftsFromDomain(items),
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

func (h *TraderShiftHandler) FutureAssignedRequisites(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.FutureAssignedRequisites(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignedRequisitesResponse{
		Items: shifts.PublicAssignedRequisites(items),
	})
}

func (h *TraderShiftHandler) HistoricalAssignedRequisites(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.HistoricalAssignedRequisites(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignedRequisitesResponse{
		Items: shifts.PublicAssignedRequisites(items),
	})
}

func (h *TraderShiftHandler) AssignedRequisitesByShift(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftID, ok := shiftRequisiteRouteID(w, r, "shiftId", "Некорректный ID смены")
	if !ok {
		return
	}

	items, err := h.service.AssignedRequisitesByShift(r.Context(), actor.TeamID, actor.ID, shiftID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignedRequisitesResponse{
		Items: shifts.PublicAssignedRequisites(items),
	})
}

func (h *TraderShiftHandler) ShiftReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftID, ok := shiftRequisiteRouteID(w, r, "shiftId", "Некорректный ID смены")
	if !ok {
		return
	}

	report, err := h.service.ShiftReport(r.Context(), actor.TeamID, actor.ID, shiftID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftReportDetailsResponse{
		Report: shifts.PublicShiftReportDetailsFromDomain(report),
	})
}

func (h *TraderShiftHandler) AssignedRequisitesByTeamShift(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftID, ok := shiftRequisiteRouteID(w, r, "shiftId", "Некорректный ID смены")
	if !ok {
		return
	}

	items, err := h.service.AssignedRequisitesByTeamShift(r.Context(), actor.TeamID, shiftID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignedRequisitesResponse{
		Items: shifts.PublicAssignedRequisites(items),
	})
}

func (h *TraderShiftHandler) TeamShiftReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftID, ok := shiftRequisiteRouteID(w, r, "shiftId", "Некорректный ID смены")
	if !ok {
		return
	}

	report, err := h.service.TeamShiftReport(r.Context(), actor.TeamID, shiftID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftReportDetailsResponse{
		Report: shifts.PublicShiftReportDetailsFromDomain(report),
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

func (h *TraderShiftHandler) CloseShiftRequisite(w http.ResponseWriter, r *http.Request) {
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

	var request closeShiftRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	releasedAt, fields := validateCloseShiftRequisiteRequest(request)
	if len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	item, err := h.service.CloseShiftRequisite(r.Context(), shifts.CloseShiftRequisiteParams{
		ActorID:               actor.ID,
		TeamID:                actor.TeamID,
		TraderID:              actor.ID,
		ShiftRequisiteID:      shiftRequisiteID,
		InboundTurnoverMinor:  request.InboundTurnoverMinor,
		OutboundTurnoverMinor: request.OutboundTurnoverMinor,
		ClosingBalanceMinor:   request.ClosingBalanceMinor,
		Blocked:               request.Blocked,
		Comment:               request.Comment,
		ReleasedAt:            releasedAt,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftRequisiteResponse{
		ShiftRequisite: shifts.PublicShiftRequisiteFromDomain(item),
	})
}

func (h *TraderShiftHandler) CorrectShiftRequisite(w http.ResponseWriter, r *http.Request) {
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

	var request correctShiftRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateCorrectShiftRequisiteRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	item, err := h.service.CorrectShiftRequisite(r.Context(), shifts.CorrectShiftRequisiteParams{
		ActorID:               actor.ID,
		TeamID:                actor.TeamID,
		TraderID:              actor.ID,
		ShiftRequisiteID:      shiftRequisiteID,
		InboundTurnoverMinor:  request.InboundTurnoverMinor,
		OutboundTurnoverMinor: request.OutboundTurnoverMinor,
		ClosingBalanceMinor:   request.ClosingBalanceMinor,
		Comment:               request.Comment,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, shiftRequisiteResponse{
		ShiftRequisite: shifts.PublicShiftRequisiteFromDomain(item),
	})
}

func (h *TraderShiftHandler) ReturnShiftRequisiteToWork(w http.ResponseWriter, r *http.Request) {
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

	item, err := h.service.ReturnShiftRequisiteToWork(r.Context(), shifts.ReturnShiftRequisiteParams{
		ActorID:          actor.ID,
		TeamID:           actor.TeamID,
		TraderID:         actor.ID,
		ShiftRequisiteID: shiftRequisiteID,
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

func (h *TraderShiftHandler) InternalTransfersByShiftRequisite(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.service.InternalTransfersByShiftRequisite(r.Context(), actor.TeamID, actor.ID, shiftRequisiteID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, internalTransfersResponse{
		Items: shifts.PublicInternalTransfers(items),
	})
}

func (h *TraderShiftHandler) CreateInternalTransfer(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request createInternalTransferRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateCreateInternalTransferRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	transfer, err := h.service.CreateInternalTransfer(r.Context(), shifts.CreateInternalTransferParams{
		ActorID:                     actor.ID,
		TeamID:                      actor.TeamID,
		TraderID:                    actor.ID,
		SourceShiftRequisiteID:      request.SourceShiftRequisiteID,
		DestinationShiftRequisiteID: request.DestinationShiftRequisiteID,
		AmountMinor:                 request.AmountMinor,
		Comment:                     request.Comment,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, internalTransferResponse{
		Transfer: shifts.PublicInternalTransferFromDomain(transfer),
	})
}

func (h *TraderShiftHandler) CancelInternalTransfer(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	transferID, ok := shiftRequisiteRouteID(w, r, "transferId", "Некорректный ID внутреннего перевода")
	if !ok {
		return
	}

	transfer, err := h.service.CancelInternalTransfer(r.Context(), shifts.CancelInternalTransferParams{
		ActorID:    actor.ID,
		TeamID:     actor.TeamID,
		TraderID:   actor.ID,
		TransferID: transferID,
	})
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}

	WriteJSON(w, http.StatusOK, internalTransferResponse{
		Transfer: shifts.PublicInternalTransferFromDomain(transfer),
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

func validateCreateInternalTransferRequest(request createInternalTransferRequest) map[string]string {
	fields := map[string]string{}
	if request.SourceShiftRequisiteID <= 0 {
		fields["sourceShiftRequisiteId"] = "Некорректный ID реквизита-источника"
	}
	if request.DestinationShiftRequisiteID <= 0 {
		fields["destinationShiftRequisiteId"] = "Некорректный ID реквизита-получателя"
	}
	if request.SourceShiftRequisiteID > 0 && request.SourceShiftRequisiteID == request.DestinationShiftRequisiteID {
		fields["destinationShiftRequisiteId"] = "Нельзя перелить сумму на тот же реквизит"
	}
	if request.AmountMinor <= 0 {
		fields["amountMinor"] = "Сумма должна быть больше нуля"
	}

	return fields
}

func validateCloseShiftRequisiteRequest(request closeShiftRequisiteRequest) (*time.Time, map[string]string) {
	fields := map[string]string{}
	if request.InboundTurnoverMinor < 0 {
		fields["inboundTurnoverMinor"] = "Оборот по оплатам не может быть отрицательным"
	}
	if request.OutboundTurnoverMinor < 0 {
		fields["outboundTurnoverMinor"] = "Оборот по выплатам не может быть отрицательным"
	}
	if request.ClosingBalanceMinor < 0 {
		fields["closingBalanceMinor"] = "Остаток не может быть отрицательным"
	}

	releasedAt := parseOptionalRequestTime(request.ReleasedAt, "releasedAt", fields)
	if releasedAt != nil && releasedAt.After(time.Now().Add(5*time.Minute)) {
		fields["releasedAt"] = "Дата закрытия не может быть в будущем"
	}

	return releasedAt, fields
}

func parseOptionalRequestTime(raw *string, field string, fields map[string]string) *time.Time {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}

	value := strings.TrimSpace(*raw)
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &parsed
	}
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return &parsed
	}

	fields[field] = "Дата должна быть в формате RFC3339 или YYYY-MM-DD"
	return nil
}

func validateCorrectShiftRequisiteRequest(request correctShiftRequisiteRequest) map[string]string {
	fields := map[string]string{}
	if request.InboundTurnoverMinor < 0 {
		fields["inboundTurnoverMinor"] = "Оборот по оплатам не может быть отрицательным"
	}
	if request.OutboundTurnoverMinor < 0 {
		fields["outboundTurnoverMinor"] = "Оборот по выплатам не может быть отрицательным"
	}
	if request.ClosingBalanceMinor < 0 {
		fields["closingBalanceMinor"] = "Остаток не может быть отрицательным"
	}
	if strings.TrimSpace(request.Comment) == "" {
		fields["comment"] = "Комментарий к корректировке обязателен"
	}

	return fields
}

func mapShiftError(err error) error {
	switch {
	case errors.Is(err, shifts.ErrInvalidInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		}).WithCause(err)
	case errors.Is(err, shifts.ErrRequisiteNotAssigned):
		return DomainError("REQUISITE_NOT_ASSIGNED", "Реквизит не назначен текущему трейдеру").WithCause(err)
	case errors.Is(err, shifts.ErrShiftRequisiteExists):
		return DomainError("REQUISITE_ALREADY_IN_WORK", "Реквизит уже взят в работу в текущей смене").WithCause(err)
	case errors.Is(err, shifts.ErrShiftRequisiteNotFound):
		return NotFoundError().WithCause(err)
	case errors.Is(err, shifts.ErrTurnoverTargetNotFound):
		return NotFoundError().WithCause(err)
	case errors.Is(err, shifts.ErrInternalTransferNotFound):
		return NotFoundError().WithCause(err)
	case errors.Is(err, shifts.ErrInternalTransferTargetNotFound):
		return DomainError("INTERNAL_TRANSFER_TARGET_NOT_FOUND", "Реквизиты для внутреннего перевода не найдены в текущей смене").WithCause(err)
	case errors.Is(err, shifts.ErrCurrentShiftNotFound):
		return NotFoundError().WithCause(err)
	case errors.Is(err, shifts.ErrCloseBlocked):
		return DomainError("SHIFT_CANNOT_BE_CLOSED", "Смену нельзя закрыть: checklist не выполнен").WithCause(err)
	case errors.Is(err, shifts.ErrShiftCannotBeClosed):
		return DomainError("SHIFT_CANNOT_BE_CLOSED", "Смену нельзя закрыть: checklist не выполнен").WithCause(err)
	default:
		return err
	}
}
