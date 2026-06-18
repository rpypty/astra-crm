package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/orders"
)

type OrderReadService interface {
	ListTraderOrders(ctx context.Context, teamID int64, traderID int64, direction string, filters orders.Filters) (orders.ListResult, error)
	ListTeamleadOrders(ctx context.Context, teamID int64, direction string, filters orders.Filters) (orders.ListResult, error)
	TraderDashboard(ctx context.Context, teamID int64, traderID int64, direction string, filters orders.Filters) (orders.Dashboard, error)
	TeamleadDashboard(ctx context.Context, teamID int64, direction string, filters orders.Filters) (orders.Dashboard, error)
}

type OrderReadHandler struct {
	service OrderReadService
}

func NewOrderReadHandler(service OrderReadService) *OrderReadHandler {
	return &OrderReadHandler{service: service}
}

type orderListResponse struct {
	Items    []orders.PublicOrder `json:"items"`
	Page     int64                `json:"page"`
	PageSize int64                `json:"pageSize"`
	Total    int64                `json:"total"`
}

type orderDashboardResponse struct {
	Dashboard orders.PublicDashboard `json:"dashboard"`
}

func (h *OrderReadHandler) TraderInboundDashboard(w http.ResponseWriter, r *http.Request) {
	h.traderDashboard(w, r, orders.DirectionInbound)
}

func (h *OrderReadHandler) TraderInboundOrders(w http.ResponseWriter, r *http.Request) {
	h.traderOrders(w, r, orders.DirectionInbound)
}

func (h *OrderReadHandler) TraderOutboundDashboard(w http.ResponseWriter, r *http.Request) {
	h.traderDashboard(w, r, orders.DirectionOutbound)
}

func (h *OrderReadHandler) TraderOutboundOrders(w http.ResponseWriter, r *http.Request) {
	h.traderOrders(w, r, orders.DirectionOutbound)
}

func (h *OrderReadHandler) TeamleadInboundDashboard(w http.ResponseWriter, r *http.Request) {
	h.teamleadDashboard(w, r, orders.DirectionInbound)
}

func (h *OrderReadHandler) TeamleadInboundOrders(w http.ResponseWriter, r *http.Request) {
	h.teamleadOrders(w, r, orders.DirectionInbound)
}

func (h *OrderReadHandler) TeamleadOutboundDashboard(w http.ResponseWriter, r *http.Request) {
	h.teamleadDashboard(w, r, orders.DirectionOutbound)
}

func (h *OrderReadHandler) TeamleadOutboundOrders(w http.ResponseWriter, r *http.Request) {
	h.teamleadOrders(w, r, orders.DirectionOutbound)
}

func (h *OrderReadHandler) traderDashboard(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	filters, ok := orderFiltersFromRequest(w, r)
	if !ok {
		return
	}

	dashboard, err := h.service.TraderDashboard(r.Context(), actor.TeamID, actor.ID, direction, filters)
	if err != nil {
		RespondError(w, mapOrderReadError(err))
		return
	}

	WriteJSON(w, http.StatusOK, orderDashboardResponse{
		Dashboard: orders.PublicDashboardFromDomain(dashboard),
	})
}

func (h *OrderReadHandler) traderOrders(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	filters, ok := orderFiltersFromRequest(w, r)
	if !ok {
		return
	}

	result, err := h.service.ListTraderOrders(r.Context(), actor.TeamID, actor.ID, direction, filters)
	if err != nil {
		RespondError(w, mapOrderReadError(err))
		return
	}

	publicResult := orders.PublicListFromDomain(result)
	WriteJSON(w, http.StatusOK, orderListResponse{
		Items:    publicResult.Items,
		Page:     publicResult.Page,
		PageSize: publicResult.PageSize,
		Total:    publicResult.Total,
	})
}

func (h *OrderReadHandler) teamleadDashboard(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	filters, ok := orderFiltersFromRequest(w, r)
	if !ok {
		return
	}

	dashboard, err := h.service.TeamleadDashboard(r.Context(), actor.TeamID, direction, filters)
	if err != nil {
		RespondError(w, mapOrderReadError(err))
		return
	}

	WriteJSON(w, http.StatusOK, orderDashboardResponse{
		Dashboard: orders.PublicDashboardFromDomain(dashboard),
	})
}

