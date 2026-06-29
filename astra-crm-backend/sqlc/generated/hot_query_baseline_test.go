package sqlc

import (
	"strings"
	"testing"
)

// SQL-OPT-001 inventory: orders readmodels, dashboards, imports/reimports,
// shift report, trader reconciliation, teamlead reconciliation, requisites.
// These tests intentionally pin current SQL contracts while SQL-OPT-003+
// moves heavy filtering/matching/counting from SQL into Go step by step.
func TestOrderReadmodelHotQueryBaseline(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "trader count",
			query: countTraderOrders,
			want: []string{
				"SELECT count(*)::bigint",
				"osi.scope_type = 'trader_shift'",
				"osi.shift_id = $2",
				"osi.direction = $3",
				"osi.is_active = TRUE",
				"osi.created_at_external >= $4::timestamptz",
				"osi.created_at_external < $5::timestamptz",
				"osi.worker_name ILIKE '%' || $6::text || '%'",
				"osi.requisite_raw ILIKE '%' || $7::text || '%'",
				"osi.method_type = $8::text",
				"osi.raw_status = $9::text",
				"osi.normalized_status = $9::text",
				"osi.amount_minor >= $10::bigint",
				"osi.amount_minor <= $11::bigint",
			},
		},
		{
			name:  "trader created desc list",
			query: listTraderOrdersCreatedDesc,
			want: []string{
				"osi.scope_type = 'trader_shift'",
				"osi.shift_id = $2",
				"osi.direction = $3",
				"osi.is_active = TRUE",
				"osi.created_at_external >= $4::timestamptz",
				"osi.created_at_external < $5::timestamptz",
				"ORDER BY osi.created_at_external DESC, osi.id DESC",
			},
		},
		{
			name:  "trader created asc list",
			query: listTraderOrdersCreatedAsc,
			want: []string{
				"ORDER BY osi.created_at_external ASC, osi.id DESC",
			},
		},
		{
			name:  "trader amount asc list",
			query: listTraderOrdersAmountAsc,
			want: []string{
				"ORDER BY osi.amount_minor ASC, osi.created_at_external DESC, osi.id DESC",
			},
		},
		{
			name:  "trader amount desc list",
			query: listTraderOrdersAmountDesc,
			want: []string{
				"ORDER BY osi.amount_minor DESC, osi.created_at_external DESC, osi.id DESC",
			},
		},
		{
			name:  "teamlead count",
			query: countTeamleadOrders,
			want: []string{
				"SELECT count(*)::bigint",
				"JOIN trader_shifts ts ON ts.id = osi.shift_id",
				"osi.scope_type = 'trader_shift'",
				"osi.direction = $2",
				"osi.is_active = TRUE",
				"ts.status IN ('closed', 'closed_with_discrepancy')",
				"ts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')",
				"ts.outbound_reconciliation_status IN ('matched', 'accepted_with_comment')",
				"osi.created_at_external >= $4::timestamptz",
				"osi.created_at_external < $5::timestamptz",
				"osi.trader_id = $6::bigint",
				"osi.trader_id = ANY($7::bigint[])",
				"osi.worker_name ILIKE '%' || $8::text || '%'",
				"osi.requisite_raw ILIKE '%' || $9::text || '%'",
				"osi.method_type = $10::text",
				"osi.raw_status = $11::text",
				"osi.normalized_status = $11::text",
				"osi.amount_minor >= $12::bigint",
				"osi.amount_minor <= $13::bigint",
			},
		},
		{
			name:  "teamlead created desc list",
			query: listTeamleadOrdersCreatedDesc,
			want: []string{
				"JOIN trader_shifts ts ON ts.id = osi.shift_id",
				"osi.scope_type = 'trader_shift'",
				"osi.direction = $2",
				"osi.is_active = TRUE",
				"osi.created_at_external >= $4::timestamptz",
				"osi.created_at_external < $5::timestamptz",
				"ORDER BY osi.created_at_external DESC, osi.id DESC",
			},
		},
		{
			name:  "teamlead created asc list",
			query: listTeamleadOrdersCreatedAsc,
			want: []string{
				"ORDER BY osi.created_at_external ASC, osi.id DESC",
			},
		},
		{
			name:  "teamlead amount asc list",
			query: listTeamleadOrdersAmountAsc,
			want: []string{
				"ORDER BY osi.amount_minor ASC, osi.created_at_external DESC, osi.id DESC",
			},
		},
		{
			name:  "teamlead amount desc list",
			query: listTeamleadOrdersAmountDesc,
			want: []string{
				"ORDER BY osi.amount_minor DESC, osi.created_at_external DESC, osi.id DESC",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSQLContainsAll(t, tc.query, tc.want)
		})
	}

	orderQueries := []string{
		countTraderOrders,
		listTraderOrdersCreatedDesc,
		listTraderOrdersCreatedAsc,
		listTraderOrdersAmountAsc,
		listTraderOrdersAmountDesc,
		countTeamleadOrders,
		listTeamleadOrdersCreatedDesc,
		listTeamleadOrdersCreatedAsc,
		listTeamleadOrdersAmountAsc,
		listTeamleadOrdersAmountDesc,
	}
	for _, query := range orderQueries {
		assertSQLNotContainsAny(t, query, []string{
			"created_at_external::date",
			"count(*) OVER()",
			"ORDER BY\n    CASE",
		})
	}
}

