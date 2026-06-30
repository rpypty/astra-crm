-- name: ListActiveBanks :many
SELECT id, code, name, status, sort_order, created_at, csv_alias, updated_at
FROM banks
WHERE status = 'active'
ORDER BY sort_order, name;

-- name: UpdateBankCSVAlias :one
UPDATE banks
SET
    csv_alias = NULLIF(btrim(sqlc.arg(csv_alias)::text), ''),
    updated_at = now()
WHERE code = sqlc.arg(code)
  AND status = 'active'
RETURNING id, code, name, status, sort_order, created_at, csv_alias, updated_at;
