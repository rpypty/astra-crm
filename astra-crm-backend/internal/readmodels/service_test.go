package readmodels

import (
	"strings"
	"testing"
)

func TestTraderProfilePeriodSuccessUsesTraderShiftScope(t *testing.T) {
	periodSuccessStart := strings.Index(traderProfileQuery, "period_success AS")
	if periodSuccessStart < 0 {
		t.Fatal("period_success CTE is missing")
	}
	selectStart := strings.Index(traderProfileQuery[periodSuccessStart:], "\nSELECT\n")
	if selectStart < 0 {
		t.Fatal("period_success CTE end is missing")
	}
	periodSuccess := traderProfileQuery[periodSuccessStart : periodSuccessStart+selectStart]

	if !strings.Contains(periodSuccess, "scope_type = 'trader_shift'") {
		t.Fatalf("period_success should use trader shift scope: %s", periodSuccess)
	}
	if strings.Contains(periodSuccess, "scope_type = 'teamlead_period'") {
		t.Fatalf("period_success should not depend on teamlead period imports: %s", periodSuccess)
	}
	if !strings.Contains(periodSuccess, "osi.trader_id = $2") {
		t.Fatalf("period_success should be scoped to current trader: %s", periodSuccess)
	}
	if !strings.Contains(periodSuccess, "ts.status IN ('closed', 'closed_with_discrepancy')") {
		t.Fatalf("period_success should require closed shift: %s", periodSuccess)
	}
	if !strings.Contains(periodSuccess, "ts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')") {
		t.Fatalf("period_success should require confirmed inbound reconciliation: %s", periodSuccess)
	}
	if !strings.Contains(periodSuccess, "osi.created_at_external::date >= $3::date") {
		t.Fatalf("period_success should apply date_from filter: %s", periodSuccess)
	}
	if !strings.Contains(periodSuccess, "osi.created_at_external::date <= $4::date") {
		t.Fatalf("period_success should apply date_to filter: %s", periodSuccess)
	}
}
