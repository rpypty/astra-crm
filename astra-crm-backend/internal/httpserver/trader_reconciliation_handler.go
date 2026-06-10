package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/go-chi/chi/v5"
)

type TraderReconciliationService interface {
	LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (reconciliation.Run, error)
	AcceptTraderInbound(ctx context.Context, params reconciliation.AcceptTraderInboundParams) (reconciliation.Run, error)
	LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (reconciliation.Run, error)
	AcceptTraderOutbound(ctx context.Context, params reconciliation.AcceptTraderOutboundParams) (reconciliation.Run, error)
}

type TraderReconciliationShiftService interface {
	Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error)
}

type TraderReconciliationHandler struct {
	service      TraderReconciliationService
	shiftService TraderReconciliationShiftService
}

func NewTraderReconciliationHandler(service TraderReconciliationService, shiftService TraderReconciliationShiftService) *TraderReconciliationHandler {
	return &TraderReconciliationHandler{
		service:      service,
		shiftService: shiftService,
	}
}

type reconciliationRunResponse struct {
	Run reconciliation.PublicRun `json:"run"`
}

type acceptReconciliationRequest struct {
	Comment string `json:"comment"`
}

func (h *TraderReconciliationHandler) LatestInbound(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil || h.shiftService == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shift, ok := h.currentShift(w, r, actor.TeamID, actor.ID)
	if !ok {
		return
	}

	run, err := h.service.LatestTraderInbound(r.Context(), actor.TeamID, actor.ID, shift.ID)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TraderReconciliationHandler) LatestOutbound(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil || h.shiftService == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shift, ok := h.currentShift(w, r, actor.TeamID, actor.ID)
	if !ok {
		return
	}

	run, err := h.service.LatestTraderOutbound(r.Context(), actor.TeamID, actor.ID, shift.ID)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TraderReconciliationHandler) AcceptInbound(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil || h.shiftService == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	if _, ok := h.currentShift(w, r, actor.TeamID, actor.ID); !ok {
		return
	}

	runID, ok := reconciliationRunIDFromRequest(w, r)
	if !ok {
		return
	}

	var request acceptReconciliationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Comment) == "" {
		RespondError(w, ValidationError(map[string]string{
			"comment": "Комментарий обязателен при подтверждении расхождения",
		}))
		return
	}

	run, err := h.service.AcceptTraderInbound(r.Context(), reconciliation.AcceptTraderInboundParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		TraderID: actor.ID,
		RunID:    runID,
		Comment:  request.Comment,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TraderReconciliationHandler) AcceptOutbound(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil || h.shiftService == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	if _, ok := h.currentShift(w, r, actor.TeamID, actor.ID); !ok {
		return
	}

	runID, ok := reconciliationRunIDFromRequest(w, r)
	if !ok {
		return
	}

	var request acceptReconciliationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Comment) == "" {
		RespondError(w, ValidationError(map[string]string{
			"comment": "Комментарий обязателен при подтверждении расхождения",
		}))
		return
	}

	run, err := h.service.AcceptTraderOutbound(r.Context(), reconciliation.AcceptTraderOutboundParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		TraderID: actor.ID,
		RunID:    runID,
		Comment:  request.Comment,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TraderReconciliationHandler) currentShift(w http.ResponseWriter, r *http.Request, teamID int64, traderID int64) (*shifts.Shift, bool) {
	shift, err := h.shiftService.Current(r.Context(), teamID, traderID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return nil, false
	}
	if shift == nil {
		RespondError(w, DomainError("CURRENT_SHIFT_REQUIRED", "Нужна открытая смена для сверки"))
		return nil, false
	}

	return shift, true
}

func reconciliationRunIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "runId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"runId": "Некорректный ID сверки",
		}))
		return 0, false
	}

	return id, true
}

func mapReconciliationError(err error) error {
	switch {
	case errors.Is(err, reconciliation.ErrInvalidInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		})
	case errors.Is(err, reconciliation.ErrRunNotFound):
		return NotFoundError()
	case errors.Is(err, reconciliation.ErrRepositoryNotConfigured):
		return ServiceUnavailableError()
	default:
		return err
	}
}
