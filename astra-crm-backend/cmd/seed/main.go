package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

const demoPassword = "demo123"

var (
	demoPeriodFrom = dateUTC(2026, 6, 17)
	demoPeriodTo   = dateUTC(2026, 6, 23)
)

type requisiteSeed struct {
	Phone           string
	MethodType      string
	BankCode        string
	Proxy           string
	EmployeeComment string
	CardNumber      string
	HolderName      string
}

type demoRequisite struct {
	ID         int64
	Phone      string
	MethodType string
	BankCode   string
	BankName   string
	CardNumber string
	HolderName string
}

type orderSeed struct {
	Direction         string
	ExternalID        string
	InnerID           string
	WorkerName        string
	TraderID          int64
	RequisiteID       int64
	RequisitePhone    string
	RequisiteRaw      string
	RequisiteExternal string
	DeviceName        string
	MethodType        string
	MethodName        string
	AmountMinor       int64
	RawStatus         string
	NormalizedStatus  string
	CreatedAt         time.Time
	Comment           string
	Initials          string
}

type importScope struct {
	ScopeType          string
	Direction          string
	UploadedBy         int64
	ShiftID            *int64
	AccountingPeriodID *int64
	TraderID           *int64
	FileName           string
	FileHash           string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer conn.Close(context.Background())

	passwordHash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer tx.Rollback(ctx)

	teamID, err := upsertTeam(ctx, tx)
	if err != nil {
		return err
	}
	teamleadID, err := upsertUser(ctx, tx, teamID, "teamlead", "teamlead", passwordHash)
	if err != nil {
		return err
	}

	traderIvanID, err := upsertTrader(ctx, tx, teamID, "trader_ivan", passwordHash, 75, "Bliss_OP2")
	if err != nil {
		return err
	}
	traderAnnaID, err := upsertTrader(ctx, tx, teamID, "trader_anna", passwordHash, 65, "Bliss_OP5")
	if err != nil {
		return err
	}
	traderOlegID, err := upsertTrader(ctx, tx, teamID, "trader_oleg", passwordHash, 50, "Bliss_OP7")
	if err != nil {
		return err
	}

	ivanReqs, err := upsertDemoRequisites(ctx, tx, teamID, teamleadID, []requisiteSeed{
		{
			Phone:           "+79021001001",
			MethodType:      "SBP",
			BankCode:        "sber",
			Proxy:           "192.168.10.11:8080",
			EmployeeComment: "Seed v2: активная смена Иван, Сбер",
			CardNumber:      "427638001001",
			HolderName:      "IVAN DEMO",
		},
		{
			Phone:           "+79021001002",
			MethodType:      "C2C",
			BankCode:        "tbank",
			Proxy:           "192.168.10.12:8080",
			EmployeeComment: "Seed v2: активная смена Иван, Т-Банк",
			CardNumber:      "553691001002",
			HolderName:      "IVAN DEMO",
		},
		{
			Phone:           "+79021001003",
			MethodType:      "SBP",
			BankCode:        "vtb",
			Proxy:           "192.168.10.13:8080",
			EmployeeComment: "Seed v2: активная смена Иван, ВТБ",
			CardNumber:      "220024001003",
			HolderName:      "IVAN DEMO",
		},
	})
	if err != nil {
		return err
	}

	annaReqs, err := upsertDemoRequisites(ctx, tx, teamID, teamleadID, []requisiteSeed{
		{
			Phone:           "+79021002001",
			MethodType:      "SBP",
			BankCode:        "sber",
			Proxy:           "192.168.20.21:8080",
			EmployeeComment: "Seed v2: закрытая смена Анна, расхождение",
			CardNumber:      "427638002001",
			HolderName:      "ANNA DEMO",
		},
		{
			Phone:           "+79021002002",
			MethodType:      "C2C",
			BankCode:        "alfa",
			Proxy:           "192.168.20.22:8080",
			EmployeeComment: "Seed v2: закрытая смена Анна, заблокирован остаток",
			CardNumber:      "521178002002",
			HolderName:      "ANNA DEMO",
		},
	})
	if err != nil {
		return err
	}

	olegReqs, err := upsertDemoRequisites(ctx, tx, teamID, teamleadID, []requisiteSeed{
		{
			Phone:           "+79021003001",
			MethodType:      "SBP",
			BankCode:        "raif",
			Proxy:           "192.168.30.31:8080",
			EmployeeComment: "Seed v2: резервный реквизит Олег",
			CardNumber:      "462729003001",
			HolderName:      "OLEG DEMO",
		},
	})
	if err != nil {
		return err
	}

	periodID, err := ensureAccountingPeriod(ctx, tx, teamID, teamleadID)
	if err != nil {
		return err
	}

	if err := seedActiveShiftScenario(ctx, tx, teamID, teamleadID, traderIvanID, ivanReqs); err != nil {
		return err
	}
	annaShiftID, annaInboundBatchID, annaOutboundBatchID, err := seedClosedDiscrepancyScenario(ctx, tx, teamID, teamleadID, traderAnnaID, annaReqs, periodID)
	if err != nil {
		return err
	}
	if err := seedReserveAssignment(ctx, tx, teamID, teamleadID, traderOlegID, olegReqs[0]); err != nil {
		return err
	}
	if err := seedTeamleadReconciliationV2(ctx, tx, teamID, teamleadID, periodID, annaShiftID, annaInboundBatchID, annaOutboundBatchID, traderAnnaID, annaReqs); err != nil {
		return err
	}
	if err := seedAuditEvents(ctx, tx, teamID, teamleadID, traderIvanID, traderAnnaID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}

	fmt.Println("seed applied")
	fmt.Println("demo users: teamlead / trader_ivan / trader_anna / trader_oleg")
	fmt.Println("demo password: demo123")
	return nil
}

func seedActiveShiftScenario(ctx context.Context, tx pgx.Tx, teamID int64, teamleadID int64, traderID int64, reqs []demoRequisite) error {
	startedAt := timeUTC(2026, 6, 18, 9, 0)
	shiftID, err := ensureShift(ctx, tx, teamID, traderID, startedAt, nil, "open", "matched", "matched", nil)
	if err != nil {
		return err
	}

	turnovers := []struct {
		InboundMinor  int64
		OutboundMinor int64
		ClosingMinor  int64
	}{
		{InboundMinor: 3300000, OutboundMinor: 600000, ClosingMinor: 2700000},
		{InboundMinor: 2850000, OutboundMinor: 400000, ClosingMinor: 2450000},
		{InboundMinor: 3100000, OutboundMinor: 200000, ClosingMinor: 2900000},
	}

	shiftRequisiteIDs := make([]int64, 0, len(reqs))
	for index, req := range reqs {
		assignmentID, err := ensureActiveAssignment(ctx, tx, teamID, req.ID, traderID, teamleadID, demoPeriodTo, 5000000, "Seed v2: активная смена")
		if err != nil {
			return err
		}
		takenAt := startedAt.Add(time.Duration(index*12) * time.Minute)
		shiftRequisiteID, err := ensureShiftRequisite(ctx, tx, teamID, shiftID, traderID, req, assignmentID, "active", turnovers[index].InboundMinor, turnovers[index].OutboundMinor, turnovers[index].ClosingMinor, takenAt, nil, "not_checked", nil)
		if err != nil {
			return err
		}
		shiftRequisiteIDs = append(shiftRequisiteIDs, shiftRequisiteID)
		if err := ensureTurnoverEntry(ctx, tx, teamID, shiftID, shiftRequisiteID, req.ID, traderID, turnovers[index].InboundMinor, traderID, "Seed v2: накопленный входящий оборот"); err != nil {
			return err
		}
		if err := refreshRequisiteAggregates(ctx, tx, req.ID); err != nil {
			return err
		}
	}

	inboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:  "trader_shift",
		Direction:  "inbound",
		UploadedBy: traderID,
		ShiftID:    &shiftID,
		TraderID:   &traderID,
		FileName:   "seed-ivan-active-inbound.csv",
		FileHash:   "seed-v2-ivan-active-inbound",
	}, []orderSeed{
		activeOrder("inbound", "910101", "seed-in-ivan-001", "Bliss_OP2", traderID, reqs[0], 3300000, "hand_success", timeUTC(2026, 6, 18, 9, 25), "A1"),
		activeOrder("inbound", "910102", "seed-in-ivan-002", "Bliss_OP2", traderID, reqs[1], 2850000, "hand_success", timeUTC(2026, 6, 18, 10, 15), "A2"),
		activeOrder("inbound", "910103", "seed-in-ivan-003", "Bliss_OP2", traderID, reqs[2], 3100000, "corrected", timeUTC(2026, 6, 18, 11, 5), "A3"),
	})
	if err != nil {
		return err
	}
	outboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:  "trader_shift",
		Direction:  "outbound",
		UploadedBy: traderID,
		ShiftID:    &shiftID,
		TraderID:   &traderID,
		FileName:   "seed-ivan-active-outbound.csv",
		FileHash:   "seed-v2-ivan-active-outbound",
	}, []orderSeed{
		activeOrder("outbound", "920101", "seed-out-ivan-001", "Bliss_OP2", traderID, reqs[0], 1200000, "paid", timeUTC(2026, 6, 18, 12, 20), "P1"),
	})
	if err != nil {
		return err
	}

	if _, err := ensurePayoutOrder(ctx, tx, teamID, shiftID, traderID, "Сбер", "+79990001001", 1200000, "paid", []payoutTransferSeed{
		{SourceShiftRequisiteID: shiftRequisiteIDs[0], SourceRequisiteID: reqs[0].ID, AmountMinor: 600000, CreatedBy: traderID, Comment: "Seed v2: выплата из первого река"},
		{SourceShiftRequisiteID: shiftRequisiteIDs[1], SourceRequisiteID: reqs[1].ID, AmountMinor: 600000, CreatedBy: traderID, Comment: "Seed v2: выплата из второго река"},
	}); err != nil {
		return err
	}

	if _, err := ensureReconciliationRun(ctx, tx, reconciliationRunSeed{
		TeamID:              teamID,
		Type:                "trader_shift_inbound",
		ScopeType:           "trader_shift",
		ShiftID:             &shiftID,
		TraderID:            &traderID,
		ImportBatchID:       &inboundBatchID,
		ExpectedAmountMinor: 9250000,
		ActualAmountMinor:   9250000,
		DiffAmountMinor:     0,
		SuccessAmountMinor:  9250000,
		SuccessCount:        3,
		TotalAmountMinor:    9250000,
		TotalCount:          3,
		Status:              "matched",
	}); err != nil {
		return err
	}
	if _, err := ensureReconciliationRun(ctx, tx, reconciliationRunSeed{
		TeamID:              teamID,
		Type:                "trader_shift_outbound",
		ScopeType:           "trader_shift",
		ShiftID:             &shiftID,
		TraderID:            &traderID,
		ImportBatchID:       &outboundBatchID,
		ExpectedAmountMinor: 1200000,
		ActualAmountMinor:   1200000,
		DiffAmountMinor:     0,
		SuccessAmountMinor:  1200000,
		SuccessCount:        1,
		TotalAmountMinor:    1200000,
		TotalCount:          1,
		Status:              "matched",
	}); err != nil {
		return err
	}

	if err := ensureAudit(ctx, tx, teamID, traderID, audit.ActionShiftCreated, "trader_shift", shiftID, map[string]any{"status": "open"}, "Seed v2: активная смена"); err != nil {
		return err
	}
	if err := ensureAudit(ctx, tx, teamID, traderID, audit.ActionImportApplied, "import_batch", inboundBatchID, map[string]any{"direction": "inbound", "scope": "trader_shift"}, "Seed v2: импорт входов активной смены"); err != nil {
		return err
	}
	if err := ensureAudit(ctx, tx, teamID, traderID, audit.ActionManualPayoutTransferAdded, "manual_payout", shiftID, map[string]any{"amountMinor": 1200000}, "Seed v2: выплата закрыта переводами"); err != nil {
		return err
	}
	return nil
}

