package payouts

import (
	"testing"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

func TestTransferRowEnrichesBankName(t *testing.T) {
	item := fromListTransferRow(db.ListPayoutTransfersRow{SourceBankCode: "SBER"}, map[string]string{
		"SBER": "Сбербанк",
	})
	if item.SourceBankName != "Сбербанк" {
		t.Fatalf("SourceBankName = %q, want %q", item.SourceBankName, "Сбербанк")
	}

	item = fromListTransferRow(db.ListPayoutTransfersRow{SourceBankCode: "ARCHIVED"}, map[string]string{
		"SBER": "Сбербанк",
	})
	if item.SourceBankName != "" {
		t.Fatalf("archived SourceBankName = %q, want empty", item.SourceBankName)
	}
}
