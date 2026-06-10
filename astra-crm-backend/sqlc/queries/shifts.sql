-- name: GetCurrentTraderShift :one
SELECT id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at
FROM trader_shifts
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND status IN ('open', 'closing')
ORDER BY started_at DESC, id DESC
LIMIT 1;

-- name: CreateTraderShift :one
INSERT INTO trader_shifts (team_id, trader_id)
VALUES (sqlc.arg(team_id), sqlc.arg(trader_id))
RETURNING id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at;

-- name: GetActiveAssignmentForTraderRequisite :one
SELECT id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment
FROM requisite_assignments
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND requisite_id = sqlc.arg(requisite_id)
  AND unassigned_at IS NULL;

-- name: ListAssignedRequisitesForTrader :many
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.proxy,
    r.status,
    ra.id AS assignment_id,
    sr.id AS shift_requisite_id,
    sr.card_number,
    sr.holder_name,
    sr.status AS shift_requisite_status,
    sr.taken_at
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
LEFT JOIN current_shift cs ON true
LEFT JOIN shift_requisites sr ON sr.shift_id = cs.id AND sr.requisite_id = r.id
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.trader_id = sqlc.arg(trader_id)
  AND ra.unassigned_at IS NULL
  AND r.deleted_at IS NULL
  AND r.status = 'active'
ORDER BY r.id;

-- name: CreateShiftRequisite :one
INSERT INTO shift_requisites (team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name)
VALUES (sqlc.arg(team_id), sqlc.arg(shift_id), sqlc.arg(trader_id), sqlc.arg(requisite_id), sqlc.arg(assignment_id), sqlc.arg(card_number), sqlc.arg(holder_name))
RETURNING id, team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name, taken_at, released_at, status, created_at, updated_at;

-- name: ListShiftRequisitesByTrader :many
SELECT sr.id, sr.team_id, sr.shift_id, sr.trader_id, sr.requisite_id, sr.assignment_id, sr.card_number, sr.holder_name, sr.taken_at, sr.released_at, sr.status, sr.created_at, sr.updated_at
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND ts.status IN ('open', 'closing')
ORDER BY sr.taken_at DESC, sr.id DESC;

-- name: UpdateShiftRequisiteDetails :one
UPDATE shift_requisites sr
SET
    card_number = sqlc.arg(card_number),
    holder_name = sqlc.arg(holder_name),
    updated_at = now()
FROM trader_shifts ts
WHERE sr.id = sqlc.arg(shift_requisite_id)
  AND sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.status = 'active'
  AND ts.id = sr.shift_id
  AND ts.status IN ('open', 'closing')
RETURNING sr.id, sr.team_id, sr.shift_id, sr.trader_id, sr.requisite_id, sr.assignment_id, sr.card_number, sr.holder_name, sr.taken_at, sr.released_at, sr.status, sr.created_at, sr.updated_at;

-- name: CreateTurnoverEntry :one
WITH target_shift_requisite AS (
    SELECT sr.id, sr.team_id, sr.shift_id, sr.requisite_id, sr.trader_id
    FROM shift_requisites sr
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    WHERE sr.id = sqlc.arg(shift_requisite_id)
      AND sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.status = 'active'
      AND ts.status IN ('open', 'closing')
)
INSERT INTO requisite_turnover_entries (
    team_id,
    shift_id,
    shift_requisite_id,
    requisite_id,
    trader_id,
    amount_minor,
    created_by,
    comment
)
SELECT
    target_shift_requisite.team_id,
    target_shift_requisite.shift_id,
    target_shift_requisite.id,
    target_shift_requisite.requisite_id,
    target_shift_requisite.trader_id,
    sqlc.arg(amount_minor),
    sqlc.arg(created_by),
    sqlc.arg(comment)
FROM target_shift_requisite
RETURNING id, team_id, shift_id, shift_requisite_id, requisite_id, trader_id, amount_minor, created_by, created_at, comment;

-- name: ListLatestTurnoversForCurrentShift :many
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
SELECT DISTINCT ON (e.shift_requisite_id)
    e.id, e.team_id, e.shift_id, e.shift_requisite_id, e.requisite_id, e.trader_id, e.amount_minor, e.created_by, e.created_at, e.comment
