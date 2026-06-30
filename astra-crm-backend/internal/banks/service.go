package banks

import (
	"context"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

type Store interface {
	ListActive(ctx context.Context) ([]Bank, error)
	UpdateCSVAlias(ctx context.Context, code string, alias string) (Bank, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type Service struct {
	store Store
	audit AuditService
}

func NewService(store Store, auditServices ...AuditService) *Service {
	var auditService AuditService
	if len(auditServices) > 0 {
		auditService = auditServices[0]
	}
	return &Service{store: store, audit: auditService}
}

func (s *Service) ListActive(ctx context.Context) ([]Bank, error) {
	return s.store.ListActive(ctx)
}

type UpdateCSVAliasParams struct {
	ActorID  int64
	TeamID   int64
	BankCode string
	CSVAlias string
}

func (s *Service) UpdateCSVAlias(ctx context.Context, params UpdateCSVAliasParams) (Bank, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || strings.TrimSpace(params.BankCode) == "" {
		return Bank{}, ErrInvalidBankInput
	}

	alias := strings.TrimSpace(params.CSVAlias)
	if len([]rune(alias)) > 128 {
		return Bank{}, ErrInvalidBankInput
	}

	bank, err := s.store.UpdateCSVAlias(ctx, strings.TrimSpace(params.BankCode), alias)
	if err != nil {
		return Bank{}, err
	}

	if s.audit != nil {
		if err := s.audit.Write(ctx, audit.Event{
			TeamID:     params.TeamID,
			ActorID:    params.ActorID,
			Action:     audit.ActionBankUpdated,
			EntityType: "bank",
			EntityID:   strconv.FormatInt(bank.ID, 10),
			After:      PublicBankFromDomain(bank),
			ChangedFields: map[string]any{
				"csvAlias": bank.CSVAlias,
			},
		}); err != nil {
			return Bank{}, err
		}
	}

	return bank, nil
}
