-- name: InsertAuditLog :one
INSERT INTO audit_logs (
    team_id,
    actor_id,
    action,
    entity_type,
    entity_id,
    before_json,
    after_json,
    changed_fields_json,
    comment
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id, team_id, actor_id, action, entity_type, entity_id, before_json, after_json, changed_fields_json, comment, created_at;

-- name: ListAuditLogsByTeam :many
SELECT id, team_id, actor_id, action, entity_type, entity_id, before_json, after_json, changed_fields_json, comment, created_at
FROM audit_logs
WHERE team_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByEntity :many
SELECT id, team_id, actor_id, action, entity_type, entity_id, before_json, after_json, changed_fields_json, comment, created_at
FROM audit_logs
WHERE team_id = $1
  AND entity_type = $2
  AND entity_id = $3
ORDER BY created_at DESC, id DESC
LIMIT $4 OFFSET $5;
