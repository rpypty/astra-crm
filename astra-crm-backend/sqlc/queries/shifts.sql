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

-- name: ListTraderShiftHistory :many
SELECT id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at
FROM trader_shifts
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND status IN ('closed', 'closed_with_discrepancy')
ORDER BY COALESCE(closed_at, ended_at, updated_at) DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- name: ListTeamShiftHistory :many
SELECT id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at
FROM trader_shifts
WHERE team_id = sqlc.arg(team_id)
  AND status IN ('closed', 'closed_with_discrepancy')
ORDER BY COALESCE(closed_at, ended_at, updated_at) DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- name: GetTraderShiftByIDForTrader :one
SELECT id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at
FROM trader_shifts
WHERE team_id = sqlc.arg(team_id)
  AND trader_id = sqlc.arg(trader_id)
  AND id = sqlc.arg(shift_id)
LIMIT 1;

-- name: GetTraderShiftByIDForTeam :one
SELECT id, team_id, trader_id, started_at, ended_at, status, inbound_reconciliation_status, outbound_reconciliation_status, close_comment, created_at, updated_at, closed_at
FROM trader_shifts
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(shift_id)
LIMIT 1;

-- name: GetActiveAssignmentForTraderRequisite :one
SELECT ra.id, ra.team_id, ra.requisite_id, ra.trader_id, ra.assigned_by, ra.assigned_at, ra.unassigned_at, ra.comment
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.trader_id = sqlc.arg(trader_id)
  AND ra.requisite_id = sqlc.arg(requisite_id)
  AND ra.unassigned_at IS NULL
  AND ra.status IN ('planned', 'assigned')
  AND ra.assigned_for_date <= sqlc.arg(today)::date
  AND r.status = 'active'
  AND r.deleted_at IS NULL;

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
    r.bank_code,
    b.name AS bank_name,
    r.proxy,
    r.employee_comment,
    r.status,
    ra.id AS assignment_id,
    ra.status AS assignment_status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    COALESCE(sr.id, 0) AS shift_requisite_id,
    COALESCE(sr.card_number, r.card_number, '') AS card_number,
    COALESCE(sr.holder_name, r.holder_name, '') AS holder_name,
    COALESCE(sr.status, '') AS shift_requisite_status,
    sr.taken_at,
    sr.released_at,
    COALESCE(sr.inbound_turnover_minor, 0) AS inbound_turnover_minor,
    COALESCE(sr.outbound_turnover_minor, 0) AS outbound_turnover_minor,
    COALESCE(sr.closing_balance_minor, 0) AS closing_balance_minor
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
JOIN banks b ON b.code = r.bank_code
LEFT JOIN current_shift cs ON true
LEFT JOIN LATERAL (
    SELECT sr.*
    FROM shift_requisites sr
    WHERE sr.shift_id = cs.id
      AND sr.requisite_id = r.id
    ORDER BY sr.taken_at DESC, sr.id DESC
    LIMIT 1
) sr ON true
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.trader_id = sqlc.arg(trader_id)
  AND ra.unassigned_at IS NULL
  AND ra.status IN ('planned', 'assigned', 'in_work')
  AND ra.assigned_for_date <= sqlc.arg(today)::date
  AND r.deleted_at IS NULL
  AND r.status = 'active'
ORDER BY ra.assigned_for_date DESC, r.created_at DESC, r.id DESC;

-- name: ListFutureAssignedRequisitesForTrader :many
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    b.name AS bank_name,
    r.proxy,
    r.employee_comment,
    r.status,
    ra.id AS assignment_id,
    ra.status AS assignment_status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    0::bigint AS shift_requisite_id,
    COALESCE(r.card_number, '') AS card_number,
    COALESCE(r.holder_name, '') AS holder_name,
    ''::text AS shift_requisite_status,
    NULL::timestamptz AS taken_at,
    NULL::timestamptz AS released_at,
    0::bigint AS inbound_turnover_minor,
    0::bigint AS outbound_turnover_minor,
    0::bigint AS closing_balance_minor
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
JOIN banks b ON b.code = r.bank_code
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.trader_id = sqlc.arg(trader_id)
  AND ra.unassigned_at IS NULL
  AND ra.status IN ('planned', 'assigned')
  AND ra.assigned_for_date > sqlc.arg(today)::date
  AND r.deleted_at IS NULL
  AND r.status = 'active'