func seedClosedDiscrepancyScenario(ctx context.Context, tx pgx.Tx, teamID int64, teamleadID int64, traderID int64, reqs []demoRequisite, periodID int64) (int64, int64, int64, error) {
	startedAt := timeUTC(2026, 6, 17, 9, 0)
	endedAt := timeUTC(2026, 6, 17, 20, 30)
	closeComment := "Seed v2: расхождение принято тимлидом после сверки"
	shiftID, err := ensureShift(ctx, tx, teamID, traderID, startedAt, &endedAt, "closed_with_discrepancy", "accepted_with_comment", "accepted_with_comment", &closeComment)
	if err != nil {
		return 0, 0, 0, err
	}

	shiftRequisiteIDs := make([]int64, 0, len(reqs))
	for index, req := range reqs {
		assignmentID, err := ensureWorkedAssignment(ctx, tx, teamID, req.ID, traderID, teamleadID, demoPeriodFrom, startedAt, endedAt, 4500000, "Seed v2: закрытая смена с расхождением")
		if err != nil {
			return 0, 0, 0, err
		}
		status := "worked_discrepancy"
		if index == 1 {
			status = "blocked"
		}
		releasedAt := endedAt.Add(-time.Duration(index) * 5 * time.Minute)
		inbound := []int64{2000000, 1500000}[index]
		outbound := []int64{1200000, 700000}[index]
		closing := []int64{800000, 800000}[index]
		shiftRequisiteID, err := ensureShiftRequisite(ctx, tx, teamID, shiftID, traderID, req, assignmentID, status, inbound, outbound, closing, startedAt.Add(time.Duration(index*10)*time.Minute), &releasedAt, "tl_accepted", nil)
		if err != nil {
			return 0, 0, 0, err
		}
		shiftRequisiteIDs = append(shiftRequisiteIDs, shiftRequisiteID)
		if err := ensureTurnoverEntry(ctx, tx, teamID, shiftID, shiftRequisiteID, req.ID, traderID, inbound, traderID, "Seed v2: финальный оборот закрытой смены"); err != nil {
			return 0, 0, 0, err
		}
		if err := refreshRequisiteAggregates(ctx, tx, req.ID); err != nil {
			return 0, 0, 0, err
		}
	}

	inboundOrders := []orderSeed{
		activeOrder("inbound", "910201", "seed-in-anna-001", "Bliss_OP5", traderID, reqs[0], 2000000, "hand_success", timeUTC(2026, 6, 17, 9, 45), "B1"),
		activeOrder("inbound", "910202", "seed-in-anna-002", "Bliss_OP5", traderID, reqs[1], 1400000, "hand_success", timeUTC(2026, 6, 17, 11, 10), "B2"),
		activeOrder("inbound", "910203", "seed-in-anna-003", "Bliss_OP5", traderID, reqs[1], 300000, "auto_decline", timeUTC(2026, 6, 17, 13, 35), "B3"),
	}
	inboundOrders[2].NormalizedStatus = "failed"
	inboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:  "trader_shift",
		Direction:  "inbound",
		UploadedBy: traderID,
		ShiftID:    &shiftID,
		TraderID:   &traderID,
		FileName:   "seed-anna-closed-inbound.csv",
		FileHash:   "seed-v2-anna-closed-inbound",
	}, inboundOrders)
	if err != nil {
		return 0, 0, 0, err
	}
	outboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:  "trader_shift",
		Direction:  "outbound",
		UploadedBy: traderID,
		ShiftID:    &shiftID,
		TraderID:   &traderID,
		FileName:   "seed-anna-closed-outbound.csv",
		FileHash:   "seed-v2-anna-closed-outbound",
	}, []orderSeed{
		activeOrder("outbound", "920201", "seed-out-anna-001", "Bliss_OP5", traderID, reqs[0], 1200000, "paid", timeUTC(2026, 6, 17, 15, 10), "Q1"),
		activeOrder("outbound", "920202", "seed-out-anna-002", "Bliss_OP5", traderID, reqs[1], 800000, "paid", timeUTC(2026, 6, 17, 17, 40), "Q2"),
	})
	if err != nil {
		return 0, 0, 0, err
	}

	if _, err := ensurePayoutOrder(ctx, tx, teamID, shiftID, traderID, "Сбер", "+79990002001", 1200000, "paid", []payoutTransferSeed{
		{SourceShiftRequisiteID: shiftRequisiteIDs[0], SourceRequisiteID: reqs[0].ID, AmountMinor: 1200000, CreatedBy: traderID, Comment: "Seed v2: выплата закрытой смены"},
	}); err != nil {
		return 0, 0, 0, err
	}
	if _, err := ensurePayoutOrder(ctx, tx, teamID, shiftID, traderID, "Альфа-Банк", "+79990002002", 700000, "paid", []payoutTransferSeed{
		{SourceShiftRequisiteID: shiftRequisiteIDs[1], SourceRequisiteID: reqs[1].ID, AmountMinor: 400000, CreatedBy: traderID, Comment: "Seed v2: частичный перевод"},
		{SourceShiftRequisiteID: shiftRequisiteIDs[0], SourceRequisiteID: reqs[0].ID, AmountMinor: 300000, CreatedBy: traderID, Comment: "Seed v2: добивка выплаты"},
	}); err != nil {
		return 0, 0, 0, err
	}

	inboundRunID, err := ensureReconciliationRun(ctx, tx, reconciliationRunSeed{
		TeamID:              teamID,
		Type:                "trader_shift_inbound",
		ScopeType:           "trader_shift",
		ShiftID:             &shiftID,
		TraderID:            &traderID,
		ImportBatchID:       &inboundBatchID,
		ExpectedAmountMinor: 3500000,
		ActualAmountMinor:   3400000,
		DiffAmountMinor:     -100000,
		SuccessAmountMinor:  3400000,
		SuccessCount:        2,
		FailedAmountMinor:   300000,
		FailedCount:         1,
		TotalAmountMinor:    3700000,
		TotalCount:          3,
		Status:              "accepted_with_comment",
		Comment:             strPtr("Seed v2: принято расхождение входящих оборотов 1 000 RUB"),
		ConfirmedBy:         &teamleadID,
	})
	if err != nil {
		return 0, 0, 0, err
	}
	if err := replaceReconciliationItems(ctx, tx, inboundRunID, []reconciliationItemSeed{
		{IssueType: "amount_mismatch", ExternalInnerID: "seed-in-anna-002", TeamleadValue: map[string]any{"csvAmountMinor": 1400000}, TraderValue: map[string]any{"turnoverAmountMinor": 1500000}, Message: "Seed v2: входящий оборот отличается от CSV"},
	}); err != nil {
		return 0, 0, 0, err
	}

	outboundRunID, err := ensureReconciliationRun(ctx, tx, reconciliationRunSeed{
		TeamID:              teamID,
		Type:                "trader_shift_outbound",
		ScopeType:           "trader_shift",
		ShiftID:             &shiftID,
		TraderID:            &traderID,
		ImportBatchID:       &outboundBatchID,
		ExpectedAmountMinor: 1900000,
		ActualAmountMinor:   2000000,
		DiffAmountMinor:     100000,
		SuccessAmountMinor:  2000000,
		SuccessCount:        2,
		TotalAmountMinor:    2000000,
		TotalCount:          2,
		Status:              "accepted_with_comment",
		Comment:             strPtr("Seed v2: принято расхождение выплат 1 000 RUB"),
		ConfirmedBy:         &teamleadID,
	})
	if err != nil {
		return 0, 0, 0, err
	}
	if err := replaceReconciliationItems(ctx, tx, outboundRunID, []reconciliationItemSeed{
		{IssueType: "amount_mismatch", ExternalInnerID: "seed-out-anna-002", TeamleadValue: map[string]any{"csvAmountMinor": 800000}, TraderValue: map[string]any{"manualTransfersAmountMinor": 700000}, Message: "Seed v2: сумма выплат отличается от переводов"},
	}); err != nil {
		return 0, 0, 0, err
	}

	if _, err := ensureReconciliationRun(ctx, tx, reconciliationRunSeed{
		TeamID:              teamID,
		Type:                "teamlead_period_inbound",
		ScopeType:           "teamlead_period",
		AccountingPeriodID:  &periodID,
		ImportBatchID:       &inboundBatchID,
		ExpectedAmountMinor: 3500000,
		ActualAmountMinor:   3400000,
		DiffAmountMinor:     -100000,
		SuccessAmountMinor:  3400000,
		SuccessCount:        2,
		FailedAmountMinor:   300000,
		FailedCount:         1,
		TotalAmountMinor:    3700000,
		TotalCount:          3,
		Status:              "accepted_with_comment",
		Comment:             strPtr("Seed v2: period inbound mismatch accepted"),
		ConfirmedBy:         &teamleadID,
	}); err != nil {
		return 0, 0, 0, err
	}
	if _, err := ensureReconciliationRun(ctx, tx, reconciliationRunSeed{
		TeamID:              teamID,
		Type:                "teamlead_period_outbound",
		ScopeType:           "teamlead_period",
		AccountingPeriodID:  &periodID,
		ImportBatchID:       &outboundBatchID,
		ExpectedAmountMinor: 1900000,
		ActualAmountMinor:   2000000,
		DiffAmountMinor:     100000,
		SuccessAmountMinor:  2000000,
		SuccessCount:        2,
		TotalAmountMinor:    2000000,
		TotalCount:          2,
		Status:              "accepted_with_comment",
		Comment:             strPtr("Seed v2: period outbound mismatch accepted"),
		ConfirmedBy:         &teamleadID,
	}); err != nil {
		return 0, 0, 0, err
	}

	if err := ensureAudit(ctx, tx, teamID, traderID, audit.ActionImportApplied, "import_batch", inboundBatchID, map[string]any{"direction": "inbound", "scope": "trader_shift"}, "Seed v2: импорт закрытой смены"); err != nil {
		return 0, 0, 0, err
	}
	if err := ensureAudit(ctx, tx, teamID, teamleadID, audit.ActionReconciliationAcceptedWithComment, "reconciliation_run", inboundRunID, map[string]any{"diffAmountMinor": -100000}, "Seed v2: принято входящее расхождение"); err != nil {
		return 0, 0, 0, err
	}
	if err := ensureAudit(ctx, tx, teamID, teamleadID, audit.ActionShiftClosed, "trader_shift", shiftID, map[string]any{"status": "closed_with_discrepancy"}, closeComment); err != nil {
		return 0, 0, 0, err
	}

	return shiftID, inboundBatchID, outboundBatchID, nil
}

