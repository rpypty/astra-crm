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
	TraderProfile(ctx context.Context, teamID int64, traderID int64, filters readmodels.PeriodFilter) (readmodels.TraderProfile, error)
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

type traderProfileResponse struct {
	Profile publicTraderProfile `json:"profile"`
}

type publicAccountingPeriod struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	DateFrom       string `json:"dateFrom"`
	DateTo         string `json:"dateTo"`
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

type publicTraderProfile struct {
	ID                              int64   `json:"id"`
	Login                           string  `json:"login"`
	SalaryRateBps                   int64   `json:"salaryRateBps"`
	ExternalWorkerName              string  `json:"externalWorkerName"`
	CurrentShiftSuccessInboundMinor int64   `json:"currentShiftSuccessInboundMinor"`
	CurrentShiftSalaryMinor         int64   `json:"currentShiftSalaryMinor"`
	PeriodID                        *int64  `json:"periodId,omitempty"`
	PeriodTitle                     *string `json:"periodTitle,omitempty"`
	PeriodSuccessInboundMinor       int64   `json:"periodSuccessInboundMinor"`
	PeriodSalaryMinor               int64   `json:"periodSalaryMinor"`
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

func (h *TeamleadReadmodelHandler) TraderProfile(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	fields := map[string]string{}
	dateFrom, ok := optionalDate(r.URL.Query().Get("dateFrom"), "dateFrom", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return
	}
	dateTo, ok := optionalDate(r.URL.Query().Get("dateTo"), "dateTo", fields)
	if !ok {
		RespondError(w, ValidationError(fields))
		return
	}
	if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
		RespondError(w, ValidationError(map[string]string{
			"dateTo": "Дата окончания должна быть не раньше даты начала",
		}))
		return
	}

	profile, err := h.service.TraderProfile(r.Context(), actor.TeamID, actor.ID, readmodels.PeriodFilter{
		DateFrom: dateFrom,
		DateTo:   dateTo,
	})
	if err != nil {
		RespondError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, traderProfileResponse{Profile: publicTraderProfileFromDomain(profile)})
}

func publicPeriods(items []readmodels.AccountingPeriod) []publicAccountingPeriod {
	result := make([]publicAccountingPeriod, 0, len(items))
	for _, item := range items {
		result = append(result, publicAccountingPeriod{
			ID:             item.ID,
			Title:          item.Title,
			DateFrom:       item.DateFrom.Format("2006-01-02"),
			DateTo:         item.DateTo.Format("2006-01-02"),
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

func publicTraderProfileFromDomain(profile readmodels.TraderProfile) publicTraderProfile {
	return publicTraderProfile{
		ID:                              profile.ID,
		Login:                           profile.Login,
		SalaryRateBps:                   profile.SalaryRateBps,
		ExternalWorkerName:              profile.ExternalWorkerName,
		CurrentShiftSuccessInboundMinor: profile.CurrentShiftSuccessInboundMinor,
		CurrentShiftSalaryMinor:         profile.CurrentShiftSalaryMinor,
		PeriodID:                        profile.PeriodID,
		PeriodTitle:                     profile.PeriodTitle,
		PeriodSuccessInboundMinor:       profile.PeriodSuccessInboundMinor,
		PeriodSalaryMinor:               profile.PeriodSalaryMinor,
	}
}
