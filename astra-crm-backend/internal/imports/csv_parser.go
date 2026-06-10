package imports

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	ColumnSetAuto     = "auto"
	ColumnSetTeamlead = "teamlead"
	ColumnSetTrader   = "trader"

	NormalizedStatusSuccess   = "success"
	NormalizedStatusCorrected = "corrected"
	NormalizedStatusFailed    = "failed"
	NormalizedStatusCancelled = "cancelled"
	NormalizedStatusUnknown   = "unknown"

	ParseCodeMissingRequiredColumn = "CSV_MISSING_REQUIRED_COLUMN"
	ParseCodeInvalidColumnSet      = "CSV_INVALID_COLUMN_SET"
	ParseCodeInvalidRowWidth       = "CSV_INVALID_ROW_WIDTH"
	ParseCodeMissingRequiredValue  = "CSV_MISSING_REQUIRED_VALUE"
	ParseCodeInvalidMoney          = "CSV_INVALID_MONEY"
	ParseCodeInvalidDate           = "CSV_INVALID_DATE"
	ParseCodeInvalidBool           = "CSV_INVALID_BOOL"
	ParseCodeDuplicateInnerID      = "CSV_DUPLICATE_INNER_ID"
)

var (
	ErrValidation = errors.New("csv validation failed")

	requiredColumns = []string{
		"id",
		"innerId",
		"amount",
		"currency",
		"status",
		"createdAt",
		"workerName",
	}
)

type ParseOptions struct {
	ColumnSet string
	Location  *time.Location
}

type ParseResult struct {
	ColumnSet              string
	Rows                   []ParsedOrderRow
	Errors                 []ParseError
	RawStatusCounts        map[string]int
	NormalizedStatusCounts map[string]int
}

func (r ParseResult) Valid() bool {
	return len(r.Errors) == 0
}

