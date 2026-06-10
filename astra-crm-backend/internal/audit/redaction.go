package audit

import (
	"encoding/json"
	"strings"
	"unicode"
)

const redactedValue = "[REDACTED]"

func MarshalRedacted(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	return json.Marshal(Redact(decoded))
}

func Redact(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			switch {
			case isCardNumberKey(key):
				result[key] = maskCardNumber(item)
			case isSensitiveKey(key):
				result[key] = redactedValue
			default:
				result[key] = Redact(item)
			}
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, Redact(item))
		}
		return result
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	normalized := normalizeKey(key)
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}

func isCardNumberKey(key string) bool {
	normalized := normalizeKey(key)
	return normalized == "cardnumber"
}

func normalizeKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func maskCardNumber(value any) string {
	raw, ok := value.(string)
	if !ok {
		return redactedValue
	}

	digits := make([]rune, 0, len(raw))
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}

	if len(digits) < 4 {
		return redactedValue
	}

	return "**** **** **** " + string(digits[len(digits)-4:])
}
