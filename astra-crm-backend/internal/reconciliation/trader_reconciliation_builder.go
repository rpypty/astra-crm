package reconciliation

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	traderInboundRequisiteMismatchIssue = "requisite_invoice_amount_mismatch"

	traderOutboundSourceRequisiteMismatchIssue = "source_requisite_outbound_mismatch"
	traderOutboundMissingManualPayoutIssue     = "missing_manual_payout_order"
	traderOutboundExtraManualPayoutIssue       = "extra_manual_payout_order"
	traderOutboundManualPayoutNotPaidIssue     = "manual_payout_not_fully_paid"

	traderInboundRequisiteMismatchMessage = "Closed requisite turnover differs from invoice CSV amount"
	traderOutboundSourceMismatchMessage   = "Closed requisite outbound turnover differs from payout transfers from this source requisite"
)

type traderReconciliationItemRecord struct {
	issueType         string
	externalOrderID   *int64
	externalInnerID   *string
	teamleadValueJSON json.RawMessage
	traderValueJSON   json.RawMessage
	message           string
}

type traderInboundReconciliationPlan struct {
	items    []traderReconciliationItemRecord
	statuses []traderInboundReviewStatusRecord
}

type traderInboundRequisiteRow struct {
	shiftRequisiteID int64
	requisiteID      int64
	phone            string
	bankName         string
	actualAmount     int64
}

type traderInboundReviewRequisiteRow struct {
	shiftRequisiteID int64
	requisiteID      int64
	phone            string
	actualAmount     int64
}

type traderInboundScopeItem struct {
	csvRequisite     string
	amountMinor      int64
	normalizedStatus string
}

type traderInboundReviewStatusRecord struct {
	shiftRequisiteID int64
	status           string
}

type traderInboundCSVAggregate struct {
	rawRequisite        *string
	expectedAmountMinor int64
	successCount        int64
	exists              bool
}

type traderInboundTeamleadPayload struct {
	Requisite           *string `json:"requisite"`
	ExpectedAmountMinor int64   `json:"expectedAmountMinor"`
	SuccessCount        int64   `json:"successCount"`
}

type traderInboundTraderPayload struct {
	ShiftRequisiteID  int64  `json:"shiftRequisiteId"`
	RequisiteID       int64  `json:"requisiteId"`
	Phone             string `json:"phone"`
	BankName          string `json:"bankName"`
	ActualAmountMinor int64  `json:"actualAmountMinor"`
}

func buildTraderInboundReconciliationPlan(requisites []traderInboundRequisiteRow, reviewRequisites []traderInboundReviewRequisiteRow, scopeItems []traderInboundScopeItem) (traderInboundReconciliationPlan, error) {
	csvByKey := buildTraderInboundCSVAggregates(scopeItems)

	crmByKey := make(map[string][]traderInboundRequisiteRow, len(requisites))
	keys := make(map[string]bool, len(requisites)+len(csvByKey))
	for _, requisite := range requisites {
		key := traderCRMRequisiteMatchKey(requisite.phone, requisite.requisiteID)
		crmByKey[key] = append(crmByKey[key], requisite)
		keys[key] = true
	}
	for key := range csvByKey {
		keys[key] = true
	}

	orderedKeys := orderedStringKeys(keys)
	items := make([]traderReconciliationItemRecord, 0)
	for _, key := range orderedKeys {
		csv := csvByKey[key]
		crmRows := crmByKey[key]
		if len(crmRows) == 0 {
			if csv.expectedAmountMinor == 0 {
				continue
			}
			item, err := buildTraderInboundMismatchItem(nil, csv)
			if err != nil {
				return traderInboundReconciliationPlan{}, err
			}
			items = append(items, item)
			continue
		}

		for _, crm := range crmRows {
			if crm.actualAmount == csv.expectedAmountMinor {
				continue
			}
			crmCopy := crm
			item, err := buildTraderInboundMismatchItem(&crmCopy, csv)
			if err != nil {
				return traderInboundReconciliationPlan{}, err
			}
			items = append(items, item)
		}
	}

	statuses := make([]traderInboundReviewStatusRecord, 0, len(reviewRequisites))
	for _, requisite := range reviewRequisites {
		key := traderCRMRequisiteMatchKey(requisite.phone, requisite.requisiteID)
		expectedAmount := csvByKey[key].expectedAmountMinor
		status := "worked_discrepancy"
		if requisite.actualAmount == expectedAmount {
			status = "worked_verified"
		}
		statuses = append(statuses, traderInboundReviewStatusRecord{
			shiftRequisiteID: requisite.shiftRequisiteID,
			status:           status,
		})
	}

	return traderInboundReconciliationPlan{
		items:    items,
		statuses: statuses,
	}, nil
}

