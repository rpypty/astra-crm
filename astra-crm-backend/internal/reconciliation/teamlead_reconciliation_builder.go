package reconciliation

import (
	"encoding/json"
	"sort"
	"time"
)

const (
	teamleadTotalAmountMismatchIssue     = "total_amount_mismatch"
	teamleadRequisiteAmountMismatchIssue = "requisite_amount_mismatch"
	teamleadMissingTraderImportIssue     = "missing_in_trader_import"
	teamleadAmountMismatchIssue          = "amount_mismatch"
	teamleadStatusMismatchIssue          = "status_mismatch"
	teamleadWorkerMismatchIssue          = "worker_mismatch"
)

type teamleadReconciliationBuildResult struct {
	summary teamleadReconciliationSummary
	items   []traderReconciliationItemRecord
}

type teamleadReconciliationSummary struct {
	ExpectedAmountMinor  int64
	ExpectedSuccessCount int64
	FailedAmountMinor    int64
	FailedCount          int64
	TotalAmountMinor     int64
	TotalCount           int64
	ActualAmountMinor    int64
	ActualSuccessCount   int64
}

type teamleadScopeOrderRow struct {
	id                int64
	externalOrderID   int64
	externalInnerID   string
	workerName        string
	traderID          *int64
	traderLogin       *string
	requisiteRaw      *string
	requisitePhone    *string
	amountMinor       int64
	normalizedStatus  string
	rawStatus         string
	createdAtExternal time.Time
	createdAt         time.Time
}

type teamleadPeriodInboundShiftRequisiteRow struct {
	shiftRequisiteID     int64
	requisiteID          int64
	requisitePhone       string
	bankCode             string
	traderID             int64
	traderLogin          string
	amountMinor          int64
	shiftRequisiteStatus string
}

type teamleadPeriodPayoutOrderRow struct {
	id                   int64
	destinationBank      string
	destinationRequisite string
	amountMinor          int64
	traderID             int64
	traderLogin          string
	createdAt            time.Time
}

type teamleadPeriodPayoutTransferRow struct {
	manualPayoutOrderID int64
	amountMinor         int64
}

type teamleadSuccessTotalPayload struct {
	SuccessAmountMinor int64 `json:"successAmountMinor"`
	SuccessCount       int64 `json:"successCount"`
}

type teamleadPeriodInboundValue struct {
	RequisitePhone     *string `json:"requisitePhone"`
	SuccessAmountMinor int64   `json:"successAmountMinor"`
	SuccessCount       int64   `json:"successCount"`
}

type teamleadPeriodInboundTraderValue struct {
	RequisiteID        int64  `json:"requisiteId"`
	RequisitePhone     string `json:"requisitePhone"`
	BankCode           string `json:"bankCode"`
	TraderID           int64  `json:"traderId"`
	TraderLogin        string `json:"traderLogin"`
	SuccessAmountMinor int64  `json:"successAmountMinor"`
	SuccessCount       int64  `json:"successCount"`
}

type teamleadPeriodOutboundValue struct {
	WorkerName       string `json:"workerName"`
	TraderID         *int64 `json:"traderId"`
	Destination      string `json:"destination"`
	AmountMinor      int64  `json:"amountMinor"`
	NormalizedStatus string `json:"normalizedStatus"`
}

type teamleadPeriodOutboundTraderValue struct {
	ManualPayoutOrderID  int64  `json:"manualPayoutOrderId"`
	DestinationBank      string `json:"destinationBank"`
	DestinationRequisite string `json:"destinationRequisite"`
	TraderID             int64  `json:"traderId"`
	TraderLogin          string `json:"traderLogin"`
	AmountMinor          int64  `json:"amountMinor"`
	PaidAmountMinor      int64  `json:"paidAmountMinor"`
}

type teamleadCurrentValue struct {
	WorkerName        string    `json:"workerName"`
	TraderID          *int64    `json:"traderId"`
	RequisitePhone    *string   `json:"requisitePhone"`
	Requisite         *string   `json:"requisite"`
	AmountMinor       int64     `json:"amountMinor"`
	RawStatus         string    `json:"rawStatus"`
	NormalizedStatus  string    `json:"normalizedStatus"`
	CreatedAtExternal time.Time `json:"createdAtExternal"`
}

