-- name: CalculateTraderInboundExpected :one
SELECT
    COALESCE(sum(CASE WHEN osi.normalized_status IN ('success', 'corrected') THEN osi.amount_minor ELSE 0 END), 0)::bigint AS expected_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status IN ('success', 'corrected'))::bigint AS success_count,
    COALESCE(sum(CASE WHEN osi.normalized_status IN ('failed', 'cancelled') THEN osi.amount_minor ELSE 0 END), 0)::bigint AS failed_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status IN ('failed', 'cancelled'))::bigint AS failed_count,
    COALESCE(sum(osi.amount_minor), 0)::bigint AS total_amount_minor,
    count(*)::bigint AS total_count
FROM order_scope_items osi
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'trader_shift'
  AND osi.shift_id = sqlc.arg(shift_id)
  AND osi.direction = 'inbound'
  AND osi.is_active = TRUE;

-- name: CalculateTraderInboundActual :one
WITH latest AS (
    SELECT DISTINCT ON (e.shift_requisite_id)
        e.amount_minor
    FROM requisite_turnover_entries e
    JOIN trader_shifts ts ON ts.id = e.shift_id
    WHERE e.team_id = sqlc.arg(team_id)
      AND e.trader_id = sqlc.arg(trader_id)
      AND e.shift_id = sqlc.arg(shift_id)
      AND ts.status IN ('open', 'closing')
    ORDER BY e.shift_requisite_id, e.created_at DESC, e.id DESC
)
SELECT COALESCE(sum(amount_minor), 0)::bigint AS actual_amount_minor
FROM latest;

-- name: CreateTraderInboundReconciliationRun :one
INSERT INTO reconciliation_runs (
    team_id,
    type,
    scope_type,
    shift_id,
    accounting_period_id,
    trader_id,
    import_batch_id,
    expected_amount_minor,
    actual_amount_minor,
    diff_amount_minor,
    success_amount_minor,
    success_count,
    failed_amount_minor,
    failed_count,
    total_amount_minor,
    total_count,
    status
)
VALUES (
    sqlc.arg(team_id),
    'trader_shift_inbound',
    'trader_shift',
    sqlc.arg(shift_id),
    NULL,
    sqlc.arg(trader_id),
    sqlc.narg(import_batch_id),
    sqlc.arg(expected_amount_minor),
    sqlc.arg(actual_amount_minor),
    sqlc.arg(diff_amount_minor),
    sqlc.arg(success_amount_minor),
    sqlc.arg(success_count),
    sqlc.arg(failed_amount_minor),
    sqlc.arg(failed_count),
    sqlc.arg(total_amount_minor),
    sqlc.arg(total_count),
    sqlc.arg(status)
)
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: LatestTraderInboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
  AND type = 'trader_shift_inbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpdateTraderShiftInboundReconciliationStatus :exec
UPDATE trader_shifts
SET inbound_reconciliation_status = sqlc.arg(status),
    updated_at = now()
