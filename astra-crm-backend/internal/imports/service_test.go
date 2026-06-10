package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
)

func TestServiceApplyCSVAppliesParsedRowsAndAudits(t *testing.T) {
	store := &fakeStore{}
	auditService := &fakeAuditService{}
	hook := &fakeHook{}
	service := NewService(store, auditService, hook)
	shiftID := int64(10)

	csvBody := strings.Join([]string{
		"id|innerId|amount|currency|status|createdAt|workerName",
		"1|in-1|10.0|RUB|hand_success|28.05.2026 11:21:35|Bliss_OP2",
	}, "\n")

	result, err := service.ApplyCSV(context.Background(), ApplyCSVParams{
		ActorID:  3,
		TeamID:   2,
		FileName: "trader-inbound.csv",
		Scope: Scope{
			Type:      ScopeTypeTraderShift,
			Direction: DirectionInbound,
			ShiftID:   &shiftID,
		},
		Reader:       strings.NewReader(csvBody),
		ParseOptions: parseOptions(),
	})
	if err != nil {
		t.Fatalf("ApplyCSV() error = %v", err)
	}

	wantHashBytes := sha256.Sum256([]byte(csvBody))
	wantHash := hex.EncodeToString(wantHashBytes[:])
	if store.record.FileHash != wantHash {
		t.Fatalf("file hash = %q, want %q", store.record.FileHash, wantHash)
	}
	if store.record.Scope.TraderID == nil || *store.record.Scope.TraderID != 3 {
		t.Fatalf("trader id = %v, want actor id 3", store.record.Scope.TraderID)
	}
	if len(store.record.Rows) != 1 {
		t.Fatalf("stored rows count = %d, want 1", len(store.record.Rows))
	}
	if result.RowsCount != 1 || result.ActiveScopeItems != 1 {
		t.Fatalf("result rows/active = %d/%d, want 1/1", result.RowsCount, result.ActiveScopeItems)
	}
	if !hook.called {
		t.Fatal("reconciliation hook was not called")
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionImportApplied {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionImportApplied)
	}
}

func TestServiceApplyCSVRejectsDuplicateBeforeStore(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil, nil)
	shiftID := int64(10)

	csvBody := strings.Join([]string{
		"id|innerId|amount|currency|status|createdAt|workerName",
		"1|dup-1|10.0|RUB|hand_success|28.05.2026 11:21:35|Bliss_OP2",
		"2|dup-1|20.0|RUB|hand_success|28.05.2026 11:22:35|Bliss_OP2",
	}, "\n")

	result, err := service.ApplyCSV(context.Background(), ApplyCSVParams{
		ActorID:  3,
		TeamID:   2,
		FileName: "trader-inbound.csv",
		Scope: Scope{
			Type:      ScopeTypeTraderShift,
			Direction: DirectionInbound,
			ShiftID:   &shiftID,
		},
		Reader:       strings.NewReader(csvBody),
		ParseOptions: parseOptions(),
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ApplyCSV() error = %v, want ErrValidation", err)
	}
	if store.called {
		t.Fatal("store was called, want parser validation to stop before applying")
	}
	if len(result.Parse.Errors) != 1 || result.Parse.Errors[0].Code != ParseCodeDuplicateInnerID {
		t.Fatalf("parse errors = %+v, want duplicate innerId", result.Parse.Errors)
	}
}

func TestServiceApplyCSVRejectsInvalidScope(t *testing.T) {
	service := NewService(&fakeStore{}, nil, nil)

	_, err := service.ApplyCSV(context.Background(), ApplyCSVParams{
		ActorID:  3,
		TeamID:   2,
		FileName: "bad.csv",
		Scope: Scope{
			Type:      ScopeTypeTraderShift,
			Direction: DirectionInbound,
		},
		Reader: strings.NewReader(""),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ApplyCSV() error = %v, want ErrInvalidInput", err)
	}
}

type fakeStore struct {
	called bool
	record ApplyImportRecord
}

func (s *fakeStore) ApplyImport(ctx context.Context, record ApplyImportRecord) (ApplyResult, error) {
	s.called = true
	s.record = record
	return ApplyResult{
		Batch: ImportBatch{
			ID:         100,
			TeamID:     record.TeamID,
			UploadedBy: record.UploadedBy,
			ScopeType:  record.Scope.Type,
			Direction:  record.Scope.Direction,
			ShiftID:    record.Scope.ShiftID,
			TraderID:   record.Scope.TraderID,
			FileName:   record.FileName,
			FileHash:   record.FileHash,
			RowsCount:  int64(len(record.Rows)),
			Status:     BatchStatusApplied,
			CreatedAt:  time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
		RowsCount:        int64(len(record.Rows)),
		CreatedOrders:    int64(len(record.Rows)),
		ActiveScopeItems: int64(len(record.Rows)),
	}, nil
}

type fakeAuditService struct {
	events []audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.events = append(s.events, event)
	return nil
}

type fakeHook struct {
	called bool
}

func (h *fakeHook) AfterImportApplied(ctx context.Context, result ApplyResult) error {
	h.called = true
	return nil
}