func TestDashboardReadmodelHotQueryBaseline(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "trader summary",
			query: traderOrdersSummary,
			want: []string{
				"sum(CASE WHEN osi.normalized_status IN ('success', 'corrected') THEN osi.amount_minor ELSE 0 END)",
				"count(*) FILTER (WHERE osi.normalized_status IN ('success', 'corrected'))",
				"osi.scope_type = 'trader_shift'",
				"osi.shift_id = $2",
				"osi.direction = $3",
				"osi.is_active = TRUE",
				"osi.created_at_external >= $4::timestamptz",
				"osi.created_at_external < $5::timestamptz",
			},
		},
		{
			name:  "teamlead summary",
			query: teamleadOrdersSummary,
			want: []string{
				"JOIN trader_shifts ts ON ts.id = osi.shift_id",
				"ts.status IN ('closed', 'closed_with_discrepancy')",
				"osi.scope_type = 'trader_shift'",
				"osi.direction = $2",
				"osi.is_active = TRUE",
				"osi.created_at_external >= $4::timestamptz",
				"osi.created_at_external < $5::timestamptz",
				"osi.trader_id = $6::bigint",
				"osi.trader_id = ANY($7::bigint[])",
			},
		},
		{
			name:  "teamlead recent imports",
			query: teamleadRecentImports,
			want: []string{
				"scope_type = 'teamlead_period'",
				"accounting_period_id IS NULL",
				"direction = $2",
				"uploaded_by = $3",
				"ORDER BY created_at DESC, id DESC",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSQLContainsAll(t, tc.query, tc.want)
		})
	}
}

func TestImportReimportHotQueryBaseline(t *testing.T) {
	assertSQLContainsAll(t, batchInsertImportRows, []string{
		"INSERT INTO import_rows",
		"jsonb_to_recordset($2::jsonb)",
		"ORDER BY input.row_number",
	})
	assertSQLContainsAll(t, batchUpsertExternalOrders, []string{
		"INSERT INTO external_orders",
		"jsonb_to_recordset($5::jsonb)",
		"ON CONFLICT (team_id, direction, external_inner_id)",
		"RETURNING id, external_inner_id, (xmax = 0) AS inserted",
	})
	assertSQLContainsAll(t, createTraderShiftScopeItemsBatch, []string{
		"INSERT INTO order_scope_items",
		"'trader_shift'",
		"$3",
		"NULL",
		"external_orders.external_inner_id",
		"external_orders.amount_minor",
		"external_orders.normalized_status",
		"external_orders.created_at_external",
		"TRUE",
		"FROM import_rows",
		"JOIN external_orders",
		"WHERE import_rows.import_batch_id = $4",
		"ORDER BY import_rows.row_number",
	})
	assertSQLContainsAll(t, createTeamleadPeriodScopeItemsBatch, []string{
		"INSERT INTO order_scope_items",
		"'teamlead_period'",
		"NULL",
		"$3",
		"external_orders.external_inner_id",
		"external_orders.amount_minor",
		"external_orders.normalized_status",
		"external_orders.created_at_external",
		"TRUE",
		"FROM import_rows",
		"JOIN external_orders",
		"WHERE import_rows.import_batch_id = $4",
		"ORDER BY import_rows.row_number",
	})
	assertSQLContainsAll(t, deactivateTraderShiftScopeItems, []string{
		"SET is_active = FALSE",
		"deactivated_at = now()",
		"team_id = $1",
		"scope_type = 'trader_shift'",
		"shift_id = $2",
		"direction = $3",
		"is_active = TRUE",
	})
	assertSQLContainsAll(t, deactivateTeamleadPeriodScopeItems, []string{
		"SET is_active = FALSE",
		"deactivated_at = now()",
		"order_scope_items.scope_type = 'teamlead_period'",
		"ib.uploaded_by = $2",
		"order_scope_items.accounting_period_id = $3",
		"$3::bigint IS NULL AND order_scope_items.accounting_period_id IS NULL",
		"order_scope_items.direction = $4",
		"order_scope_items.is_active = TRUE",
	})
}