ORDER BY ra.assigned_for_date ASC, r.created_at DESC, r.id DESC;

-- name: ListHistoricalAssignedRequisitesForTrader :many
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    b.name AS bank_name,
    r.proxy,
    r.employee_comment,
    r.status,
    ra.id AS assignment_id,
    ra.status AS assignment_status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    COALESCE(sr.id, 0) AS shift_requisite_id,
    COALESCE(sr.card_number, r.card_number, '') AS card_number,
    COALESCE(sr.holder_name, r.holder_name, '') AS holder_name,
    COALESCE(sr.status, ra.status) AS shift_requisite_status,
    sr.taken_at,
    sr.released_at,
    COALESCE(sr.inbound_turnover_minor, 0) AS inbound_turnover_minor,
    COALESCE(sr.outbound_turnover_minor, 0) AS outbound_turnover_minor,
    COALESCE(sr.closing_balance_minor, 0) AS closing_balance_minor
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
JOIN banks b ON b.code = r.bank_code
LEFT JOIN shift_requisites sr ON sr.id = ra.shift_requisite_id
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.trader_id = sqlc.arg(trader_id)
  AND ra.status IN ('worked', 'blocked')
  AND sr.id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1
      FROM requisite_assignments active_ra
      WHERE active_ra.team_id = ra.team_id
        AND active_ra.trader_id = ra.trader_id
        AND active_ra.requisite_id = ra.requisite_id
        AND active_ra.id <> ra.id
        AND active_ra.unassigned_at IS NULL
        AND active_ra.status IN ('planned', 'assigned', 'in_work')
        AND active_ra.assigned_for_date <= sqlc.arg(today)::date
  )
ORDER BY COALESCE(ra.completed_at, ra.cancelled_at, ra.unassigned_at, ra.updated_at) DESC, ra.id DESC;

-- name: ListAssignedRequisitesForShift :many
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    b.name AS bank_name,
    r.proxy,
    r.employee_comment,
    r.status,
    ra.id AS assignment_id,
    ra.status AS assignment_status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    sr.id AS shift_requisite_id,
    sr.card_number,
    sr.holder_name,
    sr.status AS shift_requisite_status,
    sr.taken_at,
    sr.released_at,
    COALESCE(sr.inbound_turnover_minor, 0) AS inbound_turnover_minor,
    COALESCE(sr.outbound_turnover_minor, 0) AS outbound_turnover_minor,
    COALESCE(sr.closing_balance_minor, 0) AS closing_balance_minor
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
JOIN requisites r ON r.id = sr.requisite_id
JOIN banks b ON b.code = r.bank_code
LEFT JOIN requisite_assignments ra ON ra.id = sr.assignment_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND ts.team_id = sqlc.arg(team_id)
  AND ts.trader_id = sqlc.arg(trader_id)
ORDER BY sr.taken_at DESC, sr.id DESC;

-- name: ListAssignedRequisitesForTeamShift :many
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    b.name AS bank_name,
    r.proxy,
    r.employee_comment,
    r.status,
    ra.id AS assignment_id,
    ra.status AS assignment_status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    sr.id AS shift_requisite_id,
    sr.card_number,
    sr.holder_name,
    sr.status AS shift_requisite_status,
    sr.taken_at,
    sr.released_at,
    COALESCE(sr.inbound_turnover_minor, 0) AS inbound_turnover_minor,
    COALESCE(sr.outbound_turnover_minor, 0) AS outbound_turnover_minor,
    COALESCE(sr.closing_balance_minor, 0) AS closing_balance_minor
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
JOIN requisites r ON r.id = sr.requisite_id
JOIN banks b ON b.code = r.bank_code
LEFT JOIN requisite_assignments ra ON ra.id = sr.assignment_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.shift_id = sqlc.arg(shift_id)
  AND ts.team_id = sqlc.arg(team_id)
ORDER BY sr.taken_at DESC, sr.id DESC;

