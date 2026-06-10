package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/payouts"
	"github.com/go-chi/chi/v5"
)

type TraderPayoutService interface {
	ListOrders(ctx context.Context, teamID int64, traderID int64) ([]payouts.Order, error)
	GetOrder(ctx context.Context, teamID int64, traderID int64, payoutID int64) (payouts.Order, []payouts.Transfer, error)
	CreateOrder(ctx context.Context, params payouts.CreateOrderParams) (payouts.Order, error)
	PatchOrder(ctx context.Context, params payouts.PatchOrderParams) (payouts.Order, error)
	CancelOrder(ctx context.Context, actorID int64, teamID int64, traderID int64, payoutID int64) error
	AddTransfer(ctx context.Context, params payouts.AddTransferParams) (payouts.Transfer, error)
	DeleteTransfer(ctx context.Context, actorID int64, teamID int64, traderID int64, payoutID int64, transferID int64) error
}

type TraderPayoutHandler struct {
	service TraderPayoutService
}

func NewTraderPayoutHandler(service TraderPayoutService) *TraderPayoutHandler {
	return &TraderPayoutHandler{service: service}
}

type payoutOrdersResponse struct {
	Items []payouts.PublicOrder `json:"items"`
}

type payoutOrderResponse struct {
	Payout payouts.PublicOrder `json:"payout"`
}

type payoutDetailsResponse struct {
	Payout    payouts.PublicOrder      `json:"payout"`
	Transfers []payouts.PublicTransfer `json:"transfers"`
}

type payoutTransferResponse struct {
	Transfer payouts.PublicTransfer `json:"transfer"`
}

type createPayoutOrderRequest struct {
	DestinationBank      string `json:"destinationBank"`
	DestinationRequisite string `json:"destinationRequisite"`
	AmountMinor          int64  `json:"amountMinor"`
}

type patchPayoutOrderRequest struct {
	DestinationBank      *string `json:"destinationBank"`
	DestinationRequisite *string `json:"destinationRequisite"`
	AmountMinor          *int64  `json:"amountMinor"`
}

type addPayoutTransferRequest struct {
	SourceShiftRequisiteID int64   `json:"sourceShiftRequisiteId"`
	AmountMinor            int64   `json:"amountMinor"`
	Comment                *string `json:"comment"`
}

func (h *TraderPayoutHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.ListOrders(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusOK, payoutOrdersResponse{
		Items: payouts.PublicOrders(items),
	})
}

func (h *TraderPayoutHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	payoutID, ok := payoutIDFromRequest(w, r)
	if !ok {
		return
	}

	order, transfers, err := h.service.GetOrder(r.Context(), actor.TeamID, actor.ID, payoutID)
	if err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusOK, payoutDetailsResponse{
		Payout:    payouts.PublicOrderFromDomain(order),
		Transfers: payouts.PublicTransfers(transfers),
	})
}

func (h *TraderPayoutHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	var request createPayoutOrderRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateCreatePayoutOrderRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	order, err := h.service.CreateOrder(r.Context(), payouts.CreateOrderParams{
		ActorID:              actor.ID,
		TeamID:               actor.TeamID,
		TraderID:             actor.ID,
		DestinationBank:      request.DestinationBank,
		DestinationRequisite: request.DestinationRequisite,
		AmountMinor:          request.AmountMinor,
	})
	if err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, payoutOrderResponse{
		Payout: payouts.PublicOrderFromDomain(order),
	})
}

func (h *TraderPayoutHandler) Patch(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	payoutID, ok := payoutIDFromRequest(w, r)
	if !ok {
		return
	}

	var request patchPayoutOrderRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validatePatchPayoutOrderRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	order, err := h.service.PatchOrder(r.Context(), payouts.PatchOrderParams{
		ActorID:              actor.ID,
		TeamID:               actor.TeamID,
		TraderID:             actor.ID,
		PayoutID:             payoutID,
		DestinationBank:      request.DestinationBank,
		DestinationRequisite: request.DestinationRequisite,
		AmountMinor:          request.AmountMinor,
	})
	if err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusOK, payoutOrderResponse{
		Payout: payouts.PublicOrderFromDomain(order),
	})
}