func seedReserveAssignment(ctx context.Context, tx pgx.Tx, teamID int64, teamleadID int64, traderID int64, req demoRequisite) error {
	_, err := ensureActiveAssignment(ctx, tx, teamID, req.ID, traderID, teamleadID, demoPeriodTo, 2500000, "Seed v2: резервный реквизит")
	return err
}

func seedTeamleadReconciliationV2(ctx context.Context, tx pgx.Tx, teamID int64, teamleadID int64, periodID int64, annaShiftID int64, traderInboundBatchID int64, traderOutboundBatchID int64, traderAnnaID int64, annaReqs []demoRequisite) error {
	currentInboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:  "teamlead_period",
		Direction:  "inbound",
		UploadedBy: teamleadID,
		FileName:   "seed-teamlead-current-inbound.csv",
		FileHash:   "seed-v2-teamlead-current-inbound",
	}, []orderSeed{
		activeOrder("inbound", "910201", "seed-in-anna-001", "Bliss_OP5", traderAnnaID, annaReqs[0], 2000000, "hand_success", timeUTC(2026, 6, 17, 9, 45), "B1"),
		activeOrder("inbound", "910202", "seed-in-anna-002", "Bliss_OP5", traderAnnaID, annaReqs[1], 1500000, "hand_success", timeUTC(2026, 6, 17, 11, 10), "B2"),
	})
	if err != nil {
		return err
	}
	currentOutboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:  "teamlead_period",
		Direction:  "outbound",
		UploadedBy: teamleadID,
		FileName:   "seed-teamlead-current-outbound.csv",
		FileHash:   "seed-v2-teamlead-current-outbound",
	}, []orderSeed{
		activeOrder("outbound", "920201", "seed-out-anna-001", "Bliss_OP5", traderAnnaID, annaReqs[0], 1200000, "paid", timeUTC(2026, 6, 17, 15, 10), "Q1"),
		activeOrder("outbound", "920202", "seed-out-anna-002", "Bliss_OP5", traderAnnaID, annaReqs[1], 800000, "paid", timeUTC(2026, 6, 17, 17, 40), "Q2"),
	})
	if err != nil {
		return err
	}

	periodInboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:          "teamlead_period",
		Direction:          "inbound",
		UploadedBy:         teamleadID,
		AccountingPeriodID: &periodID,
		FileName:           "seed-teamlead-period-inbound.csv",
		FileHash:           "seed-v2-teamlead-period-inbound",
	}, []orderSeed{
		activeOrder("inbound", "910201", "seed-in-anna-001", "Bliss_OP5", traderAnnaID, annaReqs[0], 2000000, "hand_success", timeUTC(2026, 6, 17, 9, 45), "B1"),
		activeOrder("inbound", "910202", "seed-in-anna-002", "Bliss_OP5", traderAnnaID, annaReqs[1], 1500000, "hand_success", timeUTC(2026, 6, 17, 11, 10), "B2"),
	})
	if err != nil {
		return err
	}
	periodOutboundBatchID, _, err := ensureOrdersImport(ctx, tx, importScope{
		ScopeType:          "teamlead_period",
		Direction:          "outbound",
		UploadedBy:         teamleadID,
		AccountingPeriodID: &periodID,
		FileName:           "seed-teamlead-period-outbound.csv",
		FileHash:           "seed-v2-teamlead-period-outbound",
	}, []orderSeed{
		activeOrder("outbound", "920201", "seed-out-anna-001", "Bliss_OP5", traderAnnaID, annaReqs[0], 1200000, "paid", timeUTC(2026, 6, 17, 15, 10), "Q1"),
		activeOrder("outbound", "920202", "seed-out-anna-002", "Bliss_OP5", traderAnnaID, annaReqs[1], 800000, "paid", timeUTC(2026, 6, 17, 17, 40), "Q2"),
	})
	if err != nil {
		return err
	}
	if err := updateReconciliationRunImportBatch(ctx, tx, teamID, "teamlead_period_inbound", periodID, periodInboundBatchID); err != nil {
		return err
	}
	if err := updateReconciliationRunImportBatch(ctx, tx, teamID, "teamlead_period_outbound", periodID, periodOutboundBatchID); err != nil {
		return err
	}

	runID, err := ensureTeamleadReconciliation(ctx, tx, teamID, teamleadID, currentInboundBatchID, currentOutboundBatchID)
	if err != nil {
		return err
	}
	if err := replaceTeamleadReconciliationItems(ctx, tx, runID, teamID, []teamleadReconciliationItemSeed{
		{
			Direction:       "inbound",
			Stage:           "transaction_check",
			IssueType:       "amount_mismatch",
			Severity:        "warning",
			ExternalInnerID: "seed-in-anna-002",
			TraderID:        &traderAnnaID,
			RequisiteID:     &annaReqs[1].ID,
			ShiftID:         &annaShiftID,
			Before:          map[string]any{"traderImportBatchId": traderInboundBatchID, "amountMinor": 1400000},
			After:           map[string]any{"teamleadImportBatchId": currentInboundBatchID, "amountMinor": 1500000},
			Message:         "Seed v2: тимлид обновил сумму входящего ордера",
			IsBlocking:      false,
		},
		{
			Direction:       "outbound",
			Stage:           "transaction_check",
			IssueType:       "manual_transfer_mismatch",
			Severity:        "warning",
			ExternalInnerID: "seed-out-anna-002",
			TraderID:        &traderAnnaID,
			RequisiteID:     &annaReqs[1].ID,
			ShiftID:         &annaShiftID,
			Before:          map[string]any{"manualTransfersAmountMinor": 700000, "traderImportBatchId": traderOutboundBatchID},
			After:           map[string]any{"teamleadImportBatchId": currentOutboundBatchID, "amountMinor": 800000},
			Message:         "Seed v2: сумма выплаты больше ручных переводов",
			IsBlocking:      false,
		},
		{
			Direction:  "inbound",
			Stage:      "preview",
			IssueType:  "preview_summary",
			Severity:   "info",
			After:      map[string]any{"createCount": 0, "updateCount": 1, "unchangedCount": 1, "blockedCount": 0},
			Message:    "Seed v2: preview changes calculated",
			IsBlocking: false,
		},
	}); err != nil {
		return err
	}

	if err := applyTeamleadStatuses(ctx, tx, teamID, runID, annaShiftID); err != nil {
		return err
	}
	if err := ensureAudit(ctx, tx, teamID, teamleadID, audit.ActionReconciliationCreated, "teamlead_reconciliation", runID, map[string]any{"status": "mismatch"}, "Seed v2: создана сверка тимлида"); err != nil {
		return err
	}
	if err := ensureAudit(ctx, tx, teamID, teamleadID, audit.ActionReconciliationConfirmed, "teamlead_reconciliation", runID, map[string]any{"status": "apply_queued"}, "Seed v2: сверка принята"); err != nil {
		return err
	}
	if err := ensureAudit(ctx, tx, teamID, teamleadID, audit.ActionReconciliationApplied, "teamlead_reconciliation", runID, map[string]any{"status": "applied"}, "Seed v2: изменения сверки применены"); err != nil {
		return err
	}
	return nil
}

