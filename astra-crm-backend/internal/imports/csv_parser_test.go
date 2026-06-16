package imports

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseCSVTeamleadColumns(t *testing.T) {
	result := parseFixture(t, "../../testdata/csv/teamlead.csv")

	if result.ColumnSet != ColumnSetTeamlead {
		t.Fatalf("column set = %q, want %q", result.ColumnSet, ColumnSetTeamlead)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("rows count = %d, want 2", len(result.Rows))
	}

	first := result.Rows[0]
	if first.ExternalID != "100" || first.ExternalInnerID != "in-100" {
		t.Fatalf("external ids = %q/%q, want 100/in-100", first.ExternalID, first.ExternalInnerID)
	}
	if first.AmountMinor != 300100 {
		t.Fatalf("amount minor = %d, want 300100", first.AmountMinor)
	}
	if first.ClosedAtExternal != nil {
		t.Fatalf("closedAt = %v, want nil", first.ClosedAtExternal)
	}
	if first.OldAmountMinor != nil {
		t.Fatalf("oldAmount = %v, want nil", *first.OldAmountMinor)
	}
	if first.Receipt != nil {
		t.Fatalf("receipt = %v, want nil", *first.Receipt)
	}
	if first.NormalizedStatus != NormalizedStatusSuccess {
		t.Fatalf("normalized status = %q, want %q", first.NormalizedStatus, NormalizedStatusSuccess)
	}
	if first.CreatedAtExternal.Location().String() != "UTC+3" {
		t.Fatalf("createdAt location = %q, want UTC+3", first.CreatedAtExternal.Location().String())
	}

	second := result.Rows[1]
	if second.AmountMinor != 306850 {
		t.Fatalf("second amount minor = %d, want 306850", second.AmountMinor)
	}
	if second.OldAmountMinor == nil || *second.OldAmountMinor != 300000 {
		t.Fatalf("second oldAmount = %v, want 300000", second.OldAmountMinor)
	}
	if second.HadDispute == nil || !*second.HadDispute {
		t.Fatalf("hadDispute = %v, want true", second.HadDispute)
	}
	if second.NormalizedStatus != NormalizedStatusCorrected {
		t.Fatalf("normalized status = %q, want %q", second.NormalizedStatus, NormalizedStatusCorrected)
	}
	if result.RawStatusCounts["corrected"] != 1 {
		t.Fatalf("corrected raw count = %d, want 1", result.RawStatusCounts["corrected"])
	}
}

func TestParseCSVTraderColumnsAndUnknownStatus(t *testing.T) {
	result := parseFixture(t, "../../testdata/csv/trader.csv")

	if result.ColumnSet != ColumnSetTrader {
		t.Fatalf("column set = %q, want %q", result.ColumnSet, ColumnSetTrader)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("rows count = %d, want 2", len(result.Rows))
	}

	first := result.Rows[0]
	if first.RequisiteExternalID == nil || *first.RequisiteExternalID != "req-1" {
		t.Fatalf("requisiteId = %v, want req-1", first.RequisiteExternalID)
	}
	if first.WorkerProfit == nil || *first.WorkerProfit != "50.0" {
		t.Fatalf("workerProfit = %v, want 50.0", first.WorkerProfit)
	}
	if first.NormalizedStatus != NormalizedStatusFailed {
		t.Fatalf("normalized status = %q, want %q", first.NormalizedStatus, NormalizedStatusFailed)
	}

	second := result.Rows[1]
	if second.NormalizedStatus != NormalizedStatusUnknown {
		t.Fatalf("normalized status = %q, want %q", second.NormalizedStatus, NormalizedStatusUnknown)
	}
	if second.Ordered != nil || second.Counted != nil || second.Initials != nil {
		t.Fatalf("None trader optional fields must become nil")
	}
	if result.NormalizedStatusCounts[NormalizedStatusUnknown] != 1 {
		t.Fatalf("unknown count = %d, want 1", result.NormalizedStatusCounts[NormalizedStatusUnknown])
	}
}

func TestParseCSVTraderPayoutReportWithoutWorkerName(t *testing.T) {
	result := parseFixtureWithOptions(t, "../../../docs/dealsPayoutReport-1781288762.csv", ParseOptions{
		ColumnSet:         ColumnSetTrader,
		Location:          time.FixedZone("UTC+3", 3*60*60),
		DefaultWorkerName: "trader:3",
	})

	if result.ColumnSet != ColumnSetTrader {
		t.Fatalf("column set = %q, want %q", result.ColumnSet, ColumnSetTrader)
	}
	if len(result.Rows) != 659 {
		t.Fatalf("rows count = %d, want 659", len(result.Rows))
	}
	first := result.Rows[0]
	if first.ExternalInnerID != "4c647d41-da94-41a8-914a-3342b92e5727" {
		t.Fatalf("innerId = %q, want first payout innerId", first.ExternalInnerID)
	}
	if first.WorkerName != "trader:3" {
		t.Fatalf("worker name = %q, want fallback trader:3", first.WorkerName)
	}
	if first.AmountMinor != 700000 {
		t.Fatalf("amount minor = %d, want 700000", first.AmountMinor)
	}
	if first.NormalizedStatus != NormalizedStatusSuccess {
		t.Fatalf("normalized status = %q, want success", first.NormalizedStatus)
	}
	if result.RawStatusCounts["paid"] != 507 || result.RawStatusCounts["cancelled"] != 152 {
		t.Fatalf("raw status counts = %+v, want paid=507 cancelled=152", result.RawStatusCounts)
	}
}

