-- name: CreateTeamleadReconciliation :one
INSERT INTO teamlead_reconciliations (
    team_id,
    date_from,
    date_to,
    status,
    created_by,
    inbound_import_batch_id,
    outbound_import_batch_id,
    pipeline_json,
    inbound_summary_json,
    outbound_summary_json,
    preview_json
)
VALUES (
    sqlc.arg(team_id),
    sqlc.arg(date_from),
    sqlc.arg(date_to),
    sqlc.arg(status),
    sqlc.arg(created_by),
    sqlc.narg(inbound_import_batch_id),
    sqlc.narg(outbound_import_batch_id),
    COALESCE(sqlc.narg(pipeline_json), '[]'::jsonb),
    COALESCE(sqlc.narg(inbound_summary_json), '{}'::jsonb),
    COALESCE(sqlc.narg(outbound_summary_json), '{}'::jsonb),
    COALESCE(sqlc.narg(preview_json), '{}'::jsonb)
)
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: GetTeamleadReconciliation :one
SELECT id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at
FROM teamlead_reconciliations
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id);

-- name: QueueTeamleadReconciliationApply :one
UPDATE teamlead_reconciliations
SET
    status = 'apply_queued',
    confirmed_by = sqlc.arg(actor_id),
    comment = sqlc.narg(comment),
    confirmed_at = now(),
    apply_queued_at = now(),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id)
  AND status IN ('matched', 'mismatch', 'apply_failed')
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: RejectTeamleadReconciliation :one
UPDATE teamlead_reconciliations
SET
    status = 'rejected',
    rejected_by = sqlc.arg(actor_id),
    comment = sqlc.arg(comment),
    rejected_at = now(),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id)
  AND status IN ('matched', 'mismatch', 'apply_failed')
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: MarkTeamleadReconciliationApplying :one
UPDATE teamlead_reconciliations
SET
    status = 'applying',
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id)
  AND status = 'apply_queued'
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: MarkTeamleadReconciliationApplied :one
UPDATE teamlead_reconciliations
SET
    status = 'applied',
    apply_result_json = sqlc.arg(apply_result_json),
    error_message = NULL,
    applied_at = now(),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id)
  AND status = 'applying'
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: MarkTeamleadReconciliationApplyFailed :one
UPDATE teamlead_reconciliations
SET
    status = 'apply_failed',
    apply_result_json = COALESCE(sqlc.narg(apply_result_json), apply_result_json),
    error_message = sqlc.arg(error_message),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id)
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: UpsertTeamleadReconciliationApplyJobQueued :exec
INSERT INTO teamlead_reconciliation_apply_jobs (
    teamlead_reconciliation_id,
    team_id,
    status,
    attempts,
    result_json,
    error_message,
    started_at,
    finished_at
)
VALUES (
    sqlc.arg(teamlead_reconciliation_id),
    sqlc.arg(team_id),
    'queued',
    0,
    '{}'::jsonb,
    NULL,
    NULL,
    NULL
)
ON CONFLICT (teamlead_reconciliation_id)
DO UPDATE SET
    status = 'queued',
    attempts = 0,
    result_json = '{}'::jsonb,
    error_message = NULL,
    started_at = NULL,
    finished_at = NULL;

-- name: MarkTeamleadReconciliationApplyJobRunning :exec
UPDATE teamlead_reconciliation_apply_jobs
SET
    status = 'running',
    attempts = attempts + 1,
    started_at = now(),
    finished_at = NULL,
    error_message = NULL
WHERE teamlead_reconciliation_id = sqlc.arg(teamlead_reconciliation_id)
  AND team_id = sqlc.arg(team_id);

-- name: MarkTeamleadReconciliationApplyJobSucceeded :exec
UPDATE teamlead_reconciliation_apply_jobs
SET
    status = 'succeeded',
    result_json = sqlc.arg(result_json),
    finished_at = now(),
    error_message = NULL
WHERE teamlead_reconciliation_id = sqlc.arg(teamlead_reconciliation_id)
  AND team_id = sqlc.arg(team_id);

-- name: MarkTeamleadReconciliationApplyJobFailed :exec
UPDATE teamlead_reconciliation_apply_jobs
SET
    status = 'failed',
    result_json = COALESCE(sqlc.narg(result_json), result_json),
    error_message = sqlc.arg(error_message),
    finished_at = now()
WHERE teamlead_reconciliation_id = sqlc.arg(teamlead_reconciliation_id)
  AND team_id = sqlc.arg(team_id);

-- name: UpdateTeamleadReconciliationAnalysisResult :one
UPDATE teamlead_reconciliations
SET
    status = sqlc.arg(status),
    mismatch_count = sqlc.arg(mismatch_count),
    conflict_count = sqlc.arg(conflict_count),
    blocked_count = sqlc.arg(blocked_count),
    analyzed_at = now(),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(id)
RETURNING id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at;

-- name: ListTeamleadReconciliations :many
SELECT id, team_id, date_from, date_to, status, created_by, confirmed_by, rejected_by, inbound_import_batch_id, outbound_import_batch_id, comment, mismatch_count, conflict_count, blocked_count, pipeline_json, inbound_summary_json, outbound_summary_json, preview_json, apply_result_json, error_message, created_at, updated_at, analyzed_at, confirmed_at, rejected_at, apply_queued_at, applied_at
FROM teamlead_reconciliations
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountTeamleadReconciliations :one
SELECT count(*)::bigint
FROM teamlead_reconciliations
WHERE team_id = sqlc.arg(team_id);