-- name: CreateShiftRequisite :one
INSERT INTO shift_requisites (team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name)
VALUES (sqlc.arg(team_id), sqlc.arg(shift_id), sqlc.arg(trader_id), sqlc.arg(requisite_id), sqlc.arg(assignment_id), sqlc.arg(card_number), sqlc.arg(holder_name))
RETURNING id, team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name, taken_at, released_at, status, inbound_turnover_minor, outbound_turnover_minor, closing_balance_minor, created_at, updated_at;

-- name: ListShiftRequisitesByTrader :many
SELECT sr.id, sr.team_id, sr.shift_id, sr.trader_id, sr.requisite_id, sr.assignment_id, sr.card_number, sr.holder_name, sr.taken_at, sr.released_at, sr.status, sr.inbound_turnover_minor, sr.outbound_turnover_minor, sr.closing_balance_minor, sr.created_at, sr.updated_at
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id)
  AND ts.status IN ('open', 'closing')
ORDER BY sr.taken_at DESC, sr.id DESC;

-- name: ListShiftReportRows :many
WITH crm_requisites AS (
    SELECT
        ('crm:' || sr.id::text) AS row_key,
        sr.id AS shift_requisite_id,
        sr.requisite_id,
        r.phone,
        r.method_type,
        r.bank_code,
        b.name AS bank_name,
        r.proxy,
        r.employee_comment,
        sr.card_number,
        sr.holder_name,
        sr.status,
        COALESCE(sr.inbound_turnover_minor, 0)::bigint AS inbound_turnover_minor,
        COALESCE(sr.outbound_turnover_minor, 0)::bigint AS outbound_turnover_minor,
        COALESCE(sr.closing_balance_minor, 0)::bigint AS closing_balance_minor,
        COALESCE(ra.target_turnover_minor, 0)::bigint AS target_turnover_minor,
        COALESCE(
            NULLIF(right(regexp_replace(r.phone, '[^0-9]', '', 'g'), 10), ''),
            'requisite:' || r.id::text
        ) AS match_key
    FROM shift_requisites sr
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN banks b ON b.code = r.bank_code
    LEFT JOIN requisite_assignments ra ON ra.id = sr.assignment_id
    WHERE sr.team_id = sqlc.arg(team_id)
      AND sr.shift_id = sqlc.arg(shift_id)
      AND ts.team_id = sqlc.arg(team_id)
),
csv_inbound AS (
    SELECT
        COALESCE(
            NULLIF(right(regexp_replace(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), ''), '[^0-9]', '', 'g'), 10), ''),
            'csv:' || lower(btrim(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''), 'unknown')))
        ) AS match_key,
        max(COALESCE(NULLIF(requisite_phone, ''), NULLIF(requisite_raw, ''))) AS csv_requisite,
        COALESCE(sum(CASE WHEN normalized_status IN ('success', 'corrected') THEN amount_minor ELSE 0 END), 0)::bigint AS csv_inbound_minor
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
payout_transfer_outbound AS (
    SELECT
        mpt.source_shift_requisite_id AS shift_requisite_id,
        COALESCE(sum(mpt.amount_minor), 0)::bigint AS transfer_outbound_minor
    FROM manual_payout_transfers mpt
    WHERE mpt.team_id = sqlc.arg(team_id)
      AND mpt.shift_id = sqlc.arg(shift_id)
    GROUP BY mpt.source_shift_requisite_id
),
all_keys AS (
    SELECT match_key FROM crm_requisites
    UNION
    SELECT match_key FROM csv_inbound
),
report_rows AS (
    SELECT
        COALESCE(crm.row_key, 'csv:' || all_keys.match_key) AS row_key,
        crm.shift_requisite_id,
        crm.requisite_id,
        COALESCE(crm.phone, csv_inbound.csv_requisite, 'Без реквизита') AS phone,
        COALESCE(crm.method_type, '') AS method_type,
        COALESCE(crm.bank_code, '') AS bank_code,
        COALESCE(crm.bank_name, '') AS bank_name,
        crm.proxy,
        crm.employee_comment,
        crm.card_number,
        crm.holder_name,
        CASE WHEN crm.shift_requisite_id IS NULL THEN 'csv_only' ELSE crm.status END AS status,
        COALESCE(crm.inbound_turnover_minor, 0)::bigint AS inbound_turnover_minor,
        COALESCE(crm.outbound_turnover_minor, 0)::bigint AS outbound_turnover_minor,
        COALESCE(crm.closing_balance_minor, 0)::bigint AS closing_balance_minor,
        COALESCE(crm.target_turnover_minor, 0)::bigint AS target_turnover_minor,
        COALESCE(csv_inbound.csv_inbound_minor, 0)::bigint AS csv_inbound_minor,
        COALESCE(payout_transfer_outbound.transfer_outbound_minor, 0)::bigint AS csv_outbound_minor,
        (COALESCE(crm.inbound_turnover_minor, 0) - COALESCE(csv_inbound.csv_inbound_minor, 0))::bigint AS inbound_diff_minor,
        (COALESCE(crm.outbound_turnover_minor, 0) - COALESCE(payout_transfer_outbound.transfer_outbound_minor, 0))::bigint AS outbound_diff_minor,
        (
            COALESCE(crm.inbound_turnover_minor, 0) <> COALESCE(csv_inbound.csv_inbound_minor, 0)
            OR COALESCE(crm.outbound_turnover_minor, 0) <> COALESCE(payout_transfer_outbound.transfer_outbound_minor, 0)
        ) AS has_mismatch,
        crm.shift_requisite_id IS NULL AS csv_only
    FROM all_keys
    LEFT JOIN crm_requisites crm ON crm.match_key = all_keys.match_key
    LEFT JOIN csv_inbound ON csv_inbound.match_key = all_keys.match_key
    LEFT JOIN payout_transfer_outbound ON payout_transfer_outbound.shift_requisite_id = crm.shift_requisite_id
)
SELECT
    row_key::text AS row_key,
    shift_requisite_id,
    requisite_id,
    phone::text AS phone,
    method_type::text AS method_type,
    bank_code::text AS bank_code,
    bank_name::text AS bank_name,
    proxy,
    employee_comment,
    card_number,
    holder_name,
    status::text AS status,
    inbound_turnover_minor,
    outbound_turnover_minor,
    closing_balance_minor,
    target_turnover_minor,
    csv_inbound_minor,
    csv_outbound_minor,
    inbound_diff_minor,
    outbound_diff_minor,
    has_mismatch::bool AS has_mismatch,
    csv_only::bool AS csv_only
