-- name: GetRequisiteByIDForTeam :one
SELECT id, team_id, phone, method_type, bank_code, proxy, employee_comment, holder_name, card_number, details_filled_at, details_filled_by, status, created_by, created_at, updated_at, deleted_at
FROM requisites
WHERE id = $1
  AND team_id = $2
  AND deleted_at IS NULL;

-- name: CreateRequisite :one
INSERT INTO requisites (team_id, phone, method_type, bank_code, card_number, proxy, employee_comment, created_by)
VALUES (sqlc.arg(team_id), sqlc.arg(phone), sqlc.arg(method_type), sqlc.arg(bank_code), sqlc.arg(card_number), sqlc.arg(proxy), sqlc.arg(employee_comment), sqlc.arg(created_by))
RETURNING id, team_id, phone, method_type, bank_code, proxy, employee_comment, holder_name, card_number, details_filled_at, details_filled_by, status, created_by, created_at, updated_at, deleted_at;

-- name: GetRequisiteDetailsByIDForTeam :one
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    ''::text AS bank_name,
    r.proxy,
    r.employee_comment,
    r.holder_name,
    r.card_number,
    r.details_filled_at,
    r.details_filled_by,
    r.status,
    r.created_by,
    r.created_at,
    r.updated_at,
    r.deleted_at,
    COALESCE(ra.id, 0) AS active_assignment_id,
    COALESCE(ra.trader_id, 0) AS assigned_trader_id,
    COALESCE(ra.trader_login, '') AS assigned_trader_login,
    COALESCE(ra.status, '') AS assignment_status,
    ra.assigned_for_date,
    COALESCE(ra.target_turnover_minor, 0) AS target_turnover_minor
FROM requisites r
LEFT JOIN LATERAL (
    SELECT ra.id, ra.trader_id, u.login AS trader_login, ra.status, ra.assigned_for_date, ra.target_turnover_minor
    FROM requisite_assignments ra
    JOIN users u ON u.id = ra.trader_id
    WHERE ra.team_id = r.team_id
      AND ra.requisite_id = r.id
      AND ra.unassigned_at IS NULL
      AND ra.status IN ('planned', 'assigned', 'in_work')
    ORDER BY ra.assigned_for_date DESC, ra.assigned_at DESC, ra.id DESC
    LIMIT 1
) ra ON true
WHERE r.team_id = sqlc.arg(team_id)
  AND r.id = sqlc.arg(requisite_id)
  AND r.deleted_at IS NULL;

-- name: ListRequisiteDetailsByTeam :many
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    ''::text AS bank_name,
    r.proxy,
    r.employee_comment,
    r.holder_name,
    r.card_number,
    r.details_filled_at,
    r.details_filled_by,
    r.status,
    r.created_by,
    r.created_at,
    r.updated_at,
    r.deleted_at,
    COALESCE(ra.id, 0) AS active_assignment_id,
    COALESCE(ra.trader_id, 0) AS assigned_trader_id,
    COALESCE(ra.trader_login, '') AS assigned_trader_login,
    COALESCE(ra.status, '') AS assignment_status,
    ra.assigned_for_date,
    COALESCE(ra.target_turnover_minor, 0) AS target_turnover_minor
