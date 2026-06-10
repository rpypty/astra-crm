-- name: GetCurrentShiftIDForReadModel :one
SELECT id
FROM trader_shifts
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND status IN ('open', 'closing')
ORDER BY started_at DESC, id DESC
LIMIT 1;

-- name: ListTraderOrders :many
SELECT
    osi.id AS scope_item_id,
    osi.external_order_id,
    osi.external_id,
    osi.external_inner_id,
    osi.worker_name,
    osi.trader_id,
    u.login AS trader_login,
    osi.requisite_raw,
    osi.requisite_phone,
    osi.method_type,
    osi.method_name,
    osi.amount_minor,
    osi.currency,
    osi.raw_status,
    osi.normalized_status,
    osi.created_at_external,
    osi.import_batch_id,
    count(*) OVER()::bigint AS total_count
FROM order_scope_items osi
LEFT JOIN users u ON u.id = osi.trader_id
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'trader_shift'
  AND osi.shift_id = sqlc.arg(shift_id)
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND (sqlc.narg(date_from)::date IS NULL OR osi.created_at_external::date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR osi.created_at_external::date <= sqlc.narg(date_to)::date)
  AND (sqlc.narg(worker_name)::text IS NULL OR osi.worker_name ILIKE '%' || sqlc.narg(worker_name)::text || '%')
  AND (
      sqlc.narg(requisite)::text IS NULL
      OR osi.requisite_raw ILIKE '%' || sqlc.narg(requisite)::text || '%'
      OR osi.requisite_phone ILIKE '%' || sqlc.narg(requisite)::text || '%'
  )
  AND (sqlc.narg(method_type)::text IS NULL OR osi.method_type = sqlc.narg(method_type)::text)
  AND (
      sqlc.narg(status)::text IS NULL
      OR osi.raw_status = sqlc.narg(status)::text
      OR osi.normalized_status = sqlc.narg(status)::text
  )
  AND (sqlc.narg(amount_from)::bigint IS NULL OR osi.amount_minor >= sqlc.narg(amount_from)::bigint)
  AND (sqlc.narg(amount_to)::bigint IS NULL OR osi.amount_minor <= sqlc.narg(amount_to)::bigint)
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'amount_asc' THEN osi.amount_minor END ASC,
    CASE WHEN sqlc.arg(sort)::text = 'amount_desc' THEN osi.amount_minor END DESC,
    CASE WHEN sqlc.arg(sort)::text = 'created_at_asc' THEN osi.created_at_external END ASC,
    osi.created_at_external DESC,
    osi.id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: ListTeamleadOrders :many
SELECT
    osi.id AS scope_item_id,
    osi.external_order_id,
    osi.external_id,
    osi.external_inner_id,
    osi.worker_name,
    osi.trader_id,
    u.login AS trader_login,
    osi.requisite_raw,
    osi.requisite_phone,
    osi.method_type,
    osi.method_name,
    osi.amount_minor,
    osi.currency,
    osi.raw_status,
    osi.normalized_status,
    osi.created_at_external,
    osi.import_batch_id,
    count(*) OVER()::bigint AS total_count
FROM order_scope_items osi
LEFT JOIN users u ON u.id = osi.trader_id
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'teamlead_period'
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND (sqlc.narg(date_from)::date IS NULL OR osi.created_at_external::date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR osi.created_at_external::date <= sqlc.narg(date_to)::date)
  AND (sqlc.narg(trader_id)::bigint IS NULL OR osi.trader_id = sqlc.narg(trader_id)::bigint)
  AND (sqlc.narg(worker_name)::text IS NULL OR osi.worker_name ILIKE '%' || sqlc.narg(worker_name)::text || '%')
  AND (
      sqlc.narg(requisite)::text IS NULL
      OR osi.requisite_raw ILIKE '%' || sqlc.narg(requisite)::text || '%'
      OR osi.requisite_phone ILIKE '%' || sqlc.narg(requisite)::text || '%'
  )
  AND (sqlc.narg(method_type)::text IS NULL OR osi.method_type = sqlc.narg(method_type)::text)
  AND (
      sqlc.narg(status)::text IS NULL
      OR osi.raw_status = sqlc.narg(status)::text
      OR osi.normalized_status = sqlc.narg(status)::text
  )
  AND (sqlc.narg(amount_from)::bigint IS NULL OR osi.amount_minor >= sqlc.narg(amount_from)::bigint)
  AND (sqlc.narg(amount_to)::bigint IS NULL OR osi.amount_minor <= sqlc.narg(amount_to)::bigint)
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'amount_asc' THEN osi.amount_minor END ASC,
    CASE WHEN sqlc.arg(sort)::text = 'amount_desc' THEN osi.amount_minor END DESC,
    CASE WHEN sqlc.arg(sort)::text = 'created_at_asc' THEN osi.created_at_external END ASC,
    osi.created_at_external DESC,
    osi.id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: TraderOrdersSummary :one