func (h *OrderReadHandler) teamleadOrders(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	filters, ok := orderFiltersFromRequest(w, r)
	if !ok {
		return
	}

	result, err := h.service.ListTeamleadOrders(r.Context(), actor.TeamID, direction, filters)
	if err != nil {
		RespondError(w, mapOrderReadError(err))
		return
	}

	publicResult := orders.PublicListFromDomain(result)
	WriteJSON(w, http.StatusOK, orderListResponse{
		Items:    publicResult.Items,
		Page:     publicResult.Page,
		PageSize: publicResult.PageSize,
		Total:    publicResult.Total,
	})
}

func orderFiltersFromRequest(w http.ResponseWriter, r *http.Request) (orders.Filters, bool) {
	query := r.URL.Query()
	fields := map[string]string{}

	dateFrom, ok := optionalDate(query.Get("dateFrom"), "dateFrom", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	dateTo, ok := optionalDate(query.Get("dateTo"), "dateTo", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	traderID, ok := optionalInt64(query.Get("traderId"), "traderId", fields, false)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	traderIDs, ok := optionalInt64List(query["traderIds"], "traderIds", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	amountFrom, ok := optionalInt64(query.Get("amountFrom"), "amountFrom", fields, true)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	amountTo, ok := optionalInt64(query.Get("amountTo"), "amountTo", fields, true)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	confirmedOnly, ok := optionalBool(query.Get("confirmedOnly"), "confirmedOnly", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	page, ok := optionalPositiveInt64(query.Get("page"), "page", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}
	pageSize, ok := optionalPositiveInt64(query.Get("pageSize"), "pageSize", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return orders.Filters{}, false
	}

	return orders.Filters{
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		TraderID:      traderID,
		TraderIDs:     traderIDs,
		WorkerName:    optionalCleanString(query.Get("workerName")),
		Requisite:     optionalCleanString(query.Get("requisite")),
		MethodType:    optionalCleanString(query.Get("methodType")),
		Status:        optionalCleanString(query.Get("status")),
		AmountFrom:    amountFrom,
		AmountTo:      amountTo,
		ConfirmedOnly: confirmedOnly,
		Page:          page,
		PageSize:      pageSize,
		Sort:          strings.TrimSpace(query.Get("sort")),
	}, true
}

func optionalBool(raw string, field string, fields map[string]string) (bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, true
	}
	switch raw {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		fields[field] = "Ожидается true или false"
		return false, false
	}
}

func optionalInt64List(rawValues []string, field string, fields map[string]string) ([]int64, bool) {
	if len(rawValues) == 0 {
		return nil, true
	}

	values := make([]int64, 0)
	seen := map[int64]bool{}
	for _, rawValue := range rawValues {
		for _, rawPart := range strings.Split(rawValue, ",") {
			rawPart = strings.TrimSpace(rawPart)
			if rawPart == "" || rawPart == "all" {
				continue
			}
			value, err := strconv.ParseInt(rawPart, 10, 64)
			if err != nil || value <= 0 {
				fields[field] = "Некорректный список числовых значений"
				return nil, false
			}
			if seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}

	return values, true
}

func optionalDate(raw string, field string, fields map[string]string) (*time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}

	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		fields[field] = "Дата должна быть в формате YYYY-MM-DD"
		return nil, false
	}

	return &value, true
}

func optionalInt64(raw string, field string, fields map[string]string, allowZero bool) (*int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 || (!allowZero && value == 0) {
		fields[field] = "Некорректное числовое значение"
		return nil, false
	}

	return &value, true
}

func optionalPositiveInt64(raw string, field string, fields map[string]string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, true
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		fields[field] = "Некорректное положительное число"
		return 0, false
	}

	return value, true
}

func optionalCleanString(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	return &value
}

func mapOrderReadError(err error) error {
	switch {
	case errors.Is(err, orders.ErrInvalidInput):
		return ValidationError(map[string]string{
			"query": "Некоторые параметры фильтра заполнены неверно",
		})
	default:
		return err
	}
}