func (h *TraderPayoutHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	payoutID, ok := payoutIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.CancelOrder(r.Context(), actor.ID, actor.TeamID, actor.ID, payoutID); err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *TraderPayoutHandler) AddTransfer(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	payoutID, ok := payoutIDFromRequest(w, r)
	if !ok {
		return
	}

	var request addPayoutTransferRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if fields := validateAddPayoutTransferRequest(request); len(fields) > 0 {
		RespondError(w, ValidationError(fields))
		return
	}

	transfer, err := h.service.AddTransfer(r.Context(), payouts.AddTransferParams{
		ActorID:                actor.ID,
		TeamID:                 actor.TeamID,
		TraderID:               actor.ID,
		PayoutID:               payoutID,
		SourceShiftRequisiteID: request.SourceShiftRequisiteID,
		AmountMinor:            request.AmountMinor,
		Comment:                request.Comment,
	})
	if err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusCreated, payoutTransferResponse{
		Transfer: payouts.PublicTransferFromDomain(transfer),
	})
}

func (h *TraderPayoutHandler) DeleteTransfer(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	payoutID, ok := payoutIDFromRequest(w, r)
	if !ok {
		return
	}
	transferID, ok := transferIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteTransfer(r.Context(), actor.ID, actor.TeamID, actor.ID, payoutID, transferID); err != nil {
		RespondError(w, mapPayoutError(err))
		return
	}

	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func payoutIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "payoutId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"payoutId": "Некорректный ID выплаты",
		}))
		return 0, false
	}

	return id, true
}

func transferIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "transferId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"transferId": "Некорректный ID перевода",
		}))
		return 0, false
	}

	return id, true
}

func validateCreatePayoutOrderRequest(request createPayoutOrderRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(request.DestinationBank) == "" {
		fields["destinationBank"] = "Банк обязателен"
	}
	if strings.TrimSpace(request.DestinationRequisite) == "" {
		fields["destinationRequisite"] = "Реквизит получателя обязателен"
	}
	if request.AmountMinor <= 0 {
		fields["amountMinor"] = "Сумма выплаты должна быть положительной"
	}

	return fields
}

func validatePatchPayoutOrderRequest(request patchPayoutOrderRequest) map[string]string {
	fields := map[string]string{}
	if request.DestinationBank != nil && strings.TrimSpace(*request.DestinationBank) == "" {
		fields["destinationBank"] = "Банк обязателен"
	}
	if request.DestinationRequisite != nil && strings.TrimSpace(*request.DestinationRequisite) == "" {
		fields["destinationRequisite"] = "Реквизит получателя обязателен"
	}
	if request.AmountMinor != nil && *request.AmountMinor <= 0 {
		fields["amountMinor"] = "Сумма выплаты должна быть положительной"
	}

	return fields
}

func validateAddPayoutTransferRequest(request addPayoutTransferRequest) map[string]string {
	fields := map[string]string{}
	if request.SourceShiftRequisiteID <= 0 {
		fields["sourceShiftRequisiteId"] = "Некорректный ID реквизита смены"
	}
	if request.AmountMinor <= 0 {
		fields["amountMinor"] = "Сумма перевода должна быть положительной"
	}

	return fields
}

func mapPayoutError(err error) error {
	switch {
	case errors.Is(err, payouts.ErrInvalidInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		})
	case errors.Is(err, payouts.ErrOrderNotFound):
		return NotFoundError()
	case errors.Is(err, payouts.ErrTransferNotFound):
		return NotFoundError()
	case errors.Is(err, payouts.ErrNoCurrentShift):
		return DomainError("CURRENT_SHIFT_REQUIRED", "Нужно открыть смену перед созданием выплаты")
	case errors.Is(err, payouts.ErrTransferRejected):
		return DomainError("PAYOUT_TRANSFER_REJECTED", "Перевод не может превышать остаток выплаты или реквизит недоступен")
	case errors.Is(err, payouts.ErrOrderUpdateRejected):
		return DomainError("PAYOUT_UPDATE_REJECTED", "Выплату нельзя обновить с указанными параметрами")
	default:
		return err
	}
}
