-- name: GetTeamByID :one
SELECT id, name, status, created_at, updated_at
FROM teams
WHERE id = $1;