func TestShiftReportHotQueryBaseline(t *testing.T) {
	assertSQLContainsAll(t, listShiftReportRequisites, []string{
		"FROM shift_requisites sr",
		"JOIN trader_shifts ts ON ts.id = sr.shift_id",
		"JOIN requisites r ON r.id = sr.requisite_id",
		"LEFT JOIN requisite_assignments ra ON ra.id = sr.assignment_id",
		"sr.team_id = $1",
		"sr.shift_id = $2",
		"ORDER BY sr.taken_at DESC, sr.id DESC",
	})
	assertSQLContainsAll(t, listShiftReportInboundScopeItems, []string{
		"FROM order_scope_items",
		"scope_type = 'trader_shift'",
		"direction = 'inbound'",
		"is_active = TRUE",
	})
	assertSQLContainsAll(t, listShiftReportOutboundTransfers, []string{
		"FROM manual_payout_transfers",
		"team_id = $1",
		"shift_id = $2",
		"source_shift_requisite_id",
	})
}

func TestTraderReconciliationHotQueryBaseline(t *testing.T) {
	assertSQLContainsAll(t, calculateTraderInboundExpected, []string{
		"normalized_status IN ('success', 'corrected')",
		"scope_type = 'trader_shift'",
		"direction = 'inbound'",
		"is_active = TRUE",
	})
	assertSQLContainsAll(t, calculateTraderInboundActual, []string{
		"FROM shift_requisites sr",
		"sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')",
		"ts.status IN ('open', 'closing')",
	})
	assertSQLContainsAll(t, listTraderInboundReconciliationRequisites, []string{
		"FROM shift_requisites sr",
		"JOIN requisites r ON r.id = sr.requisite_id",
		"JOIN trader_shifts ts ON ts.id = sr.shift_id",
		"r.bank_code",
		"sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction')",
		"ts.status IN ('open', 'closing')",
		"ORDER BY sr.id",
	})
	assertSQLContainsAll(t, listTraderInboundReconciliationScopeItems, []string{
		"FROM order_scope_items",
		"scope_type = 'trader_shift'",
		"direction = 'inbound'",
		"is_active = TRUE",
		"ORDER BY id",
	})
	assertSQLContainsAll(t, updateShiftRequisiteInboundReviewStatus, []string{
		"UPDATE shift_requisites",
		"SET status = $1",
		"id = $2",
		"team_id = $3",
		"trader_id = $4",
		"shift_id = $5",
	})
	assertSQLContainsAll(t, calculateTraderOutboundExpected, []string{
		"normalized_status IN ('success', 'corrected')",
		"scope_type = 'trader_shift'",
		"direction = 'outbound'",
		"is_active = TRUE",
	})
	assertSQLContainsAll(t, calculateTraderOutboundActual, []string{
		"FROM manual_payout_transfers mpt",
		"mpt.team_id = $1",
		"mpt.trader_id = $2",
		"mpt.shift_id = $3",
		"ts.status IN ('open', 'closing')",
	})
	assertSQLContainsAll(t, listTraderOutboundReconciliationSourceRequisites, []string{
		"FROM shift_requisites sr",
		"JOIN requisites r ON r.id = sr.requisite_id",
		"r.bank_code",
		"sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')",
		"ORDER BY sr.id",
	})
	assertSQLContainsAll(t, listTraderOutboundReconciliationScopeItems, []string{
		"FROM order_scope_items",
		"scope_type = 'trader_shift'",
		"direction = 'outbound'",
		"is_active = TRUE",
		"ORDER BY external_inner_id, created_at DESC, id DESC",
	})
	assertSQLContainsAll(t, listTraderOutboundReconciliationPayoutOrders, []string{
		"FROM manual_payout_orders",
		"deleted_at IS NULL",
		"status <> 'cancelled'",
		"ORDER BY amount_minor, created_at, id",
	})
	assertSQLContainsAll(t, listTraderOutboundReconciliationTransfers, []string{
		"FROM manual_payout_transfers",
		"team_id = $1",
		"trader_id = $2",
		"shift_id = $3",
		"ORDER BY id",
	})
	assertSQLContainsAll(t, createReconciliationItem, []string{
		"INSERT INTO reconciliation_items",
		"reconciliation_run_id",
		"issue_type",
		"teamlead_value_json",
		"trader_value_json",
	})

	traderReconciliationQueries := []string{
		listTraderInboundReconciliationRequisites,
		listTraderInboundReconciliationReviewRequisites,
		listTraderInboundReconciliationScopeItems,
		updateShiftRequisiteInboundReviewStatus,
		listTraderOutboundReconciliationSourceRequisites,
		listTraderOutboundReconciliationScopeItems,
		listTraderOutboundReconciliationPayoutOrders,
		listTraderOutboundReconciliationTransfers,
		createReconciliationItem,
	}
	for _, query := range traderReconciliationQueries {
		assertSQLNotContainsAny(t, query, []string{
			"regexp_replace",
			"jsonb_build_object",
			"FULL JOIN",
			"row_number()",
			"JOIN banks",
		})
	}
}

