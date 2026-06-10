-- name: GetRequisiteByIDForTeam :one
SELECT id, team_id, phone, method_type, proxy, status, created_by, created_at, updated_at, deleted_at
FROM requisites
WHERE id = $1
  AND team_id = $2
  AND deleted_at IS NULL;

-- name: CreateRequisite :one
INSERT INTO requisites (team_id, phone, method_type, proxy, created_by)
VALUES (sqlc.arg(team_id), sqlc.arg(phone), sqlc.arg(method_type), sqlc.arg(proxy), sqlc.arg(created_by))
RETURNING id, team_id, phone, method_type, proxy, status, created_by, created_at, updated_at, deleted_at;

-- name: GetRequisiteDetailsByIDForTeam :one
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.proxy,
    r.status,
    r.created_by,
    r.created_at,
    r.updated_at,
    r.deleted_at,
    ra.id AS active_assignment_id,
    ra.trader_id AS assigned_trader_id,
    u.login AS assigned_trader_login
FROM requisites r
LEFT JOIN requisite_assignments ra ON ra.requisite_id = r.id AND ra.unassigned_at IS NULL
LEFT JOIN users u ON u.id = ra.trader_id
WHERE r.team_id = sqlc.arg(team_id)
  AND r.id = sqlc.arg(requisite_id)
  AND r.deleted_at IS NULL;

-- name: ListRequisiteDetailsByTeam :many
SELECT
    r.id,
    r.team_id,
    r.phone,
    r.method_type,
    r.proxy,
    r.status,
    r.created_by,
    r.created_at,
    r.updated_at,
    r.deleted_at,
    ra.id AS active_assignment_id,
    ra.trader_id AS assigned_trader_id,
    u.login AS assigned_trader_login
FROM requisites r
LEFT JOIN requisite_assignments ra ON ra.requisite_id = r.id AND ra.unassigned_at IS NULL
LEFT JOIN users u ON u.id = ra.trader_id
WHERE r.team_id = sqlc.arg(team_id)
  AND r.deleted_at IS NULL
ORDER BY r.id;

-- name: UpdateRequisite :one
UPDATE requisites
SET
    phone = sqlc.arg(phone),
    method_type = sqlc.arg(method_type),
    proxy = sqlc.arg(proxy),
    status = sqlc.arg(status),
    updated_at = now(),
    deleted_at = CASE
        WHEN sqlc.arg(status) = 'archived' THEN COALESCE(requisites.deleted_at, now())
        ELSE NULL
    END
WHERE requisites.team_id = sqlc.arg(team_id)
  AND requisites.id = sqlc.arg(requisite_id)
  AND requisites.deleted_at IS NULL
RETURNING id, team_id, phone, method_type, proxy, status, created_by, created_at, updated_at, deleted_at;

-- name: AssignRequisite :one
WITH closed_assignment AS (
    UPDATE requisite_assignments
    SET unassigned_at = now()
    WHERE requisite_assignments.team_id = sqlc.arg(team_id)
      AND requisite_assignments.requisite_id = sqlc.arg(requisite_id)
      AND requisite_assignments.unassigned_at IS NULL
    RETURNING id
),
created_assignment AS (
    INSERT INTO requisite_assignments (team_id, requisite_id, trader_id, assigned_by, comment)
    VALUES (sqlc.arg(team_id), sqlc.arg(requisite_id), sqlc.arg(trader_id), sqlc.arg(assigned_by), sqlc.arg(comment))
    RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment
)
SELECT
    created_assignment.id,
    created_assignment.team_id,
    created_assignment.requisite_id,
    created_assignment.trader_id,
    created_assignment.assigned_by,
    created_assignment.assigned_at,
    created_assignment.unassigned_at,
    created_assignment.comment,
    EXISTS(SELECT 1 FROM closed_assignment) AS was_reassign
FROM created_assignment;

-- name: UnassignRequisite :one
UPDATE requisite_assignments
SET unassigned_at = now()
WHERE requisite_assignments.team_id = sqlc.arg(team_id)
  AND requisite_assignments.requisite_id = sqlc.arg(requisite_id)
  AND requisite_assignments.unassigned_at IS NULL
RETURNING id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment;

-- name: ListRequisiteAssignmentHistory :many
SELECT id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment
FROM requisite_assignments
WHERE team_id = sqlc.arg(team_id)
  AND requisite_id = sqlc.arg(requisite_id)
ORDER BY assigned_at DESC, id DESC;

-- name: ListActiveRequisiteAssignmentsByTrader :many
SELECT id, team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment
FROM requisite_assignments
WHERE team_id = $1
  AND trader_id = $2
  AND unassigned_at IS NULL
ORDER BY assigned_at DESC, id DESC;
