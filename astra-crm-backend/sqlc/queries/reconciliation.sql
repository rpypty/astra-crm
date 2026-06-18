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
SELECT COALESCE(sum(sr.inbound_turnover_minor), 0)::bigint AS actual_amount_minor
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')
  AND ts.status IN ('open', 'closing');

-- name: CountTraderInboundRequisiteMismatches :one
WITH crm_requisites AS (
    SELECT
        sr.id AS shift_requisite_id,
        r.phone,
        sr.inbound_turnover_minor AS actual_amount_minor,
        COALESCE(
            NULLIF(right(regexp_replace(r.phone, '[^0-9]', '', 'g'), 10), ''),
            'requisite:' || r.id::text
        ) AS match_key
    FROM shift_requisites sr
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    WHERE sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.shift_id = sqlc.arg(shift_id)
      AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction')
      AND ts.status IN ('open', 'closing')
),
csv_requisites AS (
    SELECT
        COALESCE(
            NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
            'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
        ) AS match_key,
        COALESCE(sum(CASE WHEN normalized_status IN ('success', 'corrected') THEN amount_minor ELSE 0 END), 0)::bigint AS expected_amount_minor
    FROM order_scope_items
    WHERE team_id = sqlc.arg(team_id)
      AND scope_type = 'trader_shift'
      AND shift_id = sqlc.arg(shift_id)
      AND direction = 'inbound'
      AND is_active = TRUE
    GROUP BY COALESCE(
        NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
        'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
    )
),
all_keys AS (
    SELECT match_key FROM crm_requisites
    UNION
    SELECT match_key FROM csv_requisites
)
SELECT count(*)::bigint
FROM all_keys
LEFT JOIN crm_requisites crm ON crm.match_key = all_keys.match_key
LEFT JOIN csv_requisites csv ON csv.match_key = all_keys.match_key
WHERE COALESCE(crm.actual_amount_minor, 0) <> COALESCE(csv.expected_amount_minor, 0);

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

-- name: CreateTraderInboundRequisiteMismatchItems :many
WITH crm_requisites AS (
    SELECT
        sr.id AS shift_requisite_id,
        r.id AS requisite_id,
        r.phone,
        b.name AS bank_name,
        sr.inbound_turnover_minor AS actual_amount_minor,
        COALESCE(
            NULLIF(right(regexp_replace(r.phone, '[^0-9]', '', 'g'), 10), ''),
            'requisite:' || r.id::text
        ) AS match_key
    FROM shift_requisites sr
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN banks b ON b.code = r.bank_code
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    WHERE sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.shift_id = sqlc.arg(shift_id)
      AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction')
      AND ts.status IN ('open', 'closing')
),
csv_requisites AS (
    SELECT
        COALESCE(
            NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
            'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
        ) AS match_key,
        max(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''))) AS requisite_raw,
        COALESCE(sum(CASE WHEN normalized_status IN ('success', 'corrected') THEN amount_minor ELSE 0 END), 0)::bigint AS expected_amount_minor,
        count(*) FILTER (WHERE normalized_status IN ('success', 'corrected'))::bigint AS success_count
    FROM order_scope_items
    WHERE team_id = sqlc.arg(team_id)
      AND scope_type = 'trader_shift'
      AND shift_id = sqlc.arg(shift_id)
      AND direction = 'inbound'
      AND is_active = TRUE
    GROUP BY COALESCE(
        NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
        'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
    )
),
all_keys AS (
    SELECT match_key FROM crm_requisites
    UNION
    SELECT match_key FROM csv_requisites
)
INSERT INTO reconciliation_items (
    reconciliation_run_id,
    issue_type,
    teamlead_value_json,
    trader_value_json,
    message
)
SELECT
    sqlc.arg(run_id),
    'requisite_invoice_amount_mismatch',
    CASE
        WHEN csv.match_key IS NULL THEN NULL::jsonb
        ELSE jsonb_build_object(
            'requisite', csv.requisite_raw,
            'expectedAmountMinor', csv.expected_amount_minor,
            'successCount', csv.success_count
        )
    END,
    CASE
        WHEN crm.match_key IS NULL THEN NULL::jsonb
        ELSE jsonb_build_object(
            'shiftRequisiteId', crm.shift_requisite_id,
            'requisiteId', crm.requisite_id,
            'phone', crm.phone,
            'bankName', crm.bank_name,
            'actualAmountMinor', crm.actual_amount_minor
        )
    END,
    'Closed requisite turnover differs from invoice CSV amount'
