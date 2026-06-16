-- name: ListActiveBanks :many
SELECT id, code, name, status, sort_order, created_at
FROM banks
WHERE status = 'active'
ORDER BY sort_order, name;
