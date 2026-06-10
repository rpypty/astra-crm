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
	LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error)
	ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64) ([]reconciliation.Item, error)
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
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	run, err := h.service.LatestTeamleadInbound(r.Context(), actor.TeamID)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) PeriodInbound(w http.ResponseWriter, r *http.Request) {
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

	run, err := h.service.LatestTeamleadPeriodInbound(r.Context(), actor.TeamID, periodID)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) PeriodItems(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.service.ListTeamleadPeriodInboundItems(r.Context(), actor.TeamID, periodID)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationItemsResponse{
		Items: reconciliation.PublicItemsFromDomain(items),
	})
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
