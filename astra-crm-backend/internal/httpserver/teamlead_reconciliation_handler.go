package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/go-chi/chi/v5"
)

type TeamleadReconciliationService interface {
	LatestTeamleadInbound(ctx context.Context, teamID int64, actorID int64) (reconciliation.Run, error)
	LatestTeamleadOutbound(ctx context.Context, teamID int64, actorID int64) (reconciliation.Run, error)
	RecalculateTeamleadCurrent(ctx context.Context, params reconciliation.RecalculateTeamleadCurrentParams) (reconciliation.Run, error)
	ListTeamleadCurrentRuns(ctx context.Context, params reconciliation.ListTeamleadCurrentRunsParams) (pagination.Result[reconciliation.Run], error)
	GetTeamleadCurrentRun(ctx context.Context, params reconciliation.GetTeamleadCurrentRunParams) (reconciliation.Run, error)
	ListTeamleadCurrentItems(ctx context.Context, params reconciliation.GetTeamleadCurrentRunParams, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
	ListTeamleadInboundItems(ctx context.Context, teamID int64, actorID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
	ListTeamleadOutboundItems(ctx context.Context, teamID int64, actorID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
	AcceptTeamleadCurrent(ctx context.Context, params reconciliation.AcceptTeamleadCurrentParams) (reconciliation.Run, error)
	LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (reconciliation.Run, error)
	LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (reconciliation.Run, error)
	ListTraderInboundItemsByShift(ctx context.Context, teamID int64, shiftID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
	ListTraderOutboundItemsByShift(ctx context.Context, teamID int64, shiftID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
	LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error)
	LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (reconciliation.Run, error)
	ListTeamleadPeriodInboundItems(ctx context.Context, teamID int64, accountingPeriodID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
	ListTeamleadPeriodOutboundItems(ctx context.Context, teamID int64, accountingPeriodID int64, filters reconciliation.ItemFilters, page pagination.Params) (pagination.Result[reconciliation.Item], error)
}

type TeamleadReconciliationHandler struct {
	service TeamleadReconciliationService
}

func NewTeamleadReconciliationHandler(service TeamleadReconciliationService) *TeamleadReconciliationHandler {
	return &TeamleadReconciliationHandler{service: service}
}

func (h *TeamleadReconciliationHandler) LatestInbound(w http.ResponseWriter, r *http.Request) {
	h.latest(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) LatestOutbound(w http.ResponseWriter, r *http.Request) {
	h.latest(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) StartInbound(w http.ResponseWriter, r *http.Request) {
	h.start(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) StartOutbound(w http.ResponseWriter, r *http.Request) {
	h.start(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) InboundItems(w http.ResponseWriter, r *http.Request) {
	h.currentItems(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) OutboundItems(w http.ResponseWriter, r *http.Request) {
	h.currentItems(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) InboundHistory(w http.ResponseWriter, r *http.Request) {
	h.history(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) OutboundHistory(w http.ResponseWriter, r *http.Request) {
	h.history(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) InboundRun(w http.ResponseWriter, r *http.Request) {
	h.currentRun(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) OutboundRun(w http.ResponseWriter, r *http.Request) {
	h.currentRun(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) InboundRunItems(w http.ResponseWriter, r *http.Request) {
	h.currentRunItems(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) OutboundRunItems(w http.ResponseWriter, r *http.Request) {
	h.currentRunItems(w, r, "outbound")
}

func (h *TeamleadReconciliationHandler) AcceptInbound(w http.ResponseWriter, r *http.Request) {
	h.accept(w, r, "inbound")
}

func (h *TeamleadReconciliationHandler) AcceptOutbound(w http.ResponseWriter, r *http.Request) {
	h.accept(w, r, "outbound")
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
		run, err = h.service.LatestTeamleadInbound(r.Context(), actor.TeamID, actor.ID)
	case "outbound":
		run, err = h.service.LatestTeamleadOutbound(r.Context(), actor.TeamID, actor.ID)
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

func (h *TeamleadReconciliationHandler) start(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	run, err := h.service.RecalculateTeamleadCurrent(r.Context(), reconciliation.RecalculateTeamleadCurrentParams{
		TeamID:    actor.TeamID,
		ActorID:   actor.ID,
		Direction: direction,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) currentItems(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}
	page, ok := paginationFromRequest(w, r)
	if !ok {
		return
	}
	filters, ok := reconciliationItemFiltersFromRequest(w, r)
	if !ok {
		return
	}

	var (
		items pagination.Result[reconciliation.Item]
		err   error
	)
	switch direction {
	case "inbound":
		items, err = h.service.ListTeamleadInboundItems(r.Context(), actor.TeamID, actor.ID, filters, page)
	case "outbound":
		items, err = h.service.ListTeamleadOutboundItems(r.Context(), actor.TeamID, actor.ID, filters, page)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(reconciliation.PublicItemsFromDomain(items.Items), page, items.Total)))
}

func (h *TeamleadReconciliationHandler) history(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}
	page, ok := paginationFromRequest(w, r)
	if !ok {
		return
	}

	runs, err := h.service.ListTeamleadCurrentRuns(r.Context(), reconciliation.ListTeamleadCurrentRunsParams{
		TeamID:    actor.TeamID,
		ActorID:   actor.ID,
		Direction: direction,
		Page:      page,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(reconciliation.PublicRunsFromDomain(runs.Items), page, runs.Total)))
}

func (h *TeamleadReconciliationHandler) currentRun(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	runID, ok := reconciliationRunIDFromRequest(w, r)
	if !ok {
		return
	}

	run, err := h.service.GetTeamleadCurrentRun(r.Context(), reconciliation.GetTeamleadCurrentRunParams{
		TeamID:    actor.TeamID,
		ActorID:   actor.ID,
		Direction: direction,
		RunID:     runID,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, reconciliationRunResponse{
		Run: reconciliation.PublicRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) currentRunItems(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	runID, ok := reconciliationRunIDFromRequest(w, r)
	if !ok {
		return
	}
	page, ok := paginationFromRequest(w, r)
	if !ok {
		return
	}
	filters, ok := reconciliationItemFiltersFromRequest(w, r)
	if !ok {
		return
	}

	items, err := h.service.ListTeamleadCurrentItems(r.Context(), reconciliation.GetTeamleadCurrentRunParams{
		TeamID:    actor.TeamID,
		ActorID:   actor.ID,
		Direction: direction,
		RunID:     runID,
	}, filters, page)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(reconciliation.PublicItemsFromDomain(items.Items), page, items.Total)))
}

func (h *TeamleadReconciliationHandler) accept(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
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

	run, err := h.service.AcceptTeamleadCurrent(r.Context(), reconciliation.AcceptTeamleadCurrentParams{
		ActorID:   actor.ID,
		TeamID:    actor.TeamID,
		Direction: direction,
		RunID:     runID,
		Comment:   request.Comment,
	})
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
	page, ok := paginationFromRequest(w, r)
	if !ok {
		return
	}
	filters, ok := reconciliationItemFiltersFromRequest(w, r)
	if !ok {
		return
	}

	var (
		items pagination.Result[reconciliation.Item]
		err   error
	)
	switch direction {
	case "inbound":
		items, err = h.service.ListTraderInboundItemsByShift(r.Context(), actor.TeamID, shiftID, filters, page)
	case "outbound":
		items, err = h.service.ListTraderOutboundItemsByShift(r.Context(), actor.TeamID, shiftID, filters, page)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(reconciliation.PublicItemsFromDomain(items.Items), page, items.Total)))
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
	page, ok := paginationFromRequest(w, r)
	if !ok {
		return
	}
	filters, ok := reconciliationItemFiltersFromRequest(w, r)
	if !ok {
		return
	}

	var (
		items pagination.Result[reconciliation.Item]
		err   error
	)
	switch direction {
	case "inbound":
		items, err = h.service.ListTeamleadPeriodInboundItems(r.Context(), actor.TeamID, periodID, filters, page)
	case "outbound":
		items, err = h.service.ListTeamleadPeriodOutboundItems(r.Context(), actor.TeamID, periodID, filters, page)
	default:
		RespondError(w, NotFoundError())
		return
	}
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(reconciliation.PublicItemsFromDomain(items.Items), page, items.Total)))
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

func limitFromQuery(r *http.Request, fallback int32) int32 {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 100 {
		return 100
	}

	return int32(parsed)
}

func reconciliationItemFiltersFromRequest(w http.ResponseWriter, r *http.Request) (reconciliation.ItemFilters, bool) {
	fields := map[string]string{}
	onlyMismatch, ok := optionalBool(r.URL.Query().Get("onlyMismatch"), "onlyMismatch", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return reconciliation.ItemFilters{}, false
	}

	return reconciliation.ItemFilters{
		Status:       r.URL.Query().Get("status"),
		OnlyMismatch: onlyMismatch,
	}, true
}