func traderReconciliationStatus(diffAmountMinor int64, itemCount int) string {
	if diffAmountMinor == 0 && itemCount == 0 {
		return StatusMatched
	}
	return StatusMismatch
}

func buildTraderInboundCSVAggregates(scopeItems []traderInboundScopeItem) map[string]traderInboundCSVAggregate {
	csvByKey := make(map[string]traderInboundCSVAggregate)
	for _, item := range scopeItems {
		key := traderCSVRequisiteMatchKey(item.csvRequisite)
		aggregate := csvByKey[key]
		aggregate.exists = true
		if item.csvRequisite != "" && (aggregate.rawRequisite == nil || item.csvRequisite > *aggregate.rawRequisite) {
			raw := item.csvRequisite
			aggregate.rawRequisite = &raw
		}
		if isSuccessfulReconciliationStatus(item.normalizedStatus) {
			aggregate.expectedAmountMinor += item.amountMinor
			aggregate.successCount++
		}
		csvByKey[key] = aggregate
	}
	return csvByKey
}

func buildTraderInboundMismatchItem(crm *traderInboundRequisiteRow, csv traderInboundCSVAggregate) (traderReconciliationItemRecord, error) {
	var teamleadValue json.RawMessage
	if csv.exists {
		value, err := marshalReconciliationItemValue(traderInboundTeamleadPayload{
			Requisite:           csv.rawRequisite,
			ExpectedAmountMinor: csv.expectedAmountMinor,
			SuccessCount:        csv.successCount,
		})
		if err != nil {
			return traderReconciliationItemRecord{}, err
		}
		teamleadValue = value
	}

	var traderValue json.RawMessage
	if crm != nil {
		value, err := marshalReconciliationItemValue(traderInboundTraderPayload{
			ShiftRequisiteID:  crm.shiftRequisiteID,
			RequisiteID:       crm.requisiteID,
			Phone:             crm.phone,
			BankName:          crm.bankName,
			ActualAmountMinor: crm.actualAmount,
		})
		if err != nil {
			return traderReconciliationItemRecord{}, err
		}
		traderValue = value
	}

	return traderReconciliationItemRecord{
		issueType:         traderInboundRequisiteMismatchIssue,
		teamleadValueJSON: teamleadValue,
		traderValueJSON:   traderValue,
		message:           traderInboundRequisiteMismatchMessage,
	}, nil
}

type traderOutboundSourceRequisiteRow struct {
	shiftRequisiteID            int64
	requisiteID                 int64
	phone                       string
	bankName                    string
	closedOutboundTurnoverMinor int64
}

type traderOutboundScopeItem struct {
	id               int64
	externalOrderID  int64
	externalInnerID  string
	amountMinor      int64
	normalizedStatus string
	createdAt        time.Time
}

type traderOutboundPayoutOrderRow struct {
	id                   int64
	destinationBank      string
	destinationRequisite string
	amountMinor          int64
	createdAt            time.Time
}

type traderOutboundTransferRow struct {
	manualPayoutOrderID    int64
	sourceShiftRequisiteID int64
	amountMinor            int64
}

type traderOutboundSourcePayload struct {
	ShiftRequisiteID            int64  `json:"shiftRequisiteId"`
	RequisiteID                 int64  `json:"requisiteId"`
	RequisitePhone              string `json:"requisitePhone"`
	BankName                    string `json:"bankName"`
	ClosedOutboundTurnoverMinor int64  `json:"closedOutboundTurnoverMinor"`
	TransferAmountMinor         int64  `json:"transferAmountMinor"`
	DiffAmountMinor             int64  `json:"diffAmountMinor"`
}

type traderOutboundCSVPayload struct {
	AmountMinor      int64  `json:"amountMinor"`
	NormalizedStatus string `json:"normalizedStatus"`
}

type traderOutboundPayoutPayload struct {
	ManualPayoutOrderID  int64  `json:"manualPayoutOrderId"`
	DestinationBank      string `json:"destinationBank"`
	DestinationRequisite string `json:"destinationRequisite"`
	AmountMinor          int64  `json:"amountMinor"`
	PaidAmountMinor      int64  `json:"paidAmountMinor"`
	RemainingAmountMinor int64  `json:"remainingAmountMinor"`
}