func seedAuditEvents(ctx context.Context, tx pgx.Tx, teamID int64, teamleadID int64, traderIvanID int64, traderAnnaID int64) error {
	events := []struct {
		ActorID int64
		Action  string
		Entity  string
		ID      int64
		Payload map[string]any
		Comment string
	}{
		{teamleadID, audit.ActionUserCreated, "user", traderIvanID, map[string]any{"login": "trader_ivan"}, "Seed v2: демо трейдер"},
		{teamleadID, audit.ActionUserCreated, "user", traderAnnaID, map[string]any{"login": "trader_anna"}, "Seed v2: демо трейдер"},
		{teamleadID, audit.ActionRequisiteAssigned, "requisite_assignment", traderIvanID, map[string]any{"assignedTo": "trader_ivan"}, "Seed v2: назначение реквизитов"},
	}
	for _, event := range events {
		if err := ensureAudit(ctx, tx, teamID, event.ActorID, event.Action, event.Entity, event.ID, event.Payload, event.Comment); err != nil {
			return err
		}
	}
	return nil
}

func upsertTeam(ctx context.Context, tx pgx.Tx) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM teams WHERE name = 'Demo P2P Team' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select team: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO teams(name, status)
VALUES ('Demo P2P Team', 'active')
RETURNING id`).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert team: %w", err)
	}
	return id, nil
}

func upsertUser(ctx context.Context, tx pgx.Tx, teamID int64, role string, login string, passwordHash string) (int64, error) {
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO users(team_id, role, login, password_hash, status)
VALUES ($1, $2, $3, $4, 'active')
ON CONFLICT (login)
DO UPDATE SET team_id = EXCLUDED.team_id, role = EXCLUDED.role, password_hash = EXCLUDED.password_hash, status = 'active', updated_at = now(), deleted_at = NULL
RETURNING id`, teamID, role, login, passwordHash).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert user %s: %w", login, err)
	}
	return id, nil
}

func upsertTrader(ctx context.Context, tx pgx.Tx, teamID int64, login string, passwordHash string, salaryRateBps int64, workerName string) (int64, error) {
	userID, err := upsertUser(ctx, tx, teamID, "trader", login, passwordHash)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO trader_profiles(user_id, salary_rate_bps, external_worker_name)
VALUES ($1, $2, $3)
ON CONFLICT (user_id)
DO UPDATE SET salary_rate_bps = EXCLUDED.salary_rate_bps, external_worker_name = EXCLUDED.external_worker_name, updated_at = now()`,
		userID, salaryRateBps, workerName); err != nil {
		return 0, fmt.Errorf("upsert trader profile %s: %w", login, err)
	}
	return userID, nil
}

func upsertDemoRequisites(ctx context.Context, tx pgx.Tx, teamID int64, createdBy int64, seeds []requisiteSeed) ([]demoRequisite, error) {
	result := make([]demoRequisite, 0, len(seeds))
	for _, seed := range seeds {
		id, err := upsertRequisite(ctx, tx, teamID, createdBy, seed)
		if err != nil {
			return nil, err
		}
		bankName, err := bankName(ctx, tx, seed.BankCode)
		if err != nil {
			return nil, err
		}
		result = append(result, demoRequisite{
			ID:         id,
			Phone:      seed.Phone,
			MethodType: seed.MethodType,
			BankCode:   seed.BankCode,
			BankName:   bankName,
			CardNumber: seed.CardNumber,
			HolderName: seed.HolderName,
		})
	}
	return result, nil
}

func upsertRequisite(ctx context.Context, tx pgx.Tx, teamID int64, createdBy int64, seed requisiteSeed) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM requisites
WHERE team_id = $1
  AND deleted_at IS NULL
  AND (phone = $2 OR proxy = $3)
ORDER BY CASE WHEN phone = $2 THEN 0 ELSE 1 END, id
LIMIT 1`, teamID, seed.Phone, seed.Proxy).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select requisite %s: %w", seed.Phone, err)
	}
	detailsAt := any(nil)
	detailsBy := any(nil)
	holderName := any(nil)
	cardNumber := any(nil)
	if strings.TrimSpace(seed.HolderName) != "" {
		detailsAt = timeUTC(2026, 6, 17, 8, 30)
		detailsBy = createdBy
		holderName = seed.HolderName
	}
	if strings.TrimSpace(seed.CardNumber) != "" {
		cardNumber = seed.CardNumber
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO requisites(team_id, phone, method_type, bank_code, proxy, employee_comment, card_number, holder_name, details_filled_at, details_filled_by, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'active', $11)
RETURNING id`,
			teamID, seed.Phone, seed.MethodType, seed.BankCode, seed.Proxy, seed.EmployeeComment, cardNumber, holderName, detailsAt, detailsBy, createdBy).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert requisite %s: %w", seed.Phone, err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE requisites
SET phone = $2,
    method_type = $3,
    bank_code = $4,
    proxy = $5,
    employee_comment = $6,
    card_number = $7,
    holder_name = $8,
    details_filled_at = $9,
    details_filled_by = $10,
    status = 'active',
    updated_at = now(),
    deleted_at = NULL
WHERE id = $1`,
		id, seed.Phone, seed.MethodType, seed.BankCode, seed.Proxy, seed.EmployeeComment, cardNumber, holderName, detailsAt, detailsBy); err != nil {
		return 0, fmt.Errorf("update requisite %s: %w", seed.Phone, err)
	}
	return id, nil
}

func bankName(ctx context.Context, tx pgx.Tx, code string) (string, error) {
	var name string
	if err := tx.QueryRow(ctx, `SELECT name FROM banks WHERE code = $1`, code).Scan(&name); err != nil {
		return "", fmt.Errorf("select bank %s: %w", code, err)
	}
	return name, nil
}

func ensureActiveAssignment(ctx context.Context, tx pgx.Tx, teamID int64, requisiteID int64, traderID int64, assignedBy int64, assignedForDate time.Time, targetTurnoverMinor int64, comment string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM requisite_assignments
WHERE requisite_id = $1
  AND unassigned_at IS NULL
  AND status IN ('planned', 'assigned', 'in_work')
ORDER BY assigned_at DESC, id DESC
LIMIT 1`, requisiteID).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select active assignment for requisite %d: %w", requisiteID, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO requisite_assignments(team_id, requisite_id, trader_id, assigned_by, comment, status, assigned_for_date, target_turnover_minor)
VALUES ($1, $2, $3, $4, $5, 'assigned', $6, $7)
RETURNING id`, teamID, requisiteID, traderID, assignedBy, comment, assignedForDate, targetTurnoverMinor).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert active assignment for requisite %d: %w", requisiteID, err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE requisite_assignments
SET trader_id = $2,
    assigned_by = $3,
    comment = $4,
    status = 'assigned',
    assigned_for_date = $5,
    target_turnover_minor = $6,
    unassigned_at = NULL,
    started_at = NULL,
    completed_at = NULL,
    cancelled_at = NULL,
    updated_at = now()
