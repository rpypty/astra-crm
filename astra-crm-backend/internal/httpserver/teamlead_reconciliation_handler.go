package httpserver

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/go-chi/chi/v5"
)

type TeamleadReconciliationService interface {
	LatestTeamleadInbound(ctx context.Context, teamID int64) (reconciliation.Run, error)
	LatestTeamleadOutbound(ctx context.Context, teamID int64) (reconciliation.Run, error)
	LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (reconciliation.Run, error)
	LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (reconciliation.Run, error)
	ListTraderInboundItemsByShift(ctx context.Context, teamID int64, shiftID int64) ([]reconciliation.Item, error)
	ListTraderOutboundItemsByShift(ctx context.Context, teamID int64, shiftID int64) ([]reconciliation.Item, error)
	LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error)
	LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error)
	ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64) ([]reconciliation.Item, error)
	ListTeamleadPeriodOutboundItems(ctx context.Context, teamID int64, accountingPeriodID int64) ([]reconciliation.Item, error)
}

type TeamleadReconciliationHandler struct {
	service TeamleadReconciliationService
}

func NewTeamleadReconciliationHandler(service TeamleadReconciliationService) *TeamleadReconciliationHandler {
	return &TeamleadReconciliationHandler{service: service}
}

type reconciliationItemsResponse struct {
	Items []reconciliation.PublicItem `json:"items"`
}

func (h *TeamleadReconciliationHandler) LatestInbound(w http.ResponseWriter, r *http.Request) {
	h.latest(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) LatestOutbound(w http.ResponseWriter, r *http.Request) {
	h.latest(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) latest(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var (
		run reconciliation.Run
		err error
	)
	switch direction {
	case "inbound":
		run, err = h.service.LatestTeamleadInbound(r.Context(), actor.TeamID)
	case "outbound":
		run, err = h.service.LatestTeamleadOutbound(r.Context(), actor.TeamID)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) PeriodInbound(w http.ResponseWriter, r *http.Request) {
	h.period(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) PeriodOutbound(w http.ResponseWriter, r *http.Request) {
	h.period(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) period(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	periodID, ok := periodIDFromRequest(w, r)
	if !ok {
		return
	}

	var (
		run reconciliation.Run
		err error
	)
	switch direction {
	case "inbound":
		run, err = h.service.LatestTeamleadPeriodInbound(r.Context(), actor.TeamID, periodID)
	case "outbound":
		run, err = h.service.LatestTeamleadPeriodOutbound(r.Context(), actor.TeamID, periodID)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) PeriodInboundItems(w http.ResponseWriter, r *http.Request) {
	h.periodItems(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) PeriodOutboundItems(w http.ResponseWriter, r *http.Request) {
	h.periodItems(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) PeriodItems(w http.ResponseWriter, r *http.Request) {
	h.PeriodInboundItems(w, r)
}

func (h *TeamleadReconciliationHandler) ShiftInbound(w http.ResponseWriter, r *http.Request) {
	h.shift(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) ShiftOutbound(w http.ResponseWriter, r *http.Request) {
	h.shift(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) ShiftInboundItems(w http.ResponseWriter, r *http.Request) {
	h.shiftItems(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) ShiftOutboundItems(w http.ResponseWriter, r *http.Request) {
	h.shiftItems(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) shift(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftID, ok := shiftIDFromRequest(w, r)
	if !ok {
		return
	}

	var (
		run reconciliation.Run
		err error
	)
	switch direction {
	case "inbound":
		run, err = h.service.LatestTraderInboundByShift(r.Context(), actor.TeamID, shiftID)
	case "outbound":
		run, err = h.service.LatestTraderOutboundByShift(r.Context(), actor.TeamID, shiftID)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) shiftItems(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shiftID, ok := shiftIDFromRequest(w, r)
	if !ok {
		return
	}

	var (
		items []reconciliation.Item
		err   error
	)
	switch direction {
	case "inbound":
		items, err = h.service.ListTraderInboundItemsByShift(r.Context(), actor.TeamID, shiftID)
	case "outbound":
		items, err = h.service.ListTraderOutboundItemsByShift(r.Context(), actor.TeamID, shiftID)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationItemsResponse{
		Items: reconciliation.PublicItemsFromDomain(items),
	})
}

func (h *TeamleadReconciliationHandler) periodItems(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	periodID, ok := periodIDFromRequest(w, r)
	if !ok {
		return
	}

	var (
		items []reconciliation.Item
		err   error
	)
	switch direction {
	case "inbound":
		items, err = h.service.ListTeamleadPeriodInboundItems(r.Context(), actor.TeamID, periodID)
	case "outbound":
		items, err = h.service.ListTeamleadPeriodOutboundItems(r.Context(), actor.TeamID, periodID)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationItemsResponse{
		Items: reconciliation.PublicItemsFromDomain(items),
	})
}

func shiftIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "shiftId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"shiftId": "Некорректный ID смены",
		}))
		return 0, false
	}

	return id, true
}

func periodIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "periodId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"periodId": "Некорректный ID периода",
		}))
		return 0, false
	}

	return id, true
}
