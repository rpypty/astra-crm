package imports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBatchImportRowsJSONIncludesRawPayload(t *testing.T) {
	rows := []ParsedOrderRow{
		{
			RowNumber:       2,
			ExternalID:      "ext-2",
			ExternalInnerID: "inner-2",
			RawPayload: map[string]string{
				"innerId": "inner-2",
				"amount":  "10.00",
			},
		},
	}

	payload, err := batchImportRowsJSON(rows)
	if err != nil {
		t.Fatalf("batchImportRowsJSON() error = %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("decoded rows = %d, want 1", len(decoded))
	}
	if decoded[0]["external_inner_id"] != "inner-2" {
		t.Fatalf("external_inner_id = %v, want inner-2", decoded[0]["external_inner_id"])
	}
	rawPayload, ok := decoded[0]["raw_payload_json"].(map[string]any)
	if !ok {
		t.Fatalf("raw_payload_json = %#v, want object", decoded[0]["raw_payload_json"])
	}
	if rawPayload["amount"] != "10.00" {
		t.Fatalf("raw amount = %v, want 10.00", rawPayload["amount"])
	}
}

func TestBatchExternalOrderRowsJSONPreservesRows(t *testing.T) {
	course := "92.50"
	workerAmount := "925.00"
	closedAt := time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
	rows := make([]ParsedOrderRow, 0, 512)
	for i := 0; i < 512; i++ {
		rows = append(rows, ParsedOrderRow{
			ExternalID:        "ext",
			ExternalInnerID:   "inner",
			WorkerName:        "worker",
			AmountMinor:       int64(i + 1),
			Currency:          "RUB",
			Course:            &course,
			WorkerAmount:      &workerAmount,
			RawStatus:         "hand_success",
			NormalizedStatus:  NormalizedStatusSuccess,
			CreatedAtExternal: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
			ClosedAtExternal:  &closedAt,
		})
	}

	payload, err := batchExternalOrderRowsJSON(rows)
	if err != nil {
		t.Fatalf("batchExternalOrderRowsJSON() error = %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(decoded) != len(rows) {
		t.Fatalf("decoded rows = %d, want %d", len(decoded), len(rows))
	}
	if decoded[0]["course"] != "92.50" {
		t.Fatalf("course = %v, want 92.50", decoded[0]["course"])
	}
	if decoded[511]["amount_minor"] != float64(512) {
		t.Fatalf("last amount_minor = %v, want 512", decoded[511]["amount_minor"])
	}
}

func TestBatchExternalOrderRowsJSONValidatesNumericFields(t *testing.T) {
	workerProfit := "not-a-number"
	_, err := batchExternalOrderRowsJSON([]ParsedOrderRow{
		{
			ExternalID:        "ext",
			ExternalInnerID:   "inner",
			WorkerName:        "worker",
			AmountMinor:       100,
			Currency:          "RUB",
			WorkerProfit:      &workerProfit,
			RawStatus:         "hand_success",
			NormalizedStatus:  NormalizedStatusSuccess,
			CreatedAtExternal: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		},
	})
	if err == nil {
		t.Fatal("batchExternalOrderRowsJSON() error = nil, want numeric error")
	}
	if !strings.Contains(err.Error(), "workerProfit") {
		t.Fatalf("error = %q, want workerProfit context", err.Error())
	}
}
