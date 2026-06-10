-- name: CreateImportBatch :one
INSERT INTO import_batches (
    team_id,
    uploaded_by,
    scope_type,
    direction,
    shift_id,
    accounting_period_id,
    trader_id,
    file_name,
    file_hash,
    rows_count,
    status
)
VALUES (
    sqlc.arg(team_id),
    sqlc.arg(uploaded_by),
    sqlc.arg(scope_type),
    sqlc.arg(direction),
    sqlc.narg(shift_id),
    sqlc.narg(accounting_period_id),
    sqlc.narg(trader_id),
    sqlc.arg(file_name),
    sqlc.arg(file_hash),
    sqlc.arg(rows_count),
    'parsed'
)
RETURNING id, team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, superseded_by_batch_id, error_message, created_at, applied_at;

-- name: InsertImportRow :one
INSERT INTO import_rows (
    import_batch_id,
    row_number,
    external_id,
    external_inner_id,
    raw_payload_json,
    parse_status,
    parse_error
)
VALUES (
    sqlc.arg(import_batch_id),
    sqlc.arg(row_number),
    sqlc.narg(external_id),
    sqlc.narg(external_inner_id),
    sqlc.arg(raw_payload_json),
    'parsed',
    NULL
)
RETURNING id, import_batch_id, row_number, external_id, external_inner_id, raw_payload_json, parse_status, parse_error, created_at;

-- name: UpsertExternalOrder :one
INSERT INTO external_orders (
    team_id,
    direction,
    external_id,
    external_inner_id,
    external_foreign_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    requisite_external_id,
    requisite_id,
    device_name,
    method_type,
    method_name,
    amount_minor,
    currency,
    course,
    course_worker,
    worker_amount,
    worker_profit,
    raw_status,
    normalized_status,
    created_at_external,
    closed_at_external,
    updated_at_external,
    old_amount_minor,
    had_dispute,
    receipt,
    order_comment,
    ordered,
    counted,
    initials,
    last_import_batch_id
)
VALUES (
    sqlc.arg(team_id),
    sqlc.arg(direction),
    sqlc.arg(external_id),
    sqlc.arg(external_inner_id),
    sqlc.narg(external_foreign_id),
    sqlc.arg(worker_name),
    sqlc.narg(trader_id),
    sqlc.narg(requisite_raw),
    sqlc.narg(requisite_phone),
    sqlc.narg(requisite_external_id),
    sqlc.narg(requisite_id),
    sqlc.narg(device_name),
    sqlc.narg(method_type),
    sqlc.narg(method_name),
    sqlc.arg(amount_minor),
    sqlc.arg(currency),
    sqlc.narg(course)::numeric,
    sqlc.narg(course_worker)::numeric,
    sqlc.narg(worker_amount)::numeric,
    sqlc.narg(worker_profit)::numeric,
    sqlc.arg(raw_status),
    sqlc.arg(normalized_status),
    sqlc.arg(created_at_external),
    sqlc.narg(closed_at_external),
    sqlc.narg(updated_at_external),
    sqlc.narg(old_amount_minor),
    sqlc.narg(had_dispute),
    sqlc.narg(receipt),
    sqlc.narg(order_comment),
    sqlc.narg(ordered),
    sqlc.narg(counted),
    sqlc.narg(initials),
    sqlc.arg(last_import_batch_id)
)
ON CONFLICT (team_id, direction, external_inner_id)
DO UPDATE SET
    external_id = EXCLUDED.external_id,
    external_foreign_id = EXCLUDED.external_foreign_id,
    worker_name = EXCLUDED.worker_name,
    trader_id = EXCLUDED.trader_id,
    requisite_raw = EXCLUDED.requisite_raw,
    requisite_phone = EXCLUDED.requisite_phone,
    requisite_external_id = EXCLUDED.requisite_external_id,
    requisite_id = EXCLUDED.requisite_id,
    device_name = EXCLUDED.device_name,
    method_type = EXCLUDED.method_type,
    method_name = EXCLUDED.method_name,
    amount_minor = EXCLUDED.amount_minor,
    currency = EXCLUDED.currency,
    course = EXCLUDED.course,
    course_worker = EXCLUDED.course_worker,
    worker_amount = EXCLUDED.worker_amount,
    worker_profit = EXCLUDED.worker_profit,
    raw_status = EXCLUDED.raw_status,
    normalized_status = EXCLUDED.normalized_status,
    created_at_external = EXCLUDED.created_at_external,
    closed_at_external = EXCLUDED.closed_at_external,
    updated_at_external = EXCLUDED.updated_at_external,
    old_amount_minor = EXCLUDED.old_amount_minor,
    had_dispute = EXCLUDED.had_dispute,
    receipt = EXCLUDED.receipt,
    order_comment = EXCLUDED.order_comment,
    ordered = EXCLUDED.ordered,
    counted = EXCLUDED.counted,
    initials = EXCLUDED.initials,
    last_import_batch_id = EXCLUDED.last_import_batch_id,
    updated_at = now()
RETURNING id, (xmax = 0) AS inserted;

