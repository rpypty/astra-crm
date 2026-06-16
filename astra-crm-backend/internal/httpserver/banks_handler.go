package httpserver

import (
	"context"
	"net/http"

	"github.com/ashpak/astra-crm-backend/internal/banks"
)

type BankService interface {
	ListActive(ctx context.Context) ([]banks.Bank, error)
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