func TestParseCSVTeamleadPayoutReport(t *testing.T) {
	result := parseFixtureWithOptions(t, "../../../docs/payoutReport-1781288788.csv", ParseOptions{
		ColumnSet: ColumnSetTeamlead,
		Location:  time.FixedZone("UTC+3", 3*60*60),
	})

	if result.ColumnSet != ColumnSetTeamlead {
		t.Fatalf("column set = %q, want %q", result.ColumnSet, ColumnSetTeamlead)
	}
	if len(result.Rows) != 8029 {
		t.Fatalf("rows count = %d, want 8029", len(result.Rows))
	}
	first := result.Rows[0]
	if first.WorkerName != "Bliss_OP8" {
		t.Fatalf("worker name = %q, want Bliss_OP8", first.WorkerName)
	}
	if first.OrderComment == nil || *first.OrderComment != "Максимум 3 чека" {
		t.Fatalf("comment = %v, want payout comment", first.OrderComment)
	}
	if first.NormalizedStatus != NormalizedStatusUnknown {
		t.Fatalf("first normalized status = %q, want unknown for pending", first.NormalizedStatus)
	}
	if result.RawStatusCounts["paid"] != 7077 || result.RawStatusCounts["pending"] != 5 || result.RawStatusCounts["chargeback"] != 3 {
		t.Fatalf("raw status counts = %+v, want paid=7077 pending=5 chargeback=3", result.RawStatusCounts)
	}
}

func TestParseMoneyMinor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{name: "one fractional digit", input: "3001.0", want: 300100},
		{name: "two fractional digits", input: "10.25", want: 1025},
		{name: "integer", input: "0", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseMoneyMinor(test.input)
			if err != nil {
				t.Fatalf("ParseMoneyMinor() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ParseMoneyMinor() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestParseCSVRejectsDuplicateInnerID(t *testing.T) {
	csvBody := strings.Join([]string{
		"id|innerId|amount|currency|status|createdAt|workerName",
		"1|dup-1|10.0|RUB|hand_success|28.05.2026 11:21:35|Bliss_OP2",
		"2|dup-1|20.0|RUB|hand_success|28.05.2026 11:22:35|Bliss_OP2",
	}, "\n")

	result, err := ParseCSV(strings.NewReader(csvBody), parseOptions())
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ParseCSV() error = %v, want ErrValidation", err)
	}
	if result.Valid() {
		t.Fatal("result is valid, want validation errors")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors count = %d, want 1", len(result.Errors))
	}
	validationError := result.Errors[0]
	if validationError.Code != ParseCodeDuplicateInnerID {
		t.Fatalf("error code = %q, want %q", validationError.Code, ParseCodeDuplicateInnerID)
	}
	if validationError.InnerID != "dup-1" {
		t.Fatalf("innerId = %q, want dup-1", validationError.InnerID)
	}
	if len(validationError.Rows) != 2 || validationError.Rows[0] != 2 || validationError.Rows[1] != 3 {
		t.Fatalf("duplicate rows = %v, want [2 3]", validationError.Rows)
	}
}

func TestParseCSVRejectsMissingRequiredColumns(t *testing.T) {
	csvBody := strings.Join([]string{
		"id|innerId|amount",
		"1|in-1|10.0",
	}, "\n")

	result, err := ParseCSV(strings.NewReader(csvBody), parseOptions())
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ParseCSV() error = %v, want ErrValidation", err)
	}
	if result.Valid() {
		t.Fatal("result is valid, want validation errors")
	}
	if len(result.Errors) == 0 {
		t.Fatal("errors are empty, want missing required column errors")
	}
	for _, validationError := range result.Errors {
		if validationError.Code != ParseCodeMissingRequiredColumn {
			t.Fatalf("error code = %q, want %q", validationError.Code, ParseCodeMissingRequiredColumn)
		}
	}
}

func TestParseCSVRejectsInvalidMoney(t *testing.T) {
	csvBody := strings.Join([]string{
		"id|innerId|amount|currency|status|createdAt|workerName",
		"1|in-1|10.234|RUB|hand_success|28.05.2026 11:21:35|Bliss_OP2",
	}, "\n")

	result, err := ParseCSV(strings.NewReader(csvBody), parseOptions())
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ParseCSV() error = %v, want ErrValidation", err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors count = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Code != ParseCodeInvalidMoney {
		t.Fatalf("error code = %q, want %q", result.Errors[0].Code, ParseCodeInvalidMoney)
	}
}

func parseFixture(t *testing.T, path string) ParseResult {
	t.Helper()

	return parseFixtureWithOptions(t, path, parseOptions())
}

func parseFixtureWithOptions(t *testing.T, path string, options ParseOptions) ParseResult {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	result, err := ParseCSV(file, options)
	if err != nil {
		t.Fatalf("ParseCSV() error = %v; validation errors: %+v", err, result.Errors)
	}

	return result
}

func parseOptions() ParseOptions {
	return ParseOptions{
		ColumnSet: ColumnSetAuto,
		Location:  time.FixedZone("UTC+3", 3*60*60),
	}
}