type teamleadCurrentTraderValue struct {
	WorkerName        string    `json:"workerName"`
	TraderID          *int64    `json:"traderId"`
	TraderLogin       *string   `json:"traderLogin"`
	RequisitePhone    *string   `json:"requisitePhone"`
	Requisite         *string   `json:"requisite"`
	AmountMinor       int64     `json:"amountMinor"`
	RawStatus         string    `json:"rawStatus"`
	NormalizedStatus  string    `json:"normalizedStatus"`
	CreatedAtExternal time.Time `json:"createdAtExternal"`
}

type teamleadRequisiteTotal struct {
	requisitePhone     *string
	requisiteID        int64
	bankCode           string
	traderID           int64
	traderLogin        string
	successAmountMinor int64
	successCount       int64
	exists             bool
}

func buildTeamleadPeriodInboundReconciliation(teamleadRows []teamleadScopeOrderRow, traderRows []teamleadPeriodInboundShiftRequisiteRow) (teamleadReconciliationBuildResult, error) {
	teamleadOrders := latestTeamleadScopeOrders(teamleadRows)
	summary := summarizeTeamleadScopeOrders(teamleadOrders)
	for _, row := range traderRows {
		summary.ActualAmountMinor += row.amountMinor
		summary.ActualSuccessCount++
	}

	items, err := buildTeamleadTotalItems(summary, "Teamlead period invoice success total differs from CRM requisite turnover")
	if err != nil {
		return teamleadReconciliationBuildResult{}, err
	}

	requisiteItems, err := buildTeamleadPeriodInboundRequisiteItems(teamleadOrders, traderRows)
	if err != nil {
		return teamleadReconciliationBuildResult{}, err
	}
	items = append(items, requisiteItems...)

	return teamleadReconciliationBuildResult{summary: summary, items: items}, nil
}

