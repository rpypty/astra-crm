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

-- name: ListTraderInboundReconciliationRequisites :many
SELECT
    sr.id AS shift_requisite_id,
    sr.requisite_id,
    r.phone,
    r.bank_code,
    sr.inbound_turnover_minor AS actual_amount_minor
FROM shift_requisites sr
JOIN requisites r ON r.id = sr.requisite_id
JOIN trader_shifts ts ON ts.id = sr.shift_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction')
  AND ts.status IN ('open', 'closing')
ORDER BY sr.id;

-- name: ListTraderInboundReconciliationReviewRequisites :many
SELECT
    sr.id AS shift_requisite_id,
    sr.requisite_id,
    r.phone,
    sr.inbound_turnover_minor AS actual_amount_minor
FROM shift_requisites sr
JOIN requisites r ON r.id = sr.requisite_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction')
ORDER BY sr.id;

-- name: ListTraderInboundReconciliationScopeItems :many
SELECT
    COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), '')::text AS csv_requisite,
    amount_minor,
    normalized_status
FROM order_scope_items
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'trader_shift'
  AND shift_id = sqlc.arg(shift_id)
  AND direction = 'inbound'
  AND is_active = TRUE
ORDER BY id;

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

-- name: UpdateShiftRequisiteInboundReviewStatus :exec
UPDATE shift_requisites
SET status = sqlc.arg(status),
    updated_at = now()
WHERE id = sqlc.arg(shift_requisite_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
  AND status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction');

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

-- name: GetTraderInboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND type = 'trader_shift_inbound';

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

-- name: ListTraderOutboundReconciliationSourceRequisites :many
SELECT
    sr.id AS shift_requisite_id,
    sr.requisite_id,
    r.phone,
    r.bank_code,
    COALESCE(sr.outbound_turnover_minor, 0)::bigint AS closed_outbound_turnover_minor
FROM shift_requisites sr
JOIN requisites r ON r.id = sr.requisite_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')
ORDER BY sr.id;

-- name: ListTraderOutboundReconciliationScopeItems :many
SELECT
    id,
    external_order_id,
    external_inner_id,
    amount_minor,
    normalized_status,
    created_at
FROM order_scope_items
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'trader_shift'
  AND shift_id = sqlc.arg(shift_id)
  AND direction = 'outbound'
  AND is_active = TRUE
ORDER BY external_inner_id, created_at DESC, id DESC;

-- name: ListTraderOutboundReconciliationPayoutOrders :many
SELECT
    id,
    destination_bank,
    destination_requisite,
    amount_minor,
    created_at
FROM manual_payout_orders
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
  AND deleted_at IS NULL
  AND status <> 'cancelled'
ORDER BY amount_minor, created_at, id;

-- name: ListTraderOutboundReconciliationTransfers :many
SELECT
    manual_payout_order_id,
    source_shift_requisite_id,
    amount_minor
FROM manual_payout_transfers
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND shift_id = sqlc.arg(shift_id)
ORDER BY id;

-- name: CreateReconciliationItem :one
INSERT INTO reconciliation_items (
    reconciliation_run_id,
    issue_type,
    external_order_id,
    external_inner_id,
    teamlead_value_json,
    trader_value_json,
    message
)
VALUES (
    sqlc.arg(run_id),
    sqlc.arg(issue_type),
    sqlc.narg(external_order_id),
    sqlc.narg(external_inner_id),
    sqlc.narg(teamlead_value_json),
    sqlc.narg(trader_value_json),
    sqlc.narg(message)
)
RETURNING id, reconciliation_run_id, issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message, created_at;

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

-- name: GetTraderOutboundReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE id = sqlc.arg(run_id)
  AND team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND type = 'trader_shift_outbound';

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

-- name: GetReconciliationAccountingPeriod :one
SELECT date_from, date_to
FROM accounting_periods
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(accounting_period_id);

-- name: ListTeamleadPeriodReconciliationScopeItems :many
SELECT
    id,
    external_order_id,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    amount_minor,
    normalized_status,
    raw_status,
    created_at_external,
    created_at
FROM order_scope_items
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND accounting_period_id = sqlc.arg(accounting_period_id)
  AND direction = sqlc.arg(direction)
  AND is_active = TRUE
ORDER BY external_inner_id, created_at DESC, id DESC;

-- name: ListTeamleadPeriodInboundShiftRequisites :many
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
WHERE sr.team_id = sqlc.arg(team_id)
  AND COALESCE(sr.released_at, sr.taken_at) >= sqlc.arg(date_from)::timestamptz
  AND COALESCE(sr.released_at, sr.taken_at) < sqlc.arg(date_to_exclusive)::timestamptz
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked');

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
WHERE reconciliation_runs.team_id = sqlc.arg(team_id)
  AND reconciliation_runs.type = 'teamlead_period_inbound'
  AND reconciliation_runs.accounting_period_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM import_batches ib
      WHERE ib.id = reconciliation_runs.import_batch_id
        AND ib.uploaded_by = sqlc.arg(uploaded_by)
  )