FROM report_rows
ORDER BY has_mismatch DESC, csv_only DESC, (abs(inbound_diff_minor) + abs(outbound_diff_minor)) DESC, phone ASC, row_key ASC;

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
RETURNING sr.id, sr.team_id, sr.shift_id, sr.trader_id, sr.requisite_id, sr.assignment_id, sr.card_number, sr.holder_name, sr.taken_at, sr.released_at, sr.status, sr.inbound_turnover_minor, sr.outbound_turnover_minor, sr.closing_balance_minor, sr.created_at, sr.updated_at;

-- name: CloseShiftRequisite :one
WITH target AS (
    SELECT
        sr.id,
        sr.team_id,
        sr.trader_id,
        sr.shift_id,
        sr.requisite_id,
        sr.inbound_turnover_minor AS old_inbound_turnover_minor,
        sr.outbound_turnover_minor AS old_outbound_turnover_minor
    FROM shift_requisites sr
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    WHERE sr.id = sqlc.arg(shift_requisite_id)
      AND sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.status IN ('active', 'correction')
      AND ts.id = sr.shift_id
      AND ts.status IN ('open', 'closing')
),
updated_sr AS (
    UPDATE shift_requisites sr
    SET
        inbound_turnover_minor = sqlc.arg(inbound_turnover_minor),
        outbound_turnover_minor = sqlc.arg(outbound_turnover_minor),
        closing_balance_minor = sqlc.arg(closing_balance_minor),
        released_at = COALESCE(sr.released_at, COALESCE(sqlc.narg(released_at), now())),
        status = CASE WHEN sqlc.arg(blocked)::boolean THEN 'blocked' ELSE 'worked_pending_review' END,
        updated_at = now()
    FROM target
    WHERE sr.id = target.id
    RETURNING
        sr.id,
        sr.team_id,
        sr.shift_id,
        sr.trader_id,
        sr.requisite_id,
        sr.assignment_id,
        sr.card_number,
        sr.holder_name,
        sr.taken_at,
        sr.released_at,
        sr.status,
        sr.inbound_turnover_minor,
        sr.outbound_turnover_minor,
        sr.closing_balance_minor,
        sr.created_at,
        sr.updated_at,
        target.old_inbound_turnover_minor,
        target.old_outbound_turnover_minor
),
final_turnover_entry AS (
    INSERT INTO requisite_turnover_entries (
        team_id,
        shift_id,
        shift_requisite_id,
        requisite_id,
        trader_id,
        amount_minor,
        created_by,
        comment,
        created_at
    )
    SELECT
        updated_sr.team_id,
        updated_sr.shift_id,
        updated_sr.id,
        updated_sr.requisite_id,
        updated_sr.trader_id,
        updated_sr.inbound_turnover_minor,
        sqlc.arg(created_by),
        'Финальный оборот при закрытии реквизита',
        COALESCE(sqlc.narg(released_at), now())
    FROM updated_sr
    RETURNING id
),
updated_requisite AS (
    UPDATE requisites r
    SET
        total_inbound_turnover_minor = GREATEST(0, r.total_inbound_turnover_minor + updated_sr.inbound_turnover_minor - updated_sr.old_inbound_turnover_minor),
        total_outbound_turnover_minor = GREATEST(0, r.total_outbound_turnover_minor + updated_sr.outbound_turnover_minor - updated_sr.old_outbound_turnover_minor),
        last_closing_balance_minor = updated_sr.closing_balance_minor,
        last_activity_status = updated_sr.status,
        last_activity_at = COALESCE(updated_sr.released_at, updated_sr.taken_at, updated_sr.updated_at),
        last_shift_requisite_id = updated_sr.id,
        updated_at = now()
    FROM updated_sr, final_turnover_entry
    WHERE r.team_id = updated_sr.team_id
      AND r.id = updated_sr.requisite_id
    RETURNING r.id
)
SELECT id, team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name, taken_at, released_at, status, inbound_turnover_minor, outbound_turnover_minor, closing_balance_minor, created_at, updated_at
FROM updated_sr;