FROM requisite_turnover_entries e
JOIN current_shift cs ON cs.id = e.shift_id
ORDER BY e.shift_requisite_id, e.created_at DESC, e.id DESC;

-- name: ListTurnoversByShiftRequisite :many
SELECT e.id, e.team_id, e.shift_id, e.shift_requisite_id, e.requisite_id, e.trader_id, e.amount_minor, e.created_by, e.created_at, e.comment
FROM requisite_turnover_entries e
JOIN shift_requisites sr ON sr.id = e.shift_requisite_id
JOIN trader_shifts ts ON ts.id = e.shift_id
WHERE e.team_id = sqlc.arg(team_id)
  AND e.trader_id = sqlc.arg(trader_id)
  AND e.shift_requisite_id = sqlc.arg(shift_requisite_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND ts.status IN ('open', 'closing')
ORDER BY e.created_at DESC, e.id DESC;

-- name: GetCurrentShiftChecklist :one
SELECT
    ts.id,
    ts.team_id,
    ts.trader_id,
    ts.started_at,
    ts.ended_at,
    ts.status,
    ts.inbound_reconciliation_status,
    ts.outbound_reconciliation_status,
    ts.close_comment,
    ts.created_at,
    ts.updated_at,
    ts.closed_at,
    (ts.inbound_reconciliation_status <> 'not_started') AS inbound_imported,
    (ts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')) AS inbound_ok,
    (ts.outbound_reconciliation_status <> 'not_started') AS outbound_imported,
    (ts.outbound_reconciliation_status IN ('matched', 'accepted_with_comment')) AS outbound_ok,
    NOT EXISTS (
        SELECT 1
        FROM manual_payout_orders mpo
        LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
        WHERE mpo.shift_id = ts.id
          AND mpo.deleted_at IS NULL
          AND mpo.status <> 'cancelled'
        GROUP BY mpo.id
        HAVING COALESCE(sum(mpt.amount_minor), 0)::bigint <> mpo.amount_minor
    ) AS all_payouts_fully_paid,
    (
        SELECT count(*)::bigint
        FROM (
            SELECT mpo.id
            FROM manual_payout_orders mpo
            LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
            WHERE mpo.shift_id = ts.id
              AND mpo.deleted_at IS NULL
              AND mpo.status <> 'cancelled'
            GROUP BY mpo.id
            HAVING COALESCE(sum(mpt.amount_minor), 0)::bigint <> mpo.amount_minor
        ) unpaid_payouts
    ) AS unpaid_payout_count
FROM trader_shifts ts
WHERE ts.team_id = sqlc.arg(team_id)
  AND ts.trader_id = sqlc.arg(trader_id)
  AND ts.status IN ('open', 'closing')
ORDER BY ts.started_at DESC, ts.id DESC
LIMIT 1;

-- name: CloseCurrentTraderShift :one
UPDATE trader_shifts
SET
    status = CASE
        WHEN inbound_reconciliation_status = 'accepted_with_comment'
          OR outbound_reconciliation_status = 'accepted_with_comment'
        THEN 'closed_with_discrepancy'
        ELSE 'closed'
    END,
    ended_at = now(),
    closed_at = now(),
    updated_at = now(),
    close_comment = sqlc.arg(close_comment)
WHERE trader_shifts.id = sqlc.arg(shift_id)
  AND trader_shifts.team_id = sqlc.arg(team_id)
  AND trader_shifts.trader_id = sqlc.arg(trader_id)
  AND trader_shifts.status IN ('open', 'closing')
  AND trader_shifts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')
  AND trader_shifts.outbound_reconciliation_status IN ('matched', 'accepted_with_comment')
  AND NOT EXISTS (
      SELECT 1
      FROM manual_payout_orders mpo
      LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
      WHERE mpo.shift_id = trader_shifts.id
        AND mpo.deleted_at IS NULL
        AND mpo.status <> 'cancelled'
      GROUP BY mpo.id
      HAVING COALESCE(sum(mpt.amount_minor), 0)::bigint <> mpo.amount_minor
  )
RETURNING id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at;
