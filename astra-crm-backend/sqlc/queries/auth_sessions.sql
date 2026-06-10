-- name: CreateAuthSession :one
INSERT INTO auth_sessions (
    user_id,
    token_hash,
    user_agent,
    ip,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, user_id, token_hash, user_agent, ip, expires_at, created_at, revoked_at;

-- name: GetValidAuthSessionByTokenHash :one
SELECT id, user_id, token_hash, user_agent, ip, expires_at, created_at, revoked_at
FROM auth_sessions
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RevokeAuthSessionByTokenHash :execrows
UPDATE auth_sessions
SET revoked_at = now()
WHERE token_hash = $1
  AND revoked_at IS NULL;

-- name: RevokeAuthSessionsByUserID :execrows
UPDATE auth_sessions
SET revoked_at = now()
WHERE user_id = $1
  AND revoked_at IS NULL;