FROM all_keys
LEFT JOIN crm_requisites crm ON crm.match_key = all_keys.match_key
LEFT JOIN csv_requisites csv ON csv.match_key = all_keys.match_key
WHERE COALESCE(crm.actual_amount_minor, 0) <> COALESCE(csv.expected_amount_minor, 0)
RETURNING id;

-- name: UpdateTraderInboundRequisiteReviewStatuses :exec
WITH csv_requisites AS (
    SELECT
        COALESCE(
            NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
            'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
        ) AS match_key,
        COALESCE(sum(CASE WHEN normalized_status IN ('success', 'corrected') THEN amount_minor ELSE 0 END), 0)::bigint AS expected_amount_minor
    FROM order_scope_items
    WHERE team_id = sqlc.arg(team_id)
      AND scope_type = 'trader_shift'
      AND shift_id = sqlc.arg(shift_id)
      AND direction = 'inbound'
      AND is_active = TRUE
    GROUP BY COALESCE(
        NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
        'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
    )
)
UPDATE shift_requisites sr
SET status = CASE
        WHEN sr.inbound_turnover_minor = COALESCE(csv_requisites.expected_amount_minor, 0) THEN 'worked_verified'
        ELSE 'worked_discrepancy'
    END,
    updated_at = now()
FROM requisites r
LEFT JOIN csv_requisites ON csv_requisites.match_key = COALESCE(
    NULLIF(right(regexp_replace(r.phone, '[^0-9]', '', 'g'), 10), ''),
    'requisite:' || r.id::text
)
WHERE sr.requisite_id = r.id
  AND sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction');

