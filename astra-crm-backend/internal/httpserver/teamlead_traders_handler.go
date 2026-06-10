package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
)

type TeamleadTraderService interface {
	Create(ctx context.Context, params users.CreateTraderParams) (users.Trader, error)
	List(ctx context.Context, teamID int64) ([]users.Trader, error)
	Get(ctx context.Context, teamID int64, traderID int64) (users.Trader, error)
	Patch(ctx context.Context, params users.PatchTraderParams) (users.Trader, error)
	Delete(ctx context.Context, actorID int64, teamID int64, traderID int64) error
	ResetPassword(ctx context.Context, params users.ResetTraderPasswordParams) (users.ResetTraderPasswordResult, error)
}

type TeamleadTradersHandler struct {
	service TeamleadTraderService
}

func NewTeamleadTradersHandler(service TeamleadTraderService) *TeamleadTradersHandler {
	return &TeamleadTradersHandler{service: service}
}

type tradersListResponse struct {
	Items []users.PublicTrader `json:"items"`
}

type traderResponse struct {
	Trader users.PublicTrader `json:"trader"`
}

type resetTraderPasswordResponse struct {
	Trader            users.PublicTrader `json:"trader"`
	TemporaryPassword string             `json:"temporaryPassword"`
}

type createTraderRequest struct {
	Login              string `json:"login"`
	Password           string `json:"password"`
	SalaryRateBps      int64  `json:"salaryRateBps"`
	ExternalWorkerName string `json:"externalWorkerName"`
}

type patchTraderRequest struct {
	Status             *string `json:"status"`
	SalaryRateBps      *int64  `json:"salaryRateBps"`
	ExternalWorkerName *string `json:"externalWorkerName"`
}

func (h *TeamleadTradersHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	traders, err := h.service.List(r.Context(), actor.TeamID)
	if err != nil {
		RespondError(w, mapTraderError(err))
		return
	}

	WriteJSON(w, http.StatusOK, tradersListResponse{
		Items: users.ToPublicTraders(traders),
	})
}

func (h *TeamleadTradersHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request createTraderRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateCreateTraderRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	trader, err := h.service.Create(r.Context(), users.CreateTraderParams{
		ActorID:            actor.ID,
		TeamID:             actor.TeamID,
		Login:              request.Login,
		Password:           request.Password,
		SalaryRateBps:      request.SalaryRateBps,
		ExternalWorkerName: request.ExternalWorkerName,
	})
	if err != nil {
		RespondError(w, mapTraderError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, traderResponse{
		Trader: users.ToPublicTrader(trader),
	})
}

func (h *TeamleadTradersHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	traderID, ok := traderIDFromRequest(w, r)
	if !ok {
		return
	}

	trader, err := h.service.Get(r.Context(), actor.TeamID, traderID)
	if err != nil {
		RespondError(w, mapTraderError(err))
		return
	}

	WriteJSON(w, http.StatusOK, traderResponse{
		Trader: users.ToPublicTrader(trader),
	})
}

func (h *TeamleadTradersHandler) Patch(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	traderID, ok := traderIDFromRequest(w, r)
	if !ok {
		return
	}

	var request patchTraderRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validatePatchTraderRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	trader, err := h.service.Patch(r.Context(), users.PatchTraderParams{
		ActorID:            actor.ID,
		TeamID:             actor.TeamID,
		TraderID:           traderID,
		Status:             request.Status,
		SalaryRateBps:      request.SalaryRateBps,
		ExternalWorkerName: request.ExternalWorkerName,
	})
	if err != nil {
		RespondError(w, mapTraderError(err))
		return
	}

	WriteJSON(w, http.StatusOK, traderResponse{
		Trader: users.ToPublicTrader(trader),
	})
}

func (h *TeamleadTradersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	traderID, ok := traderIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), actor.ID, actor.TeamID, traderID); err != nil {
		RespondError(w, mapTraderError(err))
		return
	}

	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *TeamleadTradersHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	traderID, ok := traderIDFromRequest(w, r)
	if !ok {
		return
	}

	result, err := h.service.ResetPassword(r.Context(), users.ResetTraderPasswordParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		TraderID: traderID,
	})
	if err != nil {
		RespondError(w, mapTraderError(err))
		return
	}

	WriteJSON(w, http.StatusOK, resetTraderPasswordResponse{
		Trader:            users.ToPublicTrader(result.Trader),
		TemporaryPassword: result.TemporaryPassword,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		RespondError(w, ValidationError(map[string]string{
			"body": "Некорректный JSON",
		}))
		return false
	}

	return true
}

func traderIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "traderId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"traderId": "Некорректный ID трейдера",
		}))
		return 0, false
	}

	return id, true
}

func validateCreateTraderRequest(request createTraderRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(request.Login) == "" {
		fields["login"] = "Логин обязателен"
	}
	if request.Password == "" {
		fields["password"] = "Пароль обязателен"
	}
	if request.SalaryRateBps < 0 {
		fields["salaryRateBps"] = "Процент ЗП не может быть отрицательным"
	}
	if strings.TrimSpace(request.ExternalWorkerName) == "" {
		fields["externalWorkerName"] = "Worker name обязателен"
	}

	return fields
}

func validatePatchTraderRequest(request patchTraderRequest) map[string]string {
	fields := map[string]string{}
	if request.Status == nil && request.SalaryRateBps == nil && request.ExternalWorkerName == nil {
		fields["body"] = "Нужно передать хотя бы одно поле для изменения"
	}
	if request.Status != nil && !validPatchStatus(*request.Status) {
		fields["status"] = "Некорректный статус трейдера"
	}
	if request.SalaryRateBps != nil && *request.SalaryRateBps < 0 {
		fields["salaryRateBps"] = "Процент ЗП не может быть отрицательным"
	}
	if request.ExternalWorkerName != nil && strings.TrimSpace(*request.ExternalWorkerName) == "" {
		fields["externalWorkerName"] = "Worker name обязателен"
	}

	return fields
}

func validPatchStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case users.StatusActive, users.StatusDisabled, users.StatusDeleted:
		return true
	default:
		return false
	}
}

func mapTraderError(err error) error {
	switch {
	case errors.Is(err, users.ErrTraderNotFound):
		return NotFoundError()
	case errors.Is(err, users.ErrDuplicateLogin):
		return DomainError("TRADER_LOGIN_EXISTS", "Трейдер с таким логином уже существует")
	case errors.Is(err, users.ErrDuplicateWorkerName):
		return DomainError("TRADER_WORKER_NAME_EXISTS", "Трейдер с таким workerName уже существует")
	case errors.Is(err, users.ErrInvalidTraderInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		})
	default:
		return err
	}
}
