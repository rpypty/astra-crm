package requisites

import (
	"testing"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

func TestMatchingBankCodes(t *testing.T) {
	banks := map[string]string{
		"sber":  "Сбер",
		"tbank": "Т-Банк",
		"ozon":  "Ozon Банк",
	}

	cases := []struct {
		name   string
		search string
		want   string
	}{
		{name: "name cyrillic", search: "сбе", want: "sber"},
		{name: "code latin", search: "tbank", want: "tbank"},
		{name: "name latin", search: "ozon", want: "ozon"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchingBankCodes(banks, tc.search)
			if !containsString(got, tc.want) {
				t.Fatalf("matchingBankCodes(%q) = %v, want %q", tc.search, got, tc.want)
			}
		})
	}
}

func TestFromListDetailsRowEnrichesBankName(t *testing.T) {
	row := db.ListRequisiteDetailsByTeamRow{
		ID:         10,
		TeamID:     20,
		Phone:      "+79991112233",
		MethodType: "card",
		BankCode:   "sber",
		Status:     StatusActive,
	}

	item := fromListDetailsRow(row, map[string]string{"sber": "Сбер"})
	if item.BankName != "Сбер" {
		t.Fatalf("BankName = %q, want Сбер", item.BankName)
	}

	unknown := fromListDetailsRow(row, map[string]string{})
	if unknown.BankName != "" {
		t.Fatalf("unknown BankName = %q, want empty", unknown.BankName)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
