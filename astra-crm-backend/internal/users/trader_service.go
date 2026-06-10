package users

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

var (
	ErrTraderNotFound        = errors.New("trader not found")
	ErrDuplicateLogin        = errors.New("trader login already exists")
	ErrDuplicateWorkerName   = errors.New("external worker name already exists")
	ErrInvalidTraderInput    = errors.New("invalid trader input")
	ErrPasswordResetNotSaved = errors.New("password reset was not saved")
)

type TraderStore interface {
	CreateTrader(ctx context.Context, params CreateTraderRecord) (Trader, error)
	GetTraderByID(ctx context.Context, teamID int64, traderID int64) (Trader, error)
	ListTraderDetailsByTeam(ctx context.Context, teamID int64) ([]Trader, error)
	UpdateTrader(ctx context.Context, params UpdateTraderRecord) (Trader, error)
	UpdateTraderPasswordHash(ctx context.Context, teamID int64, traderID int64, passwordHash string) error
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type PasswordHasher func(password string) (string, error)
type PasswordGenerator func() (string, error)

type TraderService struct {
	store            TraderStore
	audit            AuditService
	hashPassword     PasswordHasher
	generatePassword PasswordGenerator
}

func NewTraderService(store TraderStore, auditService AuditService, hashPassword PasswordHasher, generatePassword PasswordGenerator) *TraderService {
	return &TraderService{
		store:            store,
		audit:            auditService,
		hashPassword:     hashPassword,
		generatePassword: generatePassword,
	}
}

type CreateTraderParams struct {
	ActorID            int64
	TeamID             int64
	Login              string
	Password           string
	SalaryRateBps      int64
	ExternalWorkerName string
}

type CreateTraderRecord struct {
	TeamID             int64
	Login              string
	PasswordHash       string
	SalaryRateBps      int64
	ExternalWorkerName string
}

type PatchTraderParams struct {
	ActorID            int64
	TeamID             int64
	TraderID           int64
	Status             *string
	SalaryRateBps      *int64
	ExternalWorkerName *string
}

type UpdateTraderRecord struct {
	TeamID             int64
	TraderID           int64
	Status             string
	SalaryRateBps      int64
	ExternalWorkerName string
}

type ResetTraderPasswordParams struct {
	ActorID  int64
	TeamID   int64
	TraderID int64
}

type ResetTraderPasswordResult struct {
	Trader            Trader
	TemporaryPassword string
}

func (s *TraderService) Create(ctx context.Context, params CreateTraderParams) (Trader, error) {
	if err := validateCreateTrader(params); err != nil {
		return Trader{}, err
	}

	passwordHash, err := s.hashPassword(params.Password)
	if err != nil {
		return Trader{}, fmt.Errorf("hash trader password: %w", err)
	}

	trader, err := s.store.CreateTrader(ctx, CreateTraderRecord{
		TeamID:             params.TeamID,
		Login:              strings.TrimSpace(params.Login),
		PasswordHash:       passwordHash,
		SalaryRateBps:      params.SalaryRateBps,
		ExternalWorkerName: strings.TrimSpace(params.ExternalWorkerName),
	})
	if err != nil {
		return Trader{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionUserCreated,
		EntityType: "user",
		EntityID:   strconv.FormatInt(trader.ID, 10),
		After:      ToPublicTrader(trader),
	}); err != nil {
		return Trader{}, err
	}

	return trader, nil
}

func (s *TraderService) List(ctx context.Context, teamID int64) ([]Trader, error) {
	return s.store.ListTraderDetailsByTeam(ctx, teamID)
}

func (s *TraderService) Get(ctx context.Context, teamID int64, traderID int64) (Trader, error) {
	return s.store.GetTraderByID(ctx, teamID, traderID)
}

func (s *TraderService) Patch(ctx context.Context, params PatchTraderParams) (Trader, error) {
	current, err := s.store.GetTraderByID(ctx, params.TeamID, params.TraderID)
	if err != nil {
		return Trader{}, err
	}

	next := current
	if params.Status != nil {
		status := strings.TrimSpace(*params.Status)
		if !validTraderStatus(status) {
			return Trader{}, ErrInvalidTraderInput
		}
		next.Status = status
	}
	if params.SalaryRateBps != nil {
		if *params.SalaryRateBps < 0 {
			return Trader{}, ErrInvalidTraderInput
		}
		next.SalaryRateBps = *params.SalaryRateBps
	}
	if params.ExternalWorkerName != nil {
		workerName := strings.TrimSpace(*params.ExternalWorkerName)
		if workerName == "" {
			return Trader{}, ErrInvalidTraderInput
		}
		next.ExternalWorkerName = workerName
	}

	updated, err := s.store.UpdateTrader(ctx, UpdateTraderRecord{
		TeamID:             params.TeamID,
		TraderID:           params.TraderID,
		Status:             next.Status,
		SalaryRateBps:      next.SalaryRateBps,
		ExternalWorkerName: next.ExternalWorkerName,
	})
	if err != nil {
		return Trader{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionUserUpdated,
		EntityType: "user",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		Before:     ToPublicTrader(current),
		After:      ToPublicTrader(updated),
		ChangedFields: map[string]any{
			"status":             updated.Status,
			"salaryRateBps":      updated.SalaryRateBps,
			"externalWorkerName": updated.ExternalWorkerName,
		},
	}); err != nil {
		return Trader{}, err
	}

	return updated, nil
}

func (s *TraderService) Delete(ctx context.Context, actorID int64, teamID int64, traderID int64) error {
	status := StatusDeleted
	_, err := s.Patch(ctx, PatchTraderParams{
		ActorID:  actorID,
		TeamID:   teamID,
		TraderID: traderID,
		Status:   &status,
	})
	return err
}

func (s *TraderService) ResetPassword(ctx context.Context, params ResetTraderPasswordParams) (ResetTraderPasswordResult, error) {
	trader, err := s.store.GetTraderByID(ctx, params.TeamID, params.TraderID)
	if err != nil {
		return ResetTraderPasswordResult{}, err
	}

	temporaryPassword, err := s.generatePassword()
	if err != nil {
		return ResetTraderPasswordResult{}, fmt.Errorf("generate temporary trader password: %w", err)
	}

	passwordHash, err := s.hashPassword(temporaryPassword)
	if err != nil {
		return ResetTraderPasswordResult{}, fmt.Errorf("hash temporary trader password: %w", err)
	}

	if err := s.store.UpdateTraderPasswordHash(ctx, params.TeamID, params.TraderID, passwordHash); err != nil {
		return ResetTraderPasswordResult{}, err
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionUserPasswordReset,
		EntityType: "user",
		EntityID:   strconv.FormatInt(params.TraderID, 10),
		After: map[string]any{
			"userId":            params.TraderID,
			"temporaryPassword": temporaryPassword,
		},
	}); err != nil {
		return ResetTraderPasswordResult{}, err
	}

	return ResetTraderPasswordResult{
		Trader:            trader,
		TemporaryPassword: temporaryPassword,
	}, nil
}

func validateCreateTrader(params CreateTraderParams) error {
	if params.TeamID <= 0 || params.ActorID <= 0 {
		return ErrInvalidTraderInput
	}
	if strings.TrimSpace(params.Login) == "" {
		return ErrInvalidTraderInput
	}
	if params.Password == "" {
		return ErrInvalidTraderInput
	}
	if params.SalaryRateBps < 0 {
		return ErrInvalidTraderInput
	}
	if strings.TrimSpace(params.ExternalWorkerName) == "" {
		return ErrInvalidTraderInput
	}

	return nil
}

func validTraderStatus(status string) bool {
	switch status {
	case StatusActive, StatusDisabled, StatusDeleted:
		return true
	default:
		return false
	}
}

func (s *TraderService) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}