func TestTeamleadReconciliationHotQueryBaseline(t *testing.T) {
	assertSQLContainsAll(t, getReconciliationAccountingPeriod, []string{
		"FROM accounting_periods",
		"team_id = $1",
		"id = $2",
	})
	assertSQLContainsAll(t, listTeamleadPeriodReconciliationScopeItems, []string{
		"FROM order_scope_items",
		"scope_type = 'teamlead_period'",
		"accounting_period_id = $2",
		"direction = $3",
		"is_active = TRUE",
		"ORDER BY external_inner_id, created_at DESC, id DESC",
	})
	assertSQLContainsAll(t, listTeamleadPeriodInboundShiftRequisites, []string{
		"FROM shift_requisites sr",
		"JOIN requisites r ON r.id = sr.requisite_id",
		"JOIN users u ON u.id = sr.trader_id",
		"COALESCE(sr.released_at, sr.taken_at) >= $2::timestamptz",
		"COALESCE(sr.released_at, sr.taken_at) < $3::timestamptz",
		"sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')",
	})
	assertSQLContainsAll(t, listTeamleadPeriodPayoutOrders, []string{
		"FROM manual_payout_orders mpo",
		"JOIN users u ON u.id = mpo.trader_id",
		"JOIN trader_shifts ts ON ts.id = mpo.shift_id",
		"mpo.created_at >= $2::timestamptz",
		"mpo.created_at < $3::timestamptz",
		"mpo.deleted_at IS NULL",
		"mpo.status <> 'cancelled'",
	})
	assertSQLContainsAll(t, listTeamleadPeriodPayoutTransfers, []string{
		"FROM manual_payout_transfers mpt",
		"JOIN manual_payout_orders mpo ON mpo.id = mpt.manual_payout_order_id",
		"mpo.created_at >= $2::timestamptz",
		"mpo.created_at < $3::timestamptz",
	})
	assertSQLContainsAll(t, listTeamleadCurrentReconciliationScopeItems, []string{
		"FROM order_scope_items",
		"accounting_period_id IS NULL",
		"direction = $2",
		"import_batch_id = $3",
		"ORDER BY external_inner_id, created_at DESC, id DESC",
	})
	assertSQLContainsAll(t, listTraderReconciliationScopeItemsByInnerIDs, []string{
		"FROM order_scope_items osi",
		"LEFT JOIN users u ON u.id = osi.trader_id",
		"osi.scope_type = 'trader_shift'",
		"osi.direction = $2",
		"osi.external_inner_id = ANY($3::text[])",
	})
	assertSQLContainsAll(t, listTeamleadReconciliationExternalOrdersInPeriod, []string{
		"FROM external_orders",
		"team_id = $1",
		"direction = $2",
		"created_at_external >= $3::timestamptz",
		"created_at_external < $4::timestamptz",
	})
	assertSQLNotContainsAny(t, listTeamleadReconciliationExternalOrdersInPeriod, []string{
		"created_at_external::date",
	})
	assertSQLContainsAll(t, listTeamleadReconciliationItems, []string{
		"FROM teamlead_reconciliation_items",
		"teamlead_reconciliation_id = $2",
		"direction = $3",
		"stage = $4",
		"issue_type = $5",
		"severity = $6",
		"trader_id = $7",
		"requisite_id = $8",
		"NOT $9::boolean OR severity <> 'info'",
		"ORDER BY id",
	})

	teamleadReconciliationQueries := []string{
		getReconciliationAccountingPeriod,
		listTeamleadPeriodReconciliationScopeItems,
		listTeamleadPeriodInboundShiftRequisites,
		listTeamleadPeriodPayoutOrders,
		listTeamleadPeriodPayoutTransfers,
		listTeamleadCurrentReconciliationScopeItems,
		listTraderReconciliationScopeItemsByInnerIDs,
	}
	for _, query := range teamleadReconciliationQueries {
		assertSQLNotContainsAny(t, query, []string{
			"WITH ",
			"jsonb_build_object",
			"FULL JOIN",
			"row_number()",
			"::date",
		})
	}
}

