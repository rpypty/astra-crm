package shifts

import "testing"

func TestBuildShiftReportRowsMatchesCsvCardToShiftRequisite(t *testing.T) {
	card := "1234567890123456"

	rows := buildShiftReportRows(
		[]shiftReportRequisiteRow{
			{
				ShiftRequisiteID:       10,
				RequisiteID:            20,
				Phone:                  "+7 999 111-22-33",
				MethodType:             "card",
				BankCode:               "sber",
				CardNumber:             card,
				HolderName:             "IVAN IVANOV",
				Status:                 RequisiteStatusVerified,
				TLReconciliationStatus: "not_checked",
				InboundTurnoverMinor:   10000,
				OutboundTurnoverMinor:  4000,
				ClosingBalanceMinor:    6000,
				TargetTurnoverMinor:    15000,
			},
		},
		[]shiftReportInboundItem{
			{CSVRequisite: "1234 5678 9012 3456", NormalizedStatus: "success", AmountMinor: 10000},
		},
		[]shiftReportOutboundTransfer{
			{SourceShiftRequisiteID: 10, AmountMinor: 4000},
		},
		map[string]string{"sber": "Сбер"},
	)

	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1: %+v", len(rows), rows)
	}
	row := rows[0]
	if row.RowKey != "crm:10" || row.CSVOnly || row.HasMismatch {
		t.Fatalf("row identity/mismatch = %+v, want matched crm row", row)
	}
	if row.BankName != "Сбер" || row.CSVInboundMinor != 10000 || row.CSVOutboundMinor != 4000 {
		t.Fatalf("row values = %+v, want bank and CSV amounts", row)
	}
}

func TestBuildShiftReportRowsCreatesCsvOnlyRows(t *testing.T) {
	rows := buildShiftReportRows(
		nil,
		[]shiftReportInboundItem{
			{CSVRequisite: "+7 (999) 111-22-33", NormalizedStatus: "corrected", AmountMinor: 7000},
		},
		nil,
		nil,
	)

	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1: %+v", len(rows), rows)
	}
	row := rows[0]
	if !row.CSVOnly || !row.HasMismatch || row.RowKey != "csv:csv_phone:9991112233" {
		t.Fatalf("row = %+v, want csv-only phone mismatch", row)
	}
	if row.CSVInboundMinor != 7000 || row.InboundDiffMinor != -7000 || row.Phone != "+7 (999) 111-22-33" {
		t.Fatalf("row amounts/phone = %+v, want CSV inbound diff", row)
	}
}

func TestBuildShiftReportRowsDoesNotAutoMatchAmbiguousPhone(t *testing.T) {
	requisites := []shiftReportRequisiteRow{
		{
			ShiftRequisiteID:      10,
			RequisiteID:           20,
			Phone:                 "+7 999 111-22-33",
			Status:                RequisiteStatusVerified,
			InboundTurnoverMinor:  1000,
			OutboundTurnoverMinor: 0,
		},
		{
			ShiftRequisiteID:      11,
			RequisiteID:           21,
			Phone:                 "8 (999) 111-22-33",
			Status:                RequisiteStatusVerified,
			InboundTurnoverMinor:  2000,
			OutboundTurnoverMinor: 0,
		},
	}

	rows := buildShiftReportRows(
		requisites,
		[]shiftReportInboundItem{
			{CSVRequisite: "+7 999 111-22-33", NormalizedStatus: "success", AmountMinor: 1000},
		},
		nil,
		nil,
	)

	if len(rows) != 3 {
		t.Fatalf("rows count = %d, want two CRM rows plus one csv-only row: %+v", len(rows), rows)
	}
	for _, row := range rows {
		if row.CSVOnly && row.RowKey != "csv:csv_phone:9991112233" {
			t.Fatalf("csv row = %+v, want ambiguous phone to stay csv-only", row)
		}
		if !row.CSVOnly && row.CSVInboundMinor != 0 {
			t.Fatalf("crm row = %+v, want no ambiguous auto-match", row)
		}
	}
}

func TestBuildShiftReportRowsSortsMismatchesBeforeMatchedRows(t *testing.T) {
	rows := buildShiftReportRows(
		[]shiftReportRequisiteRow{
			{
				ShiftRequisiteID:      10,
				RequisiteID:           20,
				Phone:                 "+7 900 000-00-01",
				Status:                RequisiteStatusVerified,
				InboundTurnoverMinor:  1000,
				OutboundTurnoverMinor: 0,
			},
			{
				ShiftRequisiteID:      11,
				RequisiteID:           21,
				Phone:                 "+7 900 000-00-02",
				Status:                RequisiteStatusVerified,
				InboundTurnoverMinor:  5000,
				OutboundTurnoverMinor: 0,
			},
		},
		[]shiftReportInboundItem{
			{CSVRequisite: "+7 900 000-00-01", NormalizedStatus: "success", AmountMinor: 1000},
			{CSVRequisite: "+7 900 000-00-02", NormalizedStatus: "success", AmountMinor: 2000},
		},
		nil,
		nil,
	)

	if len(rows) != 2 {
		t.Fatalf("rows count = %d, want 2", len(rows))
	}
	if rows[0].RowKey != "crm:11" || !rows[0].HasMismatch {
		t.Fatalf("first row = %+v, want larger mismatch first", rows[0])
	}
	if rows[1].RowKey != "crm:10" || rows[1].HasMismatch {
		t.Fatalf("second row = %+v, want matched row last", rows[1])
	}
}
