package orders

import (
	"testing"
	"time"
)

func TestCreatedRangeValuesUsesExclusiveDateTo(t *testing.T) {
	dateFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)

	from, toExclusive := createdRangeValues(Filters{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	})

	if !from.Valid || !from.Time.Equal(dateFrom) {
		t.Fatalf("from = %v/%t, want %s/valid", from.Time, from.Valid, dateFrom)
	}
	wantToExclusive := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	if !toExclusive.Valid || !toExclusive.Time.Equal(wantToExclusive) {
		t.Fatalf("toExclusive = %v/%t, want %s/valid", toExclusive.Time, toExclusive.Valid, wantToExclusive)
	}
}
