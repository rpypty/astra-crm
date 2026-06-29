package reconciliation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
)

func TestServiceAfterImportAppliedRecalculatesTraderInbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionInbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if store.recalculateRecord.TeamID != 2 || store.recalculateRecord.TraderID != 3 || store.recalculateRecord.ShiftID != 10 {
		t.Fatalf("recalculate record = %+v, want team/trader/shift 2/3/10", store.recalculateRecord)
	}
	if store.recalculateRecord.ImportBatchID == nil || *store.recalculateRecord.ImportBatchID != 100 {
		t.Fatalf("import batch id = %v, want 100", store.recalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterImportAppliedRecalculatesActiveTeamleadPeriodsAfterTraderInbound(t *testing.T) {
	store := &fakeStore{
		activeTeamleadScopes: []TeamleadInboundPeriodScope{
			{AccountingPeriodID: 55, ImportBatchID: 500},
			{AccountingPeriodID: 56, ImportBatchID: 501},
		},
	}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionInbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if len(store.teamleadPeriodRecalculateRecords) != 2 {
		t.Fatalf("teamlead period recalculate count = %d, want 2", len(store.teamleadPeriodRecalculateRecords))
	}
	if store.teamleadPeriodRecalculateRecords[0].AccountingPeriodID != 55 || store.teamleadPeriodRecalculateRecords[1].AccountingPeriodID != 56 {
		t.Fatalf("teamlead period recalculate records = %+v, want periods 55 and 56", store.teamleadPeriodRecalculateRecords)
	}
}

func TestServiceAfterImportAppliedRecalculatesActiveTeamleadPeriodsAfterTraderOutbound(t *testing.T) {
	store := &fakeStore{
		activeTeamleadOutboundScopes: []TeamleadOutboundPeriodScope{
			{AccountingPeriodID: 65, ImportBatchID: 600},
			{AccountingPeriodID: 66, ImportBatchID: 601},
		},
	}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionOutbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if len(store.teamleadPeriodOutboundRecalculateRecords) != 2 {
		t.Fatalf("teamlead outbound period recalculate count = %d, want 2", len(store.teamleadPeriodOutboundRecalculateRecords))
	}
	if store.teamleadPeriodOutboundRecalculateRecords[0].AccountingPeriodID != 65 || store.teamleadPeriodOutboundRecalculateRecords[1].AccountingPeriodID != 66 {
		t.Fatalf("teamlead outbound period recalculate records = %+v, want periods 65 and 66", store.teamleadPeriodOutboundRecalculateRecords)
	}
}

func TestServiceAfterImportAppliedRecalculatesTraderOutbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	shiftID := int64(10)
	traderID := int64(3)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:        100,
			TeamID:    2,
			ScopeType: imports.ScopeTypeTraderShift,
			Direction: imports.DirectionOutbound,
			ShiftID:   &shiftID,
			TraderID:  &traderID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}
	if store.outboundRecalculateRecord.TeamID != 2 || store.outboundRecalculateRecord.TraderID != 3 || store.outboundRecalculateRecord.ShiftID != 10 {
		t.Fatalf("outbound recalculate record = %+v, want team/trader/shift 2/3/10", store.outboundRecalculateRecord)
	}
	if store.outboundRecalculateRecord.ImportBatchID == nil || *store.outboundRecalculateRecord.ImportBatchID != 100 {
		t.Fatalf("outbound import batch id = %v, want 100", store.outboundRecalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterImportAppliedRecalculatesTeamleadPeriodInbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	periodID := int64(55)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:                 500,
			TeamID:             2,
			ScopeType:          imports.ScopeTypeTeamleadPeriod,
			Direction:          imports.DirectionInbound,
			AccountingPeriodID: &periodID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if len(store.teamleadPeriodRecalculateRecords) != 1 {
		t.Fatalf("teamlead period recalculate count = %d, want 1", len(store.teamleadPeriodRecalculateRecords))
	}
	record := store.teamleadPeriodRecalculateRecords[0]
	if record.TeamID != 2 || record.AccountingPeriodID != 55 {
		t.Fatalf("teamlead period recalculate record = %+v, want team/period 2/55", record)
	}
	if record.ImportBatchID == nil || *record.ImportBatchID != 500 {
		t.Fatalf("teamlead import batch id = %v, want 500", record.ImportBatchID)
	}
}

func TestServiceAfterImportAppliedRecalculatesTeamleadPeriodOutbound(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	periodID := int64(55)

	err := service.AfterImportApplied(context.Background(), imports.ApplyResult{
		Batch: imports.ImportBatch{
			ID:                 500,
			TeamID:             2,
			ScopeType:          imports.ScopeTypeTeamleadPeriod,
			Direction:          imports.DirectionOutbound,
			AccountingPeriodID: &periodID,
		},
	})
	if err != nil {
		t.Fatalf("AfterImportApplied() error = %v", err)
	}

	if len(store.teamleadPeriodOutboundRecalculateRecords) != 1 {
		t.Fatalf("teamlead outbound period recalculate count = %d, want 1", len(store.teamleadPeriodOutboundRecalculateRecords))
	}
	record := store.teamleadPeriodOutboundRecalculateRecords[0]
	if record.TeamID != 2 || record.AccountingPeriodID != 55 {
		t.Fatalf("teamlead outbound period recalculate record = %+v, want team/period 2/55", record)
	}
	if record.ImportBatchID == nil || *record.ImportBatchID != 500 {
		t.Fatalf("teamlead outbound import batch id = %v, want 500", record.ImportBatchID)
	}
}

func TestServiceCreateTeamleadReconciliationAlwaysRunsTransactionDiff(t *testing.T) {
	cardNumber := "1234567890123456"
	store := &fakeStore{
		teamleadTraders: []TeamleadTraderMatch{
			{TraderID: 3, ExternalWorkerName: "Bliss_OP1"},
		},
		teamleadRequisites: []TeamleadRequisiteMatch{
			{
				ID:                   20,
				BankCode:             "sber",
				Phone:                "79991234567",
				CardNumber:           &cardNumber,
				NormalizedPhone:      "79991234567",
				NormalizedCardNumber: cardNumber,
			},
		},
		teamleadInboundTurnover: TeamleadTurnoverSnapshot{
			AmountMinor: 1000,
			Count:       1,
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)
	inboundCSV := []byte("id|foreignId|innerId|requisite|requisitePhone|methodName|amount|currency|status|createdAt|workerName\n" +
		"1|f1|in-1|1234567890123456|+7 (999) 123-45-67|Сбер|10.00|RUB|hand_success|10.06.2026 12:00:00|Bliss_OP1\n" +
		"2|f2|outside-period|1234567890123456|+7 (999) 123-45-67|Сбер|20.00|RUB|hand_success|01.05.2026 12:00:00|Bliss_OP1\n")

	run, err := service.CreateTeamleadReconciliation(context.Background(), CreateTeamleadReconciliationParams{
		ActorID:  1,
		TeamID:   2,
		DateFrom: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		DateTo:   time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Inbound: &TeamleadCSVInput{
			FileName: "inbound.csv",
			Payload:  inboundCSV,
		},
	})
	if err != nil {
		t.Fatalf("CreateTeamleadReconciliation() error = %v", err)
	}
	if run.Status != TeamleadRunStatusMismatch {
		t.Fatalf("run status = %q, want mismatch", run.Status)
	}
	if len(store.createdTeamleadAnalysis.Directions) != 1 {
		t.Fatalf("directions count = %d, want 1", len(store.createdTeamleadAnalysis.Directions))
	}
	summary := store.createdTeamleadAnalysis.Directions[0].Summary
	if summary.RowsTotal != 2 || summary.RowsInPeriod != 1 {
		t.Fatalf("summary rows = total:%d period:%d, want 2/1", summary.RowsTotal, summary.RowsInPeriod)
	}
	if summary.SuccessAmountMinor != 1000 || summary.CRMAmountMinor != 1000 || summary.DiffAmountMinor != 0 {
		t.Fatalf("summary amounts = success:%d crm:%d diff:%d, want 1000/1000/0", summary.SuccessAmountMinor, summary.CRMAmountMinor, summary.DiffAmountMinor)
	}
	if summary.CreateCount != 1 || summary.UpdateCount != 0 || summary.UnchangedCount != 0 {
		t.Fatalf("preview counts = create:%d update:%d unchanged:%d, want 1/0/0", summary.CreateCount, summary.UpdateCount, summary.UnchangedCount)
	}
	if !hasTeamleadItem(store.createdTeamleadAnalysis.Items, TeamleadItemStageTransactionCheck, "missing_in_crm") {
		t.Fatalf("items = %+v, want missing_in_crm transaction diff item", store.createdTeamleadAnalysis.Items)
	}
	if hasTeamleadItem(store.createdTeamleadAnalysis.Items, TeamleadItemStageTurnoverCheck, "turnover_mismatch") {
		t.Fatalf("items = %+v, did not expect turnover mismatch when totals match", store.createdTeamleadAnalysis.Items)
	}
	if len(auditService.events) != 1 || auditService.events[0].Action != audit.ActionReconciliationCreated {
		t.Fatalf("audit events = %+v, want reconciliation.created", auditService.events)
	}
}

func TestServiceCreateTeamleadReconciliationRejectsDuplicateInnerID(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	inboundCSV := []byte("id|innerId|amount|currency|status|createdAt|workerName\n" +
		"1|dup-1|10.00|RUB|hand_success|10.06.2026 12:00:00|Bliss_OP1\n" +
		"2|dup-1|20.00|RUB|hand_success|10.06.2026 13:00:00|Bliss_OP1\n")

	_, err := service.CreateTeamleadReconciliation(context.Background(), CreateTeamleadReconciliationParams{
		ActorID:  1,
		TeamID:   2,
		DateFrom: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		DateTo:   time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Inbound: &TeamleadCSVInput{
			FileName: "inbound.csv",
			Payload:  inboundCSV,
		},
	})
	var csvErr *TeamleadCSVValidationError
	if !errors.As(err, &csvErr) {
		t.Fatalf("CreateTeamleadReconciliation() error = %v, want TeamleadCSVValidationError", err)
	}
	if len(csvErr.Parse.Errors) != 1 || csvErr.Parse.Errors[0].Code != imports.ParseCodeDuplicateInnerID {
		t.Fatalf("parse errors = %+v, want duplicate innerId", csvErr.Parse.Errors)
	}
	if store.createdTeamleadAnalysis.TeamID != 0 {
		t.Fatalf("created analysis = %+v, want no persisted run", store.createdTeamleadAnalysis)
	}
}

func TestServiceCreateTeamleadReconciliationReportsTurnoverMismatchAndTransactionDiff(t *testing.T) {
	cardNumber := "1234567890123456"
	traderID := int64(3)
	requisiteID := int64(20)
	store := &fakeStore{
		teamleadTraders: []TeamleadTraderMatch{
			{TraderID: traderID, ExternalWorkerName: "Bliss_OP1"},
		},
		teamleadRequisites: []TeamleadRequisiteMatch{
			{
				ID:                   requisiteID,
				BankCode:             "sber",
				Phone:                "79991234567",
				CardNumber:           &cardNumber,
				NormalizedPhone:      "79991234567",
				NormalizedCardNumber: cardNumber,
			},
		},
		teamleadInboundTurnover: TeamleadTurnoverSnapshot{
			AmountMinor: 700,
			Count:       1,
		},
		teamleadExternalOrders: []TeamleadExternalOrderSnapshot{
			{
				ID:                900,
				Direction:         imports.DirectionInbound,
				ExternalInnerID:   "in-1",
				WorkerName:        "Bliss_OP1",
				TraderID:          &traderID,
				RequisiteID:       &requisiteID,
				AmountMinor:       500,
				RawStatus:         "hand_success",
				NormalizedStatus:  imports.NormalizedStatusSuccess,
				CreatedAtExternal: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	service := NewService(store, nil)
	inboundCSV := []byte("id|foreignId|innerId|requisite|requisitePhone|methodName|amount|currency|status|createdAt|workerName\n" +
		"1|f1|in-1|1234567890123456|+7 (999) 123-45-67|Сбер|10.00|RUB|hand_success|10.06.2026 12:00:00|Bliss_OP1\n")

	run, err := service.CreateTeamleadReconciliation(context.Background(), CreateTeamleadReconciliationParams{
		ActorID:  1,
		TeamID:   2,
		DateFrom: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		DateTo:   time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Inbound: &TeamleadCSVInput{
			FileName: "inbound.csv",
			Payload:  inboundCSV,
		},
	})
	if err != nil {
		t.Fatalf("CreateTeamleadReconciliation() error = %v", err)
	}
	if run.Status != TeamleadRunStatusMismatch {
		t.Fatalf("run status = %q, want mismatch", run.Status)
	}
	if !hasTeamleadItem(store.createdTeamleadAnalysis.Items, TeamleadItemStageTurnoverCheck, "turnover_mismatch") {
		t.Fatalf("items = %+v, want turnover_mismatch", store.createdTeamleadAnalysis.Items)
	}
	if !hasTeamleadItem(store.createdTeamleadAnalysis.Items, TeamleadItemStageTransactionCheck, "amount_changed") {
		t.Fatalf("items = %+v, want amount_changed transaction diff", store.createdTeamleadAnalysis.Items)
	}
	summary := store.createdTeamleadAnalysis.Directions[0].Summary
	if summary.DiffAmountMinor != -300 || summary.UpdateCount != 1 {
		t.Fatalf("summary diff/update = %d/%d, want -300/1", summary.DiffAmountMinor, summary.UpdateCount)
	}
}

func TestServiceCreateTeamleadReconciliationDetectsAmbiguousPhoneOnlyRequisite(t *testing.T) {
	firstCard := "1111222233334444"
	secondCard := "5555666677778888"
	store := &fakeStore{
		teamleadTraders: []TeamleadTraderMatch{
			{TraderID: 3, ExternalWorkerName: "Bliss_OP1"},
		},
		teamleadRequisites: []TeamleadRequisiteMatch{
			{
				ID:                   20,
				BankCode:             "sber",
				Phone:                "79991234567",
				CardNumber:           &firstCard,
				NormalizedPhone:      "79991234567",
				NormalizedCardNumber: firstCard,
			},
			{
				ID:                   21,
				BankCode:             "sber",
				Phone:                "79991234567",
				CardNumber:           &secondCard,
				NormalizedPhone:      "79991234567",
				NormalizedCardNumber: secondCard,
			},
		},
		teamleadInboundTurnover: TeamleadTurnoverSnapshot{
			AmountMinor: 1000,
			Count:       1,
		},
	}
	service := NewService(store, nil)
	inboundCSV := []byte("id|foreignId|innerId|requisite|requisitePhone|methodName|amount|currency|status|createdAt|workerName\n" +
		"1|f1|in-1||+7 (999) 123-45-67|Сбер|10.00|RUB|hand_success|10.06.2026 12:00:00|Bliss_OP1\n")

	run, err := service.CreateTeamleadReconciliation(context.Background(), CreateTeamleadReconciliationParams{
		ActorID:  1,
		TeamID:   2,
		DateFrom: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		DateTo:   time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Inbound: &TeamleadCSVInput{
			FileName: "inbound.csv",
			Payload:  inboundCSV,
		},
	})
	if err != nil {
		t.Fatalf("CreateTeamleadReconciliation() error = %v", err)
	}
	if run.BlockedCount == 0 {
		t.Fatalf("blocked count = %d, want blocker for ambiguous requisite", run.BlockedCount)
	}
	if !hasTeamleadItem(store.createdTeamleadAnalysis.Items, TeamleadItemStageMatching, "ambiguous_requisite") {
		t.Fatalf("items = %+v, want ambiguous_requisite", store.createdTeamleadAnalysis.Items)
	}
}

func TestServiceCreateTeamleadReconciliationDetectsCardPhoneConflict(t *testing.T) {
	cardNumber := "1234567890123456"
	store := &fakeStore{
		teamleadTraders: []TeamleadTraderMatch{
			{TraderID: 3, ExternalWorkerName: "Bliss_OP1"},
		},
		teamleadRequisites: []TeamleadRequisiteMatch{
			{
				ID:                   20,
				BankCode:             "sber",
				Phone:                "70000000000",
				CardNumber:           &cardNumber,
				NormalizedPhone:      "70000000000",
				NormalizedCardNumber: cardNumber,
			},
		},
		teamleadInboundTurnover: TeamleadTurnoverSnapshot{
			AmountMinor: 1000,
			Count:       1,
		},
	}
	service := NewService(store, nil)
	inboundCSV := []byte("id|foreignId|innerId|requisite|requisitePhone|methodName|amount|currency|status|createdAt|workerName\n" +
		"1|f1|in-1|1234567890123456|+7 (999) 123-45-67|Сбер|10.00|RUB|hand_success|10.06.2026 12:00:00|Bliss_OP1\n")

	run, err := service.CreateTeamleadReconciliation(context.Background(), CreateTeamleadReconciliationParams{
		ActorID:  1,
		TeamID:   2,
		DateFrom: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		DateTo:   time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Inbound: &TeamleadCSVInput{
			FileName: "inbound.csv",
			Payload:  inboundCSV,
		},
	})
	if err != nil {
		t.Fatalf("CreateTeamleadReconciliation() error = %v", err)
	}
	if run.ConflictCount == 0 || run.BlockedCount == 0 {
		t.Fatalf("conflict/blocker counts = %d/%d, want both > 0", run.ConflictCount, run.BlockedCount)
	}
	if !hasTeamleadItem(store.createdTeamleadAnalysis.Items, TeamleadItemStageMatching, "conflict_requisite") {
		t.Fatalf("items = %+v, want conflict_requisite", store.createdTeamleadAnalysis.Items)
	}
}

func TestServiceConfirmTeamleadReconciliationRequiresCommentForMismatch(t *testing.T) {
	store := &fakeStore{
		teamleadRun: TeamleadRun{
			ID:            500,
			TeamID:        2,
			Status:        TeamleadRunStatusMismatch,
			MismatchCount: 1,
		},
	}
	scheduler := &fakeTeamleadApplyScheduler{}
	service := NewService(store, nil, scheduler)

	_, err := service.ConfirmTeamleadReconciliation(context.Background(), ConfirmTeamleadReconciliationParams{
		ActorID: 1,
		TeamID:  2,
		RunID:   500,
		Comment: " ",
	})
	if err != ErrInvalidInput {
		t.Fatalf("ConfirmTeamleadReconciliation() error = %v, want ErrInvalidInput", err)
	}
	if store.queuedTeamleadApplyRecord.RunID != 0 {
		t.Fatalf("queued record = %+v, want no queue", store.queuedTeamleadApplyRecord)
	}
	if scheduler.runID != 0 {
		t.Fatalf("scheduled run id = %d, want 0", scheduler.runID)
	}
}

func TestServiceConfirmTeamleadReconciliationQueuesAsyncApplyAndAudits(t *testing.T) {
	store := &fakeStore{
		teamleadRun: TeamleadRun{
			ID:     500,
			TeamID: 2,
			Status: TeamleadRunStatusMatched,
		},
		queuedTeamleadRun: TeamleadRun{
			ID:            500,
			TeamID:        2,
			Status:        TeamleadRunStatusApplyQueued,
			CreatedBy:     1,
			ApplyQueuedAt: timeValuePtr(time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)),
		},
	}
	auditService := &fakeAuditService{}
	scheduler := &fakeTeamleadApplyScheduler{}
	service := NewService(store, auditService, scheduler)

	run, err := service.ConfirmTeamleadReconciliation(context.Background(), ConfirmTeamleadReconciliationParams{
		ActorID: 1,
		TeamID:  2,
		RunID:   500,
	})
	if err != nil {
		t.Fatalf("ConfirmTeamleadReconciliation() error = %v", err)
	}
	if run.Status != TeamleadRunStatusApplyQueued {
		t.Fatalf("run status = %q, want apply_queued", run.Status)
	}
	if store.queuedTeamleadApplyRecord.RunID != 500 || store.queuedTeamleadApplyRecord.ActorID != 1 {
		t.Fatalf("queued record = %+v, want run 500 actor 1", store.queuedTeamleadApplyRecord)
	}
	if scheduler.teamID != 2 || scheduler.runID != 500 {
		t.Fatalf("scheduler call = team:%d run:%d, want 2/500", scheduler.teamID, scheduler.runID)
	}
	if len(auditService.events) != 1 || auditService.events[0].Action != audit.ActionReconciliationConfirmed {
		t.Fatalf("audit events = %+v, want reconciliation.confirmed", auditService.events)
	}
}

func TestServiceRejectTeamleadReconciliationAuditsWithoutSchedulingApply(t *testing.T) {
	store := &fakeStore{
		teamleadRun: TeamleadRun{
			ID:     500,
			TeamID: 2,
			Status: TeamleadRunStatusMismatch,
		},
		rejectedTeamleadRun: TeamleadRun{
			ID:        500,
			TeamID:    2,
			Status:    TeamleadRunStatusRejected,
			CreatedBy: 1,
			Comment:   stringPtr("bad export"),
		},
	}
	auditService := &fakeAuditService{}
	scheduler := &fakeTeamleadApplyScheduler{}
	service := NewService(store, auditService, scheduler)

	run, err := service.RejectTeamleadReconciliation(context.Background(), RejectTeamleadReconciliationParams{
		ActorID: 1,
		TeamID:  2,
		RunID:   500,
		Comment: " bad export ",
	})
	if err != nil {
		t.Fatalf("RejectTeamleadReconciliation() error = %v", err)
	}
	if run.Status != TeamleadRunStatusRejected {
		t.Fatalf("run status = %q, want rejected", run.Status)
	}
	if store.rejectedTeamleadRecord.Comment != "bad export" {
		t.Fatalf("reject comment = %q, want trimmed comment", store.rejectedTeamleadRecord.Comment)
	}
	if scheduler.runID != 0 {
		t.Fatalf("scheduled run id = %d, want 0", scheduler.runID)
	}
	if len(auditService.events) != 1 || auditService.events[0].Action != audit.ActionReconciliationRejected {
		t.Fatalf("audit events = %+v, want reconciliation.rejected", auditService.events)
	}
}

func TestServiceApplyQueuedTeamleadReconciliationAuditsMutation(t *testing.T) {
	store := &fakeStore{
		appliedTeamleadRun: TeamleadRun{
			ID:     500,
			TeamID: 2,
			Status: TeamleadRunStatusApplied,
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	run, err := service.ApplyQueuedTeamleadReconciliation(context.Background(), 2, 500)
	if err != nil {
		t.Fatalf("ApplyQueuedTeamleadReconciliation() error = %v", err)
	}
	if run.Status != TeamleadRunStatusApplied {
		t.Fatalf("run status = %q, want applied", run.Status)
	}
	if store.appliedTeamleadTeamID != 2 || store.appliedTeamleadRunID != 500 {
		t.Fatalf("apply call = team:%d run:%d, want 2/500", store.appliedTeamleadTeamID, store.appliedTeamleadRunID)
	}
	if len(auditService.events) != 1 || auditService.events[0].Action != audit.ActionReconciliationApplied {
		t.Fatalf("audit events = %+v, want reconciliation.applied", auditService.events)
	}
}

func TestServiceAfterTurnoverCreatedRerunsExistingInboundRun(t *testing.T) {
	importBatchID := int64(100)
	store := &fakeStore{
		latestRun: Run{
			ID:            50,
			TeamID:        2,
			ShiftID:       int64Ptr(10),
			TraderID:      int64Ptr(3),
			ImportBatchID: &importBatchID,
			Status:        StatusMismatch,
			CreatedAt:     time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	service := NewService(store, nil)

	err := service.AfterTurnoverCreated(context.Background(), shifts.TurnoverEntry{
		ID:       30,
		TeamID:   2,
		ShiftID:  10,
		TraderID: 3,
	})
	if err != nil {
		t.Fatalf("AfterTurnoverCreated() error = %v", err)
	}
	if !store.latestCalled || !store.recalculateCalled {
		t.Fatalf("latest/recalculate called = %v/%v, want true/true", store.latestCalled, store.recalculateCalled)
	}
	if store.recalculateRecord.ImportBatchID == nil || *store.recalculateRecord.ImportBatchID != 100 {
		t.Fatalf("import batch id = %v, want 100", store.recalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterTurnoverCreatedIgnoresMissingInboundRun(t *testing.T) {
	store := &fakeStore{latestErr: ErrRunNotFound}
	service := NewService(store, nil)

	err := service.AfterTurnoverCreated(context.Background(), shifts.TurnoverEntry{
		ID:       30,
		TeamID:   2,
		ShiftID:  10,
		TraderID: 3,
	})
	if err != nil {
		t.Fatalf("AfterTurnoverCreated() error = %v", err)
	}
	if store.recalculateCalled {
		t.Fatal("recalculate was called without existing inbound run")
	}
}

func TestServiceAfterManualPayoutChangedRerunsExistingOutboundRun(t *testing.T) {
	importBatchID := int64(200)
	store := &fakeStore{
		latestOutboundRun: Run{
			ID:            70,
			TeamID:        2,
			ShiftID:       int64Ptr(10),
			TraderID:      int64Ptr(3),
			ImportBatchID: &importBatchID,
			Status:        StatusMismatch,
			CreatedAt:     time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	service := NewService(store, nil)

	err := service.AfterManualPayoutChanged(context.Background(), 2, 3, 10)
	if err != nil {
		t.Fatalf("AfterManualPayoutChanged() error = %v", err)
	}
	if !store.latestOutboundCalled || !store.outboundRecalculateCalled {
		t.Fatalf("latest outbound/recalculate outbound called = %v/%v, want true/true", store.latestOutboundCalled, store.outboundRecalculateCalled)
	}
	if store.outboundRecalculateRecord.ImportBatchID == nil || *store.outboundRecalculateRecord.ImportBatchID != 200 {
		t.Fatalf("outbound import batch id = %v, want 200", store.outboundRecalculateRecord.ImportBatchID)
	}
}

func TestServiceAfterManualPayoutChangedIgnoresMissingOutboundRun(t *testing.T) {
	store := &fakeStore{latestOutboundErr: ErrRunNotFound}
	service := NewService(store, nil)

	err := service.AfterManualPayoutChanged(context.Background(), 2, 3, 10)
	if err != nil {
		t.Fatalf("AfterManualPayoutChanged() error = %v", err)
	}
	if store.outboundRecalculateCalled {
		t.Fatal("outbound recalculate was called without existing outbound run")
	}
}

func TestServiceAcceptTraderInboundRequiresComment(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.AcceptTraderInbound(context.Background(), AcceptTraderInboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    50,
		Comment:  " ",
	})
	if err != ErrInvalidInput {
		t.Fatalf("AcceptTraderInbound() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceAcceptTraderOutboundRequiresComment(t *testing.T) {
	service := NewService(&fakeStore{}, nil)

	_, err := service.AcceptTraderOutbound(context.Background(), AcceptTraderOutboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    50,
		Comment:  " ",
	})
	if err != ErrInvalidInput {
		t.Fatalf("AcceptTraderOutbound() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceAcceptTraderInboundAuditsMutation(t *testing.T) {
	store := &fakeStore{
		acceptedRun: Run{
			ID:        50,
			TeamID:    2,
			Type:      TypeTraderShiftInbound,
			ScopeType: imports.ScopeTypeTraderShift,
			ShiftID:   int64Ptr(10),
			TraderID:  int64Ptr(3),
			Status:    StatusAcceptedWithComment,
			Comment:   stringPtr("accepted"),
			CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	run, err := service.AcceptTraderInbound(context.Background(), AcceptTraderInboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    50,
		Comment:  " accepted ",
	})
	if err != nil {
		t.Fatalf("AcceptTraderInbound() error = %v", err)
	}
	if run.Status != StatusAcceptedWithComment {
		t.Fatalf("run status = %q, want accepted_with_comment", run.Status)
	}
	if store.acceptRecord.Comment != "accepted" {
		t.Fatalf("stored comment = %q, want trimmed comment", store.acceptRecord.Comment)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionReconciliationAcceptedWithComment {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionReconciliationAcceptedWithComment)
	}
}

func TestServiceAcceptTraderOutboundAuditsMutation(t *testing.T) {
	store := &fakeStore{
		acceptedOutboundRun: Run{
			ID:        60,
			TeamID:    2,
			Type:      TypeTraderShiftOutbound,
			ScopeType: imports.ScopeTypeTraderShift,
			ShiftID:   int64Ptr(10),
			TraderID:  int64Ptr(3),
			Status:    StatusAcceptedWithComment,
			Comment:   stringPtr("accepted"),
			CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		},
	}
	auditService := &fakeAuditService{}
	service := NewService(store, auditService)

	run, err := service.AcceptTraderOutbound(context.Background(), AcceptTraderOutboundParams{
		ActorID:  3,
		TeamID:   2,
		TraderID: 3,
		RunID:    60,
		Comment:  " accepted ",
	})
	if err != nil {
		t.Fatalf("AcceptTraderOutbound() error = %v", err)
	}
	if run.Status != StatusAcceptedWithComment {
		t.Fatalf("run status = %q, want accepted_with_comment", run.Status)
	}
	if store.acceptOutboundRecord.Comment != "accepted" {
		t.Fatalf("stored outbound comment = %q, want trimmed comment", store.acceptOutboundRecord.Comment)
	}
	if len(auditService.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(auditService.events))
	}
	if auditService.events[0].Action != audit.ActionReconciliationAcceptedWithComment {
		t.Fatalf("audit action = %q, want %q", auditService.events[0].Action, audit.ActionReconciliationAcceptedWithComment)
	}
}

func hasTeamleadItem(items []TeamleadItemRecord, stage string, issueType string) bool {
	for _, item := range items {
		if item.Stage == stage && item.IssueType == issueType {
			return true
		}
	}
	return false
}

type fakeStore struct {
	recalculateCalled                        bool
	recalculateRecord                        RecalculateTraderInboundRecord
	outboundRecalculateCalled                bool
	outboundRecalculateRecord                RecalculateTraderOutboundRecord
	teamleadPeriodRecalculateRecords         []RecalculateTeamleadPeriodInboundRecord
	teamleadPeriodOutboundRecalculateRecords []RecalculateTeamleadPeriodOutboundRecord
	teamleadCurrentRecalculateRecord         RecalculateTeamleadCurrentRecord
	latestCalled                             bool
	latestRun                                Run
	latestErr                                error
	latestOutboundCalled                     bool
	latestOutboundRun                        Run
	latestOutboundErr                        error
	acceptRecord                             AcceptTraderInboundRecord
	acceptedRun                              Run
	acceptOutboundRecord                     AcceptTraderOutboundRecord
	acceptedOutboundRun                      Run
	acceptTeamleadCurrentRecord              AcceptTeamleadCurrentRecord
	acceptedTeamleadCurrentRun               Run
	teamleadCurrentRuns                      []Run
	teamleadCurrentRun                       Run
	activeTeamleadScopes                     []TeamleadInboundPeriodScope
	activeTeamleadOutboundScopes             []TeamleadOutboundPeriodScope
	items                                    []Item
	createdTeamleadAnalysis                  CreateTeamleadAnalysisRecord
	teamleadRun                              TeamleadRun
	teamleadRuns                             []TeamleadRun
	teamleadItems                            []TeamleadItem
	teamleadItemsFilters                     TeamleadItemFilters
	queuedTeamleadApplyRecord                QueueTeamleadApplyRecord
	queuedTeamleadRun                        TeamleadRun
	rejectedTeamleadRecord                   RejectTeamleadReconciliationRecord
	rejectedTeamleadRun                      TeamleadRun
	appliedTeamleadTeamID                    int64
	appliedTeamleadRunID                     int64
	appliedTeamleadRun                       TeamleadRun
	teamleadTraders                          []TeamleadTraderMatch
	teamleadRequisites                       []TeamleadRequisiteMatch
	teamleadExternalOrders                   []TeamleadExternalOrderSnapshot
	teamleadExternalOrdersInPeriod           []TeamleadExternalOrderSnapshot
	teamleadInboundTurnover                  TeamleadTurnoverSnapshot
	teamleadOutboundTransfers                TeamleadTurnoverSnapshot
}

func (s *fakeStore) RecalculateTraderInbound(ctx context.Context, record RecalculateTraderInboundRecord) (Run, error) {
	s.recalculateCalled = true
	s.recalculateRecord = record
	return Run{
		ID:        51,
		TeamID:    record.TeamID,
		Type:      TypeTraderShiftInbound,
		ScopeType: imports.ScopeTypeTraderShift,
		ShiftID:   &record.ShiftID,
		TraderID:  &record.TraderID,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) RecalculateTraderOutbound(ctx context.Context, record RecalculateTraderOutboundRecord) (Run, error) {
	s.outboundRecalculateCalled = true
	s.outboundRecalculateRecord = record
	return Run{
		ID:        61,
		TeamID:    record.TeamID,
		Type:      TypeTraderShiftOutbound,
		ScopeType: imports.ScopeTypeTraderShift,
		ShiftID:   &record.ShiftID,
		TraderID:  &record.TraderID,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) RecalculateTeamleadPeriodInbound(ctx context.Context, record RecalculateTeamleadPeriodInboundRecord) (Run, error) {
	s.teamleadPeriodRecalculateRecords = append(s.teamleadPeriodRecalculateRecords, record)
	return Run{
		ID:                 71,
		TeamID:             record.TeamID,
		Type:               TypeTeamleadPeriodInbound,
		ScopeType:          imports.ScopeTypeTeamleadPeriod,
		AccountingPeriodID: &record.AccountingPeriodID,
		ImportBatchID:      record.ImportBatchID,
		Status:             StatusMatched,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) RecalculateTeamleadPeriodOutbound(ctx context.Context, record RecalculateTeamleadPeriodOutboundRecord) (Run, error) {
	s.teamleadPeriodOutboundRecalculateRecords = append(s.teamleadPeriodOutboundRecalculateRecords, record)
	return Run{
		ID:                 81,
		TeamID:             record.TeamID,
		Type:               TypeTeamleadPeriodOutbound,
		ScopeType:          imports.ScopeTypeTeamleadPeriod,
		AccountingPeriodID: &record.AccountingPeriodID,
		ImportBatchID:      record.ImportBatchID,
		Status:             StatusMatched,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) RecalculateTeamleadCurrent(ctx context.Context, record RecalculateTeamleadCurrentRecord) (Run, error) {
	s.teamleadCurrentRecalculateRecord = record
	runType := TypeTeamleadPeriodInbound
	if record.Direction == imports.DirectionOutbound {
		runType = TypeTeamleadPeriodOutbound
	}
	return Run{
		ID:        91,
		TeamID:    record.TeamID,
		Type:      runType,
		ScopeType: imports.ScopeTypeTeamleadPeriod,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTraderInbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	s.latestCalled = true
	if s.latestErr != nil {
		return Run{}, s.latestErr
	}
	return s.latestRun, nil
}

func (s *fakeStore) LatestTraderInboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error) {
	s.latestCalled = true
	if s.latestErr != nil {
		return Run{}, s.latestErr
	}
	return s.latestRun, nil
}

func (s *fakeStore) GetTraderInboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error) {
	s.latestCalled = true
	if s.latestErr != nil {
		return Run{}, s.latestErr
	}
	run := s.latestRun
	run.ID = runID
	return run, nil
}

func (s *fakeStore) LatestTraderOutbound(ctx context.Context, teamID int64, traderID int64, shiftID int64) (Run, error) {
	s.latestOutboundCalled = true
	if s.latestOutboundErr != nil {
		return Run{}, s.latestOutboundErr
	}
	return s.latestOutboundRun, nil
}

func (s *fakeStore) LatestTraderOutboundByShift(ctx context.Context, teamID int64, shiftID int64) (Run, error) {
	s.latestOutboundCalled = true
	if s.latestOutboundErr != nil {
		return Run{}, s.latestOutboundErr
	}
	return s.latestOutboundRun, nil
}

func (s *fakeStore) GetTraderOutboundRun(ctx context.Context, teamID int64, traderID int64, runID int64) (Run, error) {
	s.latestOutboundCalled = true
	if s.latestOutboundErr != nil {
		return Run{}, s.latestOutboundErr
	}
	run := s.latestOutboundRun
	run.ID = runID
	return run, nil
}

func (s *fakeStore) LatestTeamleadPeriodInbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	return Run{
		ID:                 71,
		TeamID:             teamID,
		Type:               TypeTeamleadPeriodInbound,
		ScopeType:          imports.ScopeTypeTeamleadPeriod,
		AccountingPeriodID: &accountingPeriodID,
		Status:             StatusMatched,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTeamleadPeriodOutbound(ctx context.Context, teamID int64, accountingPeriodID int64) (Run, error) {
	return Run{
		ID:                 81,
		TeamID:             teamID,
		Type:               TypeTeamleadPeriodOutbound,
		ScopeType:          imports.ScopeTypeTeamleadPeriod,
		AccountingPeriodID: &accountingPeriodID,
		Status:             StatusMatched,
		CreatedAt:          time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTeamleadInbound(ctx context.Context, teamID int64, actorID int64) (Run, error) {
	return Run{
		ID:        71,
		TeamID:    teamID,
		Type:      TypeTeamleadPeriodInbound,
		ScopeType: imports.ScopeTypeTeamleadPeriod,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) LatestTeamleadOutbound(ctx context.Context, teamID int64, actorID int64) (Run, error) {
	return Run{
		ID:        81,
		TeamID:    teamID,
		Type:      TypeTeamleadPeriodOutbound,
		ScopeType: imports.ScopeTypeTeamleadPeriod,
		Status:    StatusMatched,
		CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeStore) ListTeamleadCurrentRuns(ctx context.Context, teamID int64, actorID int64, direction string, page pagination.Params) (pagination.Result[Run], error) {
	return pagination.FromSlice(s.teamleadCurrentRuns, page), nil
}

func (s *fakeStore) GetTeamleadCurrentRun(ctx context.Context, teamID int64, actorID int64, direction string, runID int64) (Run, error) {
	return s.teamleadCurrentRun, nil
}

func (s *fakeStore) ListItems(ctx context.Context, runID int64, filters ItemFilters, page pagination.Params) (pagination.Result[Item], error) {
	return pagination.FromSlice(s.items, page), nil
}

func (s *fakeStore) ListActiveTeamleadInboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadInboundPeriodScope, error) {
	return s.activeTeamleadScopes, nil
}

func (s *fakeStore) ListActiveTeamleadOutboundPeriodScopes(ctx context.Context, teamID int64) ([]TeamleadOutboundPeriodScope, error) {
	return s.activeTeamleadOutboundScopes, nil
}

func (s *fakeStore) AcceptTraderInbound(ctx context.Context, record AcceptTraderInboundRecord) (Run, error) {
	s.acceptRecord = record
	return s.acceptedRun, nil
}

func (s *fakeStore) AcceptTraderOutbound(ctx context.Context, record AcceptTraderOutboundRecord) (Run, error) {
	s.acceptOutboundRecord = record
	return s.acceptedOutboundRun, nil
}

func (s *fakeStore) AcceptTeamleadCurrent(ctx context.Context, record AcceptTeamleadCurrentRecord) (Run, error) {
	s.acceptTeamleadCurrentRecord = record
	return s.acceptedTeamleadCurrentRun, nil
}

func (s *fakeStore) CreateTeamleadAnalysis(ctx context.Context, record CreateTeamleadAnalysisRecord) (TeamleadRun, error) {
	s.createdTeamleadAnalysis = record
	return TeamleadRun{
		ID:                  500,
		TeamID:              record.TeamID,
		DateFrom:            record.DateFrom,
		DateTo:              record.DateTo,
		Status:              record.Status,
		CreatedBy:           record.ActorID,
		MismatchCount:       record.MismatchCount,
		ConflictCount:       record.ConflictCount,
		BlockedCount:        record.BlockedCount,
		PipelineJSON:        record.PipelineJSON,
		InboundSummaryJSON:  record.InboundSummary,
		OutboundSummaryJSON: record.OutboundSummary,
		PreviewJSON:         record.PreviewJSON,
		CreatedAt:           time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
		AnalyzedAt:          timeValuePtr(time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)),
	}, nil
}

func (s *fakeStore) GetTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error) {
	if s.teamleadRun.ID == 0 {
		return TeamleadRun{}, ErrRunNotFound
	}
	return s.teamleadRun, nil
}

func (s *fakeStore) ListTeamleadReconciliations(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[TeamleadRun], error) {
	return pagination.FromSlice(s.teamleadRuns, page), nil
}

func (s *fakeStore) ListTeamleadReconciliationItems(ctx context.Context, teamID int64, runID int64, filters TeamleadItemFilters, page pagination.Params) (pagination.Result[TeamleadItem], error) {
	s.teamleadItemsFilters = filters
	return pagination.FromSlice(s.teamleadItems, page), nil
}

func (s *fakeStore) QueueTeamleadReconciliationApply(ctx context.Context, record QueueTeamleadApplyRecord) (TeamleadRun, error) {
	s.queuedTeamleadApplyRecord = record
	return s.queuedTeamleadRun, nil
}

func (s *fakeStore) RejectTeamleadReconciliation(ctx context.Context, record RejectTeamleadReconciliationRecord) (TeamleadRun, error) {
	s.rejectedTeamleadRecord = record
	return s.rejectedTeamleadRun, nil
}

func (s *fakeStore) ApplyQueuedTeamleadReconciliation(ctx context.Context, teamID int64, runID int64) (TeamleadRun, error) {
	s.appliedTeamleadTeamID = teamID
	s.appliedTeamleadRunID = runID
	return s.appliedTeamleadRun, nil
}

func (s *fakeStore) ListTeamleadReconciliationTraders(ctx context.Context, teamID int64) ([]TeamleadTraderMatch, error) {
	return s.teamleadTraders, nil
}

func (s *fakeStore) ListTeamleadReconciliationRequisites(ctx context.Context, teamID int64) ([]TeamleadRequisiteMatch, error) {
	return s.teamleadRequisites, nil
}

func (s *fakeStore) ListTeamleadReconciliationExternalOrders(ctx context.Context, teamID int64, direction string, innerIDs []string) ([]TeamleadExternalOrderSnapshot, error) {
	result := make([]TeamleadExternalOrderSnapshot, 0, len(s.teamleadExternalOrders))
	allowed := map[string]struct{}{}
	for _, innerID := range innerIDs {
		allowed[innerID] = struct{}{}
	}
	for _, order := range s.teamleadExternalOrders {
		if order.Direction != direction {
			continue
		}
		if _, ok := allowed[order.ExternalInnerID]; ok {
			result = append(result, order)
		}
	}
	return result, nil
}

func (s *fakeStore) ListTeamleadReconciliationExternalOrdersInPeriod(ctx context.Context, teamID int64, direction string, dateFrom time.Time, dateTo time.Time) ([]TeamleadExternalOrderSnapshot, error) {
	result := make([]TeamleadExternalOrderSnapshot, 0, len(s.teamleadExternalOrdersInPeriod))
	for _, order := range s.teamleadExternalOrdersInPeriod {
		if order.Direction == direction {
			result = append(result, order)
		}
	}
	return result, nil
}

func (s *fakeStore) CalculateTeamleadV2InboundTurnover(ctx context.Context, teamID int64, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error) {
	return s.teamleadInboundTurnover, nil
}

func (s *fakeStore) CalculateTeamleadV2OutboundTransfers(ctx context.Context, teamID int64, dateFrom time.Time, dateTo time.Time) (TeamleadTurnoverSnapshot, error) {
	return s.teamleadOutboundTransfers, nil
}

type fakeAuditService struct {
	events []audit.Event
}

func (s *fakeAuditService) Write(ctx context.Context, event audit.Event) error {
	s.events = append(s.events, event)
	return nil
}

type fakeTeamleadApplyScheduler struct {
	teamID int64
	runID  int64
}

func (s *fakeTeamleadApplyScheduler) EnqueueTeamleadApply(teamID int64, runID int64) {
	s.teamID = teamID
	s.runID = runID
}

func int64Ptr(value int64) *int64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func timeValuePtr(value time.Time) *time.Time {
	return &value
}
