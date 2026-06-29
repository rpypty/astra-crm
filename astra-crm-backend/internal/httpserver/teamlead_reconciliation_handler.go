package httpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/go-chi/chi/v5"
)

type TeamleadReconciliationService interface {
	CreateTeamleadReconciliation(ctx context.Context, params reconciliation.CreateTeamleadReconciliationParams) (reconciliation.TeamleadRun, error)
	ConfirmTeamleadReconciliation(ctx context.Context, params reconciliation.ConfirmTeamleadReconciliationParams) (reconciliation.TeamleadRun, error)
	RejectTeamleadReconciliation(ctx context.Context, params reconciliation.RejectTeamleadReconciliationParams) (reconciliation.TeamleadRun, error)
	ListTeamleadReconciliations(ctx context.Context, params reconciliation.ListTeamleadReconciliationsParams) (pagination.Result[reconciliation.TeamleadRun], error)
	GetTeamleadReconciliation(ctx context.Context, params reconciliation.GetTeamleadReconciliationParams) (reconciliation.TeamleadRun, error)
	ListTeamleadReconciliationItems(ctx context.Context, params reconciliation.GetTeamleadReconciliationParams, filters reconciliation.TeamleadItemFilters, page pagination.Params) (pagination.Result[reconciliation.TeamleadItem], error)
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

type teamleadReconciliationResponse struct {
	Reconciliation reconciliation.PublicTeamleadRun `json:"reconciliation"`
}

func (h *TeamleadReconciliationHandler) List(w http.ResponseWriter, r *http.Request) {
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

	runs, err := h.service.ListTeamleadReconciliations(r.Context(), reconciliation.ListTeamleadReconciliationsParams{
		TeamID: actor.TeamID,
		Page:   page,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}
	publicRuns := make([]reconciliation.PublicTeamleadRun, 0, len(runs.Items))
	for _, run := range runs.Items {
		publicRuns = append(publicRuns, reconciliation.PublicTeamleadRunFromDomain(run))
	}
	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(publicRuns, page, runs.Total)))
}