SELECT
    COALESCE(sum(osi.amount_minor), 0)::bigint AS total_amount_minor,
    count(*)::bigint AS total_count,
    COALESCE(sum(CASE WHEN osi.normalized_status IN ('success', 'corrected') THEN osi.amount_minor ELSE 0 END), 0)::bigint AS success_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status IN ('success', 'corrected'))::bigint AS success_count,
    COALESCE(sum(CASE WHEN osi.normalized_status IN ('failed', 'cancelled') THEN osi.amount_minor ELSE 0 END), 0)::bigint AS failed_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status IN ('failed', 'cancelled'))::bigint AS failed_count,
    COALESCE(sum(CASE WHEN osi.normalized_status = 'unknown' THEN osi.amount_minor ELSE 0 END), 0)::bigint AS unknown_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status = 'unknown')::bigint AS unknown_count
FROM order_scope_items osi
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'trader_shift'
  AND osi.shift_id = sqlc.arg(shift_id)
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND (sqlc.narg(date_from)::date IS NULL OR osi.created_at_external::date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR osi.created_at_external::date <= sqlc.narg(date_to)::date);

-- name: TeamleadOrdersSummary :one
SELECT
    COALESCE(sum(osi.amount_minor), 0)::bigint AS total_amount_minor,
    count(*)::bigint AS total_count,
    COALESCE(sum(CASE WHEN osi.normalized_status IN ('success', 'corrected') THEN osi.amount_minor ELSE 0 END), 0)::bigint AS success_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status IN ('success', 'corrected'))::bigint AS success_count,
    COALESCE(sum(CASE WHEN osi.normalized_status IN ('failed', 'cancelled') THEN osi.amount_minor ELSE 0 END), 0)::bigint AS failed_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status IN ('failed', 'cancelled'))::bigint AS failed_count,
    COALESCE(sum(CASE WHEN osi.normalized_status = 'unknown' THEN osi.amount_minor ELSE 0 END), 0)::bigint AS unknown_amount_minor,
    count(*) FILTER (WHERE osi.normalized_status = 'unknown')::bigint AS unknown_count
FROM order_scope_items osi
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'teamlead_period'
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND (sqlc.narg(date_from)::date IS NULL OR osi.created_at_external::date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR osi.created_at_external::date <= sqlc.narg(date_to)::date)
  AND (sqlc.narg(trader_id)::bigint IS NULL OR osi.trader_id = sqlc.narg(trader_id)::bigint);

-- name: TraderStatusBreakdown :many
SELECT
    osi.raw_status,
    osi.normalized_status,
    COALESCE(sum(osi.amount_minor), 0)::bigint AS amount_minor,
    count(*)::bigint AS count
FROM order_scope_items osi
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'trader_shift'
  AND osi.shift_id = sqlc.arg(shift_id)
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND (sqlc.narg(date_from)::date IS NULL OR osi.created_at_external::date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR osi.created_at_external::date <= sqlc.narg(date_to)::date)
GROUP BY osi.raw_status, osi.normalized_status
ORDER BY count DESC, amount_minor DESC, osi.raw_status;

-- name: TeamleadStatusBreakdown :many
SELECT
    osi.raw_status,
    osi.normalized_status,
    COALESCE(sum(osi.amount_minor), 0)::bigint AS amount_minor,
    count(*)::bigint AS count
FROM order_scope_items osi
WHERE osi.team_id = sqlc.arg(team_id)
  AND osi.scope_type = 'teamlead_period'
  AND osi.direction = sqlc.arg(direction)
  AND osi.is_active = TRUE
  AND (sqlc.narg(date_from)::date IS NULL OR osi.created_at_external::date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR osi.created_at_external::date <= sqlc.narg(date_to)::date)
  AND (sqlc.narg(trader_id)::bigint IS NULL OR osi.trader_id = sqlc.narg(trader_id)::bigint)
GROUP BY osi.raw_status, osi.normalized_status
ORDER BY count DESC, amount_minor DESC, osi.raw_status;

-- name: TraderRecentImports :many
SELECT id, team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, superseded_by_batch_id, error_message, created_at, applied_at
FROM import_batches
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'trader_shift'
  AND shift_id = sqlc.arg(shift_id)
  AND direction = sqlc.arg(direction)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- name: TeamleadRecentImports :many
SELECT id, team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, superseded_by_batch_id, error_message, created_at, applied_at
FROM import_batches
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND direction = sqlc.arg(direction)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count);