ORDER BY reconciliation_runs.created_at DESC, reconciliation_runs.id DESC
LIMIT 1;

-- name: ListTeamleadPeriodPayoutOrders :many
SELECT
    mpo.id,
    mpo.destination_bank,
    mpo.destination_requisite,
    mpo.amount_minor,
    mpo.trader_id,
    u.login AS trader_login,
    mpo.created_at
FROM manual_payout_orders mpo
JOIN users u ON u.id = mpo.trader_id
JOIN trader_shifts ts ON ts.id = mpo.shift_id
WHERE mpo.team_id = sqlc.arg(team_id)
  AND mpo.created_at >= sqlc.arg(date_from)::timestamptz
  AND mpo.created_at < sqlc.arg(date_to_exclusive)::timestamptz
  AND mpo.deleted_at IS NULL
  AND mpo.status <> 'cancelled'
  AND ts.team_id = sqlc.arg(team_id)
ORDER BY mpo.amount_minor, mpo.id;

-- name: ListTeamleadPeriodPayoutTransfers :many
SELECT
    mpt.manual_payout_order_id,
    mpt.amount_minor
FROM manual_payout_transfers mpt
JOIN manual_payout_orders mpo ON mpo.id = mpt.manual_payout_order_id
JOIN trader_shifts ts ON ts.id = mpo.shift_id
WHERE mpo.team_id = sqlc.arg(team_id)
  AND mpo.created_at >= sqlc.arg(date_from)::timestamptz
  AND mpo.created_at < sqlc.arg(date_to_exclusive)::timestamptz
  AND mpo.deleted_at IS NULL
  AND mpo.status <> 'cancelled'
  AND ts.team_id = sqlc.arg(team_id)
ORDER BY mpt.id;

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
WHERE reconciliation_runs.team_id = sqlc.arg(team_id)
  AND reconciliation_runs.type = 'teamlead_period_outbound'
  AND reconciliation_runs.accounting_period_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM import_batches ib
      WHERE ib.id = reconciliation_runs.import_batch_id
        AND ib.uploaded_by = sqlc.arg(uploaded_by)
  )
ORDER BY reconciliation_runs.created_at DESC, reconciliation_runs.id DESC
LIMIT 1;

-- name: ListTeamleadCurrentReconciliationRuns :many
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE reconciliation_runs.team_id = sqlc.arg(team_id)
  AND reconciliation_runs.type = CASE
      WHEN sqlc.arg(direction)::text = 'inbound' THEN 'teamlead_period_inbound'
      ELSE 'teamlead_period_outbound'
  END
  AND reconciliation_runs.accounting_period_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM import_batches ib
      WHERE ib.id = reconciliation_runs.import_batch_id
        AND ib.uploaded_by = sqlc.arg(uploaded_by)
  )
ORDER BY reconciliation_runs.created_at DESC, reconciliation_runs.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountTeamleadCurrentReconciliationRuns :one
SELECT count(*)::bigint
FROM reconciliation_runs
WHERE reconciliation_runs.team_id = sqlc.arg(team_id)
  AND reconciliation_runs.type = CASE
      WHEN sqlc.arg(direction)::text = 'inbound' THEN 'teamlead_period_inbound'
      ELSE 'teamlead_period_outbound'
  END
  AND reconciliation_runs.accounting_period_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM import_batches ib
      WHERE ib.id = reconciliation_runs.import_batch_id
        AND ib.uploaded_by = sqlc.arg(uploaded_by)
  );

