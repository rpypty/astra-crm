package readmodels

import (
	"strings"
	"testing"
	"time"
)

func TestTraderProfilePeriodSuccessUsesTraderShiftScope(t *testing.T) {
	if !strings.Contains(traderSuccessInboundAmountQuery, "scope_type = 'trader_shift'") {
		t.Fatalf("success query should use trader shift scope: %s", traderSuccessInboundAmountQuery)
	}
	if strings.Contains(traderSuccessInboundAmountQuery, "scope_type = 'teamlead_period'") {
		t.Fatalf("success query should not depend on teamlead period imports: %s", traderSuccessInboundAmountQuery)
	}
	if !strings.Contains(traderSuccessInboundAmountQuery, "osi.trader_id = $2") {
		t.Fatalf("success query should be scoped to current trader: %s", traderSuccessInboundAmountQuery)
	}
	if !strings.Contains(traderSuccessInboundAmountQuery, "ts.status IN ('closed', 'closed_with_discrepancy')") {
		t.Fatalf("success query should require closed shift: %s", traderSuccessInboundAmountQuery)
	}
	if !strings.Contains(traderSuccessInboundAmountQuery, "ts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')") {
		t.Fatalf("success query should require confirmed inbound reconciliation: %s", traderSuccessInboundAmountQuery)
	}
	if !strings.Contains(traderSuccessInboundAmountQuery, "osi.created_at_external >= $3::timestamptz") {
		t.Fatalf("success query should apply date_from range filter: %s", traderSuccessInboundAmountQuery)
	}
	if !strings.Contains(traderSuccessInboundAmountQuery, "osi.created_at_external < $4::timestamptz") {
		t.Fatalf("success query should apply to_exclusive range filter: %s", traderSuccessInboundAmountQuery)
	}
	if strings.Contains(traderSuccessInboundAmountQuery, "created_at_external::date") {
		t.Fatalf("success query should not cast indexed timestamp column: %s", traderSuccessInboundAmountQuery)
	}
}

func TestPeriodFormatting(t *testing.T) {
	from := time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)

	if got := periodTitle(from, to); got != "Период 01.06.2026 - 30.06.2026" {
		t.Fatalf("periodTitle() = %q", got)
	}
	if got := periodDateRange(from, to); got != "01.06.2026 - 30.06.2026" {
		t.Fatalf("periodDateRange() = %q", got)
	}
	if got := filteredPeriodTitle(&from, nil); got != "Период с 01.06.2026" {
		t.Fatalf("filteredPeriodTitle(from) = %q", got)
	}
	if got := filteredPeriodTitle(nil, &to); got != "Период до 30.06.2026" {
		t.Fatalf("filteredPeriodTitle(to) = %q", got)
	}
}

func TestDateRangesAndSalary(t *testing.T) {
	value := time.Date(2026, 6, 15, 20, 30, 0, 0, time.UTC)

	start := dayStart(value)
	if !start.Equal(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("dayStart() = %s", start)
	}
	if got := dayAfter(value); !got.Equal(time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("dayAfter() = %s", got)
	}
	if got := salaryMinor(123456, 75); got != 925 {
		t.Fatalf("salaryMinor() = %d, want 925", got)
	}
}