func buildTeamleadPeriodInboundRequisiteItems(teamleadOrders []teamleadScopeOrderRow, traderRows []teamleadPeriodInboundShiftRequisiteRow) ([]traderReconciliationItemRecord, error) {
	teamleadTotals := make(map[string]teamleadRequisiteTotal)
	for _, order := range teamleadOrders {
		if !isSuccessfulReconciliationStatus(order.normalizedStatus) {
			continue
		}
		key := firstPresentOrderString(order.requisitePhone, order.requisiteRaw, "")
		total := teamleadTotals[key]
		total.exists = true
		total.requisitePhone = maxStringPtr(total.requisitePhone, order.requisitePhone)
		total.successAmountMinor += order.amountMinor
		total.successCount++
		teamleadTotals[key] = total
	}

	traderTotals := make(map[string]teamleadRequisiteTotal)
	for _, row := range traderRows {
		key := row.requisitePhone
		total := traderTotals[key]
		total.exists = true
		total.requisiteID = maxInt64(total.requisiteID, row.requisiteID)
		total.requisitePhone = maxStringPtr(total.requisitePhone, &row.requisitePhone)
		total.bankCode = maxString(total.bankCode, row.bankCode)
		total.traderID = maxInt64(total.traderID, row.traderID)
		total.traderLogin = maxString(total.traderLogin, row.traderLogin)
		total.successAmountMinor += row.amountMinor
		total.successCount++
		traderTotals[key] = total
	}

	keys := orderedTeamleadTotalKeys(teamleadTotals, traderTotals)
	items := make([]traderReconciliationItemRecord, 0)
	for _, key := range keys {
		teamleadTotal := teamleadTotals[key]
		traderTotal := traderTotals[key]
		if teamleadTotal.successAmountMinor == traderTotal.successAmountMinor && teamleadTotal.successCount == traderTotal.successCount {
			continue
		}

		item, err := buildTeamleadPeriodInboundRequisiteItem(teamleadTotal, traderTotal)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func buildTeamleadPeriodInboundRequisiteItem(teamleadTotal teamleadRequisiteTotal, traderTotal teamleadRequisiteTotal) (traderReconciliationItemRecord, error) {
	var teamleadValue json.RawMessage
	if teamleadTotal.exists {
		value, err := marshalReconciliationItemValue(teamleadPeriodInboundValue{
			RequisitePhone:     teamleadTotal.requisitePhone,
			SuccessAmountMinor: teamleadTotal.successAmountMinor,
			SuccessCount:       teamleadTotal.successCount,
		})
		if err != nil {
			return traderReconciliationItemRecord{}, err
		}
		teamleadValue = value
	}

	var traderValue json.RawMessage
	if traderTotal.exists {
		value, err := marshalReconciliationItemValue(teamleadPeriodInboundTraderValue{
			RequisiteID:        traderTotal.requisiteID,
			RequisitePhone:     stringValue(traderTotal.requisitePhone),
			BankCode:           traderTotal.bankCode,
			TraderID:           traderTotal.traderID,
			TraderLogin:        traderTotal.traderLogin,
			SuccessAmountMinor: traderTotal.successAmountMinor,
			SuccessCount:       traderTotal.successCount,
		})
		if err != nil {
			return traderReconciliationItemRecord{}, err
		}
		traderValue = value
	}

	return traderReconciliationItemRecord{
		issueType:         teamleadRequisiteAmountMismatchIssue,
		teamleadValueJSON: teamleadValue,
		traderValueJSON:   traderValue,
		message:           "Requisite turnover differs between teamlead CSV transactions and CRM final turnover",
	}, nil
}

func buildTeamleadPeriodOutboundReconciliation(teamleadRows []teamleadScopeOrderRow, payoutOrders []teamleadPeriodPayoutOrderRow, transfers []teamleadPeriodPayoutTransferRow) (teamleadReconciliationBuildResult, error) {
	teamleadOrders := latestTeamleadScopeOrders(teamleadRows)
	summary := summarizeTeamleadScopeOrders(teamleadOrders)

	paidByOrderID := teamleadPaidByPayoutOrderID(transfers)
	for _, order := range payoutOrders {
		summary.ActualAmountMinor += paidByOrderID[order.id]
		summary.ActualSuccessCount++
	}

	items, err := buildTeamleadTotalItems(summary, "Teamlead period payout success total differs from trader manual payout transfers")
	if err != nil {
		return teamleadReconciliationBuildResult{}, err
	}

	payoutItems, err := buildTeamleadPeriodOutboundPayoutItems(teamleadOrders, payoutOrders, paidByOrderID)
	if err != nil {
		return teamleadReconciliationBuildResult{}, err
	}
	items = append(items, payoutItems...)

	return teamleadReconciliationBuildResult{summary: summary, items: items}, nil
}

func buildTeamleadPeriodOutboundPayoutItems(teamleadOrders []teamleadScopeOrderRow, payoutOrders []teamleadPeriodPayoutOrderRow, paidByOrderID map[int64]int64) ([]traderReconciliationItemRecord, error) {
	teamleadByAmount := groupTeamleadSuccessfulOrdersByAmount(teamleadOrders)
	payoutByAmount := groupTeamleadPeriodPayoutOrdersByAmount(payoutOrders)
	amounts := orderedTeamleadPayoutAmountKeys(teamleadByAmount, payoutByAmount)

	items := make([]traderReconciliationItemRecord, 0)
	for _, amount := range amounts {
		teamleadRows := teamleadByAmount[amount]
		payoutRows := payoutByAmount[amount]
		maxCount := maxInt(len(teamleadRows), len(payoutRows))
		for index := 0; index < maxCount; index++ {
			var teamleadOrder *teamleadScopeOrderRow
			if index < len(teamleadRows) {
				teamleadOrder = &teamleadRows[index]
			}
			var payoutOrder *teamleadPeriodPayoutOrderRow
			if index < len(payoutRows) {
				payoutOrder = &payoutRows[index]
			}

			item, ok, err := buildTeamleadPeriodOutboundPayoutItem(teamleadOrder, payoutOrder, paidByOrderID)
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

func buildTeamleadPeriodOutboundPayoutItem(teamleadOrder *teamleadScopeOrderRow, payoutOrder *teamleadPeriodPayoutOrderRow, paidByOrderID map[int64]int64) (traderReconciliationItemRecord, bool, error) {
	switch {
	case payoutOrder == nil && teamleadOrder != nil:
		teamleadValue, err := marshalTeamleadPeriodOutboundValue(*teamleadOrder)
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		return traderReconciliationItemRecord{
			issueType:         traderOutboundMissingManualPayoutIssue,
			externalOrderID:   traderInt64Ptr(teamleadOrder.externalOrderID),
			externalInnerID:   traderStringPtr(teamleadOrder.externalInnerID),
			teamleadValueJSON: teamleadValue,
			message:           "Payout is present in teamlead period CSV but no manual payout order with the same amount was created",
		}, true, nil
	case teamleadOrder == nil && payoutOrder != nil:
		traderValue, err := marshalTeamleadPeriodOutboundTraderValue(*payoutOrder, paidByOrderID[payoutOrder.id])
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		return traderReconciliationItemRecord{
			issueType:       traderOutboundExtraManualPayoutIssue,
			traderValueJSON: traderValue,
			message:         "Manual payout order has no matching successful payout amount in teamlead period CSV",
		}, true, nil
	case teamleadOrder != nil && payoutOrder != nil:
		paidAmount := paidByOrderID[payoutOrder.id]
		if paidAmount == payoutOrder.amountMinor {
			return traderReconciliationItemRecord{}, false, nil
		}
		teamleadValue, err := marshalTeamleadPeriodOutboundValue(*teamleadOrder)
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		traderValue, err := marshalTeamleadPeriodOutboundTraderValue(*payoutOrder, paidAmount)
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		return traderReconciliationItemRecord{
			issueType:         traderOutboundManualPayoutNotPaidIssue,
			externalOrderID:   traderInt64Ptr(teamleadOrder.externalOrderID),
			externalInnerID:   traderStringPtr(teamleadOrder.externalInnerID),
			teamleadValueJSON: teamleadValue,
			traderValueJSON:   traderValue,
			message:           "Manual payout order is not fully paid by transfers",
		}, true, nil
	default:
		return traderReconciliationItemRecord{}, false, nil
	}
}

func buildTeamleadCurrentReconciliation(teamleadRows []teamleadScopeOrderRow, traderRows []teamleadScopeOrderRow) (teamleadReconciliationBuildResult, error) {
	teamleadOrders := latestTeamleadScopeOrders(teamleadRows)
	traderOrders := latestTeamleadScopeOrders(traderRows)

	summary := summarizeTeamleadScopeOrders(teamleadOrders)
	for _, order := range traderOrders {
		if isSuccessfulReconciliationStatus(order.normalizedStatus) {
			summary.ActualAmountMinor += order.amountMinor
			summary.ActualSuccessCount++
		}
	}

	items, err := buildTeamleadTotalItems(summary, "Teamlead CSV total differs from existing CRM order snapshots")
	if err != nil {
		return teamleadReconciliationBuildResult{}, err
	}

	traderByInnerID := make(map[string]teamleadScopeOrderRow, len(traderOrders))
	for _, order := range traderOrders {
		traderByInnerID[order.externalInnerID] = order
	}
	for _, teamleadOrder := range teamleadOrders {
		traderOrder, hasTraderOrder := traderByInnerID[teamleadOrder.externalInnerID]
		item, ok, err := buildTeamleadCurrentOrderItem(teamleadOrder, traderOrder, hasTraderOrder)
		if err != nil {
			return teamleadReconciliationBuildResult{}, err
		}
		if ok {
			items = append(items, item)
		}
	}

	return teamleadReconciliationBuildResult{summary: summary, items: items}, nil
}

func buildTeamleadCurrentOrderItem(teamleadOrder teamleadScopeOrderRow, traderOrder teamleadScopeOrderRow, hasTraderOrder bool) (traderReconciliationItemRecord, bool, error) {
	issueType := ""
	message := ""
	switch {
	case !hasTraderOrder:
		issueType = teamleadMissingTraderImportIssue
		message = "Order from teamlead CSV was not found in trader imports"
	case teamleadOrder.amountMinor != traderOrder.amountMinor:
		issueType = teamleadAmountMismatchIssue
		message = "Order amount changed in teamlead CSV"
	case teamleadOrder.normalizedStatus != traderOrder.normalizedStatus:
		issueType = teamleadStatusMismatchIssue
		message = "Order status changed in teamlead CSV"
	case teamleadOrder.workerName != traderOrder.workerName:
		issueType = teamleadWorkerMismatchIssue
		message = "Order worker changed in teamlead CSV"
	default:
		return traderReconciliationItemRecord{}, false, nil
	}

	teamleadValue, err := marshalTeamleadCurrentValue(teamleadOrder)
	if err != nil {
		return traderReconciliationItemRecord{}, false, err
	}
	var traderValue json.RawMessage
	if hasTraderOrder {
		value, err := marshalTeamleadCurrentTraderValue(traderOrder)
		if err != nil {
			return traderReconciliationItemRecord{}, false, err
		}
		traderValue = value
	}

	return traderReconciliationItemRecord{
		issueType:         issueType,
		externalOrderID:   traderInt64Ptr(teamleadOrder.externalOrderID),
		externalInnerID:   traderStringPtr(teamleadOrder.externalInnerID),
		teamleadValueJSON: teamleadValue,
		traderValueJSON:   traderValue,
		message:           message,
	}, true, nil
}

func summarizeTeamleadScopeOrders(orders []teamleadScopeOrderRow) teamleadReconciliationSummary {
	var summary teamleadReconciliationSummary
	for _, order := range orders {
		summary.TotalAmountMinor += order.amountMinor
		summary.TotalCount++
		switch {
		case isSuccessfulReconciliationStatus(order.normalizedStatus):
			summary.ExpectedAmountMinor += order.amountMinor
			summary.ExpectedSuccessCount++
		case order.normalizedStatus == "failed" || order.normalizedStatus == "cancelled":
			summary.FailedAmountMinor += order.amountMinor
			summary.FailedCount++
		}
	}
	return summary
}

func buildTeamleadTotalItems(summary teamleadReconciliationSummary, message string) ([]traderReconciliationItemRecord, error) {
	if summary.ExpectedAmountMinor == summary.ActualAmountMinor && summary.ExpectedSuccessCount == summary.ActualSuccessCount {
		return nil, nil
	}
	teamleadValue, err := marshalReconciliationItemValue(teamleadSuccessTotalPayload{
		SuccessAmountMinor: summary.ExpectedAmountMinor,
		SuccessCount:       summary.ExpectedSuccessCount,
	})
	if err != nil {
		return nil, err
	}
	traderValue, err := marshalReconciliationItemValue(teamleadSuccessTotalPayload{
		SuccessAmountMinor: summary.ActualAmountMinor,
		SuccessCount:       summary.ActualSuccessCount,
	})
	if err != nil {
		return nil, err
	}
	return []traderReconciliationItemRecord{{
		issueType:         teamleadTotalAmountMismatchIssue,
		teamleadValueJSON: teamleadValue,
		traderValueJSON:   traderValue,
		message:           message,
	}}, nil
}

func teamleadReconciliationStatus(summary teamleadReconciliationSummary, itemCount int) string {
	if summary.ActualAmountMinor-summary.ExpectedAmountMinor == 0 && summary.ExpectedSuccessCount == summary.ActualSuccessCount && itemCount == 0 {
		return StatusMatched
	}
	return StatusMismatch
}

func latestTeamleadScopeOrders(rows []teamleadScopeOrderRow) []teamleadScopeOrderRow {
	ordered := append([]teamleadScopeOrderRow(nil), rows...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left.externalInnerID != right.externalInnerID {
			return left.externalInnerID < right.externalInnerID
		}
		if !left.createdAt.Equal(right.createdAt) {
			return left.createdAt.After(right.createdAt)
		}
		return left.id > right.id
	})

	seen := make(map[string]bool, len(ordered))
	result := make([]teamleadScopeOrderRow, 0, len(ordered))
	for _, row := range ordered {
		if seen[row.externalInnerID] {
			continue
		}
		seen[row.externalInnerID] = true
		result = append(result, row)
	}
	return result
}

func groupTeamleadSuccessfulOrdersByAmount(orders []teamleadScopeOrderRow) map[int64][]teamleadScopeOrderRow {
	result := make(map[int64][]teamleadScopeOrderRow)
	for _, order := range orders {
		if !isSuccessfulReconciliationStatus(order.normalizedStatus) {
			continue
		}
		result[order.amountMinor] = append(result[order.amountMinor], order)
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

func groupTeamleadPeriodPayoutOrdersByAmount(orders []teamleadPeriodPayoutOrderRow) map[int64][]teamleadPeriodPayoutOrderRow {
	rows := append([]teamleadPeriodPayoutOrderRow(nil), orders...)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].amountMinor != rows[j].amountMinor {
			return rows[i].amountMinor < rows[j].amountMinor
		}
		return rows[i].id < rows[j].id
	})
	result := make(map[int64][]teamleadPeriodPayoutOrderRow)
	for _, row := range rows {
		result[row.amountMinor] = append(result[row.amountMinor], row)
	}
	return result
}

func teamleadPaidByPayoutOrderID(transfers []teamleadPeriodPayoutTransferRow) map[int64]int64 {
	paidByOrderID := make(map[int64]int64)
	for _, transfer := range transfers {
		paidByOrderID[transfer.manualPayoutOrderID] += transfer.amountMinor
	}
	return paidByOrderID
}

func orderedTeamleadPayoutAmountKeys(left map[int64][]teamleadScopeOrderRow, right map[int64][]teamleadPeriodPayoutOrderRow) []int64 {
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

func orderedTeamleadTotalKeys(left map[string]teamleadRequisiteTotal, right map[string]teamleadRequisiteTotal) []string {
	seen := make(map[string]bool, len(left)+len(right))
	for key := range left {
		seen[key] = true
	}
	for key := range right {
		seen[key] = true
	}
	return orderedStringKeys(seen)
}

func marshalTeamleadPeriodOutboundValue(order teamleadScopeOrderRow) (json.RawMessage, error) {
	return marshalReconciliationItemValue(teamleadPeriodOutboundValue{
		WorkerName:       order.workerName,
		TraderID:         order.traderID,
		Destination:      firstPresentOrderString(order.requisitePhone, order.requisiteRaw, ""),
		AmountMinor:      order.amountMinor,
		NormalizedStatus: order.normalizedStatus,
	})
}

func marshalTeamleadPeriodOutboundTraderValue(order teamleadPeriodPayoutOrderRow, paidAmount int64) (json.RawMessage, error) {
	return marshalReconciliationItemValue(teamleadPeriodOutboundTraderValue{
		ManualPayoutOrderID:  order.id,
		DestinationBank:      order.destinationBank,
		DestinationRequisite: order.destinationRequisite,
		TraderID:             order.traderID,
		TraderLogin:          order.traderLogin,
		AmountMinor:          order.amountMinor,
		PaidAmountMinor:      paidAmount,
	})
}

func marshalTeamleadCurrentValue(order teamleadScopeOrderRow) (json.RawMessage, error) {
	return marshalReconciliationItemValue(teamleadCurrentValue{
		WorkerName:        order.workerName,
		TraderID:          order.traderID,
		RequisitePhone:    order.requisitePhone,
		Requisite:         order.requisiteRaw,
		AmountMinor:       order.amountMinor,
		RawStatus:         order.rawStatus,
		NormalizedStatus:  order.normalizedStatus,
		CreatedAtExternal: order.createdAtExternal,
	})
}

func marshalTeamleadCurrentTraderValue(order teamleadScopeOrderRow) (json.RawMessage, error) {
	return marshalReconciliationItemValue(teamleadCurrentTraderValue{
		WorkerName:        order.workerName,
		TraderID:          order.traderID,
		TraderLogin:       order.traderLogin,
		RequisitePhone:    order.requisitePhone,
		Requisite:         order.requisiteRaw,
		AmountMinor:       order.amountMinor,
		RawStatus:         order.rawStatus,
		NormalizedStatus:  order.normalizedStatus,
		CreatedAtExternal: order.createdAtExternal,
	})
}

func firstPresentOrderString(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case *string:
			if typed != nil {
				return *typed
			}
		case string:
			return typed
		}
	}
	return ""
}

func maxStringPtr(left *string, right *string) *string {
	if right == nil {
		return left
	}
	if left == nil || *right > *left {
		value := *right
		return &value
	}
	return left
}

func maxString(left string, right string) string {
	if right > left {
		return right
	}
	return left
}

func maxInt64(left int64, right int64) int64 {
	if right > left {
		return right
	}
	return left
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
