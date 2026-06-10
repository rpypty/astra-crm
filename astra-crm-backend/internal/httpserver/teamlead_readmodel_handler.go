package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/readmodels"
)

type TeamleadReadmodelService interface {
	ListPeriods(ctx context.Context, teamID int64) ([]readmodels.AccountingPeriod, error)
	ListAudit(ctx context.Context, teamID int64) ([]readmodels.AuditLogEntry, error)
}

type TeamleadReadmodelHandler struct {
	service TeamleadReadmodelService
}

func NewTeamleadReadmodelHandler(service TeamleadReadmodelService) *TeamleadReadmodelHandler {
	return &TeamleadReadmodelHandler{service: service}
}

type periodsResponse struct {
	Items []publicAccountingPeriod `json:"items"`
}

type auditResponse struct {
	Items []publicAuditLogEntry `json:"items"`
}

type publicAccountingPeriod struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	DateRange      string `json:"dateRange"`
	InboundStatus  string `json:"inboundStatus"`
	OutboundStatus string `json:"outboundStatus"`
	Status         string `json:"status"`
}

type publicAuditLogEntry struct {
	ID            int64          `json:"id"`
	CreatedAt     time.Time      `json:"createdAt"`
	ActorLogin    string         `json:"actorLogin"`
	Action        string         `json:"action"`
	EntityType    string         `json:"entityType"`
	EntityID      string         `json:"entityId"`
	Comment       *string        `json:"comment,omitempty"`
	MaskedPayload map[string]any `json:"maskedPayload"`
}

func (h *TeamleadReadmodelHandler) Periods(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.ListPeriods(r.Context(), actor.TeamID)
	if err != nil {
		RespondError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, periodsResponse{Items: publicPeriods(items)})
}

func (h *TeamleadReadmodelHandler) Audit(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.ListAudit(r.Context(), actor.TeamID)
	if err != nil {
		RespondError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, auditResponse{Items: publicAudit(items)})
}

func publicPeriods(items []readmodels.AccountingPeriod) []publicAccountingPeriod {
	result := make([]publicAccountingPeriod, 0, len(items))
	for _, item := range items {
		result = append(result, publicAccountingPeriod{
			ID:             item.ID,
			Title:          item.Title,
			DateRange:      item.DateRange,
			InboundStatus:  item.InboundStatus,
			OutboundStatus: item.OutboundStatus,
			Status:         item.Status,
		})
	}
	return result
}

func publicAudit(items []readmodels.AuditLogEntry) []publicAuditLogEntry {
	result := make([]publicAuditLogEntry, 0, len(items))
	for _, item := range items {
		result = append(result, publicAuditLogEntry{
			ID:            item.ID,
			CreatedAt:     item.CreatedAt,
			ActorLogin:    item.ActorLogin,
			Action:        item.Action,
			EntityType:    item.EntityType,
			EntityID:      item.EntityID,
			Comment:       item.Comment,
			MaskedPayload: item.MaskedPayload,
		})
	}
	return result
}
