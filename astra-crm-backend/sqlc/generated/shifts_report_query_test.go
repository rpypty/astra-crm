package sqlc

import (
	"strings"
	"testing"
)

func TestListShiftReportRowsResolvesCsvCardsToShiftRequisites(t *testing.T) {
	if !strings.Contains(listShiftReportRows, "'card:' || card_match_key") {
		t.Fatalf("report rows query should index CRM shift requisites by full card number")
	}
	if !strings.Contains(listShiftReportRows, "card_match.lookup_key = 'card:' || source.csv_digits") {
		t.Fatalf("report rows query should resolve CSV card values to CRM shift requisite rows")
	}
	if !strings.Contains(listShiftReportRows, "length(source.csv_digits) >= 12") {
		t.Fatalf("report rows query should treat long digit values as card numbers, not phone tails")
	}
	if !strings.Contains(listShiftReportRows, "HAVING count(DISTINCT shift_requisite_id) = 1") {
		t.Fatalf("report rows query should avoid auto-matching ambiguous card or phone keys")
	}
}
