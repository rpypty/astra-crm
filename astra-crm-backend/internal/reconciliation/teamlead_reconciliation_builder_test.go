package reconciliation

import (
	"testing"
	"time"
)

func TestBuildTeamleadPeriodInboundReconciliationMatched(t *testing.T) {
	result, err := buildTeamleadPeriodInboundReconciliation(
		[]teamleadScopeOrderRow{teamleadOrder("in-1", 10000, "success", withRequisitePhone("79991112233"))},
		[]teamleadPeriodInboundShiftRequisiteRow{{
			shiftRequisiteID: 10,
			requisiteID:      20,
			requisitePhone:   "79991112233",
			bankCode:         "tbank",
			traderID:         30,
			traderLogin:      "trader",
			amountMinor:      10000,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.summary.ExpectedAmountMinor != 10000 || result.summary.ActualAmountMinor != 10000 {
		t.Fatalf("summary = %+v, want matched amounts", result.summary)
	}
	if len(result.items) != 0 {
		t.Fatalf("items count = %d, want 0", len(result.items))
	}
	if got := teamleadReconciliationStatus(result.summary, len(result.items)); got != StatusMatched {
		t.Fatalf("status = %q, want %q", got, StatusMatched)
	}
}

func TestBuildTeamleadPeriodInboundReconciliationTotalMismatch(t *testing.T) {
	result, err := buildTeamleadPeriodInboundReconciliation(
		[]teamleadScopeOrderRow{teamleadOrder("in-1", 10000, "success", withRequisitePhone("79991112233"))},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssue(result.items, teamleadTotalAmountMismatchIssue) {
		t.Fatalf("issues = %v, want total mismatch", issueTypes(result.items))
	}
	if got := teamleadReconciliationStatus(result.summary, len(result.items)); got != StatusMismatch {
		t.Fatalf("status = %q, want %q", got, StatusMismatch)
	}
}

func TestBuildTeamleadPeriodInboundReconciliationRequisiteMismatch(t *testing.T) {
	result, err := buildTeamleadPeriodInboundReconciliation(
		[]teamleadScopeOrderRow{
			teamleadOrder("in-1", 10000, "success", withRequisitePhone("79991112233")),
			teamleadOrder("in-2", 5000, "corrected", withRequisitePhone("79992223344")),
		},
		[]teamleadPeriodInboundShiftRequisiteRow{
			{requisiteID: 20, requisitePhone: "79991112233", bankCode: "sber", traderID: 30, traderLogin: "trader", amountMinor: 8000},
			{requisiteID: 21, requisitePhone: "79992223344", bankCode: "tbank", traderID: 31, traderLogin: "trader2", amountMinor: 7000},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if hasIssue(result.items, teamleadTotalAmountMismatchIssue) {
		t.Fatalf("issues = %v, did not expect total mismatch", issueTypes(result.items))
	}
	count := countIssue(result.items, teamleadRequisiteAmountMismatchIssue)
	if count != 2 {
		t.Fatalf("requisite mismatch count = %d, want 2; issues=%v", count, issueTypes(result.items))
	}
}

func TestBuildTeamleadPeriodOutboundReconciliationMatched(t *testing.T) {
	result, err := buildTeamleadPeriodOutboundReconciliation(
		[]teamleadScopeOrderRow{teamleadOrder("out-1", 10000, "success")},
		[]teamleadPeriodPayoutOrderRow{{id: 10, amountMinor: 10000, destinationBank: "Сбер", destinationRequisite: "5533", traderID: 20, traderLogin: "trader"}},
		[]teamleadPeriodPayoutTransferRow{{manualPayoutOrderID: 10, amountMinor: 10000}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.items) != 0 {
		t.Fatalf("items count = %d, want 0", len(result.items))
	}
	if got := teamleadReconciliationStatus(result.summary, len(result.items)); got != StatusMatched {
		t.Fatalf("status = %q, want %q", got, StatusMatched)
	}
}

func TestBuildTeamleadPeriodOutboundReconciliationPayoutMismatches(t *testing.T) {
	result, err := buildTeamleadPeriodOutboundReconciliation(
		[]teamleadScopeOrderRow{
			teamleadOrder("out-missing", 10000, "success"),
			teamleadOrder("out-partial", 7000, "success"),
		},
		[]teamleadPeriodPayoutOrderRow{
			{id: 10, amountMinor: 5000, destinationBank: "Сбер", destinationRequisite: "1111", traderID: 20, traderLogin: "trader"},
			{id: 11, amountMinor: 7000, destinationBank: "Т-Банк", destinationRequisite: "2222", traderID: 21, traderLogin: "trader2"},
		},
		[]teamleadPeriodPayoutTransferRow{{manualPayoutOrderID: 11, amountMinor: 3000}},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range []string{
		traderOutboundMissingManualPayoutIssue,
		traderOutboundExtraManualPayoutIssue,
		traderOutboundManualPayoutNotPaidIssue,
	} {
		if !hasIssue(result.items, issue) {
			t.Fatalf("issues = %v, want %q", issueTypes(result.items), issue)
		}
	}
}

func TestBuildTeamleadCurrentReconciliationOrderMismatches(t *testing.T) {
	result, err := buildTeamleadCurrentReconciliation(
		[]teamleadScopeOrderRow{
			teamleadOrder("missing", 1000, "success"),
			teamleadOrder("amount", 2000, "success"),
			teamleadOrder("status", 3000, "success"),
			teamleadOrder("worker", 4000, "success", withWorker("alice")),
		},
		[]teamleadScopeOrderRow{
			teamleadOrder("amount", 2500, "success"),
			teamleadOrder("status", 3000, "failed"),
			teamleadOrder("worker", 4000, "success", withWorker("bob")),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range []string{
		teamleadMissingTraderImportIssue,
		teamleadAmountMismatchIssue,
		teamleadStatusMismatchIssue,
		teamleadWorkerMismatchIssue,
	} {
		if !hasIssue(result.items, issue) {
			t.Fatalf("issues = %v, want %q", issueTypes(result.items), issue)
		}
	}
}

type teamleadOrderOption func(*teamleadScopeOrderRow)

func teamleadOrder(innerID string, amount int64, status string, options ...teamleadOrderOption) teamleadScopeOrderRow {
	base := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	row := teamleadScopeOrderRow{
		id:                int64(len(innerID)),
		externalOrderID:   int64(len(innerID) + 100),
		externalInnerID:   innerID,
		workerName:        "worker",
		amountMinor:       amount,
		normalizedStatus:  status,
		rawStatus:         status,
		createdAtExternal: base,
		createdAt:         base,
	}
	for _, option := range options {
		option(&row)
	}
	return row
}

func withRequisitePhone(value string) teamleadOrderOption {
	return func(row *teamleadScopeOrderRow) {
		row.requisitePhone = traderStringPtr(value)
	}
}

func withWorker(value string) teamleadOrderOption {
	return func(row *teamleadScopeOrderRow) {
		row.workerName = value
	}
}

func issueTypes(items []traderReconciliationItemRecord) []string {
	issues := make([]string, 0, len(items))
	for _, item := range items {
		issues = append(issues, item.issueType)
	}
	return issues
}

func hasIssue(items []traderReconciliationItemRecord, issue string) bool {
	return countIssue(items, issue) > 0
}

func countIssue(items []traderReconciliationItemRecord, issue string) int {
	count := 0
	for _, item := range items {
		if item.issueType == issue {
			count++
		}
	}
	return count
}