-- name: CorrectClosedShiftRequisiteTurnovers :one
WITH target AS (
    SELECT
        sr.id,
        sr.team_id,
        sr.trader_id,
        sr.shift_id,
        sr.requisite_id,
        sr.inbound_turnover_minor AS old_inbound_turnover_minor,
        sr.outbound_turnover_minor AS old_outbound_turnover_minor
    FROM shift_requisites sr
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    WHERE sr.id = sqlc.arg(shift_requisite_id)
      AND sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy', 'correction')
      AND ts.id = sr.shift_id
      AND ts.status IN ('open', 'closing')
),
updated_sr AS (
    UPDATE shift_requisites sr
    SET
        inbound_turnover_minor = sqlc.arg(inbound_turnover_minor),
        outbound_turnover_minor = sqlc.arg(outbound_turnover_minor),
        closing_balance_minor = sqlc.arg(closing_balance_minor),
        status = 'worked_pending_review',
        updated_at = now()
    FROM target
    WHERE sr.id = target.id
    RETURNING
        sr.id,
        sr.team_id,
        sr.shift_id,
        sr.trader_id,
        sr.requisite_id,
        sr.assignment_id,
        sr.card_number,
        sr.holder_name,
        sr.taken_at,
        sr.released_at,
        sr.status,
        sr.inbound_turnover_minor,
        sr.outbound_turnover_minor,
        sr.closing_balance_minor,
        sr.created_at,
        sr.updated_at,
        target.old_inbound_turnover_minor,
        target.old_outbound_turnover_minor
),
turnover_entry AS (
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
        updated_sr.team_id,
        updated_sr.shift_id,
        updated_sr.id,
        updated_sr.requisite_id,
        updated_sr.trader_id,
        updated_sr.inbound_turnover_minor,
        sqlc.arg(created_by),
        sqlc.arg(comment)
    FROM updated_sr
    RETURNING id
),
updated_requisite AS (
    UPDATE requisites r
    SET
        total_inbound_turnover_minor = GREATEST(0, r.total_inbound_turnover_minor + updated_sr.inbound_turnover_minor - updated_sr.old_inbound_turnover_minor),
        total_outbound_turnover_minor = GREATEST(0, r.total_outbound_turnover_minor + updated_sr.outbound_turnover_minor - updated_sr.old_outbound_turnover_minor),
        last_closing_balance_minor = updated_sr.closing_balance_minor,
        last_activity_status = updated_sr.status,
        last_activity_at = COALESCE(updated_sr.released_at, updated_sr.taken_at, updated_sr.updated_at),
        last_shift_requisite_id = updated_sr.id,
        updated_at = now()
    FROM updated_sr, turnover_entry
    WHERE r.team_id = updated_sr.team_id
      AND r.id = updated_sr.requisite_id
    RETURNING r.id
)
SELECT id, team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name, taken_at, released_at, status, inbound_turnover_minor, outbound_turnover_minor, closing_balance_minor, created_at, updated_at
FROM updated_sr;

