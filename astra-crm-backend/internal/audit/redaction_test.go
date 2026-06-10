package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalRedactedRemovesSensitiveFields(t *testing.T) {
	payload := map[string]any{
		"login":         "trader_1",
		"password":      "plain",
		"password_hash": "hash",
		"sessionToken":  "raw-token",
		"nested": map[string]any{
			"token_hash": "hashed-token",
			"secret":     "value",
		},
	}

	data, err := MarshalRedacted(payload)
	if err != nil {
		t.Fatalf("MarshalRedacted() error = %v", err)
	}

	raw := string(data)
	for _, forbidden := range []string{"plain", "raw-token", "hashed-token", `"value"`} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("redacted payload contains sensitive value %q: %s", forbidden, raw)
		}
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal redacted payload: %v", err)
	}

	if decoded["password"] != redactedValue {
		t.Fatalf("password = %v, want redacted", decoded["password"])
	}
}

func TestMarshalRedactedMasksCardNumber(t *testing.T) {
	payload := map[string]any{
		"cardNumber": "1234 5678 9012 3456",
	}

	data, err := MarshalRedacted(payload)
	if err != nil {
		t.Fatalf("MarshalRedacted() error = %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, "1234 5678 9012 3456") {
		t.Fatalf("card number leaked: %s", raw)
	}
	if !strings.Contains(raw, "3456") {
		t.Fatalf("last digits are not preserved: %s", raw)
	}
}