func (h *TeamleadReconciliationHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	run, err := h.service.GetTeamleadReconciliation(r.Context(), reconciliation.GetTeamleadReconciliationParams{
		TeamID: actor.TeamID,
		RunID:  runID,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}
	WriteJSON(w, http.StatusOK, teamleadReconciliationResponse{
		Reconciliation: reconciliation.PublicTeamleadRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) Items(w http.ResponseWriter, r *http.Request) {
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
	filters, ok := teamleadReconciliationItemFiltersFromRequest(w, r)
	if !ok {
		return
	}

	items, err := h.service.ListTeamleadReconciliationItems(r.Context(), reconciliation.GetTeamleadReconciliationParams{
		TeamID: actor.TeamID,
		RunID:  runID,
	}, filters, page)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}
	WriteJSON(w, http.StatusOK, paginated(pagination.NewResult(reconciliation.PublicTeamleadItemsFromDomain(items.Items), page, items.Total)))
}

func (h *TeamleadReconciliationHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	params, ok := createTeamleadReconciliationParamsFromRequest(w, r)
	if !ok {
		return
	}
	params.ActorID = actor.ID
	params.TeamID = actor.TeamID

	run, err := h.service.CreateTeamleadReconciliation(r.Context(), params)
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, teamleadReconciliationResponse{
		Reconciliation: reconciliation.PublicTeamleadRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) Confirm(w http.ResponseWriter, r *http.Request) {
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
	run, err := h.service.ConfirmTeamleadReconciliation(r.Context(), reconciliation.ConfirmTeamleadReconciliationParams{
		ActorID: actor.ID,
		TeamID:  actor.TeamID,
		RunID:   runID,
		Comment: request.Comment,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, teamleadReconciliationResponse{
		Reconciliation: reconciliation.PublicTeamleadRunFromDomain(run),
	})
}

func (h *TeamleadReconciliationHandler) Reject(w http.ResponseWriter, r *http.Request) {
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
	run, err := h.service.RejectTeamleadReconciliation(r.Context(), reconciliation.RejectTeamleadReconciliationParams{
		ActorID: actor.ID,
		TeamID:  actor.TeamID,
		RunID:   runID,
		Comment: request.Comment,
	})
	if err != nil {
		RespondError(w, mapReconciliationError(err))
		return
	}

	WriteJSON(w, http.StatusOK, teamleadReconciliationResponse{
		Reconciliation: reconciliation.PublicTeamleadRunFromDomain(run),
	})
}

func createTeamleadReconciliationParamsFromRequest(w http.ResponseWriter, r *http.Request) (reconciliation.CreateTeamleadReconciliationParams, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportFileSize*2)
	if err := r.ParseMultipartForm(maxImportFileSize * 2); err != nil {
		RespondError(w, ValidationError(map[string]string{
			"body": "Ожидается multipart/form-data с периодом и CSV файлами",
		}))
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}

	fields := map[string]string{}
	dateFrom, ok := parseRequiredDateField(r.FormValue("dateFrom"), "dateFrom", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}
	dateTo, ok := parseRequiredDateField(r.FormValue("dateTo"), "dateTo", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}
	if dateTo.Before(dateFrom) {
		fields["dateTo"] = "Дата окончания не может быть раньше даты начала"
		RespondError(w, ValidationError(fields))
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}

	inbound, inboundOK, ok := optionalTeamleadCSVInputFromRequest(w, r, "inboundFile")
	if !ok {
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}
	outbound, outboundOK, ok := optionalTeamleadCSVInputFromRequest(w, r, "outboundFile")
	if !ok {
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}
	if !inboundOK && !outboundOK {
		RespondError(w, ValidationError(map[string]string{
			"file": "Нужно загрузить CSV входов, CSV выходов или оба файла",
		}))
		return reconciliation.CreateTeamleadReconciliationParams{}, false
	}

	params := reconciliation.CreateTeamleadReconciliationParams{
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}
	if inboundOK {
		params.Inbound = &inbound
	}
	if outboundOK {
		params.Outbound = &outbound
	}
	return params, true
}

func parseRequiredDateField(value string, field string, fields map[string]string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		fields[field] = "Дата обязательна"
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		fields[field] = "Дата должна быть в формате YYYY-MM-DD"
		return time.Time{}, false
	}
	return parsed, true
}

func optionalTeamleadCSVInputFromRequest(w http.ResponseWriter, r *http.Request, field string) (reconciliation.TeamleadCSVInput, bool, bool) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return reconciliation.TeamleadCSVInput{}, false, true
		}
		RespondError(w, ValidationError(map[string]string{
			field: "CSV файл не удалось прочитать",
		}))
		return reconciliation.TeamleadCSVInput{}, false, false
	}
	defer file.Close()

	payload, err := io.ReadAll(file)
	if err != nil {
		RespondError(w, ValidationError(map[string]string{
			field: "CSV файл не удалось прочитать",
		}))
		return reconciliation.TeamleadCSVInput{}, false, false
	}
	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		fileName = field + ".csv"
	}
	return reconciliation.TeamleadCSVInput{FileName: fileName, Payload: payload}, true, true
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

func teamleadReconciliationItemFiltersFromRequest(w http.ResponseWriter, r *http.Request) (reconciliation.TeamleadItemFilters, bool) {
	fields := map[string]string{}
	query := r.URL.Query()
	onlyMismatch, ok := optionalBool(query.Get("onlyMismatch"), "onlyMismatch", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return reconciliation.TeamleadItemFilters{}, false
	}
	traderID, ok := optionalInt64(query.Get("traderId"), "traderId", fields, false)
	if !ok {
		RespondError(w, ValidationError(fields))
		return reconciliation.TeamleadItemFilters{}, false
	}
	requisiteID, ok := optionalInt64(query.Get("requisiteId"), "requisiteId", fields, false)
	if !ok {
		RespondError(w, ValidationError(fields))
		return reconciliation.TeamleadItemFilters{}, false
	}

	return reconciliation.TeamleadItemFilters{
		Direction:    strings.TrimSpace(query.Get("direction")),
		Stage:        strings.TrimSpace(query.Get("stage")),
		IssueType:    strings.TrimSpace(query.Get("issueType")),
		Severity:     strings.TrimSpace(query.Get("severity")),
		TraderID:     traderID,
		RequisiteID:  requisiteID,
		OnlyMismatch: onlyMismatch,
	}, true
}
