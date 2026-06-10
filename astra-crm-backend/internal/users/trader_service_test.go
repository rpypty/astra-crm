package users

import (
	"context"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

func TestTraderServiceCreateHashesPasswordAndAuditsPublicPayload(t *testing.T) {
	store := &fakeTraderStore{}
	auditService := &fakeAuditService{}
	service := NewTraderService(store, auditService, fakeHashPassword, fakeGeneratePassword)

	trader, err := service.Create(context.Background(), CreateTraderParams{
		ActorID:            1,
		TeamID:             2,
		Login:              " trader_ivan ",
		Password:           "temporary-password",
		SalaryRateBps:      50,
		ExternalWorkerName: " Bliss_OP2 ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if trader.ID != 10 {
		t.Fatalf("trader ID = %d, want 10", trader.ID)
	}
	if store.created.PasswordHash == "temporary-password" {
		t.Fatal("raw password was passed to store")
	}
	if store.created.Login != "trader_ivan" {
		t.Fatalf("created login = %q, want trimmed login", store.created.Login)
	}
	if auditService.event.Action != audit.ActionUserCreated {
		t.Fatalf("audit action = %q, want %q", auditService.event.Action, audit.ActionUserCreated)
	}
	if _, ok := auditService.event.After.(PublicTrader); !ok {
		t.Fatalf("audit after payload = %T, want PublicTrader", auditService.event.After)
	}
}

func TestTraderServicePatchRejectsNegativeSalaryRate(t *testing.T) {
	store := &fakeTraderStore{
		byID: Trader{
			ID:                 10,
			TeamID:             2,
			Role:               RoleTrader,
			Login:              "trader_ivan",
			Status:             StatusActive,
			SalaryRateBps:      50,
			ExternalWorkerName: "Bliss_OP2",
		},
	}
	service := NewTraderService(store, nil, fakeHashPassword, fakeGeneratePassword)
	negativeRate := int64(-1)

	_, err := service.Patch(context.Background(), PatchTraderParams{
		ActorID:       1,
		TeamID:        2,
		TraderID:      10,
		SalaryRateBps: &negativeRate,
	})
	if err != ErrInvalidTraderInput {
		t.Fatalf("Patch() error = %v, want ErrInvalidTraderInput", err)
	}
}

func TestTraderServiceResetPasswordUpdatesHashAndAudits(t *testing.T) {
	store := &fakeTraderStore{
		byID: Trader{
			ID:                 10,
			TeamID:             2,
			Role:               RoleTrader,
			Login:              "trader_ivan",
			Status:             StatusActive,
			SalaryRateBps:      50,
			ExternalWorkerName: "Bliss_OP2",
		},
	}
	auditService := &fakeAuditService{}
	service := NewTraderService(store, auditService, fakeHashPassword, fakeGeneratePassword)

	result, err := service.ResetPassword(context.Background(), ResetTraderPasswordParams{
		ActorID:  1,
		TeamID:   2,
		TraderID: 10,
	})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if result.TemporaryPassword != "generated-temporary-password" {
		t.Fatalf("temporary password = %q", result.TemporaryPassword)
	}
	if store.updatedPasswordHash == result.TemporaryPassword {
		t.Fatal("raw temporary password was stored")
	}
	if auditService.event.Action != audit.ActionUserPasswordReset {
		t.Fatalf("audit action = %q, want %q", auditService.event.Action, audit.ActionUserPasswordReset)
	}
}

type fakeTraderStore struct {
	created             CreateTraderRecord
	updated             UpdateTraderRecord
	updatedPasswordHash string
	byID                Trader
	list                []Trader
}

func (s *fakeTraderStore) CreateTrader(ctx context.Context, params CreateTraderRecord) (Trader, error) {
	s.created = params
	return Trader{
		ID:                 10,
		TeamID:             params.TeamID,
		Role:               RoleTrader,
		Login:              params.Login,
		Status:             StatusActive,
		SalaryRateBps:      params.SalaryRateBps,
		ExternalWorkerName: params.ExternalWorkerName,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeTraderStore) GetTraderByID(ctx context.Context, teamID int64, traderID int64) (Trader, error) {
	if s.byID.ID == 0 {
		return Trader{}, ErrTraderNotFound
	}

	return s.byID, nil
}

func (s *fakeTraderStore) ListTraderDetailsByTeam(ctx context.Context, teamID int64) ([]Trader, error) {
	return s.list, nil
}

func (s *fakeTraderStore) UpdateTrader(ctx context.Context, params UpdateTraderRecord) (Trader, error) {
	s.updated = params
	return Trader{
		ID:                 params.TraderID,
		TeamID:             params.TeamID,
		Role:               RoleTrader,
		Login:              s.byID.Login,
		Status:             params.Status,
		SalaryRateBps:      params.SalaryRateBps,
		ExternalWorkerName: params.ExternalWorkerName,
	}, nil
}

func (s *fakeTraderStore) UpdateTraderPasswordHash(ctx context.Context, teamID int64, traderID int64, passwordHash string) error {
	s.updatedPasswordHash = passwordHash
	return nil
}

type fakeAuditService struct {
	event audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.event = event
	return nil
}

func fakeHashPassword(password string) (string, error) {
	return "hashed:" + password, nil
}

func fakeGeneratePassword() (string, error) {
	return "generated-temporary-password", nil
}