-- name: DeactivateTraderShiftScopeItems :many
UPDATE order_scope_items
SET is_active = FALSE,
    deactivated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'trader_shift'
  AND shift_id = sqlc.arg(shift_id)
  AND direction = sqlc.arg(direction)
  AND is_active = TRUE
RETURNING id;

-- name: DeactivateTeamleadPeriodScopeItems :many
UPDATE order_scope_items
SET is_active = FALSE,
    deactivated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND accounting_period_id = sqlc.arg(accounting_period_id)
  AND direction = sqlc.arg(direction)
  AND is_active = TRUE
RETURNING id;

-- name: SupersedeTraderShiftImportBatches :many
UPDATE import_batches
SET status = 'superseded',
    superseded_by_batch_id = sqlc.arg(new_batch_id)
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'trader_shift'
  AND shift_id = sqlc.arg(shift_id)
  AND direction = sqlc.arg(direction)
  AND id <> sqlc.arg(new_batch_id)
  AND superseded_by_batch_id IS NULL
  AND status IN ('parsed', 'applied', 'reconciled')
RETURNING id;

-- name: SupersedeTeamleadPeriodImportBatches :many
UPDATE import_batches
SET status = 'superseded',
    superseded_by_batch_id = sqlc.arg(new_batch_id)
WHERE team_id = sqlc.arg(team_id)
  AND scope_type = 'teamlead_period'
  AND accounting_period_id = sqlc.arg(accounting_period_id)
  AND direction = sqlc.arg(direction)
  AND id <> sqlc.arg(new_batch_id)
  AND superseded_by_batch_id IS NULL
  AND status IN ('parsed', 'applied', 'reconciled')
RETURNING id;

-- name: CreateTraderShiftScopeItem :one
INSERT INTO order_scope_items (
    team_id,
    scope_type,
    direction,
    shift_id,
    accounting_period_id,
    import_batch_id,
    import_row_id,
    external_order_id,
    external_id,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    method_type,
    method_name,
    amount_minor,
    currency,
    raw_status,
    normalized_status,
    created_at_external,
    is_active
)
SELECT
    sqlc.arg(team_id),
    'trader_shift',
    sqlc.arg(direction),
    sqlc.arg(shift_id),
    NULL,
    sqlc.arg(import_batch_id),
    sqlc.arg(import_row_id),
    sqlc.arg(external_order_id),
    eo.external_id,
    eo.external_inner_id,
    eo.worker_name,
    eo.trader_id,
    eo.requisite_raw,
    eo.requisite_phone,
    eo.method_type,
    eo.method_name,
    eo.amount_minor,
    eo.currency,
    eo.raw_status,
    eo.normalized_status,
    eo.created_at_external,
    TRUE
FROM external_orders eo
WHERE eo.id = sqlc.arg(external_order_id)
RETURNING id, team_id, scope_type, direction, shift_id, accounting_period_id, import_batch_id, import_row_id, external_order_id, external_id, external_inner_id, worker_name, trader_id, requisite_raw, requisite_phone, method_type, method_name, amount_minor, currency, raw_status, normalized_status, created_at_external, is_active, created_at, deactivated_at;

-- name: CreateTeamleadPeriodScopeItem :one
INSERT INTO order_scope_items (
    team_id,
    scope_type,
    direction,
    shift_id,
    accounting_period_id,
    import_batch_id,
    import_row_id,
    external_order_id,
    external_id,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_raw,
    requisite_phone,
    method_type,
    method_name,
    amount_minor,
    currency,
    raw_status,
    normalized_status,
    created_at_external,
    is_active
)
SELECT
    sqlc.arg(team_id),
    'teamlead_period',
    sqlc.arg(direction),
    NULL,
    sqlc.arg(accounting_period_id),
    sqlc.arg(import_batch_id),
    sqlc.arg(import_row_id),
    sqlc.arg(external_order_id),
    eo.external_id,
    eo.external_inner_id,
    eo.worker_name,
    eo.trader_id,
    eo.requisite_raw,
    eo.requisite_phone,
    eo.method_type,
    eo.method_name,
    eo.amount_minor,
    eo.currency,
    eo.raw_status,
    eo.normalized_status,
    eo.created_at_external,
    TRUE
FROM external_orders eo
WHERE eo.id = sqlc.arg(external_order_id)
RETURNING id, team_id, scope_type, direction, shift_id, accounting_period_id, import_batch_id, import_row_id, external_order_id, external_id, external_inner_id, worker_name, trader_id, requisite_raw, requisite_phone, method_type, method_name, amount_minor, currency, raw_status, normalized_status, created_at_external, is_active, created_at, deactivated_at;

-- name: MarkImportBatchApplied :one
UPDATE import_batches
SET status = 'applied',
    applied_at = now()
WHERE id = sqlc.arg(import_batch_id)
RETURNING id, team_id, uploaded_by, scope_type, direction, shift_id, accounting_period_id, trader_id, file_name, file_hash, rows_count, status, superseded_by_batch_id, error_message, created_at, applied_at;