func TestRequisiteReadmodelHotQueryBaseline(t *testing.T) {
	assertSQLContainsAll(t, listRequisiteDetailsByTeam, []string{
		"LEFT JOIN LATERAL",
		"ra.unassigned_at IS NULL",
		"ra.status IN ('planned', 'assigned', 'in_work')",
		"ORDER BY ra.assigned_for_date DESC, ra.assigned_at DESC, ra.id DESC",
		"r.deleted_at IS NULL",
		"r.bank_code = $2::text",
		"r.status = $3::text",
		"NOT EXISTS",
		"blocked_ra.status = 'blocked'",
		"$5::text = 'unassigned' AND ra.trader_id IS NULL",
		"lower(r.phone) LIKE '%' || lower($6::text) || '%'",
		"r.bank_code = ANY($7::text[])",
		"regexp_replace(r.phone, '[^0-9]', '', 'g') LIKE '%' || $8::text || '%'",
		"ORDER BY r.created_at DESC, r.id DESC",
	})
	assertSQLContainsAll(t, listRequisiteReportShifts, []string{
		"FROM shift_requisites sr",
		"JOIN trader_shifts ts ON ts.id = sr.shift_id",
		"LEFT JOIN requisite_assignments ra ON ra.id = sr.assignment_id",
		"sr.team_id = $1",
		"sr.requisite_id = $2",
		"ORDER BY COALESCE(ra.assigned_for_date, sr.taken_at::date) ASC, sr.taken_at ASC, sr.id ASC",
	})
	for _, query := range []string{
		getRequisiteDetailsByIDForTeam,
		listRequisiteDetailsByTeam,
		countRequisiteDetailsByTeam,
		listTeamleadRequisitePlans,
		listTeamleadRequisiteActivity,
		countTeamleadRequisiteActivity,
		getRequisiteReportSummary,
	} {
		assertSQLNotContainsAny(t, query, []string{"JOIN banks"})
	}
}

func assertSQLContainsAll(t *testing.T, query string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(query, want) {
			t.Fatalf("query is missing %q\nquery:\n%s", want, query)
		}
	}
}

func assertSQLNotContainsAny(t *testing.T, query string, values []string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(query, value) {
			t.Fatalf("query contains %q\nquery:\n%s", value, query)
		}
	}
}