-- name: LatestTraderInboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
  AND type = 'trader_shift_inbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: LatestTraderInboundReconciliationRunByShift :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
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
    sqlc.arg(success_count),
    sqlc.arg(failed_amount_minor),
    sqlc.arg(failed_count),
    sqlc.arg(total_amount_minor),
    sqlc.arg(total_count),
    sqlc.arg(status)
)
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: CreateTraderOutboundReconciliationItems :many
WITH source_requisites AS (
    SELECT
        sr.id AS shift_requisite_id,
        sr.requisite_id,
        r.phone,
        b.name AS bank_name,
        COALESCE(sr.outbound_turnover_minor, 0)::bigint AS closed_outbound_turnover_minor,
        COALESCE(sum(mpt.amount_minor), 0)::bigint AS transfer_amount_minor
    FROM shift_requisites sr
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN banks b ON b.code = r.bank_code
    LEFT JOIN manual_payout_transfers mpt ON mpt.source_shift_requisite_id = sr.id
        AND mpt.team_id = sr.team_id
        AND mpt.trader_id = sr.trader_id
        AND mpt.shift_id = sr.shift_id
    WHERE sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.shift_id = sqlc.arg(shift_id)
      AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')
    GROUP BY sr.id, r.phone, b.name
),
source_requisite_items AS (
    SELECT
        'source_requisite_outbound_mismatch'::text AS issue_type,
        NULL::bigint AS external_order_id,
        NULL::text AS external_inner_id,
        NULL::jsonb AS teamlead_value_json,
        jsonb_build_object(
            'shiftRequisiteId', shift_requisite_id,
            'requisiteId', requisite_id,
            'requisitePhone', phone,
            'bankName', bank_name,
            'closedOutboundTurnoverMinor', closed_outbound_turnover_minor,
            'transferAmountMinor', transfer_amount_minor,
            'diffAmountMinor', closed_outbound_turnover_minor - transfer_amount_minor
        ) AS trader_value_json,
        'Closed requisite outbound turnover differs from payout transfers from this source requisite'::text AS message
    FROM source_requisites
    WHERE closed_outbound_turnover_minor <> transfer_amount_minor
),
csv_orders_raw AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_order_id,
        osi.external_inner_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'trader_shift'
      AND osi.shift_id = sqlc.arg(shift_id)
      AND osi.direction = 'outbound'
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
csv_orders AS (
    SELECT
        external_order_id,
        external_inner_id,
        amount_minor,
        normalized_status,
        row_number() OVER (PARTITION BY amount_minor ORDER BY external_inner_id) AS match_number
    FROM csv_orders_raw
    WHERE normalized_status IN ('success', 'corrected')
),
payout_totals AS (
    SELECT
        mpo.id,
        mpo.destination_bank,
        mpo.destination_requisite,
        mpo.amount_minor,
        COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor,
        row_number() OVER (PARTITION BY mpo.amount_minor ORDER BY mpo.created_at, mpo.id) AS match_number
    FROM manual_payout_orders mpo
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
    WHERE mpo.team_id = sqlc.arg(team_id)
      AND mpo.trader_id = sqlc.arg(trader_id)
      AND mpo.shift_id = sqlc.arg(shift_id)
      AND mpo.deleted_at IS NULL
      AND mpo.status <> 'cancelled'
    GROUP BY mpo.id
),
payout_order_items AS (
    SELECT
        CASE
            WHEN pt.id IS NULL THEN 'missing_manual_payout_order'
            WHEN csv.external_inner_id IS NULL THEN 'extra_manual_payout_order'
            WHEN pt.paid_amount_minor <> pt.amount_minor THEN 'manual_payout_not_fully_paid'
        END AS issue_type,
        csv.external_order_id,
        csv.external_inner_id,
        CASE
            WHEN csv.external_inner_id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'amountMinor', csv.amount_minor,
                'normalizedStatus', csv.normalized_status
            )
        END AS teamlead_value_json,
        CASE
            WHEN pt.id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'manualPayoutOrderId', pt.id,
                'destinationBank', pt.destination_bank,
                'destinationRequisite', pt.destination_requisite,
                'amountMinor', pt.amount_minor,
                'paidAmountMinor', pt.paid_amount_minor,
                'remainingAmountMinor', pt.amount_minor - pt.paid_amount_minor
            )
        END AS trader_value_json,
        CASE
            WHEN pt.id IS NULL THEN 'Payout is present in trader CSV but no manual payout order with the same amount was created'
            WHEN csv.external_inner_id IS NULL THEN 'Manual payout order has no matching successful payout amount in trader CSV'
            WHEN pt.paid_amount_minor <> pt.amount_minor THEN 'Manual payout order is not fully paid by transfers'
        END AS message
    FROM csv_orders csv
    FULL JOIN payout_totals pt ON pt.amount_minor = csv.amount_minor
        AND pt.match_number = csv.match_number
    WHERE pt.id IS NULL
       OR csv.external_inner_id IS NULL
       OR pt.paid_amount_minor <> pt.amount_minor
),
items AS (
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM source_requisite_items
    UNION ALL
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM payout_order_items
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

-- name: LatestTraderOutboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
  AND type = 'trader_shift_outbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: LatestTraderOutboundReconciliationRunByShift :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
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
    SELECT
        sr.id AS shift_requisite_id,
        sr.inbound_turnover_minor AS amount_minor
    FROM shift_requisites sr
    JOIN period p ON COALESCE(sr.released_at, sr.taken_at)::date BETWEEN p.date_from AND p.date_to
    WHERE sr.team_id = sqlc.arg(team_id)
      AND sr.status IN ('worked', 'blocked', 'correction')
)
SELECT
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_success_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders), 0)::bigint AS total_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders), 0)::bigint AS total_count,
    COALESCE((SELECT sum(amount_minor) FROM trader_orders), 0)::bigint AS actual_amount_minor,
    COALESCE((SELECT count(*) FROM trader_orders), 0)::bigint AS actual_success_count;

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
        osi.requisite_raw,
        osi.requisite_phone,
        osi.amount_minor,
        osi.normalized_status,
        osi.raw_status,
        osi.created_at_external
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'teamlead_period'
      AND osi.accounting_period_id = sqlc.arg(accounting_period_id)
      AND osi.direction = 'inbound'
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT
        sr.id AS shift_requisite_id,
        sr.requisite_id,
        r.phone AS requisite_phone,
        r.bank_code,
        sr.trader_id,
        u.login AS trader_login,
        sr.inbound_turnover_minor AS amount_minor,
        sr.status AS shift_requisite_status
    FROM shift_requisites sr
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN users u ON u.id = sr.trader_id
    JOIN period p ON COALESCE(sr.released_at, sr.taken_at)::date BETWEEN p.date_from AND p.date_to
    WHERE sr.team_id = sqlc.arg(team_id)
      AND sr.status IN ('worked', 'blocked', 'correction')
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
        'Teamlead period invoice success total differs from CRM requisite turnover'::text AS message
    FROM teamlead_success_total tl
    CROSS JOIN trader_success_total tr
    WHERE tl.amount_minor <> tr.amount_minor
       OR tl.count <> tr.count
),
teamlead_requisite_totals AS (
    SELECT
        COALESCE(requisite_phone, requisite_raw, '') AS requisite_key,
        max(requisite_phone) AS requisite_phone,
        COALESCE(sum(amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM teamlead_orders
    WHERE normalized_status IN ('success', 'corrected')
    GROUP BY COALESCE(requisite_phone, requisite_raw, '')
),
trader_requisite_totals AS (
    SELECT
        requisite_phone AS requisite_key,
        max(requisite_id) AS requisite_id,
        max(requisite_phone) AS requisite_phone,
        max(bank_code) AS bank_code,
        max(trader_id) AS trader_id,
        max(trader_login) AS trader_login,
        COALESCE(sum(amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM trader_orders
    GROUP BY requisite_phone
),
requisite_total_items AS (
    SELECT
        'requisite_amount_mismatch'::text AS issue_type,
        NULL::bigint AS external_order_id,
        NULL::text AS external_inner_id,
        CASE
            WHEN tl.requisite_key IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'requisitePhone', tl.requisite_phone,
                'successAmountMinor', tl.amount_minor,
                'successCount', tl.count
            )
        END AS teamlead_value_json,
        CASE
            WHEN tr.requisite_key IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'requisiteId', tr.requisite_id,
                'requisitePhone', tr.requisite_phone,
                'bankCode', tr.bank_code,
                'traderId', tr.trader_id,
                'traderLogin', tr.trader_login,
                'successAmountMinor', tr.amount_minor,
                'successCount', tr.count
            )
        END AS trader_value_json,
        'Requisite turnover differs between teamlead CSV transactions and CRM final turnover'::text AS message
    FROM teamlead_requisite_totals tl
    FULL JOIN trader_requisite_totals tr ON tr.requisite_key = tl.requisite_key
    WHERE COALESCE(tl.amount_minor, 0) <> COALESCE(tr.amount_minor, 0)
       OR COALESCE(tl.count, 0) <> COALESCE(tr.count, 0)
),
items AS (
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM total_items
    UNION ALL
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM requisite_total_items
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
  AND accounting_period_id IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: CalculateTeamleadPeriodOutboundSummary :one
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
      AND osi.direction = 'outbound'
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT
        mpo.id AS manual_payout_order_id,
        COALESCE(sum(mpt.amount_minor), 0)::bigint AS amount_minor
    FROM manual_payout_orders mpo
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
    JOIN trader_shifts ts ON ts.id = mpo.shift_id
    JOIN period p ON mpo.created_at::date BETWEEN p.date_from AND p.date_to
    WHERE mpo.team_id = sqlc.arg(team_id)
      AND mpo.deleted_at IS NULL
      AND mpo.status <> 'cancelled'
      AND ts.team_id = sqlc.arg(team_id)
    GROUP BY mpo.id
)
SELECT
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_success_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders), 0)::bigint AS total_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders), 0)::bigint AS total_count,
    COALESCE((SELECT sum(amount_minor) FROM trader_orders), 0)::bigint AS actual_amount_minor,
    COALESCE((SELECT count(*) FROM trader_orders), 0)::bigint AS actual_success_count;