WHERE id = $1`, id, traderID, assignedBy, comment, assignedForDate, targetTurnoverMinor); err != nil {
		return 0, fmt.Errorf("update active assignment %d: %w", id, err)
	}
	return id, nil
}

func ensureWorkedAssignment(ctx context.Context, tx pgx.Tx, teamID int64, requisiteID int64, traderID int64, assignedBy int64, assignedForDate time.Time, startedAt time.Time, completedAt time.Time, targetTurnoverMinor int64, comment string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM requisite_assignments
WHERE requisite_id = $1
  AND trader_id = $2
  AND assigned_for_date = $3::date
ORDER BY id
LIMIT 1`, requisiteID, traderID, assignedForDate).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select worked assignment for requisite %d: %w", requisiteID, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO requisite_assignments(team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'worked', $8, $9, $10, $11)
RETURNING id`, teamID, requisiteID, traderID, assignedBy, startedAt.Add(-30*time.Minute), completedAt, comment, assignedForDate, targetTurnoverMinor, startedAt, completedAt).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert worked assignment for requisite %d: %w", requisiteID, err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE requisite_assignments
SET assigned_by = $2,
    assigned_at = $3,
    unassigned_at = $4,
    comment = $5,
    status = 'worked',
    assigned_for_date = $6,
    target_turnover_minor = $7,
    started_at = $8,
    completed_at = $9,
    cancelled_at = NULL,
    updated_at = now()
WHERE id = $1`, id, assignedBy, startedAt.Add(-30*time.Minute), completedAt, comment, assignedForDate, targetTurnoverMinor, startedAt, completedAt); err != nil {
		return 0, fmt.Errorf("update worked assignment %d: %w", id, err)
	}
	return id, nil
}

func ensureAccountingPeriod(ctx context.Context, tx pgx.Tx, teamID int64, createdBy int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM accounting_periods
WHERE team_id = $1
ORDER BY id
LIMIT 1`, teamID).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select accounting period: %w", err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO accounting_periods(team_id, date_from, date_to, status, created_by)
VALUES ($1, $2, $3, 'open', $4)
RETURNING id`, teamID, demoPeriodFrom, demoPeriodTo, createdBy).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert accounting period: %w", err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE accounting_periods
SET date_from = $2,
    date_to = $3,
    status = 'open',
    created_by = $4,
    closed_by = NULL,
    closed_at = NULL
WHERE id = $1`, id, demoPeriodFrom, demoPeriodTo, createdBy); err != nil {
		return 0, fmt.Errorf("update accounting period: %w", err)
	}
	return id, nil
}

func ensureShift(ctx context.Context, tx pgx.Tx, teamID int64, traderID int64, startedAt time.Time, endedAt *time.Time, status string, inboundStatus string, outboundStatus string, closeComment *string) (int64, error) {
	var id int64
	if status == "open" || status == "closing" {
		err := tx.QueryRow(ctx, `
SELECT id
FROM trader_shifts
WHERE team_id = $1
  AND trader_id = $2
  AND status IN ('open', 'closing')
ORDER BY started_at DESC, id DESC
LIMIT 1`, teamID, traderID).Scan(&id)
		if err != nil && err != pgx.ErrNoRows {
			return 0, fmt.Errorf("select open shift for trader %d: %w", traderID, err)
		}
		if err == nil {
			if err := updateShift(ctx, tx, id, startedAt, endedAt, status, inboundStatus, outboundStatus, closeComment); err != nil {
				return 0, err
			}
			return id, nil
		}
	} else {
		err := tx.QueryRow(ctx, `
SELECT id
FROM trader_shifts
WHERE team_id = $1
  AND trader_id = $2
  AND started_at = $3
LIMIT 1`, teamID, traderID, startedAt).Scan(&id)
		if err != nil && err != pgx.ErrNoRows {
			return 0, fmt.Errorf("select closed shift for trader %d: %w", traderID, err)
		}
		if err == nil {
			if err := updateShift(ctx, tx, id, startedAt, endedAt, status, inboundStatus, outboundStatus, closeComment); err != nil {
				return 0, err
			}
			return id, nil
		}
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO trader_shifts(team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, closed_at)
VALUES ($1, $2, $3, $4::timestamptz, $5, $6, $7, $8, CASE WHEN $5 IN ('closed', 'closed_with_discrepancy') THEN $4::timestamptz ELSE NULL END)
RETURNING id`, teamID, traderID, startedAt, endedAt, status, inboundStatus, outboundStatus, closeComment).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert shift for trader %d: %w", traderID, err)
	}
	return id, nil
}

func updateShift(ctx context.Context, tx pgx.Tx, id int64, startedAt time.Time, endedAt *time.Time, status string, inboundStatus string, outboundStatus string, closeComment *string) error {
	_, err := tx.Exec(ctx, `
UPDATE trader_shifts
SET started_at = $2,
    ended_at = $3::timestamptz,
    status = $4,
    inbound_reconciliation_status = $5,
    outbound_reconciliation_status = $6,
    close_comment = $7,
    closed_at = CASE WHEN $4 IN ('closed', 'closed_with_discrepancy') THEN $3::timestamptz ELSE NULL END,
    updated_at = now()
WHERE id = $1`, id, startedAt, endedAt, status, inboundStatus, outboundStatus, closeComment)
	if err != nil {
		return fmt.Errorf("update shift %d: %w", id, err)
	}
	return nil
}

func ensureShiftRequisite(ctx context.Context, tx pgx.Tx, teamID int64, shiftID int64, traderID int64, req demoRequisite, assignmentID int64, status string, inboundMinor int64, outboundMinor int64, closingMinor int64, takenAt time.Time, releasedAt *time.Time, tlStatus string, teamleadRunID *int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM shift_requisites
WHERE shift_id = $1
  AND requisite_id = $2
ORDER BY id
LIMIT 1`, shiftID, req.ID).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select shift requisite %d/%d: %w", shiftID, req.ID, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO shift_requisites(team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name, taken_at, released_at, status, inbound_turnover_minor, outbound_turnover_minor, closing_balance_minor, tl_reconciliation_status, last_teamlead_reconciliation_id, tl_reconciled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CASE WHEN $14 <> 'not_checked' THEN now() ELSE NULL END)
RETURNING id`, teamID, shiftID, traderID, req.ID, assignmentID, req.CardNumber, req.HolderName, takenAt, releasedAt, status, inboundMinor, outboundMinor, closingMinor, tlStatus, teamleadRunID).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert shift requisite %d/%d: %w", shiftID, req.ID, err)
		}
	} else if _, err := tx.Exec(ctx, `
UPDATE shift_requisites
SET assignment_id = $2,
    card_number = $3,
    holder_name = $4,
    taken_at = $5,
    released_at = $6,
    status = $7,
    inbound_turnover_minor = $8,
    outbound_turnover_minor = $9,
    closing_balance_minor = $10,
    tl_reconciliation_status = $11,
    last_teamlead_reconciliation_id = $12,
    tl_reconciled_at = CASE WHEN $11 <> 'not_checked' THEN now() ELSE NULL END,
    updated_at = now()
WHERE id = $1`, id, assignmentID, req.CardNumber, req.HolderName, takenAt, releasedAt, status, inboundMinor, outboundMinor, closingMinor, tlStatus, teamleadRunID); err != nil {
		return 0, fmt.Errorf("update shift requisite %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE requisite_assignments
SET shift_requisite_id = $2,
    started_at = COALESCE(started_at, $3),
    completed_at = CASE WHEN $4::text IN ('worked_discrepancy', 'worked_verified', 'blocked', 'released') THEN COALESCE(completed_at, $5) ELSE completed_at END,
    updated_at = now()
WHERE id = $1`, assignmentID, id, takenAt, status, releasedAt); err != nil {
		return 0, fmt.Errorf("update assignment shift requisite %d: %w", assignmentID, err)
	}
	return id, nil
}

func ensureTurnoverEntry(ctx context.Context, tx pgx.Tx, teamID int64, shiftID int64, shiftRequisiteID int64, requisiteID int64, traderID int64, amountMinor int64, createdBy int64, comment string) error {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM requisite_turnover_entries
WHERE shift_requisite_id = $1
  AND comment = $2
LIMIT 1`, shiftRequisiteID, comment).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("select turnover for shift requisite %d: %w", shiftRequisiteID, err)
	}
	if err == pgx.ErrNoRows {
		if _, err := tx.Exec(ctx, `
INSERT INTO requisite_turnover_entries(team_id, shift_id, shift_requisite_id, requisite_id, trader_id, amount_minor, created_by, comment)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, teamID, shiftID, shiftRequisiteID, requisiteID, traderID, amountMinor, createdBy, comment); err != nil {
			return fmt.Errorf("insert turnover for shift requisite %d: %w", shiftRequisiteID, err)
		}
		return nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE requisite_turnover_entries
SET amount_minor = $2,
    created_by = $3
WHERE id = $1`, id, amountMinor, createdBy); err != nil {
		return fmt.Errorf("update turnover %d: %w", id, err)
	}
	return nil
}

