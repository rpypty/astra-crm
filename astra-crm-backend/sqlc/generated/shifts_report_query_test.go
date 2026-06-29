package sqlc

import (
	"strings"
	"testing"
)

func TestShiftReportQueriesAreSimpleReads(t *testing.T) {
	reportQueries := []string{
		listShiftReportRequisites,
		listShiftReportInboundScopeItems,
		listShiftReportOutboundTransfers,
	}
	for _, query := range reportQueries {
		for _, forbidden := range []string{"regexp_replace", "UNION", "GROUP BY", "jsonb_build_object"} {
			if strings.Contains(query, forbidden) {
				t.Fatalf("report query contains %q, want simple read:\n%s", forbidden, query)
			}
		}
	}

	if !strings.Contains(listShiftReportRequisites, "r.card_number AS requisite_card_number") {
		t.Fatalf("report requisites query should load requisite card fallback:\n%s", listShiftReportRequisites)
	}
	if !strings.Contains(listShiftReportInboundScopeItems, "direction = 'inbound'") {
		t.Fatalf("report inbound query should load only active inbound scope items:\n%s", listShiftReportInboundScopeItems)
	}
	if !strings.Contains(listShiftReportOutboundTransfers, "source_shift_requisite_id") {
		t.Fatalf("report transfers query should expose source shift requisite:\n%s", listShiftReportOutboundTransfers)
	}
}