-- name: CreateTeamleadPeriodOutboundReconciliationRun :one
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
    'teamlead_period_outbound',
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

-- name: CreateTeamleadPeriodOutboundReconciliationItems :many
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
        osi.requisite_raw,
        osi.requisite_phone,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'teamlead_period'
      AND osi.accounting_period_id = sqlc.arg(accounting_period_id)
      AND osi.direction = 'outbound'
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT
        mpo.id AS manual_payout_order_id,
        mpo.destination_bank,
        mpo.destination_requisite,
        mpo.amount_minor,
        COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor,
        mpo.trader_id,
        u.login AS trader_login,
        row_number() OVER (
            PARTITION BY mpo.amount_minor
            ORDER BY mpo.id
        ) AS match_number
    FROM manual_payout_orders mpo
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
    JOIN users u ON u.id = mpo.trader_id
    JOIN trader_shifts ts ON ts.id = mpo.shift_id
    JOIN period p ON mpo.created_at::date BETWEEN p.date_from AND p.date_to
    WHERE mpo.team_id = sqlc.arg(team_id)
      AND mpo.deleted_at IS NULL
      AND mpo.status <> 'cancelled'
      AND ts.team_id = sqlc.arg(team_id)
    GROUP BY mpo.id, u.login
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
        COALESCE(sum(paid_amount_minor), 0)::bigint AS amount_minor,
        count(*)::bigint AS count
    FROM trader_orders
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
        'Teamlead period payout success total differs from trader manual payout transfers'::text AS message
    FROM teamlead_success_total tl
    CROSS JOIN trader_success_total tr
    WHERE tl.amount_minor <> tr.amount_minor
       OR tl.count <> tr.count
),
teamlead_matchable AS (
    SELECT
        tl.*,
        COALESCE(tl.requisite_phone, tl.requisite_raw, '') AS destination_key,
        row_number() OVER (
            PARTITION BY tl.amount_minor
            ORDER BY tl.external_inner_id
        ) AS match_number
    FROM teamlead_orders tl
    WHERE tl.normalized_status IN ('success', 'corrected')
),
order_items AS (
    SELECT
        CASE
            WHEN tr.manual_payout_order_id IS NULL THEN 'missing_manual_payout_order'
            WHEN tl.external_inner_id IS NULL THEN 'extra_manual_payout_order'
            WHEN tr.paid_amount_minor <> tr.amount_minor THEN 'manual_payout_not_fully_paid'
        END AS issue_type,
        tl.external_order_id AS external_order_id,
        tl.external_inner_id AS external_inner_id,
        CASE
            WHEN tl.external_inner_id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'workerName', tl.worker_name,
                'traderId', tl.trader_id,
                'destination', tl.destination_key,
                'amountMinor', tl.amount_minor,
                'normalizedStatus', tl.normalized_status
            )
        END AS teamlead_value_json,
        CASE
            WHEN tr.manual_payout_order_id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'manualPayoutOrderId', tr.manual_payout_order_id,
                'destinationBank', tr.destination_bank,
                'destinationRequisite', tr.destination_requisite,
                'traderId', tr.trader_id,
                'traderLogin', tr.trader_login,
                'amountMinor', tr.amount_minor,
                'paidAmountMinor', tr.paid_amount_minor
            )
        END AS trader_value_json,
        CASE
            WHEN tr.manual_payout_order_id IS NULL THEN 'Payout is present in teamlead period CSV but no manual payout order with the same amount was created'
            WHEN tl.external_inner_id IS NULL THEN 'Manual payout order has no matching successful payout amount in teamlead period CSV'
            WHEN tr.paid_amount_minor <> tr.amount_minor THEN 'Manual payout order is not fully paid by transfers'
        END AS message
    FROM teamlead_matchable tl
    FULL JOIN trader_orders tr ON tr.amount_minor = tl.amount_minor
        AND tr.match_number = tl.match_number
    WHERE tr.manual_payout_order_id IS NULL
       OR tl.external_inner_id IS NULL
       OR tr.paid_amount_minor <> tr.amount_minor
),
items AS (
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM total_items
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

-- name: LatestTeamleadPeriodOutboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND accounting_period_id = sqlc.arg(accounting_period_id)
  AND type = 'teamlead_period_outbound'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: LatestTeamleadOutboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND type = 'teamlead_period_outbound'
  AND accounting_period_id IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListTeamleadCurrentReconciliationRuns :many
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE team_id = sqlc.arg(team_id)
  AND type = CASE
      WHEN sqlc.arg(direction)::text = 'inbound' THEN 'teamlead_period_inbound'
      ELSE 'teamlead_period_outbound'
  END
  AND accounting_period_id IS NULL
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- name: GetTeamleadCurrentReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
  AND type = CASE
      WHEN sqlc.arg(direction)::text = 'inbound' THEN 'teamlead_period_inbound'
      ELSE 'teamlead_period_outbound'
  END
  AND accounting_period_id IS NULL
LIMIT 1;

-- name: LatestActiveTeamleadCurrentImportBatch :one
SELECT id, team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, superseded_by_batch_id, error_message, created_at, applied_at
FROM import_batches
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL
  AND direction = sqlc.arg(direction)
  AND status IN ('applied', 'reconciled')
  AND superseded_by_batch_id IS NULL
ORDER BY applied_at DESC NULLS LAST, id DESC
LIMIT 1;

-- name: CalculateTeamleadCurrentSummary :one
WITH latest_import AS (
    SELECT ib.id
    FROM import_batches ib
    WHERE ib.team_id = sqlc.arg(team_id)
      AND ib.scope_type = 'teamlead_period'
      AND ib.accounting_period_id IS NULL
      AND ib.direction = sqlc.arg(direction)
      AND ib.status IN ('applied', 'reconciled')
      AND ib.superseded_by_batch_id IS NULL
    ORDER BY ib.applied_at DESC NULLS LAST, ib.id DESC
    LIMIT 1
),
teamlead_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_inner_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'teamlead_period'
      AND osi.accounting_period_id IS NULL
      AND osi.direction = sqlc.arg(direction)
      AND osi.is_active = TRUE
      AND osi.import_batch_id = (SELECT id FROM latest_import)
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_inner_id,
        osi.amount_minor,
        osi.normalized_status
    FROM order_scope_items osi
    JOIN teamlead_orders tl ON tl.external_inner_id = osi.external_inner_id
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'trader_shift'
      AND osi.direction = sqlc.arg(direction)
      AND osi.is_active = TRUE
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
)
SELECT
    (SELECT id FROM latest_import)::bigint AS import_batch_id,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS expected_success_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders WHERE normalized_status IN ('failed', 'cancelled')), 0)::bigint AS failed_count,
    COALESCE((SELECT sum(amount_minor) FROM teamlead_orders), 0)::bigint AS total_amount_minor,
    COALESCE((SELECT count(*) FROM teamlead_orders), 0)::bigint AS total_count,
    COALESCE((SELECT sum(amount_minor) FROM trader_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS actual_amount_minor,
    COALESCE((SELECT count(*) FROM trader_orders WHERE normalized_status IN ('success', 'corrected')), 0)::bigint AS actual_success_count;

-- name: CreateTeamleadCurrentReconciliationRun :one
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
    sqlc.arg(type),
    'teamlead_period',
    NULL,
    NULL,
    NULL,
    sqlc.arg(import_batch_id),
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

-- name: CreateTeamleadCurrentReconciliationItems :many
WITH teamlead_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_order_id,
        osi.external_inner_id,
        osi.worker_name,
        osi.trader_id,
        osi.requisite_raw,
        osi.requisite_phone,
        osi.amount_minor,
        osi.normalized_status,
        osi.raw_status,
        osi.created_at_external
    FROM order_scope_items osi
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'teamlead_period'
      AND osi.accounting_period_id IS NULL
      AND osi.direction = sqlc.arg(direction)
      AND osi.is_active = TRUE
      AND osi.import_batch_id = sqlc.arg(import_batch_id)
    ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC
),
trader_orders AS (
    SELECT DISTINCT ON (osi.external_inner_id)
        osi.external_order_id,
        osi.external_inner_id,
        osi.worker_name,
        osi.trader_id,
        u.login AS trader_login,
        osi.requisite_raw,
        osi.requisite_phone,
        osi.amount_minor,
        osi.normalized_status,
        osi.raw_status,
        osi.created_at_external
    FROM order_scope_items osi
    LEFT JOIN users u ON u.id = osi.trader_id
    JOIN teamlead_orders tl ON tl.external_inner_id = osi.external_inner_id
    WHERE osi.team_id = sqlc.arg(team_id)
      AND osi.scope_type = 'trader_shift'
      AND osi.direction = sqlc.arg(direction)
      AND osi.is_active = TRUE
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
        'Teamlead CSV total differs from existing CRM order snapshots'::text AS message
    FROM teamlead_success_total tl
    CROSS JOIN trader_success_total tr
    WHERE tl.amount_minor <> tr.amount_minor
       OR tl.count <> tr.count
),
order_items AS (
    SELECT
        CASE
            WHEN tr.external_inner_id IS NULL THEN 'missing_in_trader_import'
            WHEN tl.amount_minor <> tr.amount_minor THEN 'amount_mismatch'
            WHEN tl.normalized_status <> tr.normalized_status THEN 'status_mismatch'
            WHEN COALESCE(tl.worker_name, '') <> COALESCE(tr.worker_name, '') THEN 'worker_mismatch'
        END AS issue_type,
        tl.external_order_id,
        tl.external_inner_id,
        jsonb_build_object(
            'workerName', tl.worker_name,
            'traderId', tl.trader_id,
            'requisitePhone', tl.requisite_phone,
            'requisite', tl.requisite_raw,
            'amountMinor', tl.amount_minor,
            'rawStatus', tl.raw_status,
            'normalizedStatus', tl.normalized_status,
            'createdAtExternal', tl.created_at_external
        ) AS teamlead_value_json,
        CASE
            WHEN tr.external_inner_id IS NULL THEN NULL::jsonb
            ELSE jsonb_build_object(
                'workerName', tr.worker_name,
                'traderId', tr.trader_id,
                'traderLogin', tr.trader_login,
                'requisitePhone', tr.requisite_phone,
                'requisite', tr.requisite_raw,
                'amountMinor', tr.amount_minor,
                'rawStatus', tr.raw_status,
                'normalizedStatus', tr.normalized_status,
                'createdAtExternal', tr.created_at_external
            )
        END AS trader_value_json,
        CASE
            WHEN tr.external_inner_id IS NULL THEN 'Order from teamlead CSV was not found in trader imports'
            WHEN tl.amount_minor <> tr.amount_minor THEN 'Order amount changed in teamlead CSV'
            WHEN tl.normalized_status <> tr.normalized_status THEN 'Order status changed in teamlead CSV'
            WHEN COALESCE(tl.worker_name, '') <> COALESCE(tr.worker_name, '') THEN 'Order worker changed in teamlead CSV'
        END AS message
    FROM teamlead_orders tl
    LEFT JOIN trader_orders tr ON tr.external_inner_id = tl.external_inner_id
    WHERE tr.external_inner_id IS NULL
       OR tl.amount_minor <> tr.amount_minor
       OR tl.normalized_status <> tr.normalized_status
       OR COALESCE(tl.worker_name, '') <> COALESCE(tr.worker_name, '')
),
items AS (
    SELECT issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message
    FROM total_items
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

-- name: AcceptTeamleadCurrentReconciliationRun :one
UPDATE reconciliation_runs
SET status = 'accepted_with_comment',
    comment = sqlc.arg(comment),
    confirmed_by = sqlc.arg(confirmed_by),
    confirmed_at = now()
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
  AND type IN ('teamlead_period_inbound', 'teamlead_period_outbound')
  AND accounting_period_id IS NULL
  AND status = 'mismatch'
  AND btrim(sqlc.arg(comment)) <> ''
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

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

-- name: ListActiveTeamleadOutboundPeriodScopes :many
SELECT DISTINCT ON (ib.accounting_period_id)
    ib.accounting_period_id,
    ib.id AS import_batch_id
FROM import_batches ib
WHERE ib.team_id = sqlc.arg(team_id)
  AND ib.scope_type = 'teamlead_period'
  AND ib.direction = 'outbound'
  AND ib.status IN ('applied', 'reconciled')
  AND ib.accounting_period_id IS NOT NULL
  AND ib.superseded_by_batch_id IS NULL
ORDER BY ib.accounting_period_id, ib.applied_at DESC NULLS LAST, ib.id DESC;
