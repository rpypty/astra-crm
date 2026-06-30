package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/banks"
	"github.com/go-chi/chi/v5"
)

type BankService interface {
	ListActive(ctx context.Context) ([]banks.Bank, error)
	UpdateCSVAlias(ctx context.Context, params banks.UpdateCSVAliasParams) (banks.Bank, error)
}

type BanksHandler struct {
	service BankService
}

func NewBanksHandler(service BankService) *BanksHandler {
	return &BanksHandler{service: service}
}

type banksListResponse struct {
	Items []banks.PublicBank `json:"items"`
}

type bankResponse struct {
	Bank banks.PublicBank `json:"bank"`
}

type updateBankCSVAliasRequest struct {
	CSVAlias *string `json:"csvAlias"`
}

func (h *BanksHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := CurrentUser(r.Context()); !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	items, err := h.service.ListActive(r.Context())
	if err != nil {
		RespondError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, banksListResponse{
		Items: banks.PublicBanks(items),
	})
}

func (h *BanksHandler) UpdateCSVAlias(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	bankCode := strings.TrimSpace(chi.URLParam(r, "bankCode"))
	if bankCode == "" {
		RespondError(w, ValidationError(map[string]string{
			"bankCode": "Некорректный код банка",
		}))
		return
	}

	var request updateBankCSVAliasRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		RespondError(w, ValidationError(map[string]string{
			"body": "Некорректный JSON",
		}))
		return
	}
	if request.CSVAlias == nil {
		RespondError(w, ValidationError(map[string]string{
			"csvAlias": "Передайте alias банка для CSV",
		}))
		return
	}

	bank, err := h.service.UpdateCSVAlias(r.Context(), banks.UpdateCSVAliasParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		BankCode: bankCode,
		CSVAlias: *request.CSVAlias,
	})
	if err != nil {
		RespondError(w, mapBankError(err))
		return
	}

	WriteJSON(w, http.StatusOK, bankResponse{Bank: banks.PublicBankFromDomain(bank)})
}

func mapBankError(err error) error {
	switch {
	case errors.Is(err, banks.ErrBankNotFound):
		return NotFoundError().WithCause(err)
	case errors.Is(err, banks.ErrDuplicateCSVAlias):
		return DomainError("BANK_CSV_ALIAS_EXISTS", "Такой alias CSV уже задан для другого банка").WithCause(err)
	case errors.Is(err, banks.ErrInvalidBankInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		}).WithCause(err)
	default:
		return err
	}
}
