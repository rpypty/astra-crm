package reconciliation

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildTraderInboundReconciliationPlanMatched(t *testing.T) {
	plan, err := buildTraderInboundReconciliationPlan(
		[]traderInboundRequisiteRow{{
			shiftRequisiteID: 10,
			requisiteID:      20,
			phone:            "+7 999 111-22-33",
			bankName:         "Т-Банк",
			actualAmount:     10000,
		}},
		[]traderInboundReviewRequisiteRow{{
			shiftRequisiteID: 10,
			requisiteID:      20,
			phone:            "+7 999 111-22-33",
			actualAmount:     10000,
		}},
		[]traderInboundScopeItem{
			{csvRequisite: "9991112233", amountMinor: 4000, normalizedStatus: "success"},
			{csvRequisite: "9991112233", amountMinor: 6000, normalizedStatus: "corrected"},
			{csvRequisite: "9991112233", amountMinor: 5000, normalizedStatus: "failed"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.items) != 0 {
		t.Fatalf("items count = %d, want 0", len(plan.items))
	}
	if len(plan.statuses) != 1 || plan.statuses[0].shiftRequisiteID != 10 || plan.statuses[0].status != "worked_verified" {
		t.Fatalf("statuses = %+v, want one worked_verified status", plan.statuses)
	}
}

func TestBuildTraderInboundReconciliationPlanRequisiteMismatch(t *testing.T) {
	plan, err := buildTraderInboundReconciliationPlan(
		[]traderInboundRequisiteRow{{
			shiftRequisiteID: 10,
			requisiteID:      20,
			phone:            "+7 999 111-22-33",
			bankName:         "Сбер",
			actualAmount:     8000,
		}},
		[]traderInboundReviewRequisiteRow{{
			shiftRequisiteID: 10,
			requisiteID:      20,
			phone:            "+7 999 111-22-33",
			actualAmount:     8000,
		}},
		[]traderInboundScopeItem{{csvRequisite: "79991112233", amountMinor: 10000, normalizedStatus: "success"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.items) != 1 {
		t.Fatalf("items count = %d, want 1", len(plan.items))
	}
	item := plan.items[0]
	if item.issueType != traderInboundRequisiteMismatchIssue {
		t.Fatalf("issueType = %q, want %q", item.issueType, traderInboundRequisiteMismatchIssue)
	}
	var teamlead traderInboundTeamleadPayload
	decodeRawJSON(t, item.teamleadValueJSON, &teamlead)
	if teamlead.ExpectedAmountMinor != 10000 || teamlead.SuccessCount != 1 || teamlead.Requisite == nil || *teamlead.Requisite != "79991112233" {
		t.Fatalf("teamlead payload = %+v, want CSV amount/count/requisite", teamlead)
	}
	var trader traderInboundTraderPayload
	decodeRawJSON(t, item.traderValueJSON, &trader)
	if trader.ShiftRequisiteID != 10 || trader.RequisiteID != 20 || trader.ActualAmountMinor != 8000 || trader.BankName != "Сбер" {
		t.Fatalf("trader payload = %+v, want CRM requisite payload", trader)
	}
	if len(plan.statuses) != 1 || plan.statuses[0].status != "worked_discrepancy" {
		t.Fatalf("statuses = %+v, want worked_discrepancy", plan.statuses)
	}
}

func TestBuildTraderInboundReconciliationPlanCSVOnly(t *testing.T) {
	plan, err := buildTraderInboundReconciliationPlan(nil, nil, []traderInboundScopeItem{{
		csvRequisite:     "9991112233",
		amountMinor:      12000,
		normalizedStatus: "success",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.items) != 1 {
		t.Fatalf("items count = %d, want 1", len(plan.items))
	}
	item := plan.items[0]
	if item.issueType != traderInboundRequisiteMismatchIssue {
		t.Fatalf("issueType = %q, want %q", item.issueType, traderInboundRequisiteMismatchIssue)
	}
	if len(item.traderValueJSON) != 0 {
		t.Fatalf("trader payload = %s, want nil for CSV-only mismatch", string(item.traderValueJSON))
	}
	var teamlead traderInboundTeamleadPayload
	decodeRawJSON(t, item.teamleadValueJSON, &teamlead)
	if teamlead.ExpectedAmountMinor != 12000 || teamlead.Requisite == nil || *teamlead.Requisite != "9991112233" {
		t.Fatalf("teamlead payload = %+v, want CSV-only payload", teamlead)
	}
}

func TestTraderReconciliationStatus(t *testing.T) {
	if got := traderReconciliationStatus(0, 0); got != StatusMatched {
		t.Fatalf("status for no diff/no items = %q, want %q", got, StatusMatched)
	}
	if got := traderReconciliationStatus(100, 0); got != StatusMismatch {
		t.Fatalf("status for total diff = %q, want %q", got, StatusMismatch)
	}
	if got := traderReconciliationStatus(0, 1); got != StatusMismatch {
		t.Fatalf("status for item mismatch = %q, want %q", got, StatusMismatch)
	}
}

func TestBuildTraderOutboundReconciliationItemsMatched(t *testing.T) {
	base := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	items, err := buildTraderOutboundReconciliationItems(
		[]traderOutboundSourceRequisiteRow{{
			shiftRequisiteID:            10,
			requisiteID:                 20,
			phone:                       "+7 999 111-22-33",
			bankName:                    "Т-Банк",
			closedOutboundTurnoverMinor: 10000,
		}},
		[]traderOutboundScopeItem{{
			id:               1,
			externalOrderID:  101,
			externalInnerID:  "out-1",
			amountMinor:      10000,
			normalizedStatus: "success",
			createdAt:        base,
		}},
		[]traderOutboundPayoutOrderRow{{
			id:                   30,
			destinationBank:      "Сбер",
			destinationRequisite: "5533",
			amountMinor:          10000,
			createdAt:            base,
		}},
		[]traderOutboundTransferRow{{
			manualPayoutOrderID:    30,
			sourceShiftRequisiteID: 10,
			amountMinor:            10000,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items count = %d, want 0", len(items))
	}
}

func TestBuildTraderOutboundReconciliationItemsSourceMismatch(t *testing.T) {
	items, err := buildTraderOutboundReconciliationItems(
		[]traderOutboundSourceRequisiteRow{{
			shiftRequisiteID:            10,
			requisiteID:                 20,
			phone:                       "+7 999 111-22-33",
			bankName:                    "Т-Банк",
			closedOutboundTurnoverMinor: 10000,
		}},
		nil,
		nil,
		[]traderOutboundTransferRow{{sourceShiftRequisiteID: 10, amountMinor: 4000}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items count = %d, want 1", len(items))
	}
	item := items[0]
	if item.issueType != traderOutboundSourceRequisiteMismatchIssue {
		t.Fatalf("issueType = %q, want %q", item.issueType, traderOutboundSourceRequisiteMismatchIssue)
	}
	var payload traderOutboundSourcePayload
	decodeRawJSON(t, item.traderValueJSON, &payload)
	if payload.TransferAmountMinor != 4000 || payload.DiffAmountMinor != 6000 {
		t.Fatalf("payload = %+v, want source transfer diff", payload)
	}
}

func TestBuildTraderOutboundReconciliationItemsMissingManualPayout(t *testing.T) {
	items, err := buildTraderOutboundReconciliationItems(
		nil,
		[]traderOutboundScopeItem{{
			id:               1,
			externalOrderID:  101,
			externalInnerID:  "out-1",
			amountMinor:      10000,
			normalizedStatus: "corrected",
			createdAt:        time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
		}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items count = %d, want 1", len(items))
	}
	item := items[0]
	if item.issueType != traderOutboundMissingManualPayoutIssue {
		t.Fatalf("issueType = %q, want %q", item.issueType, traderOutboundMissingManualPayoutIssue)
	}
	if item.externalInnerID == nil || *item.externalInnerID != "out-1" {
		t.Fatalf("externalInnerID = %v, want out-1", item.externalInnerID)
	}
	var payload traderOutboundCSVPayload
	decodeRawJSON(t, item.teamleadValueJSON, &payload)
	if payload.AmountMinor != 10000 || payload.NormalizedStatus != "corrected" {
		t.Fatalf("payload = %+v, want CSV payout payload", payload)
	}
}

func TestBuildTraderOutboundReconciliationItemsExtraManualPayout(t *testing.T) {
	base := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	items, err := buildTraderOutboundReconciliationItems(
		nil,
		nil,
		[]traderOutboundPayoutOrderRow{{
			id:                   30,
			destinationBank:      "Сбер",
			destinationRequisite: "5533",
			amountMinor:          10000,
			createdAt:            base,
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items count = %d, want 1", len(items))
	}
	item := items[0]
	if item.issueType != traderOutboundExtraManualPayoutIssue {
		t.Fatalf("issueType = %q, want %q", item.issueType, traderOutboundExtraManualPayoutIssue)
	}
	if len(item.teamleadValueJSON) != 0 {
		t.Fatalf("teamlead payload = %s, want nil for extra payout", string(item.teamleadValueJSON))
	}
	var payload traderOutboundPayoutPayload
	decodeRawJSON(t, item.traderValueJSON, &payload)
	if payload.ManualPayoutOrderID != 30 || payload.RemainingAmountMinor != 10000 {
		t.Fatalf("payload = %+v, want unpaid manual payout payload", payload)
	}
}

func TestBuildTraderOutboundReconciliationItemsManualPayoutNotFullyPaid(t *testing.T) {
	base := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	items, err := buildTraderOutboundReconciliationItems(
		nil,
		[]traderOutboundScopeItem{{
			id:               1,
			externalOrderID:  101,
			externalInnerID:  "out-1",
			amountMinor:      10000,
			normalizedStatus: "success",
			createdAt:        base,
		}},
		[]traderOutboundPayoutOrderRow{{
			id:                   30,
			destinationBank:      "Сбер",
			destinationRequisite: "5533",
			amountMinor:          10000,
			createdAt:            base,
		}},
		[]traderOutboundTransferRow{{manualPayoutOrderID: 30, amountMinor: 7000}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items count = %d, want 1", len(items))
	}
	item := items[0]
	if item.issueType != traderOutboundManualPayoutNotPaidIssue {
		t.Fatalf("issueType = %q, want %q", item.issueType, traderOutboundManualPayoutNotPaidIssue)
	}
	var payload traderOutboundPayoutPayload
	decodeRawJSON(t, item.traderValueJSON, &payload)
	if payload.PaidAmountMinor != 7000 || payload.RemainingAmountMinor != 3000 {
		t.Fatalf("payload = %+v, want partial payment payload", payload)
	}
}

func decodeRawJSON(t *testing.T, raw json.RawMessage, target any) {
	t.Helper()
	if len(raw) == 0 {
		t.Fatal("raw JSON is empty")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode raw JSON: %v\npayload: %s", err, string(raw))
	}
}