func buildTraderOutboundReconciliationItems(sourceRequisites []traderOutboundSourceRequisiteRow, scopeItems []traderOutboundScopeItem, payoutOrders []traderOutboundPayoutOrderRow, transfers []traderOutboundTransferRow) ([]traderReconciliationItemRecord, error) {
	items := make([]traderReconciliationItemRecord, 0)

	transferBySourceRequisiteID := make(map[int64]int64)
	paidByPayoutOrderID := make(map[int64]int64)
	for _, transfer := range transfers {
		transferBySourceRequisiteID[transfer.sourceShiftRequisiteID] += transfer.amountMinor
		paidByPayoutOrderID[transfer.manualPayoutOrderID] += transfer.amountMinor
	}

	sort.SliceStable(sourceRequisites, func(i, j int) bool {
		return sourceRequisites[i].shiftRequisiteID < sourceRequisites[j].shiftRequisiteID
	})
	for _, requisite := range sourceRequisites {
		transferAmount := transferBySourceRequisiteID[requisite.shiftRequisiteID]
		if requisite.closedOutboundTurnoverMinor == transferAmount {
			continue
		}
		value, err := marshalReconciliationItemValue(traderOutboundSourcePayload{
			ShiftRequisiteID:            requisite.shiftRequisiteID,
			RequisiteID:                 requisite.requisiteID,
			RequisitePhone:              requisite.phone,
			BankName:                    requisite.bankName,
			ClosedOutboundTurnoverMinor: requisite.closedOutboundTurnoverMinor,
			TransferAmountMinor:         transferAmount,
			DiffAmountMinor:             requisite.closedOutboundTurnoverMinor - transferAmount,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, traderReconciliationItemRecord{
			issueType:       traderOutboundSourceRequisiteMismatchIssue,
			traderValueJSON: value,
			message:         traderOutboundSourceMismatchMessage,
		})
	}

	csvByAmount := groupSuccessfulTraderOutboundCSVItems(scopeItems)
	payoutByAmount := groupTraderOutboundPayoutOrders(payoutOrders)
	amounts := orderedInt64UnionKeys(csvByAmount, payoutByAmount)
	for _, amount := range amounts {
		csvRows := csvByAmount[amount]
		payoutRows := payoutByAmount[amount]
		maxCount := maxInt(len(csvRows), len(payoutRows))
		for index := 0; index < maxCount; index++ {
			var csv *traderOutboundScopeItem
			if index < len(csvRows) {
				csv = &csvRows[index]
			}
			var payout *traderOutboundPayoutOrderRow
			if index < len(payoutRows) {
				payout = &payoutRows[index]
			}

			item, ok, err := buildTraderOutboundPayoutItem(csv, payout, paidByPayoutOrderID)
			if err != nil {
				return nil, err
			}
			if ok {
				items = append(items, item)
			}
		}
	}

	return items, nil
}

func groupSuccessfulTraderOutboundCSVItems(scopeItems []traderOutboundScopeItem) map[int64][]traderOutboundScopeItem {
	deduped := latestTraderOutboundScopeItems(scopeItems)
	result := make(map[int64][]traderOutboundScopeItem)
	for _, item := range deduped {
		if !isSuccessfulReconciliationStatus(item.normalizedStatus) {
			continue
		}
		result[item.amountMinor] = append(result[item.amountMinor], item)
	}
	for amount := range result {
		rows := result[amount]
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].externalInnerID != rows[j].externalInnerID {
				return rows[i].externalInnerID < rows[j].externalInnerID
			}
			return rows[i].id < rows[j].id
		})
		result[amount] = rows
	}
	return result
}

func latestTraderOutboundScopeItems(scopeItems []traderOutboundScopeItem) []traderOutboundScopeItem {
	rows := append([]traderOutboundScopeItem(nil), scopeItems...)
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.externalInnerID != right.externalInnerID {
			return left.externalInnerID < right.externalInnerID
		}
		if !left.createdAt.Equal(right.createdAt) {
			return left.createdAt.After(right.createdAt)
		}
		return left.id > right.id
	})

	seen := make(map[string]bool, len(rows))
	result := make([]traderOutboundScopeItem, 0, len(rows))
	for _, row := range rows {
		if seen[row.externalInnerID] {
			continue
		}
		seen[row.externalInnerID] = true
		result = append(result, row)
	}
	return result
}

func groupTraderOutboundPayoutOrders(payoutOrders []traderOutboundPayoutOrderRow) map[int64][]traderOutboundPayoutOrderRow {
	rows := append([]traderOutboundPayoutOrderRow(nil), payoutOrders...)
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.amountMinor != right.amountMinor {
			return left.amountMinor < right.amountMinor
		}
		if !left.createdAt.Equal(right.createdAt) {
			return left.createdAt.Before(right.createdAt)
		}
		return left.id < right.id
	})

	result := make(map[int64][]traderOutboundPayoutOrderRow)
	for _, row := range rows {
		result[row.amountMinor] = append(result[row.amountMinor], row)
	}
	return result
}