FROM requisites r
LEFT JOIN LATERAL (
    SELECT ra.id, ra.trader_id, u.login AS trader_login, ra.status, ra.assigned_for_date, ra.target_turnover_minor
    FROM requisite_assignments ra
    JOIN users u ON u.id = ra.trader_id
    WHERE ra.team_id = r.team_id
      AND ra.requisite_id = r.id
      AND ra.unassigned_at IS NULL
      AND ra.status IN ('planned', 'assigned', 'in_work')
    ORDER BY ra.assigned_for_date DESC, ra.assigned_at DESC, ra.id DESC
    LIMIT 1
) ra ON true
WHERE r.team_id = sqlc.arg(team_id)
  AND r.deleted_at IS NULL
  AND (
      sqlc.arg(bank_code)::text = ''
      OR sqlc.arg(bank_code)::text = 'all'
      OR r.bank_code = sqlc.arg(bank_code)::text
  )
  AND (
      sqlc.arg(status)::text = ''
      OR sqlc.arg(status)::text = 'all'
      OR r.status = sqlc.arg(status)::text
  )
  AND (
      NOT sqlc.arg(available_for_planning)::boolean
      OR (
          r.status = 'active'
          AND NOT EXISTS (
              SELECT 1
              FROM requisite_assignments blocked_ra
              WHERE blocked_ra.team_id = r.team_id
                AND blocked_ra.requisite_id = r.id
                AND blocked_ra.unassigned_at IS NULL
                AND blocked_ra.status = 'blocked'
          )
      )
  )
  AND (
      sqlc.arg(trader_filter)::text = ''
      OR sqlc.arg(trader_filter)::text = 'all'
      OR (sqlc.arg(trader_filter)::text = 'unassigned' AND ra.trader_id IS NULL)
      OR (sqlc.arg(trader_filter)::text NOT IN ('unassigned', 'all') AND ra.trader_id::text = sqlc.arg(trader_filter)::text)
  )
  AND (
      sqlc.arg(search)::text = ''
      OR lower(r.phone) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          COALESCE(cardinality(sqlc.arg(bank_search_codes)::text[]), 0) > 0
          AND r.bank_code = ANY(sqlc.arg(bank_search_codes)::text[])
      )
      OR lower(COALESCE(r.proxy, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.employee_comment, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.holder_name, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.card_number, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          sqlc.arg(search_digits)::text <> ''
          AND (
              regexp_replace(r.phone, '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
              OR regexp_replace(COALESCE(r.card_number, ''), '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
          )
      )
  )
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountRequisiteDetailsByTeam :one
SELECT count(*)::bigint
FROM requisites r
LEFT JOIN LATERAL (
    SELECT ra.id, ra.trader_id, u.login AS trader_login, ra.status, ra.assigned_for_date, ra.target_turnover_minor
    FROM requisite_assignments ra
    JOIN users u ON u.id = ra.trader_id
    WHERE ra.team_id = r.team_id
      AND ra.requisite_id = r.id
      AND ra.unassigned_at IS NULL
      AND ra.status IN ('planned', 'assigned', 'in_work')
    ORDER BY ra.assigned_for_date DESC, ra.assigned_at DESC, ra.id DESC
    LIMIT 1
) ra ON true
WHERE r.team_id = sqlc.arg(team_id)
  AND r.deleted_at IS NULL
  AND (
      sqlc.arg(bank_code)::text = ''
      OR sqlc.arg(bank_code)::text = 'all'
      OR r.bank_code = sqlc.arg(bank_code)::text
  )
  AND (
      sqlc.arg(status)::text = ''
      OR sqlc.arg(status)::text = 'all'
      OR r.status = sqlc.arg(status)::text
  )
  AND (
      NOT sqlc.arg(available_for_planning)::boolean
      OR (
          r.status = 'active'
          AND NOT EXISTS (
              SELECT 1
              FROM requisite_assignments blocked_ra
              WHERE blocked_ra.team_id = r.team_id
                AND blocked_ra.requisite_id = r.id
                AND blocked_ra.unassigned_at IS NULL
                AND blocked_ra.status = 'blocked'
          )
      )
  )
  AND (
      sqlc.arg(trader_filter)::text = ''
      OR sqlc.arg(trader_filter)::text = 'all'
      OR (sqlc.arg(trader_filter)::text = 'unassigned' AND ra.trader_id IS NULL)
      OR (sqlc.arg(trader_filter)::text NOT IN ('unassigned', 'all') AND ra.trader_id::text = sqlc.arg(trader_filter)::text)
  )
  AND (
      sqlc.arg(search)::text = ''
      OR lower(r.phone) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          COALESCE(cardinality(sqlc.arg(bank_search_codes)::text[]), 0) > 0
          AND r.bank_code = ANY(sqlc.arg(bank_search_codes)::text[])
      )
      OR lower(COALESCE(r.proxy, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.employee_comment, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.holder_name, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.card_number, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          sqlc.arg(search_digits)::text <> ''
          AND (
              regexp_replace(r.phone, '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
              OR regexp_replace(COALESCE(r.card_number, ''), '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
          )
      )
  );

-- name: UpdateRequisite :one
UPDATE requisites
SET
    phone = sqlc.arg(phone),
    method_type = sqlc.arg(method_type),
    bank_code = sqlc.arg(bank_code),
    card_number = sqlc.arg(card_number),
    proxy = sqlc.arg(proxy),
    employee_comment = sqlc.arg(employee_comment),
    status = sqlc.arg(status),
    updated_at = now(),
    deleted_at = CASE
        WHEN sqlc.arg(status) = 'archived' THEN COALESCE(requisites.deleted_at, now())
        ELSE NULL
    END
WHERE requisites.team_id = sqlc.arg(team_id)
  AND requisites.id = sqlc.arg(requisite_id)
  AND requisites.deleted_at IS NULL
RETURNING id, team_id, phone, method_type, bank_code, proxy, employee_comment, holder_name, card_number, details_filled_at, details_filled_by, status, created_by, created_at, updated_at, deleted_at;

-- name: FillRequisiteInitialDetails :exec
UPDATE requisites
SET
    holder_name = sqlc.arg(holder_name),
    card_number = COALESCE(card_number, sqlc.arg(card_number)),
    details_filled_at = now(),
    details_filled_by = sqlc.arg(details_filled_by),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(requisite_id)
  AND deleted_at IS NULL
  AND holder_name IS NULL;

-- name: LockAssignableRequisiteForAssignment :one
SELECT r.id
FROM requisites r
WHERE r.team_id = sqlc.arg(team_id)
  AND r.id = sqlc.arg(requisite_id)
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM shift_requisites sr
      JOIN trader_shifts ts ON ts.id = sr.shift_id
      WHERE sr.team_id = r.team_id
        AND sr.requisite_id = r.id
        AND sr.status = 'active'
        AND ts.status IN ('open', 'closing')
  )
FOR UPDATE;

-- name: CloseActiveRequisiteAssignment :many
UPDATE requisite_assignments
SET unassigned_at = now(),
    status = CASE WHEN status IN ('worked', 'blocked', 'cancelled', 'expired') THEN status ELSE 'cancelled' END,
    cancelled_at = CASE WHEN status IN ('worked', 'blocked', 'cancelled', 'expired') THEN cancelled_at ELSE now() END,
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND requisite_id = sqlc.arg(requisite_id)
  AND unassigned_at IS NULL
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: CreateRequisiteAssignment :one
INSERT INTO requisite_assignments (team_id, requisite_id, trader_id, assigned_by, comment, status, assigned_for_date, target_turnover_minor)
VALUES (sqlc.arg(team_id), sqlc.arg(requisite_id), sqlc.arg(trader_id), sqlc.arg(assigned_by), sqlc.arg(comment), sqlc.arg(status), sqlc.arg(assigned_for_date), sqlc.arg(target_turnover_minor))
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: UnassignRequisite :one
UPDATE requisite_assignments
SET unassigned_at = now(),
    status = CASE WHEN status IN ('worked', 'blocked', 'cancelled', 'expired') THEN status ELSE 'cancelled' END,
    cancelled_at = CASE WHEN status IN ('worked', 'blocked', 'cancelled', 'expired') THEN cancelled_at ELSE now() END,
    updated_at = now()
WHERE requisite_assignments.team_id = sqlc.arg(team_id)
  AND requisite_assignments.requisite_id = sqlc.arg(requisite_id)
  AND requisite_assignments.unassigned_at IS NULL
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: ListRequisiteAssignmentHistory :many
SELECT id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at
FROM requisite_assignments
WHERE team_id = sqlc.arg(team_id)
  AND requisite_id = sqlc.arg(requisite_id)
ORDER BY assigned_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountRequisiteAssignmentHistory :one
SELECT count(*)::bigint
FROM requisite_assignments
WHERE team_id = sqlc.arg(team_id)
  AND requisite_id = sqlc.arg(requisite_id);

-- name: ListActiveRequisiteAssignmentsByTrader :many
SELECT id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at
FROM requisite_assignments
WHERE team_id = $1
  AND trader_id = $2
  AND unassigned_at IS NULL
  AND status IN ('planned', 'assigned', 'in_work')
ORDER BY assigned_at DESC, id DESC;

-- name: GetRequisiteAssignmentByIDForTeam :one
SELECT id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at
FROM requisite_assignments
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(assignment_id);

-- name: UpdateRequisiteAssignmentPlan :one
UPDATE requisite_assignments
SET trader_id = sqlc.arg(trader_id),
    requisite_id = sqlc.arg(requisite_id),
    assigned_for_date = sqlc.arg(assigned_for_date),
    target_turnover_minor = sqlc.arg(target_turnover_minor),
    comment = sqlc.arg(comment),
    status = CASE WHEN status = 'planned' THEN 'planned' ELSE status END,
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(assignment_id)
  AND status IN ('planned', 'assigned')
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: CancelRequisiteAssignmentPlan :one
UPDATE requisite_assignments
SET status = 'cancelled',
    cancelled_at = now(),
    unassigned_at = COALESCE(unassigned_at, now()),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(assignment_id)
  AND status IN ('planned', 'assigned')
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: StartRequisiteAssignmentWork :one
UPDATE requisite_assignments
SET status = 'in_work',
    started_at = COALESCE(started_at, now()),
    shift_requisite_id = sqlc.arg(shift_requisite_id),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(assignment_id)
  AND status IN ('planned', 'assigned')
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: CompleteRequisiteAssignmentWork :one
UPDATE requisite_assignments
SET status = CASE WHEN sqlc.arg(blocked)::boolean THEN 'blocked' ELSE 'worked' END,
    completed_at = COALESCE(completed_at, COALESCE(sqlc.narg(completed_at), now())),
    unassigned_at = COALESCE(unassigned_at, COALESCE(sqlc.narg(completed_at), now())),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND shift_requisite_id = sqlc.arg(shift_requisite_id)
  AND status = 'in_work'
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment, status, assigned_for_date, target_turnover_minor, started_at, completed_at, cancelled_at, shift_requisite_id, updated_at;

-- name: CreateRequisiteAssignmentEvent :one
INSERT INTO requisite_assignment_events (team_id, assignment_id, actor_id, action, before_json, after_json, comment)
VALUES (sqlc.arg(team_id), sqlc.arg(assignment_id), sqlc.arg(actor_id), sqlc.arg(action), sqlc.arg(before_json), sqlc.arg(after_json), sqlc.arg(comment))
RETURNING id, team_id, assignment_id, actor_id, action, before_json, after_json, comment, created_at;

-- name: ListRequisiteAssignmentEvents :many
SELECT id, team_id, assignment_id, actor_id, action, before_json, after_json, comment, created_at
FROM requisite_assignment_events
WHERE team_id = sqlc.arg(team_id)
  AND assignment_id = sqlc.arg(assignment_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountRequisiteAssignmentEvents :one
SELECT count(*)::bigint
FROM requisite_assignment_events
WHERE team_id = sqlc.arg(team_id)
  AND assignment_id = sqlc.arg(assignment_id);

-- name: ListTeamleadRequisitePlans :many
SELECT
    ra.id AS assignment_id,
    ra.team_id,
    ra.requisite_id,
    r.phone,
    r.bank_code,
    ''::text AS bank_name,
    r.proxy,
    ra.trader_id,
    u.login AS trader_login,
    ra.status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    COALESCE(sr.inbound_turnover_minor, 0) AS inbound_turnover_minor,
    COALESCE(sr.outbound_turnover_minor, 0) AS outbound_turnover_minor,
    COALESCE(sr.closing_balance_minor, 0) AS closing_balance_minor,
    ra.comment,
    ra.assigned_by,
    ra.assigned_at,
    ra.started_at,
    ra.completed_at,
    ra.updated_at,
    ra.shift_requisite_id
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
JOIN users u ON u.id = ra.trader_id
LEFT JOIN shift_requisites sr ON sr.id = ra.shift_requisite_id
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.status IN ('planned', 'assigned')
ORDER BY ra.assigned_for_date DESC, ra.updated_at DESC, ra.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountTeamleadRequisitePlans :one
SELECT count(*)::bigint
FROM requisite_assignments ra
WHERE ra.team_id = sqlc.arg(team_id)
  AND ra.status IN ('planned', 'assigned');

-- name: ListTeamleadRequisiteActivity :many
SELECT
    ra.id AS assignment_id,
    ra.team_id,
    ra.requisite_id,
    r.phone,
    r.bank_code,
    ''::text AS bank_name,
    r.proxy,
    ra.trader_id,
    u.login AS trader_login,
    ra.status,
    ra.assigned_for_date,
    ra.target_turnover_minor,
    COALESCE(sr.inbound_turnover_minor, 0) AS inbound_turnover_minor,
    COALESCE(sr.outbound_turnover_minor, 0) AS outbound_turnover_minor,
    COALESCE(sr.closing_balance_minor, 0) AS closing_balance_minor,
    sr.card_number,
    sr.holder_name,
    sr.taken_at,
    sr.released_at,
    ra.comment,
    ra.assigned_at,
    ra.started_at,
    ra.completed_at,
    ra.shift_requisite_id
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
JOIN users u ON u.id = ra.trader_id
LEFT JOIN shift_requisites sr ON sr.id = ra.shift_requisite_id
WHERE ra.team_id = sqlc.arg(team_id)
  AND (
      sqlc.arg(bank_code)::text = ''
      OR sqlc.arg(bank_code)::text = 'all'
      OR r.bank_code = sqlc.arg(bank_code)::text
  )
  AND (
      sqlc.arg(status)::text = ''
      OR sqlc.arg(status)::text = 'all'
      OR r.status = sqlc.arg(status)::text
  )
  AND (
      sqlc.arg(trader_filter)::text = ''
      OR sqlc.arg(trader_filter)::text = 'all'
      OR (sqlc.arg(trader_filter)::text NOT IN ('unassigned', 'all') AND ra.trader_id::text = sqlc.arg(trader_filter)::text)
  )
  AND (
      sqlc.arg(search)::text = ''
      OR lower(r.phone) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          COALESCE(cardinality(sqlc.arg(bank_search_codes)::text[]), 0) > 0
          AND r.bank_code = ANY(sqlc.arg(bank_search_codes)::text[])
      )
      OR lower(COALESCE(r.proxy, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.employee_comment, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(sr.holder_name, r.holder_name, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(sr.card_number, r.card_number, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(u.login) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          sqlc.arg(search_digits)::text <> ''
          AND (
              regexp_replace(r.phone, '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
              OR regexp_replace(COALESCE(sr.card_number, r.card_number, ''), '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
          )
      )
  )
ORDER BY COALESCE(ra.completed_at, ra.started_at, ra.assigned_at) DESC, ra.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountTeamleadRequisiteActivity :one
SELECT count(*)::bigint
FROM requisite_assignments ra
JOIN requisites r ON r.id = ra.requisite_id
JOIN users u ON u.id = ra.trader_id
LEFT JOIN shift_requisites sr ON sr.id = ra.shift_requisite_id
WHERE ra.team_id = sqlc.arg(team_id)
  AND (
      sqlc.arg(bank_code)::text = ''
      OR sqlc.arg(bank_code)::text = 'all'
      OR r.bank_code = sqlc.arg(bank_code)::text
  )
  AND (
      sqlc.arg(status)::text = ''
      OR sqlc.arg(status)::text = 'all'
      OR r.status = sqlc.arg(status)::text
  )
  AND (
      sqlc.arg(trader_filter)::text = ''
      OR sqlc.arg(trader_filter)::text = 'all'
      OR (sqlc.arg(trader_filter)::text NOT IN ('unassigned', 'all') AND ra.trader_id::text = sqlc.arg(trader_filter)::text)
  )
  AND (
      sqlc.arg(search)::text = ''
      OR lower(r.phone) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          COALESCE(cardinality(sqlc.arg(bank_search_codes)::text[]), 0) > 0
          AND r.bank_code = ANY(sqlc.arg(bank_search_codes)::text[])
      )
      OR lower(COALESCE(r.proxy, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(r.employee_comment, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(sr.holder_name, r.holder_name, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(COALESCE(sr.card_number, r.card_number, '')) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR lower(u.login) LIKE '%' || lower(sqlc.arg(search)::text) || '%'
      OR (
          sqlc.arg(search_digits)::text <> ''
          AND (
              regexp_replace(r.phone, '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
              OR regexp_replace(COALESCE(sr.card_number, r.card_number, ''), '[^0-9]', '', 'g') LIKE '%' || sqlc.arg(search_digits)::text || '%'
          )
      )
  );

-- name: GetRequisiteReportSummary :one
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.bank_code,
    ''::text AS bank_name,
    r.proxy,
    r.employee_comment,
    r.holder_name,
    r.card_number,
    r.status,
    r.total_inbound_turnover_minor,
    r.total_outbound_turnover_minor,
    r.last_closing_balance_minor,
    COALESCE(latest_assignment.status, r.last_activity_status, '') AS latest_status,
    r.last_activity_at,
    r.last_shift_requisite_id
FROM requisites r
LEFT JOIN LATERAL (
    SELECT ra.status
    FROM requisite_assignments ra
    WHERE ra.team_id = r.team_id
      AND ra.requisite_id = r.id
    ORDER BY COALESCE(ra.completed_at, ra.cancelled_at, ra.unassigned_at, ra.updated_at, ra.assigned_at) DESC, ra.id DESC
    LIMIT 1
) latest_assignment ON true
WHERE r.team_id = sqlc.arg(team_id)
  AND r.id = sqlc.arg(requisite_id)
  AND r.deleted_at IS NULL;

-- name: ListRequisiteReportShifts :many
SELECT
    sr.id AS shift_requisite_id,
    sr.shift_id,
    sr.team_id,
    sr.requisite_id,
    sr.trader_id,
    u.login AS trader_login,
    ts.started_at AS shift_started_at,
    ts.closed_at AS shift_closed_at,
    ts.status AS shift_status,
    sr.taken_at,
    sr.released_at,
    sr.status AS requisite_status,
    sr.inbound_turnover_minor,
    sr.outbound_turnover_minor,
    COALESCE(ra.target_turnover_minor, 0) AS target_turnover_minor,
    sr.closing_balance_minor,
    sr.card_number,
    sr.holder_name,
    ra.assigned_for_date,
    COALESCE(ra.status, '') AS assignment_status
FROM shift_requisites sr
JOIN trader_shifts ts ON ts.id = sr.shift_id
JOIN users u ON u.id = sr.trader_id
LEFT JOIN requisite_assignments ra ON ra.id = sr.assignment_id
WHERE sr.team_id = sqlc.arg(team_id)
  AND sr.requisite_id = sqlc.arg(requisite_id)
ORDER BY COALESCE(ra.assigned_for_date, sr.taken_at::date) ASC, sr.taken_at ASC, sr.id ASC;