type ParseError struct {
	Code    string `json:"code"`
	Row     int    `json:"row,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	InnerID string `json:"innerId,omitempty"`
	Rows    []int  `json:"rows,omitempty"`
}

type ParsedOrderRow struct {
	RowNumber           int
	RawPayload          map[string]string
	ExternalID          string
	ExternalInnerID     string
	ExternalForeignID   *string
	RequisiteRaw        *string
	RequisitePhone      *string
	RequisiteExternalID *string
	DeviceName          *string
	MethodType          *string
	MethodName          *string
	AmountMinor         int64
	Course              *string
	CourseWorker        *string
	Currency            string
	RawStatus           string
	NormalizedStatus    string
	CreatedAtExternal   time.Time
	ClosedAtExternal    *time.Time
	UpdatedAtExternal   *time.Time
	OldAmountMinor      *int64
	HadDispute          *bool
	Receipt             *string
	OrderComment        *string
	WorkerName          string
	WorkerAmount        *string
	WorkerProfit        *string
	Ordered             *bool
	Counted             *bool
	Initials            *string
}

func ParseCSV(reader io.Reader, options ParseOptions) (ParseResult, error) {
	location := options.Location
	if location == nil {
		location = defaultLocation()
	}

	csvReader := csv.NewReader(reader)
	csvReader.Comma = '|'
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = false

	records, err := csvReader.ReadAll()
	if err != nil {
		return ParseResult{}, err
	}

	result := ParseResult{
		RawStatusCounts:        map[string]int{},
		NormalizedStatusCounts: map[string]int{},
	}
	if len(records) == 0 {
		result.Errors = append(result.Errors, ParseError{
			Code:    ParseCodeMissingRequiredColumn,
			Message: "CSV header is required",
		})
		return result, ErrValidation
	}

	header := normalizeHeader(records[0])
	columnIndex := make(map[string]int, len(header))
	for index, column := range header {
		columnIndex[column] = index
	}

	result.ColumnSet = resolveColumnSet(options.ColumnSet, columnIndex)
	if result.ColumnSet == "" {
		result.Errors = append(result.Errors, ParseError{
			Code:    ParseCodeInvalidColumnSet,
			Message: "CSV column set must be teamlead, trader or auto",
		})
		return result, ErrValidation
	}

	for _, column := range requiredColumns {
		if _, ok := columnIndex[column]; !ok {
			result.Errors = append(result.Errors, ParseError{
				Code:    ParseCodeMissingRequiredColumn,
				Field:   column,
				Message: fmt.Sprintf("required column %q is missing", column),
			})
		}
	}
	if len(result.Errors) > 0 {
		return result, ErrValidation
	}

	innerIDRows := map[string][]int{}
	for recordIndex, record := range records[1:] {
		rowNumber := recordIndex + 2
		if len(record) != len(header) {
			result.Errors = append(result.Errors, ParseError{
				Code:    ParseCodeInvalidRowWidth,
				Row:     rowNumber,
				Message: "row has different number of fields than header",
			})
			continue
		}

		rawPayload := map[string]string{}
		for index, column := range header {
			rawPayload[column] = record[index]
		}

		row, rowErrors := parseOrderRow(rowNumber, rawPayload, location)
		if len(rowErrors) > 0 {
			result.Errors = append(result.Errors, rowErrors...)
			continue
		}

		innerIDRows[row.ExternalInnerID] = append(innerIDRows[row.ExternalInnerID], rowNumber)
		result.Rows = append(result.Rows, row)
		result.RawStatusCounts[row.RawStatus]++
		result.NormalizedStatusCounts[row.NormalizedStatus]++
	}

	for innerID, rows := range innerIDRows {
		if len(rows) <= 1 {
			continue
		}
		result.Errors = append(result.Errors, ParseError{
			Code:    ParseCodeDuplicateInnerID,
			Field:   "innerId",
			Message: "duplicated innerId inside CSV",
			InnerID: innerID,
			Rows:    rows,
		})
	}

	if len(result.Errors) > 0 {
		return result, ErrValidation
	}

	return result, nil
}

func parseOrderRow(rowNumber int, raw map[string]string, location *time.Location) (ParsedOrderRow, []ParseError) {
	var errorsList []ParseError

	externalID, ok := requiredValue(raw, "id")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "id"))
	}
	innerID, ok := requiredValue(raw, "innerId")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "innerId"))
	}
	amountRaw, ok := requiredValue(raw, "amount")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "amount"))
	}
	currency, ok := requiredValue(raw, "currency")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "currency"))
	}
	rawStatus, ok := requiredValue(raw, "status")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "status"))
	}
	createdAtRaw, ok := requiredValue(raw, "createdAt")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "createdAt"))
	}
	workerName, ok := requiredValue(raw, "workerName")
	if !ok {
		errorsList = append(errorsList, missingValueError(rowNumber, "workerName"))
	}
	if len(errorsList) > 0 {
		return ParsedOrderRow{}, errorsList
	}

	amountMinor, err := ParseMoneyMinor(amountRaw)
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidMoney,
			Row:     rowNumber,
			Field:   "amount",
			Message: err.Error(),
		})
	}

	createdAt, err := parseRequiredTime(createdAtRaw, location)
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidDate,
			Row:     rowNumber,
			Field:   "createdAt",
			Message: err.Error(),
		})
	}

	closedAt, err := parseOptionalTime(raw["closedAt"], location)
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidDate,
			Row:     rowNumber,
			Field:   "closedAt",
			Message: err.Error(),
		})
	}

	updatedAt, err := parseOptionalTime(raw["updatedAt"], location)
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidDate,
			Row:     rowNumber,
			Field:   "updatedAt",
			Message: err.Error(),
		})
	}

	oldAmountMinor, err := parseOptionalMoney(raw["oldAmount"])
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidMoney,
			Row:     rowNumber,
			Field:   "oldAmount",
			Message: err.Error(),
		})
	}

	hadDispute, err := parseOptionalBool(raw["hadDispute"])
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidBool,
			Row:     rowNumber,
			Field:   "hadDispute",
			Message: err.Error(),
		})
	}

	ordered, err := parseOptionalBool(raw["ordered"])
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidBool,
			Row:     rowNumber,
			Field:   "ordered",
			Message: err.Error(),
		})
	}

	counted, err := parseOptionalBool(raw["counted"])
	if err != nil {
		errorsList = append(errorsList, ParseError{
			Code:    ParseCodeInvalidBool,
			Row:     rowNumber,
			Field:   "counted",
			Message: err.Error(),
		})
	}

	if len(errorsList) > 0 {
		return ParsedOrderRow{}, errorsList
	}

	return ParsedOrderRow{
		RowNumber:           rowNumber,
		RawPayload:          raw,
		ExternalID:          externalID,
		ExternalInnerID:     innerID,
		ExternalForeignID:   optionalString(raw["foreignId"]),
		RequisiteRaw:        optionalString(raw["requisite"]),
		RequisitePhone:      optionalString(raw["requisitePhone"]),
		RequisiteExternalID: optionalString(raw["requisiteId"]),
		DeviceName:          optionalString(raw["deviceName"]),
		MethodType:          optionalString(raw["methodType"]),
		MethodName:          optionalString(raw["methodName"]),
		AmountMinor:         amountMinor,
		Course:              optionalString(raw["course"]),
		CourseWorker:        optionalString(raw["courseWorker"]),
		Currency:            currency,
		RawStatus:           rawStatus,
		NormalizedStatus:    NormalizeStatus(rawStatus),
		CreatedAtExternal:   createdAt,
		ClosedAtExternal:    closedAt,
		UpdatedAtExternal:   updatedAt,
		OldAmountMinor:      oldAmountMinor,
		HadDispute:          hadDispute,
		Receipt:             optionalString(raw["receipt"]),
		OrderComment:        optionalString(raw["orderComment"]),
		WorkerName:          workerName,
		WorkerAmount:        optionalString(raw["workerAmount"]),
		WorkerProfit:        optionalString(raw["workerProfit"]),
		Ordered:             ordered,
		Counted:             counted,
		Initials:            optionalString(raw["initials"]),
	}, nil
}

func ParseMoneyMinor(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if isNone(trimmed) {
		return 0, fmt.Errorf("money value is required")
	}
	if strings.HasPrefix(trimmed, "-") {
		return 0, fmt.Errorf("money value must not be negative")
	}
	trimmed = strings.TrimPrefix(trimmed, "+")
	trimmed = strings.ReplaceAll(trimmed, " ", "")

	parts := strings.Split(trimmed, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("money value has invalid decimal format")
	}
	majorPart := parts[0]
	if majorPart == "" {
		return 0, fmt.Errorf("money value has empty major part")
	}
	if !allDigits(majorPart) {
		return 0, fmt.Errorf("money value contains non-digit major part")
	}

	fractionPart := ""
	if len(parts) == 2 {
		fractionPart = parts[1]
		if len(fractionPart) > 2 {
			return 0, fmt.Errorf("money value has more than 2 fractional digits")
		}
		if fractionPart != "" && !allDigits(fractionPart) {
			return 0, fmt.Errorf("money value contains non-digit fractional part")
		}
	}
	for len(fractionPart) < 2 {
		fractionPart += "0"
	}

	major, err := strconv.ParseInt(majorPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money value is too large")
	}
	if major > math.MaxInt64/100 {
		return 0, fmt.Errorf("money value is too large")
	}

	minor := int64(0)
	if fractionPart != "" {
		minor, err = strconv.ParseInt(fractionPart, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("money fractional value is invalid")
		}
	}
	if major*100 > math.MaxInt64-minor {
		return 0, fmt.Errorf("money value is too large")
	}

	return major*100 + minor, nil
}

func NormalizeStatus(rawStatus string) string {
	switch strings.TrimSpace(rawStatus) {
	case "hand_success":
		return NormalizedStatusSuccess
	case "corrected":
		return NormalizedStatusCorrected
	case "auto_decline":
		return NormalizedStatusFailed
	case "cancelled":
		return NormalizedStatusCancelled
	default:
		return NormalizedStatusUnknown
	}
}

func resolveColumnSet(columnSet string, columns map[string]int) string {
	switch columnSet {
	case "", ColumnSetAuto:
		return detectColumnSet(columns)
	case ColumnSetTeamlead, ColumnSetTrader:
		return columnSet
	default:
		return ""
	}
}

func detectColumnSet(columns map[string]int) string {
	if _, ok := columns["foreignId"]; ok {
		return ColumnSetTeamlead
	}
	if _, ok := columns["hadDispute"]; ok {
		return ColumnSetTeamlead
	}
	if _, ok := columns["requisiteId"]; ok {
		return ColumnSetTrader
	}
	if _, ok := columns["workerProfit"]; ok {
		return ColumnSetTrader
	}

	return ColumnSetTrader
}

func normalizeHeader(header []string) []string {
	normalized := make([]string, 0, len(header))
	for index, column := range header {
		if index == 0 {
			column = strings.TrimPrefix(column, "\ufeff")
		}
		normalized = append(normalized, strings.TrimSpace(column))
	}

	return normalized
}

func parseRequiredTime(value string, location *time.Location) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if isNone(trimmed) {
		return time.Time{}, fmt.Errorf("date value is required")
	}

	return time.ParseInLocation("02.01.2006 15:04:05", trimmed, location)
}

func parseOptionalTime(value string, location *time.Location) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if isNone(trimmed) {
		return nil, nil
	}

	parsed, err := time.ParseInLocation("02.01.2006 15:04:05", trimmed, location)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseOptionalMoney(value string) (*int64, error) {
	trimmed := strings.TrimSpace(value)
	if isNone(trimmed) {
		return nil, nil
	}

	parsed, err := ParseMoneyMinor(trimmed)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseOptionalBool(value string) (*bool, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if isNone(trimmed) {
		return nil, nil
	}

	switch trimmed {
	case "true", "1", "yes":
		parsed := true
		return &parsed, nil
	case "false", "0", "no":
		parsed := false
		return &parsed, nil
	default:
		return nil, fmt.Errorf("bool value is invalid")
	}
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if isNone(trimmed) {
		return nil
	}

	return &trimmed
}

func requiredValue(raw map[string]string, field string) (string, bool) {
	value := optionalString(raw[field])
	if value == nil {
		return "", false
	}

	return *value, true
}

func missingValueError(rowNumber int, field string) ParseError {
	return ParseError{
		Code:    ParseCodeMissingRequiredValue,
		Row:     rowNumber,
		Field:   field,
		Message: fmt.Sprintf("required value %q is missing", field),
	}
}

func allDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func isNone(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed == "" || strings.EqualFold(trimmed, "None")
}

func defaultLocation() *time.Location {
	location, err := time.LoadLocation("Europe/Moscow")
	if err == nil {
		return location
	}

	return time.FixedZone("Europe/Moscow", 3*60*60)
}