-- name: CreateTeamleadReconciliationItem :one
INSERT INTO teamlead_reconciliation_items (
    teamlead_reconciliation_id,
    team_id,
    direction,
    stage,
    issue_type,
    severity,
    external_order_id,
    external_inner_id,
    trader_id,
    requisite_id,
    shift_id,
    before_json,
    after_json,
    message,
    is_blocking
)
VALUES (
    sqlc.arg(teamlead_reconciliation_id),
    sqlc.arg(team_id),
    sqlc.arg(direction),
    sqlc.arg(stage),
    sqlc.arg(issue_type),
    sqlc.arg(severity),
    sqlc.narg(external_order_id),
    sqlc.narg(external_inner_id),
    sqlc.narg(trader_id),
    sqlc.narg(requisite_id),
    sqlc.narg(shift_id),
    sqlc.narg(before_json),
    sqlc.narg(after_json),
    sqlc.narg(message),
    sqlc.arg(is_blocking)
)
RETURNING id, teamlead_reconciliation_id, team_id, direction, stage, issue_type, severity, external_order_id, external_inner_id, trader_id, requisite_id, shift_id, before_json, after_json, message, is_blocking, applied_at, created_at;

-- name: ListTeamleadReconciliationItems :many
SELECT id, teamlead_reconciliation_id, team_id, direction, stage, issue_type, severity, external_order_id, external_inner_id, trader_id, requisite_id, shift_id, before_json, after_json, message, is_blocking, applied_at, created_at
FROM teamlead_reconciliation_items
WHERE team_id = sqlc.arg(team_id)
  AND teamlead_reconciliation_id = sqlc.arg(teamlead_reconciliation_id)
  AND (sqlc.narg(direction)::text IS NULL OR direction = sqlc.narg(direction))
  AND (sqlc.narg(stage)::text IS NULL OR stage = sqlc.narg(stage))
  AND (sqlc.narg(issue_type)::text IS NULL OR issue_type = sqlc.narg(issue_type))
  AND (sqlc.narg(severity)::text IS NULL OR severity = sqlc.narg(severity))
  AND (sqlc.narg(trader_id)::bigint IS NULL OR trader_id = sqlc.narg(trader_id))
  AND (sqlc.narg(requisite_id)::bigint IS NULL OR requisite_id = sqlc.narg(requisite_id))
  AND (NOT sqlc.arg(only_mismatch)::boolean OR severity <> 'info')
ORDER BY id
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountTeamleadReconciliationItems :one
SELECT count(*)::bigint
FROM teamlead_reconciliation_items
WHERE team_id = sqlc.arg(team_id)
  AND teamlead_reconciliation_id = sqlc.arg(teamlead_reconciliation_id)
  AND (sqlc.narg(direction)::text IS NULL OR direction = sqlc.narg(direction))
  AND (sqlc.narg(stage)::text IS NULL OR stage = sqlc.narg(stage))
  AND (sqlc.narg(issue_type)::text IS NULL OR issue_type = sqlc.narg(issue_type))
  AND (sqlc.narg(severity)::text IS NULL OR severity = sqlc.narg(severity))
  AND (sqlc.narg(trader_id)::bigint IS NULL OR trader_id = sqlc.narg(trader_id))
  AND (sqlc.narg(requisite_id)::bigint IS NULL OR requisite_id = sqlc.narg(requisite_id))
  AND (NOT sqlc.arg(only_mismatch)::boolean OR severity <> 'info');

-- name: ListTeamleadReconciliationTraders :many
SELECT
    u.id AS trader_id,
    p.external_worker_name
FROM users u
JOIN trader_profiles p ON p.user_id = u.id
WHERE u.team_id = sqlc.arg(team_id)
  AND u.role = 'trader'
  AND u.deleted_at IS NULL;

-- name: ListTeamleadReconciliationRequisites :many
SELECT
    id,
    bank_code,
    phone,
    card_number,
    normalized_phone,
    normalized_card_number
FROM requisites
WHERE team_id = sqlc.arg(team_id)
  AND deleted_at IS NULL;

-- name: ListTeamleadReconciliationExternalOrders :many
SELECT
    id,
    direction,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_id,
    amount_minor,
    raw_status,
    normalized_status,
    created_at_external
FROM external_orders
WHERE team_id = sqlc.arg(team_id)
  AND direction = sqlc.arg(direction)
  AND external_inner_id = ANY(sqlc.arg(inner_ids)::text[]);

-- name: ListTeamleadReconciliationExternalOrdersInPeriod :many
SELECT
    id,
    direction,
    external_inner_id,
    worker_name,
    trader_id,
    requisite_id,
    amount_minor,
    raw_status,
    normalized_status,
    created_at_external
FROM external_orders
WHERE team_id = sqlc.arg(team_id)
  AND direction = sqlc.arg(direction)
  AND created_at_external >= sqlc.arg(date_from)::timestamptz
  AND created_at_external < sqlc.arg(date_to_exclusive)::timestamptz;

-- name: CalculateTeamleadV2InboundTurnover :one
SELECT
    COALESCE(sum(sr.inbound_turnover_minor), 0)::bigint AS amount_minor,
    count(*)::bigint AS requisites_count
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction', 'blocked')
  AND ts.started_at::date <= sqlc.arg(date_to)::date
  AND COALESCE(ts.closed_at, ts.ended_at, ts.updated_at)::date >= sqlc.arg(date_from)::date;

-- name: CalculateTeamleadV2OutboundTransfers :one
SELECT
    COALESCE(sum(mpt.amount_minor), 0)::bigint AS amount_minor,
    count(*)::bigint AS transfers_count
FROM manual_payout_transfers mpt
JOIN trader_shifts ts ON ts.id = mpt.shift_id
WHERE mpt.team_id = sqlc.arg(team_id)
  AND ts.started_at::date <= sqlc.arg(date_to)::date
  AND COALESCE(ts.closed_at, ts.ended_at, ts.updated_at)::date >= sqlc.arg(date_from)::date;
