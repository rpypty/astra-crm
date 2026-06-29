package shifts

import (
	"reflect"
	"testing"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

func TestMatchingBankCodes(t *testing.T) {
	bankNames := map[string]string{
		"SBER": "Сбербанк",
		"TCS":  "Т-Банк",
		"VTB":  "ВТБ",
	}

	tests := []struct {
		name   string
		search string
		want   []string
	}{
		{name: "matches by name", search: "сбер", want: []string{"SBER"}},
		{name: "matches by code", search: "tc", want: []string{"TCS"}},
		{name: "trims search", search: "  втб ", want: []string{"VTB"}},
		{name: "unknown or archived", search: "райф", want: nil},
		{name: "empty search", search: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchingBankCodes(bankNames, tt.search)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("matchingBankCodes() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAssignedRowEnrichesBankName(t *testing.T) {
	item := fromAssignedRow(db.ListAssignedRequisitesForTraderRow{BankCode: "SBER"}, map[string]string{
		"SBER": "Сбербанк",
	})
	if item.BankName != "Сбербанк" {
		t.Fatalf("BankName = %q, want %q", item.BankName, "Сбербанк")
	}

	item = fromAssignedRow(db.ListAssignedRequisitesForTraderRow{BankCode: "ARCHIVED"}, map[string]string{
		"SBER": "Сбербанк",
	})
	if item.BankName != "" {
		t.Fatalf("archived BankName = %q, want empty", item.BankName)
	}
}

func TestInternalTransferRowEnrichesBankNames(t *testing.T) {
	item := fromListInternalTransferRow(db.ListInternalTransfersForShiftRequisiteRow{
		SourceBankCode:      "SBER",
		DestinationBankCode: "TCS",
	}, map[string]string{
		"SBER": "Сбербанк",
		"TCS":  "Т-Банк",
	})

	if item.SourceBankName != "Сбербанк" {
		t.Fatalf("SourceBankName = %q, want %q", item.SourceBankName, "Сбербанк")
	}
	if item.DestinationBankName != "Т-Банк" {
		t.Fatalf("DestinationBankName = %q, want %q", item.DestinationBankName, "Т-Банк")
	}
}