-- name: ReturnShiftRequisiteToWork :one
WITH target AS (
    SELECT
        sr.id,
        sr.team_id,
        sr.trader_id,
        sr.shift_id,
        sr.requisite_id,
        sr.assignment_id
    FROM shift_requisites sr
    JOIN trader_shifts ts ON ts.id = sr.shift_id
    WHERE sr.id = sqlc.arg(shift_requisite_id)
      AND sr.team_id = sqlc.arg(team_id)
      AND sr.trader_id = sqlc.arg(trader_id)
      AND sr.status IN ('worked_pending_review', 'worked_discrepancy', 'correction', 'blocked')
      AND ts.status IN ('open', 'closing')
),
updated_sr AS (
    UPDATE shift_requisites sr
    SET
        released_at = NULL,
        status = 'active',
        updated_at = now()
    FROM target
    WHERE sr.id = target.id
    RETURNING
        sr.id,
        sr.team_id,
        sr.shift_id,
        sr.trader_id,
        sr.requisite_id,
        sr.assignment_id,
        sr.card_number,
        sr.holder_name,
        sr.taken_at,
        sr.released_at,
        sr.status,
        sr.inbound_turnover_minor,
        sr.outbound_turnover_minor,
        sr.closing_balance_minor,
        sr.created_at,
        sr.updated_at
),
updated_assignment AS (
    UPDATE requisite_assignments ra
    SET
        status = 'in_work',
        completed_at = NULL,
        unassigned_at = NULL,
        started_at = COALESCE(started_at, now()),
        updated_at = now()
    FROM updated_sr
    WHERE ra.team_id = updated_sr.team_id
      AND ra.id = updated_sr.assignment_id
    RETURNING ra.id
),
updated_requisite AS (
    UPDATE requisites r
    SET
        last_activity_status = updated_sr.status,
        last_activity_at = COALESCE(updated_sr.taken_at, updated_sr.updated_at),
        last_shift_requisite_id = updated_sr.id,
        updated_at = now()
    FROM updated_sr
    WHERE r.team_id = updated_sr.team_id
      AND r.id = updated_sr.requisite_id
    RETURNING r.id
)
SELECT id, team_id, shift_id, trader_id, requisite_id, assignment_id, card_number, holder_name, taken_at, released_at, status, inbound_turnover_minor, outbound_turnover_minor, closing_balance_minor, created_at, updated_at
FROM updated_sr;

-- name: GetShiftRequisiteForTrader :one
SELECT sr.id, sr.team_id, sr.shift_id, sr.trader_id, sr.requisite_id, sr.assignment_id, sr.card_number, sr.holder_name, sr.taken_at, sr.released_at, sr.status, sr.inbound_turnover_minor, sr.outbound_turnover_minor, sr.closing_balance_minor, sr.created_at, sr.updated_at
FROM shift_requisites sr
WHERE sr.id = sqlc.arg(shift_requisite_id)
  AND sr.team_id = sqlc.arg(team_id)
  AND sr.trader_id = sqlc.arg(trader_id);

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