-- name: GetTeamleadCurrentReconciliationRun :one
SELECT id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at
FROM reconciliation_runs
WHERE reconciliation_runs.id = sqlc.arg(run_id)
  AND reconciliation_runs.team_id = sqlc.arg(team_id)
  AND reconciliation_runs.type = CASE
      WHEN sqlc.arg(direction)::text = 'inbound' THEN 'teamlead_period_inbound'
      ELSE 'teamlead_period_outbound'
  END
  AND reconciliation_runs.accounting_period_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM import_batches ib
      WHERE ib.id = reconciliation_runs.import_batch_id
        AND ib.uploaded_by = sqlc.arg(uploaded_by)
  )
LIMIT 1;

-- name: LatestActiveTeamleadCurrentImportBatch :one
SELECT id, team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, superseded_by_batch_id, error_message, created_at, applied_at
FROM import_batches
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL
  AND direction = sqlc.arg(direction)
  AND uploaded_by = sqlc.arg(uploaded_by)
  AND status IN ('applied', 'reconciled')
  AND superseded_by_batch_id IS NULL
ORDER BY applied_at DESC NULLS LAST, id DESC
LIMIT 1;

-- name: ListTeamleadCurrentReconciliationScopeItems :many
SELECT
    id,
    external_order_id,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    amount_minor,
    normalized_status,
    raw_status,
    created_at_external,
    created_at
FROM order_scope_items
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL
  AND direction = sqlc.arg(direction)
  AND is_active = TRUE
  AND import_batch_id = sqlc.arg(import_batch_id)
ORDER BY external_inner_id, created_at DESC, id DESC;

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

-- name: ListTraderReconciliationScopeItemsByInnerIDs :many
SELECT
    osi.id,
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
    osi.created_at_external,
    osi.created_at
FROM order_scope_items osi
LEFT JOIN users u ON u.id = osi.trader_id
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'trader_shift'
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND osi.external_inner_id = ANY(sqlc.arg(inner_ids)::text[])
ORDER BY osi.external_inner_id, osi.created_at DESC, osi.id DESC;

-- name: AcceptTeamleadCurrentReconciliationRun :one
UPDATE reconciliation_runs
SET status = 'accepted_with_comment',
    comment = sqlc.arg(comment),
    confirmed_by = sqlc.arg(confirmed_by),
    confirmed_at = now()
WHERE reconciliation_runs.id = sqlc.arg(run_id)
  AND reconciliation_runs.team_id = sqlc.arg(team_id)
  AND reconciliation_runs.type IN ('teamlead_period_inbound', 'teamlead_period_outbound')
  AND reconciliation_runs.accounting_period_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM import_batches ib
      WHERE ib.id = reconciliation_runs.import_batch_id
        AND ib.uploaded_by = sqlc.arg(actor_id)
  )
  AND reconciliation_runs.status = 'mismatch'
  AND btrim(sqlc.arg(comment)) <> ''
RETURNING id, team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id, expected_amount_minor, actual_amount_minor, diff_amount_minor, success_amount_minor, success_count, failed_amount_minor, failed_count, total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at, created_at;

-- name: ListReconciliationItemsForRun :many
SELECT id, reconciliation_run_id, issue_type, external_order_id, external_inner_id, teamlead_value_json, trader_value_json, message, created_at
FROM reconciliation_items
WHERE reconciliation_run_id = sqlc.arg(run_id)
  AND (
      (NOT sqlc.arg(only_mismatch)::boolean AND sqlc.arg(status)::text IN ('', 'all'))
      OR issue_type IS NOT NULL
  )
ORDER BY id
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountReconciliationItemsForRun :one
SELECT count(*)::bigint
FROM reconciliation_items
WHERE reconciliation_run_id = sqlc.arg(run_id)
  AND (
      (NOT sqlc.arg(only_mismatch)::boolean AND sqlc.arg(status)::text IN ('', 'all'))
      OR issue_type IS NOT NULL
  );

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