WHERE id = sqlc.arg(shift_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND status IN ('open', 'closing');

-- name: AcceptTraderInboundReconciliationRun :one
UPDATE reconciliation_runs
SET status = 'accepted_with_comment',
    comment = sqlc.arg(comment),
    confirmed_by = sqlc.arg(confirmed_by),
    confirmed_at = now()
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND type = 'trader_shift_inbound'
  AND status = 'mismatch'
  AND btrim(sqlc.arg(comment)) <> ''
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: CalculateTraderOutboundExpected :one
SELECT
    COALESCE(sum(osi.amount_minor), 0)::bigint AS expected_amount_minor,
    count(*)::bigint AS order_count
FROM order_scope_items osi
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'trader_shift'
  AND osi.shift_id = sqlc.arg(shift_id)
  AND osi.direction = 'outbound'
  AND osi.is_active = TRUE;

-- name: CalculateTraderOutboundActual :one
SELECT COALESCE(sum(mpt.amount_minor), 0)::bigint AS actual_amount_minor
FROM manual_payout_transfers mpt
JOIN trader_shifts ts ON ts.id = mpt.shift_id
WHERE mpt.team_id = sqlc.arg(team_id)
  AND mpt.trader_id = sqlc.arg(trader_id)
  AND mpt.shift_id = sqlc.arg(shift_id)
  AND ts.status IN ('open', 'closing');

-- name: CreateTraderOutboundReconciliationRun :one
INSERT INTO reconciliation_runs (
    team_id,
    type,
    scope_type,
    shift_id,
    accounting_period_id,
    trader_id,
    import_batch_id,
    expected_amount_minor,
    actual_amount_minor,
    diff_amount_minor,
    success_amount_minor,
    success_count,
    failed_amount_minor,
    failed_count,
    total_amount_minor,
    total_count,
    status
)
VALUES (
    sqlc.arg(team_id),
    'trader_shift_outbound',
    'trader_shift',
    sqlc.arg(shift_id),
    NULL,
    sqlc.arg(trader_id),
    sqlc.narg(import_batch_id),
    sqlc.arg(expected_amount_minor),
    sqlc.arg(actual_amount_minor),
    sqlc.arg(diff_amount_minor),
    sqlc.arg(expected_amount_minor),
    sqlc.arg(order_count),
    0,
    0,
    sqlc.arg(expected_amount_minor),
    sqlc.arg(order_count),
    sqlc.arg(status)
)
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: CreateTraderOutboundUnpaidPayoutItems :exec
WITH payout_totals AS (
    SELECT
        mpo.id,
        mpo.destination_bank,
        mpo.destination_requisite,
        mpo.amount_minor,
        COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor
    FROM manual_payout_orders mpo
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
    WHERE mpo.team_id = sqlc.arg(team_id)
      AND mpo.trader_id = sqlc.arg(trader_id)
      AND mpo.shift_id = sqlc.arg(shift_id)
      AND mpo.deleted_at IS NULL
      AND mpo.status <> 'cancelled'
    GROUP BY mpo.id
)
INSERT INTO reconciliation_items (
    reconciliation_run_id,
    issue_type,
    trader_value_json,
    message
)
SELECT
    sqlc.arg(run_id),
    'payout_not_fully_paid',
    jsonb_build_object(
        'manualPayoutOrderId', id,
        'destinationBank', destination_bank,
        'destinationRequisite', destination_requisite,
        'amountMinor', amount_minor,
        'paidAmountMinor', paid_amount_minor,
        'remainingAmountMinor', amount_minor - paid_amount_minor
    ),
    'Manual payout order is not fully paid'
FROM payout_totals
WHERE paid_amount_minor <> amount_minor;

-- name: LatestTraderOutboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
  AND type = 'trader_shift_outbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpdateTraderShiftOutboundReconciliationStatus :exec
UPDATE trader_shifts
SET outbound_reconciliation_status = sqlc.arg(status),
    updated_at = now()
WHERE id = sqlc.arg(shift_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND status IN ('open', 'closing');

-- name: AcceptTraderOutboundReconciliationRun :one
UPDATE reconciliation_runs
SET status = 'accepted_with_comment',
    comment = sqlc.arg(comment),
    confirmed_by = sqlc.arg(confirmed_by),
    confirmed_at = now()
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND type = 'trader_shift_outbound'
  AND status = 'mismatch'
  AND btrim(sqlc.arg(comment)) <> ''
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: CalculateTeamleadPeriodInboundSummary :one
WITH period AS (
    SELECT date_from, date_to
    FROM accounting_periods ap
    WHERE ap.team_id = sqlc.arg(team_id)
      AND ap.id = sqlc.arg(accounting_period_id)
),
teamlead_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_inner_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'teamlead_period'
      AND osi.accounting_period_id = sqlc.arg(accounting_period_id)
      AND osi.direction = 'inbound'
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_inner_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    CROSS JOIN period p
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'trader_shift'
      AND osi.direction = 'inbound'
      AND osi.is_active = TRUE
      AND osi.created_at_external::date BETWEEN p.date_from AND p.date_to
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
)
SELECT
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_success_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders), 0)::bigint AS total_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders), 0)::bigint AS total_count,
    COALESCE((SELECT sum(amount_minor) FROM trader_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS actual_amount_minor,
    COALESCE((SELECT count(*) FROM trader_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS actual_success_count;

-- name: CreateTeamleadPeriodInboundReconciliationRun :one
INSERT INTO reconciliation_runs (
    team_id,
    type,
    scope_type,
    shift_id,
    accounting_period_id,
    trader_id,
    import_batch_id,
    expected_amount_minor,
    actual_amount_minor,
    diff_amount_minor,
    success_amount_minor,
    success_count,
    failed_amount_minor,
    failed_count,
    total_amount_minor,
    total_count,
    status
)
VALUES (
    sqlc.arg(team_id),
    'teamlead_period_inbound',
    'teamlead_period',
    NULL,
    sqlc.arg(accounting_period_id),
    NULL,
    sqlc.narg(import_batch_id),
    sqlc.arg(expected_amount_minor),
    sqlc.arg(actual_amount_minor),
    sqlc.arg(diff_amount_minor),
    sqlc.arg(success_amount_minor),
    sqlc.arg(success_count),
    sqlc.arg(failed_amount_minor),
    sqlc.arg(failed_count),
    sqlc.arg(total_amount_minor),
    sqlc.arg(total_count),
    sqlc.arg(status)
)
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: CreateTeamleadPeriodInboundReconciliationItems :many
WITH period AS (
    SELECT date_from, date_to
    FROM accounting_periods ap
    WHERE ap.team_id = sqlc.arg(team_id)
      AND ap.id = sqlc.arg(accounting_period_id)
),
teamlead_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_order_id,
        osi.external_inner_id,
        osi.worker_name,
        osi.trader_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'teamlead_period'
      AND osi.accounting_period_id = sqlc.arg(accounting_period_id)
      AND osi.direction = 'inbound'
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_order_id,
        osi.external_inner_id,
        osi.worker_name,
        osi.trader_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    CROSS JOIN period p
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'trader_shift'
      AND osi.direction = 'inbound'
      AND osi.is_active = TRUE
      AND osi.created_at_external::date BETWEEN p.date_from AND p.date_to
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
teamlead_success_total AS (
    SELECT
        COALESCE(sum(amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM teamlead_orders
    WHERE normalized_status IN ('success', 'corrected')
),
trader_success_total AS (
    SELECT
        COALESCE(sum(amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM trader_orders
    WHERE normalized_status IN ('success', 'corrected')
),
total_items AS (
    SELECT
        'total_amount_mismatch'::text AS issue_type,
        NULL::bigint AS external_order_id,
        NULL::text AS external_inner_id,
        jsonb_build_object(
            'successAmountMinor', tl.amount_minor,
            'successCount', tl.count
        ) AS teamlead_value_json,
        jsonb_build_object(
            'successAmountMinor', tr.amount_minor,
            'successCount', tr.count
        ) AS trader_value_json,
        'Teamlead period success total differs from trader imports'::text AS message
    FROM teamlead_success_total tl
    CROSS JOIN trader_success_total tr
    WHERE tl.amount_minor <> tr.amount_minor
       OR tl.count <> tr.count
),
teamlead_worker_totals AS (
    SELECT
        worker_name,
        COALESCE(sum(amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM teamlead_orders
    WHERE normalized_status IN ('success', 'corrected')
    GROUP BY worker_name
),
trader_worker_totals AS (
    SELECT
        worker_name,
        COALESCE(sum(amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM trader_orders
    WHERE normalized_status IN ('success', 'corrected')
    GROUP BY worker_name
),
worker_total_items AS (
    SELECT
        'total_amount_mismatch'::text AS issue_type,
        NULL::bigint AS external_order_id,
        NULL::text AS external_inner_id,
        CASE
            WHEN tl.worker_name IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'workerName', tl.worker_name,
                'successAmountMinor', tl.amount_minor,
                'successCount', tl.count
            )
        END AS teamlead_value_json,
        CASE
            WHEN tr.worker_name IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'workerName', tr.worker_name,
                'successAmountMinor', tr.amount_minor,
                'successCount', tr.count
            )
        END AS trader_value_json,
        'Worker success total differs between teamlead period and trader imports'::text AS message
    FROM teamlead_worker_totals tl
    FULL JOIN trader_worker_totals tr ON tr.worker_name = tl.worker_name
    WHERE COALESCE(tl.amount_minor, 0) <> COALESCE(tr.amount_minor, 0)
       OR COALESCE(tl.count, 0) <> COALESCE(tr.count, 0)
),
order_items AS (
    SELECT
        CASE
            WHEN tr.external_inner_id IS NULL THEN 'missing_in_trader_import'
            WHEN tl.external_inner_id IS NULL THEN 'extra_in_trader_import'
            WHEN tl.amount_minor <> tr.amount_minor THEN 'amount_mismatch'
            WHEN tl.normalized_status <> tr.normalized_status THEN 'status_mismatch'
            WHEN tl.worker_name <> tr.worker_name THEN 'worker_mismatch'
        END AS issue_type,
        COALESCE(tl.external_order_id, tr.external_order_id) AS external_order_id,
        COALESCE(tl.external_inner_id, tr.external_inner_id) AS external_inner_id,
        CASE
            WHEN tl.external_inner_id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'workerName', tl.worker_name,
                'traderId', tl.trader_id,
                'amountMinor', tl.amount_minor,
                'normalizedStatus', tl.normalized_status
            )
        END AS teamlead_value_json,
        CASE
            WHEN tr.external_inner_id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'workerName', tr.worker_name,
                'traderId', tr.trader_id,
                'amountMinor', tr.amount_minor,
                'normalizedStatus', tr.normalized_status
            )
        END AS trader_value_json,
        CASE
            WHEN tr.external_inner_id IS NULL THEN 'Order is present in teamlead period CSV but absent from trader imports'
            WHEN tl.external_inner_id IS NULL THEN 'Order is present in trader imports but absent from teamlead period CSV'
            WHEN tl.amount_minor <> tr.amount_minor THEN 'Order amount differs between teamlead period CSV and trader import'
            WHEN tl.normalized_status <> tr.normalized_status THEN 'Order status differs between teamlead period CSV and trader import'
            WHEN tl.worker_name <> tr.worker_name THEN 'Order worker differs between teamlead period CSV and trader import'
        END AS message
    FROM teamlead_orders tl
    FULL JOIN trader_orders tr ON tr.external_inner_id = tl.external_inner_id
    WHERE tr.external_inner_id IS NULL
       OR tl.external_inner_id IS NULL
       OR tl.amount_minor <> tr.amount_minor
       OR tl.normalized_status <> tr.normalized_status
       OR tl.worker_name <> tr.worker_name
),
items AS (
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM total_items
    UNION ALL
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM worker_total_items
    UNION ALL
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM order_items
    WHERE issue_type IS NOT NULL
)
INSERT INTO reconciliation_items (
    reconciliation_run_id,
    issue_type,
    external_order_id,
    external_inner_id,
    teamlead_value_json,
    trader_value_json,
    message
)
SELECT
    sqlc.arg(run_id),
    issue_type,
    external_order_id,
    external_inner_id,
    teamlead_value_json,
    trader_value_json,
    message
FROM items
RETURNING id;

-- name: UpdateReconciliationRunStatus :one
UPDATE reconciliation_runs
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: LatestTeamleadPeriodInboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND accounting_period_id = sqlc.arg(accounting_period_id)
  AND type = 'teamlead_period_inbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: LatestTeamleadInboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND type = 'teamlead_period_inbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListReconciliationItemsForRun :many
SELECT id, reconciliation_run_id, issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message, created_at
FROM reconciliation_items
WHERE reconciliation_run_id = sqlc.arg(run_id)
ORDER BY id;

-- name: ListActiveTeamleadInboundPeriodScopes :many
SELECT DISTINCT ON (ib.accounting_period_id)
    ib.accounting_period_id,
    ib.id AS import_batch_id
FROM import_batches ib
WHERE ib.team_id = sqlc.arg(team_id)
  AND ib.scope_type = 'teamlead_period'
  AND ib.direction = 'inbound'
  AND ib.status IN ('applied', 'reconciled')
  AND ib.accounting_period_id IS NOT NULL
  AND ib.superseded_by_batch_id IS NULL
ORDER BY ib.accounting_period_id, ib.applied_at DESC NULLS LAST, ib.id DESC;