func refreshRequisiteAggregates(ctx context.Context, tx pgx.Tx, requisiteID int64) error {
	_, err := tx.Exec(ctx, `
WITH totals AS (
    SELECT
        requisite_id,
        COALESCE(SUM(inbound_turnover_minor), 0)::bigint AS total_inbound_turnover_minor,
        COALESCE(SUM(outbound_turnover_minor), 0)::bigint AS total_outbound_turnover_minor
    FROM shift_requisites
    WHERE requisite_id = $1
    GROUP BY requisite_id
),
latest AS (
    SELECT
        id,
        status,
        closing_balance_minor,
        COALESCE(released_at, taken_at, updated_at) AS activity_at
    FROM shift_requisites
    WHERE requisite_id = $1
    ORDER BY COALESCE(released_at, taken_at, updated_at) DESC, id DESC
    LIMIT 1
)
UPDATE requisites r
SET total_inbound_turnover_minor = COALESCE((SELECT total_inbound_turnover_minor FROM totals), 0),
    total_outbound_turnover_minor = COALESCE((SELECT total_outbound_turnover_minor FROM totals), 0),
    last_closing_balance_minor = COALESCE((SELECT closing_balance_minor FROM latest), 0),
    last_activity_status = (SELECT status FROM latest),
    last_activity_at = (SELECT activity_at FROM latest),
    last_shift_requisite_id = (SELECT id FROM latest),
    updated_at = now()
WHERE r.id = $1`, requisiteID)
	if err != nil {
		return fmt.Errorf("refresh requisite aggregates %d: %w", requisiteID, err)
	}
	return nil
}

func ensureOrdersImport(ctx context.Context, tx pgx.Tx, scope importScope, orders []orderSeed) (int64, []int64, error) {
	batchID, err := ensureImportBatch(ctx, tx, scope, len(orders))
	if err != nil {
		return 0, nil, err
	}
	orderIDs := make([]int64, 0, len(orders))
	for index, order := range orders {
		rowID, err := ensureImportRow(ctx, tx, batchID, int64(index+1), order)
		if err != nil {
			return 0, nil, err
		}
		externalOrderID, err := ensureExternalOrder(ctx, tx, batchID, order)
		if err != nil {
			return 0, nil, err
		}
		if err := ensureScopeItem(ctx, tx, scope, batchID, rowID, externalOrderID); err != nil {
			return 0, nil, err
		}
		orderIDs = append(orderIDs, externalOrderID)
	}
	if _, err := tx.Exec(ctx, `
UPDATE import_batches
SET rows_count = $2,
    status = 'applied',
    applied_at = COALESCE(applied_at, now())
WHERE id = $1`, batchID, len(orders)); err != nil {
		return 0, nil, fmt.Errorf("mark import batch %d applied: %w", batchID, err)
	}
	return batchID, orderIDs, nil
}

func ensureImportBatch(ctx context.Context, tx pgx.Tx, scope importScope, rowsCount int) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM import_batches
WHERE team_id = (SELECT team_id FROM users WHERE id = $1)
  AND uploaded_by = $1
  AND scope_type = $2
  AND direction = $3
  AND (($4::bigint IS NULL AND shift_id IS NULL) OR shift_id = $4)
  AND (($5::bigint IS NULL AND accounting_period_id IS NULL) OR accounting_period_id = $5)
  AND file_name = $6
ORDER BY id
LIMIT 1`, scope.UploadedBy, scope.ScopeType, scope.Direction, nullableInt64(scope.ShiftID), nullableInt64(scope.AccountingPeriodID), scope.FileName).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select import batch %s: %w", scope.FileName, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO import_batches(team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, applied_at)
SELECT team_id, $1, $2, $3, $4, $5, $6, $7, $8, $9, 'applied', now()
FROM users
WHERE id = $1
RETURNING id`, scope.UploadedBy, scope.ScopeType, scope.Direction, nullableInt64(scope.ShiftID), nullableInt64(scope.AccountingPeriodID), nullableInt64(scope.TraderID), scope.FileName, scope.FileHash, rowsCount).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert import batch %s: %w", scope.FileName, err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE import_batches
SET rows_count = $2,
    status = 'applied',
    superseded_by_batch_id = NULL,
    error_message = NULL,
    applied_at = COALESCE(applied_at, now())
WHERE id = $1`, id, rowsCount); err != nil {
		return 0, fmt.Errorf("update import batch %d: %w", id, err)
	}
	return id, nil
}

func ensureImportRow(ctx context.Context, tx pgx.Tx, batchID int64, rowNumber int64, order orderSeed) (int64, error) {
	payload, err := jsonBytes(map[string]any{
		"externalId":        order.ExternalID,
		"innerId":           order.InnerID,
		"worker":            order.WorkerName,
		"requisite":         order.RequisiteRaw,
		"amountMinor":       order.AmountMinor,
		"status":            order.RawStatus,
		"createdAtExternal": order.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return 0, err
	}
	var id int64
	err = tx.QueryRow(ctx, `
SELECT id
FROM import_rows
WHERE import_batch_id = $1
  AND row_number = $2`, batchID, rowNumber).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select import row %d/%d: %w", batchID, rowNumber, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO import_rows(import_batch_id, row_number, external_id, external_inner_id, raw_payload_json, parse_status)
VALUES ($1, $2, $3, $4, $5, 'parsed')
RETURNING id`, batchID, rowNumber, order.ExternalID, order.InnerID, payload).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert import row %d/%d: %w", batchID, rowNumber, err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE import_rows
SET external_id = $3,
    external_inner_id = $4,
    raw_payload_json = $5,
    parse_status = 'parsed',
    parse_error = NULL
WHERE import_batch_id = $1
  AND row_number = $2`, batchID, rowNumber, order.ExternalID, order.InnerID, payload); err != nil {
		return 0, fmt.Errorf("update import row %d: %w", id, err)
	}
	return id, nil
}

func ensureExternalOrder(ctx context.Context, tx pgx.Tx, batchID int64, order orderSeed) (int64, error) {
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO external_orders(
    team_id,
    direction,
    external_id,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    requisite_external_id,
    requisite_id,
    device_name,
    method_type,
    method_name,
    amount_minor,
    currency,
    raw_status,
    normalized_status,
    created_at_external,
    closed_at_external,
    updated_at_external,
    receipt,
    order_comment,
    ordered,
    counted,
    initials,
    last_import_batch_id
)
SELECT
    ib.team_id,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    'RUB',
    $15,
    $16,
    $17,
    $18,
    $18,
    'seed/' || $4 || '.pdf',
    $19,
    TRUE,
    TRUE,
    $20,
    ib.id
FROM import_batches ib
WHERE ib.id = $1
ON CONFLICT (team_id, direction, external_inner_id)
DO UPDATE SET
    external_id = EXCLUDED.external_id,
    worker_name = EXCLUDED.worker_name,
    trader_id = EXCLUDED.trader_id,
    requisite_raw = EXCLUDED.requisite_raw,
    requisite_phone = EXCLUDED.requisite_phone,
    requisite_external_id = EXCLUDED.requisite_external_id,
    requisite_id = EXCLUDED.requisite_id,
    device_name = EXCLUDED.device_name,
    method_type = EXCLUDED.method_type,
    method_name = EXCLUDED.method_name,
    amount_minor = EXCLUDED.amount_minor,
    currency = EXCLUDED.currency,
    raw_status = EXCLUDED.raw_status,
    normalized_status = EXCLUDED.normalized_status,
    created_at_external = EXCLUDED.created_at_external,
    closed_at_external = EXCLUDED.closed_at_external,
    updated_at_external = EXCLUDED.updated_at_external,
    receipt = EXCLUDED.receipt,
    order_comment = EXCLUDED.order_comment,
    ordered = EXCLUDED.ordered,
    counted = EXCLUDED.counted,
    initials = EXCLUDED.initials,
    last_import_batch_id = EXCLUDED.last_import_batch_id,
    updated_at = now()
RETURNING id`,
		batchID,
		order.Direction,
		order.ExternalID,
		order.InnerID,
		order.WorkerName,
		order.TraderID,
		order.RequisiteRaw,
		order.RequisitePhone,
		order.RequisiteExternal,
		order.RequisiteID,
		order.DeviceName,
		order.MethodType,
		order.MethodName,
		order.AmountMinor,
		order.RawStatus,
		order.NormalizedStatus,
		order.CreatedAt,
		order.CreatedAt.Add(8*time.Minute),
		order.Comment,
		order.Initials,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert external order %s: %w", order.InnerID, err)
	}
	return id, nil
}

func ensureScopeItem(ctx context.Context, tx pgx.Tx, scope importScope, batchID int64, rowID int64, externalOrderID int64) error {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM order_scope_items
WHERE import_batch_id = $1
  AND import_row_id = $2
LIMIT 1`, batchID, rowID).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("select scope item for batch %d row %d: %w", batchID, rowID, err)
	}
	if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx, `
INSERT INTO order_scope_items(
    team_id,
    scope_type,
    direction,
    shift_id,
    accounting_period_id,
    import_batch_id,
    import_row_id,
    external_order_id,
    external_id,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    method_type,
    method_name,
    amount_minor,
    currency,
    raw_status,
    normalized_status,
    created_at_external,
    is_active
)
SELECT
    eo.team_id,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    eo.id,
    eo.external_id,
    eo.external_inner_id,
    eo.worker_name,
    eo.trader_id,
    eo.requisite_raw,
    eo.requisite_phone,
    eo.method_type,
    eo.method_name,
    eo.amount_minor,
    eo.currency,
    eo.raw_status,
    eo.normalized_status,
    eo.created_at_external,
    TRUE
FROM external_orders eo
WHERE eo.id = $1`, externalOrderID, scope.ScopeType, scope.Direction, nullableInt64(scope.ShiftID), nullableInt64(scope.AccountingPeriodID), batchID, rowID)
		if err != nil {
			return fmt.Errorf("insert scope item for order %d: %w", externalOrderID, err)
		}
		return nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE order_scope_items osi
SET external_order_id = eo.id,
    external_id = eo.external_id,
    external_inner_id = eo.external_inner_id,
    worker_name = eo.worker_name,
    trader_id = eo.trader_id,
    requisite_raw = eo.requisite_raw,
    requisite_phone = eo.requisite_phone,
    method_type = eo.method_type,
    method_name = eo.method_name,
    amount_minor = eo.amount_minor,
    currency = eo.currency,
    raw_status = eo.raw_status,
    normalized_status = eo.normalized_status,
    created_at_external = eo.created_at_external,
    is_active = TRUE,
    deactivated_at = NULL
FROM external_orders eo
WHERE osi.id = $1
  AND eo.id = $2`, id, externalOrderID); err != nil {
		return fmt.Errorf("update scope item %d: %w", id, err)
	}
	return nil
}

