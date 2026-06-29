package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
)

type paginatedResponse[T any] struct {
	Items    []T   `json:"items"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
	Total    int64 `json:"total"`
}

func paginated[T any](result pagination.Result[T]) paginatedResponse[T] {
	return paginatedResponse[T]{
		Items:    result.Items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	}
}

func paginationFromRequest(w http.ResponseWriter, r *http.Request) (pagination.Params, bool) {
	fields := map[string]string{}
	page, ok := paginationQueryInt(r.URL.Query().Get("page"), "page", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return pagination.Params{}, false
	}
	pageSize, ok := paginationQueryInt(r.URL.Query().Get("pageSize"), "pageSize", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return pagination.Params{}, false
	}

	return pagination.Normalize(pagination.Params{
		Page:     page,
		PageSize: pageSize,
	}), true
}

func paginationQueryInt(raw string, field string, fields map[string]string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, true
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		fields[field] = "Ожидается положительное число"
		return 0, false
	}

	return value, true
}