func buildTraderOutboundPayoutItem(csv *traderOutboundScopeItem, payout *traderOutboundPayoutOrderRow, paidByPayoutOrderID map[int64]int64) (traderReconciliationItemRecord, bool, error) {
	switch {
	case payout == nil && csv != nil:
		teamleadValue, err := marshalReconciliationItemValue(traderOutboundCSVPayload{
			AmountMinor:      csv.amountMinor,
			NormalizedStatus: csv.normalizedStatus,
		})
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		return traderReconciliationItemRecord{
			issueType:         traderOutboundMissingManualPayoutIssue,
			externalOrderID:   traderInt64Ptr(csv.externalOrderID),
			externalInnerID:   traderStringPtr(csv.externalInnerID),
			teamleadValueJSON: teamleadValue,
			message:           "Payout is present in trader CSV but no manual payout order with the same amount was created",
		}, true, nil
	case csv == nil && payout != nil:
		traderValue, err := marshalTraderOutboundPayoutValue(*payout, paidByPayoutOrderID[payout.id])
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		return traderReconciliationItemRecord{
			issueType:       traderOutboundExtraManualPayoutIssue,
			traderValueJSON: traderValue,
			message:         "Manual payout order has no matching successful payout amount in trader CSV",
		}, true, nil
	case csv != nil && payout != nil:
		paidAmount := paidByPayoutOrderID[payout.id]
		if paidAmount == payout.amountMinor {
			return traderReconciliationItemRecord{}, false, nil
		}
		teamleadValue, err := marshalReconciliationItemValue(traderOutboundCSVPayload{
			AmountMinor:      csv.amountMinor,
			NormalizedStatus: csv.normalizedStatus,
		})
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		traderValue, err := marshalTraderOutboundPayoutValue(*payout, paidAmount)
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		return traderReconciliationItemRecord{
			issueType:         traderOutboundManualPayoutNotPaidIssue,
			externalOrderID:   traderInt64Ptr(csv.externalOrderID),
			externalInnerID:   traderStringPtr(csv.externalInnerID),
			teamleadValueJSON: teamleadValue,
			traderValueJSON:   traderValue,
			message:           "Manual payout order is not fully paid by transfers",
		}, true, nil
	default:
		return traderReconciliationItemRecord{}, false, nil
	}
}

func marshalTraderOutboundPayoutValue(payout traderOutboundPayoutOrderRow, paidAmount int64) (json.RawMessage, error) {
	return marshalReconciliationItemValue(traderOutboundPayoutPayload{
		ManualPayoutOrderID:  payout.id,
		DestinationBank:      payout.destinationBank,
		DestinationRequisite: payout.destinationRequisite,
		AmountMinor:          payout.amountMinor,
		PaidAmountMinor:      paidAmount,
		RemainingAmountMinor: payout.amountMinor - paidAmount,
	})
}

func traderCRMRequisiteMatchKey(phone string, requisiteID int64) string {
	if digits := rightTraderReconciliationDigits(phone, 10); digits != "" {
		return digits
	}
	return "requisite:" + strconv.FormatInt(requisiteID, 10)
}

func traderCSVRequisiteMatchKey(value string) string {
	if digits := rightTraderReconciliationDigits(value, 10); digits != "" {
		return digits
	}
	fallback := "unknown"
	if value != "" {
		fallback = value
	}
	return "csv:" + strings.ToLower(strings.TrimSpace(fallback))
}

func rightTraderReconciliationDigits(value string, count int) string {
	digits := digitsOnly(value)
	if digits == "" {
		return ""
	}
	if len(digits) <= count {
		return digits
	}
	return digits[len(digits)-count:]
}

func isSuccessfulReconciliationStatus(status string) bool {
	return status == "success" || status == "corrected"
}

func marshalReconciliationItemValue(value any) (json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func orderedStringKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func orderedInt64UnionKeys(left map[int64][]traderOutboundScopeItem, right map[int64][]traderOutboundPayoutOrderRow) []int64 {
	seen := make(map[int64]bool, len(left)+len(right))
	for key := range left {
		seen[key] = true
	}
	for key := range right {
		seen[key] = true
	}
	keys := make([]int64, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func traderInt64Ptr(value int64) *int64 {
	return &value
}

func traderStringPtr(value string) *string {
	return &value
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