type payoutTransferSeed struct {
	SourceShiftRequisiteID int64
	SourceRequisiteID      int64
	AmountMinor            int64
	CreatedBy              int64
	Comment                string
}

func ensurePayoutOrder(ctx context.Context, tx pgx.Tx, teamID int64, shiftID int64, traderID int64, bank string, destination string, amountMinor int64, status string, transfers []payoutTransferSeed) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM manual_payout_orders
WHERE team_id = $1
  AND shift_id = $2
  AND trader_id = $3
  AND destination_requisite = $4
LIMIT 1`, teamID, shiftID, traderID, destination).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select payout order %s: %w", destination, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO manual_payout_orders(team_id, shift_id, trader_id, destination_bank, destination_requisite, amount_minor, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id`, teamID, shiftID, traderID, bank, destination, amountMinor, status).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert payout order %s: %w", destination, err)
		}
	} else if _, err := tx.Exec(ctx, `
UPDATE manual_payout_orders
SET destination_bank = $2,
    amount_minor = $3,
    status = $4,
    deleted_at = NULL,
    updated_at = now()
WHERE id = $1`, id, bank, amountMinor, status); err != nil {
		return 0, fmt.Errorf("update payout order %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM manual_payout_transfers WHERE manual_payout_order_id = $1`, id); err != nil {
		return 0, fmt.Errorf("reset payout transfers for order %d: %w", id, err)
	}
	for _, transfer := range transfers {
		if _, err := tx.Exec(ctx, `
INSERT INTO manual_payout_transfers(team_id, manual_payout_order_id, shift_id, trader_id, source_shift_requisite_id, source_requisite_id, amount_minor, created_by, comment)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			teamID, id, shiftID, traderID, transfer.SourceShiftRequisiteID, transfer.SourceRequisiteID, transfer.AmountMinor, transfer.CreatedBy, transfer.Comment); err != nil {
			return 0, fmt.Errorf("insert payout transfer for order %d: %w", id, err)
		}
	}
	return id, nil
}

type reconciliationRunSeed struct {
	TeamID              int64
	Type                string
	ScopeType           string
	ShiftID             *int64
	AccountingPeriodID  *int64
	TraderID            *int64
	ImportBatchID       *int64
	ExpectedAmountMinor int64
	ActualAmountMinor   int64
	DiffAmountMinor     int64
	SuccessAmountMinor  int64
	SuccessCount        int64
	FailedAmountMinor   int64
	FailedCount         int64
	TotalAmountMinor    int64
	TotalCount          int64
	Status              string
	Comment             *string
	ConfirmedBy         *int64
}

func ensureReconciliationRun(ctx context.Context, tx pgx.Tx, seed reconciliationRunSeed) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
SELECT id
FROM reconciliation_runs
WHERE team_id = $1
  AND type = $2
  AND (($3::bigint IS NULL AND shift_id IS NULL) OR shift_id = $3)
  AND (($4::bigint IS NULL AND accounting_period_id IS NULL) OR accounting_period_id = $4)
ORDER BY id
LIMIT 1`, seed.TeamID, seed.Type, nullableInt64(seed.ShiftID), nullableInt64(seed.AccountingPeriodID)).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select reconciliation run %s: %w", seed.Type, err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO reconciliation_runs(team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, CASE WHEN $19::bigint IS NULL THEN NULL ELSE now() END)
RETURNING id`,
			seed.TeamID,
			seed.Type,
			seed.ScopeType,
			nullableInt64(seed.ShiftID),
			nullableInt64(seed.AccountingPeriodID),
			nullableInt64(seed.TraderID),
			nullableInt64(seed.ImportBatchID),
			seed.ExpectedAmountMinor,
			seed.ActualAmountMinor,
			seed.DiffAmountMinor,
			seed.SuccessAmountMinor,
			seed.SuccessCount,
			seed.FailedAmountMinor,
			seed.FailedCount,
			seed.TotalAmountMinor,
			seed.TotalCount,
			seed.Status,
			seed.Comment,
			nullableInt64(seed.ConfirmedBy),
		).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert reconciliation run %s: %w", seed.Type, err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE reconciliation_runs
SET import_batch_id = $2,
    expected_amount_minor = $3,
    actual_amount_minor = $4,
    diff_amount_minor = $5,
    success_amount_minor = $6,
    success_count = $7,
    failed_amount_minor = $8,
    failed_count = $9,
    total_amount_minor = $10,
    total_count = $11,
    status = $12,
    comment = $13,
    confirmed_by = $14,
    confirmed_at = CASE WHEN $14::bigint IS NULL THEN NULL ELSE COALESCE(confirmed_at, now()) END
WHERE id = $1`,
		id,
		nullableInt64(seed.ImportBatchID),
		seed.ExpectedAmountMinor,
		seed.ActualAmountMinor,
		seed.DiffAmountMinor,
		seed.SuccessAmountMinor,
		seed.SuccessCount,
		seed.FailedAmountMinor,
		seed.FailedCount,
		seed.TotalAmountMinor,
		seed.TotalCount,
		seed.Status,
		seed.Comment,
		nullableInt64(seed.ConfirmedBy),
	); err != nil {
		return 0, fmt.Errorf("update reconciliation run %d: %w", id, err)
	}
	return id, nil
}

func updateReconciliationRunImportBatch(ctx context.Context, tx pgx.Tx, teamID int64, runType string, periodID int64, importBatchID int64) error {
	_, err := tx.Exec(ctx, `
UPDATE reconciliation_runs
SET import_batch_id = $4
WHERE team_id = $1
  AND type = $2
  AND accounting_period_id = $3`, teamID, runType, periodID, importBatchID)
	if err != nil {
		return fmt.Errorf("update period reconciliation import batch %s: %w", runType, err)
	}
	return nil
}

type reconciliationItemSeed struct {
	IssueType       string
	ExternalInnerID string
	TeamleadValue   map[string]any
	TraderValue     map[string]any
	Message         string
}

