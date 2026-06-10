package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/requisites"
	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

type TeamleadRequisiteService interface {
	Create(ctx context.Context, params requisites.CreateParams) (requisites.RequisiteDetails, error)
	List(ctx context.Context, teamID int64) ([]requisites.RequisiteDetails, error)
	Get(ctx context.Context, teamID int64, requisiteID int64) (requisites.RequisiteDetails, error)
	Patch(ctx context.Context, params requisites.PatchParams) (requisites.RequisiteDetails, error)
	Delete(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error
	Assign(ctx context.Context, params requisites.AssignParams) (requisites.Assignment, error)
	Unassign(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error
	AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64) ([]requisites.Assignment, error)
}

type TeamleadRequisitesHandler struct {
	service TeamleadRequisiteService
}

func NewTeamleadRequisitesHandler(service TeamleadRequisiteService) *TeamleadRequisitesHandler {
	return &TeamleadRequisitesHandler{service: service}
}

type requisitesListResponse struct {
	Items []requisites.PublicRequisite `json:"items"`
}

type requisiteResponse struct {
	Requisite requisites.PublicRequisite `json:"requisite"`
}

type assignmentResponse struct {
	Assignment requisites.PublicRequisiteAssignment `json:"assignment"`
}

type assignmentHistoryResponse struct {
	Items []requisites.PublicRequisiteAssignment `json:"items"`
}

type createRequisiteRequest struct {
	Phone            string  `json:"phone"`
	MethodType       string  `json:"methodType"`
	Proxy            *string `json:"proxy"`
	AssignedTraderID *int64  `json:"assignedTraderId"`
}

type patchRequisiteRequest struct {
	Phone      *string `json:"phone"`
	MethodType *string `json:"methodType"`
	Proxy      *string `json:"proxy"`
	Status     *string `json:"status"`
}

type assignRequisiteRequest struct {
	TraderID int64   `json:"traderId"`
	Comment  *string `json:"comment"`
}

func (h *TeamleadRequisitesHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.List(r.Context(), actor.TeamID)
	if err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, requisitesListResponse{
		Items: requisites.PublicRequisites(items),
	})
}

func (h *TeamleadRequisitesHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request createRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateCreateRequisiteRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	requisite, err := h.service.Create(r.Context(), requisites.CreateParams{
		ActorID:          actor.ID,
		TeamID:           actor.TeamID,
		Phone:            request.Phone,
		MethodType:       request.MethodType,
		Proxy:            request.Proxy,
		AssignedTraderID: request.AssignedTraderID,
	})
	if err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, requisiteResponse{
		Requisite: requisites.PublicRequisiteFromDetails(requisite),
	})
}

func (h *TeamleadRequisitesHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := requisiteIDFromRequest(w, r)
	if !ok {
		return
	}

	requisite, err := h.service.Get(r.Context(), actor.TeamID, requisiteID)
	if err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, requisiteResponse{
		Requisite: requisites.PublicRequisiteFromDetails(requisite),
	})
}

func (h *TeamleadRequisitesHandler) Patch(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := requisiteIDFromRequest(w, r)
	if !ok {
		return
	}

	var request patchRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validatePatchRequisiteRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	requisite, err := h.service.Patch(r.Context(), requisites.PatchParams{
		ActorID:     actor.ID,
		TeamID:      actor.TeamID,
		RequisiteID: requisiteID,
		Phone:       request.Phone,
		MethodType:  request.MethodType,
		Proxy:       request.Proxy,
		Status:      request.Status,
	})
	if err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, requisiteResponse{
		Requisite: requisites.PublicRequisiteFromDetails(requisite),
	})
}

func (h *TeamleadRequisitesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := requisiteIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), actor.ID, actor.TeamID, requisiteID); err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *TeamleadRequisitesHandler) Assign(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := requisiteIDFromRequest(w, r)
	if !ok {
		return
	}

	var request assignRequisiteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateAssignRequisiteRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	assignment, err := h.service.Assign(r.Context(), requisites.AssignParams{
		ActorID:     actor.ID,
		TeamID:      actor.TeamID,
		RequisiteID: requisiteID,
		TraderID:    request.TraderID,
		Comment:     request.Comment,
	})
	if err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignmentResponse{
		Assignment: requisites.PublicAssignment(assignment),
	})
}

func (h *TeamleadRequisitesHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := requisiteIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Unassign(r.Context(), actor.ID, actor.TeamID, requisiteID); err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *TeamleadRequisitesHandler) AssignmentHistory(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	requisiteID, ok := requisiteIDFromRequest(w, r)
	if !ok {
		return
	}

	items, err := h.service.AssignmentHistory(r.Context(), actor.TeamID, requisiteID)
	if err != nil {
		RespondError(w, mapRequisiteError(err))
		return
	}

	WriteJSON(w, http.StatusOK, assignmentHistoryResponse{
		Items: requisites.PublicAssignments(items),
	})
}

func requisiteIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "requisiteId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"requisiteId": "Некорректный ID реквизита",
		}))
		return 0, false
	}

	return id, true
}

func validateCreateRequisiteRequest(request createRequisiteRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(request.Phone) == "" {
		fields["phone"] = "Телефон обязателен"
	}
	if strings.TrimSpace(request.MethodType) == "" {
		fields["methodType"] = "Метод обязателен"
	}
	if request.AssignedTraderID != nil && *request.AssignedTraderID <= 0 {
		fields["assignedTraderId"] = "Некорректный ID трейдера"
	}

	return fields
}

func validatePatchRequisiteRequest(request patchRequisiteRequest) map[string]string {
	fields := map[string]string{}
	if request.Phone == nil && request.MethodType == nil && request.Proxy == nil && request.Status == nil {
		fields["body"] = "Нужно передать хотя бы одно поле для изменения"
	}
	if request.Phone != nil && strings.TrimSpace(*request.Phone) == "" {
		fields["phone"] = "Телефон обязателен"
	}
	if request.MethodType != nil && strings.TrimSpace(*request.MethodType) == "" {
		fields["methodType"] = "Метод обязателен"
	}
	if request.Status != nil && !validRequisitePatchStatus(*request.Status) {
		fields["status"] = "Некорректный статус реквизита"
	}

	return fields
}

func validateAssignRequisiteRequest(request assignRequisiteRequest) map[string]string {
	fields := map[string]string{}
	if request.TraderID <= 0 {
		fields["traderId"] = "Некорректный ID трейдера"
	}

	return fields
}

func validRequisitePatchStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case requisites.StatusActive, requisites.StatusDisabled, requisites.StatusArchived:
		return true
	default:
		return false
	}
}

func mapRequisiteError(err error) error {
	switch {
	case errors.Is(err, requisites.ErrNotFound):
		return NotFoundError()
	case errors.Is(err, requisites.ErrAssignmentNotFound):
		return NotFoundError()
	case errors.Is(err, users.ErrTraderNotFound):
		return NotFoundError()
	case errors.Is(err, requisites.ErrInactiveTrader):
		return DomainError("TRADER_NOT_ACTIVE", "Назначить реквизит можно только активному трейдеру")
	case errors.Is(err, requisites.ErrInvalidInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		})
	default:
		return err
	}
}