func replaceReconciliationItems(ctx context.Context, tx pgx.Tx, runID int64, items []reconciliationItemSeed) error {
	if _, err := tx.Exec(ctx, `DELETE FROM reconciliation_items WHERE reconciliation_run_id = $1`, runID); err != nil {
		return fmt.Errorf("delete reconciliation items for run %d: %w", runID, err)
	}
	for _, item := range items {
		teamleadValue, err := jsonBytes(item.TeamleadValue)
		if err != nil {
			return err
		}
		traderValue, err := jsonBytes(item.TraderValue)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO reconciliation_items(reconciliation_run_id, issue_type, external_inner_id, teamlead_value_json, trader_value_json, message)
VALUES ($1, $2, $3, $4, $5, $6)`, runID, item.IssueType, item.ExternalInnerID, teamleadValue, traderValue, item.Message); err != nil {
			return fmt.Errorf("insert reconciliation item for run %d: %w", runID, err)
		}
	}
	return nil
}

func ensureTeamleadReconciliation(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, inboundBatchID int64, outboundBatchID int64) (int64, error) {
	pipeline, err := jsonBytes([]map[string]any{
		{"stage": "normalization", "status": "matched", "issuesCount": 0, "directionsCount": 2},
		{"stage": "matching", "status": "matched", "issuesCount": 0, "directionsCount": 2},
		{"stage": "turnover_check", "status": "mismatch", "issuesCount": 1, "directionsCount": 2},
		{"stage": "transaction_check", "status": "mismatch", "issuesCount": 2, "directionsCount": 2},
		{"stage": "preview", "status": "matched", "issuesCount": 0, "directionsCount": 2},
	})
	if err != nil {
		return 0, err
	}
	inboundSummary, err := jsonBytes(map[string]any{
		"rowsTotal":          2,
		"rowsInPeriod":       2,
		"successAmountMinor": 3500000,
		"successCount":       2,
		"failedAmountMinor":  0,
		"failedCount":        0,
		"totalAmountMinor":   3500000,
		"totalCount":         2,
		"crmAmountMinor":     3400000,
		"crmCount":           2,
		"diffAmountMinor":    100000,
	})
	if err != nil {
		return 0, err
	}
	outboundSummary, err := jsonBytes(map[string]any{
		"rowsTotal":          2,
		"rowsInPeriod":       2,
		"successAmountMinor": 2000000,
		"successCount":       2,
		"failedAmountMinor":  0,
		"failedCount":        0,
		"totalAmountMinor":   2000000,
		"totalCount":         2,
		"crmAmountMinor":     1900000,
		"crmCount":           2,
		"diffAmountMinor":    100000,
	})
	if err != nil {
		return 0, err
	}
	preview, err := jsonBytes(map[string]any{
		"inbound":  map[string]any{"createCount": 0, "updateCount": 1, "unchangedCount": 1, "blockedCount": 0},
		"outbound": map[string]any{"createCount": 0, "updateCount": 0, "unchangedCount": 1, "blockedCount": 1},
	})
	if err != nil {
		return 0, err
	}
	applyResult, err := jsonBytes(map[string]any{
		"runId":             0,
		"createdOrders":     0,
		"updatedOrders":     1,
		"confirmedOrders":   2,
		"discrepancyOrders": 1,
		"directions": []map[string]any{
			{"direction": "inbound", "rowsApplied": 2, "createdOrders": 0, "updatedOrders": 1, "confirmedOrders": 1, "discrepancyOrders": 0},
			{"direction": "outbound", "rowsApplied": 2, "createdOrders": 0, "updatedOrders": 0, "confirmedOrders": 1, "discrepancyOrders": 1},
		},
	})
	if err != nil {
		return 0, err
	}

	comment := "Seed v2: сверка применена, одно расхождение принято"
	var id int64
	err = tx.QueryRow(ctx, `
SELECT id
FROM teamlead_reconciliations
WHERE team_id = $1
  AND date_from = $2::date
  AND date_to = $3::date
  AND comment = $4
ORDER BY id
LIMIT 1`, teamID, demoPeriodFrom, demoPeriodTo, comment).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("select teamlead reconciliation: %w", err)
	}
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `
INSERT INTO teamlead_reconciliations(team_id, date_from, date_to, status, created_by, confirmed_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, analyzed_at, confirmed_at, apply_queued_at, applied_at)
VALUES ($1, $2, $3, 'applied', $4, $4, $5, $6, $7, 2, 0, 0, $8, $9, $10, $11, $12, now(), now(), now(), now())
RETURNING id`, teamID, demoPeriodFrom, demoPeriodTo, actorID, inboundBatchID, outboundBatchID, comment, pipeline, inboundSummary, outboundSummary, preview, applyResult).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert teamlead reconciliation: %w", err)
		}
		return id, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE teamlead_reconciliations
SET status = 'applied',
    created_by = $2,
    confirmed_by = $2,
    rejected_by = NULL,
    inbound_import_batch_id = $3,
    outbound_import_batch_id = $4,
    mismatch_count = 2,
    conflict_count = 0,
    blocked_count = 0,
    pipeline_json = $5,
    inbound_summary_json = $6,
    outbound_summary_json = $7,
    preview_json = $8,
    apply_result_json = $9,
    error_message = NULL,
    updated_at = now(),
    analyzed_at = COALESCE(analyzed_at, now()),
    confirmed_at = COALESCE(confirmed_at, now()),
    rejected_at = NULL,
    apply_queued_at = COALESCE(apply_queued_at, now()),
    applied_at = COALESCE(applied_at, now())
WHERE id = $1`, id, actorID, inboundBatchID, outboundBatchID, pipeline, inboundSummary, outboundSummary, preview, applyResult); err != nil {
		return 0, fmt.Errorf("update teamlead reconciliation %d: %w", id, err)
	}
	return id, nil
}

type teamleadReconciliationItemSeed struct {
	Direction       string
	Stage           string
	IssueType       string
	Severity        string
	ExternalInnerID string
	TraderID        *int64
	RequisiteID     *int64
	ShiftID         *int64
	Before          map[string]any
	After           map[string]any
	Message         string
	IsBlocking      bool
}

func replaceTeamleadReconciliationItems(ctx context.Context, tx pgx.Tx, runID int64, teamID int64, items []teamleadReconciliationItemSeed) error {
	if _, err := tx.Exec(ctx, `DELETE FROM teamlead_reconciliation_items WHERE teamlead_reconciliation_id = $1`, runID); err != nil {
		return fmt.Errorf("delete teamlead reconciliation items for run %d: %w", runID, err)
	}
	for _, item := range items {
		beforeJSON, err := jsonBytes(item.Before)
		if err != nil {
			return err
		}
		afterJSON, err := jsonBytes(item.After)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO teamlead_reconciliation_items(teamlead_reconciliation_id, team_id, direction, stage, issue_type, severity, external_inner_id, trader_id, requisite_id, shift_id, before_json, after_json, message, is_blocking, applied_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, now())`,
			runID,
			teamID,
			item.Direction,
			item.Stage,
			item.IssueType,
			item.Severity,
			item.ExternalInnerID,
			nullableInt64(item.TraderID),
			nullableInt64(item.RequisiteID),
			nullableInt64(item.ShiftID),
			beforeJSON,
			afterJSON,
			item.Message,
			item.IsBlocking,
		); err != nil {
			return fmt.Errorf("insert teamlead reconciliation item for run %d: %w", runID, err)
		}
	}
	return nil
}

func applyTeamleadStatuses(ctx context.Context, tx pgx.Tx, teamID int64, runID int64, annaShiftID int64) error {
	if _, err := tx.Exec(ctx, `
UPDATE external_orders
SET tl_reconciliation_status = CASE external_inner_id
    WHEN 'seed-in-anna-001' THEN 'confirmed_by_tl'
    WHEN 'seed-in-anna-002' THEN 'updated_by_tl'
    WHEN 'seed-out-anna-001' THEN 'confirmed_by_tl'
    WHEN 'seed-out-anna-002' THEN 'tl_discrepancy'
    ELSE tl_reconciliation_status
END,
last_teamlead_reconciliation_id = $2,
tl_reconciled_at = now()
WHERE team_id = $1
  AND external_inner_id IN ('seed-in-anna-001', 'seed-in-anna-002', 'seed-out-anna-001', 'seed-out-anna-002')`, teamID, runID); err != nil {
		return fmt.Errorf("update external order tl statuses: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE trader_shifts
SET tl_reconciliation_status = 'tl_accepted',
    last_teamlead_reconciliation_id = $2,
    tl_reconciled_at = now()
WHERE team_id = $1
  AND id = $3`, teamID, runID, annaShiftID); err != nil {
		return fmt.Errorf("update shift tl status: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE shift_requisites
SET tl_reconciliation_status = CASE WHEN status = 'blocked' THEN 'tl_discrepancy' ELSE 'tl_accepted' END,
    last_teamlead_reconciliation_id = $2,
    tl_reconciled_at = now(),
    updated_at = now()
WHERE team_id = $1
  AND shift_id = $3`, teamID, runID, annaShiftID); err != nil {
		return fmt.Errorf("update shift requisite tl statuses: %w", err)
	}
	return nil
}

func ensureAudit(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, action string, entityType string, entityID int64, payload map[string]any, comment string) error {
	payloadJSON, err := jsonBytes(payload)
	if err != nil {
		return err
	}
	var id int64
	err = tx.QueryRow(ctx, `
SELECT id
FROM audit_logs
WHERE team_id = $1
  AND actor_id = $2
  AND action = $3
  AND entity_type = $4
  AND entity_id = $5
  AND comment = $6
LIMIT 1`, teamID, actorID, action, entityType, fmt.Sprint(entityID), comment).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("select audit event %s/%s/%d: %w", action, entityType, entityID, err)
	}
	if err == nil {
		if _, err := tx.Exec(ctx, `
UPDATE audit_logs
SET after_json = $2,
    changed_fields_json = $2,
    created_at = now()
WHERE id = $1`, id, payloadJSON); err != nil {
			return fmt.Errorf("update audit event %d: %w", id, err)
		}
		return nil
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_logs(team_id, actor_id, action, entity_type, entity_id, after_json, changed_fields_json, comment)
VALUES ($1, $2, $3, $4, $5, $6, $6, $7)`, teamID, actorID, action, entityType, fmt.Sprint(entityID), payloadJSON, comment); err != nil {
		return fmt.Errorf("insert audit event %s/%s/%d: %w", action, entityType, entityID, err)
	}
	return nil
}

func activeOrder(direction string, externalID string, innerID string, workerName string, traderID int64, req demoRequisite, amountMinor int64, rawStatus string, createdAt time.Time, initials string) orderSeed {
	normalizedStatus := "success"
	if rawStatus == "corrected" {
		normalizedStatus = "corrected"
	}
	if rawStatus == "auto_decline" || rawStatus == "cancelled" {
		normalizedStatus = "failed"
	}
	return orderSeed{
		Direction:         direction,
		ExternalID:        externalID,
		InnerID:           innerID,
		WorkerName:        workerName,
		TraderID:          traderID,
		RequisiteID:       req.ID,
		RequisitePhone:    req.Phone,
		RequisiteRaw:      req.CardNumber,
		RequisiteExternal: fmt.Sprintf("req-%d", req.ID),
		DeviceName:        "seed-device",
		MethodType:        req.MethodType,
		MethodName:        req.BankName,
		AmountMinor:       amountMinor,
		RawStatus:         rawStatus,
		NormalizedStatus:  normalizedStatus,
		CreatedAt:         createdAt,
		Comment:           "Seed v2 demo order",
		Initials:          initials,
	}
}

func jsonBytes(value any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return payload, nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func strPtr(value string) *string {
	return &value
}

func dateUTC(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func timeUTC(year int, month time.Month, day int, hour int, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}
