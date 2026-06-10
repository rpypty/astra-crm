-- name: GetUserByID :one
SELECT id, team_id, role, login, password_hash, status, created_at, updated_at, deleted_at
FROM users
WHERE id = $1;

-- name: GetUserByLogin :one
SELECT id, team_id, role, login, password_hash, status, created_at, updated_at, deleted_at
FROM users
WHERE login = $1
LIMIT 1;

-- name: ListTradersByTeam :many
SELECT u.id, u.team_id, u.role, u.login, u.password_hash, u.status, u.created_at, u.updated_at, u.deleted_at
FROM users u
WHERE u.team_id = $1
  AND u.role = 'trader'
  AND u.deleted_at IS NULL
ORDER BY u.id;

-- name: GetTraderProfileByUserID :one
SELECT id, user_id, salary_rate_bps, external_worker_name, created_at, updated_at
FROM trader_profiles
WHERE user_id = $1;

-- name: CreateTrader :one
WITH created_user AS (
    INSERT INTO users (team_id, role, login, password_hash, status)
    VALUES (sqlc.arg(team_id), 'trader', sqlc.arg(login), sqlc.arg(password_hash), 'active')
    RETURNING id, team_id, role, login, status, created_at, updated_at, deleted_at
),
created_profile AS (
    INSERT INTO trader_profiles (user_id, salary_rate_bps, external_worker_name)
    SELECT id, sqlc.arg(salary_rate_bps), sqlc.arg(external_worker_name)
    FROM created_user
    RETURNING id, user_id, salary_rate_bps, external_worker_name, created_at, updated_at
)
SELECT
    created_user.id,
    created_user.team_id,
    created_user.role,
    created_user.login,
    created_user.status,
    created_user.created_at,
    created_user.updated_at,
    created_user.deleted_at,
    created_profile.id AS profile_id,
    created_profile.salary_rate_bps,
    created_profile.external_worker_name,
    created_profile.created_at AS profile_created_at,
    created_profile.updated_at AS profile_updated_at
FROM created_user
CROSS JOIN created_profile;

-- name: GetTraderByIDForTeam :one
SELECT
    u.id,
    u.team_id,
    u.role,
    u.login,
    u.status,
    u.created_at,
    u.updated_at,
    u.deleted_at,
    p.id AS profile_id,
    p.salary_rate_bps,
    p.external_worker_name,
    p.created_at AS profile_created_at,
    p.updated_at AS profile_updated_at
FROM users u
JOIN trader_profiles p ON p.user_id = u.id
WHERE u.team_id = sqlc.arg(team_id)
  AND u.id = sqlc.arg(trader_id)
  AND u.role = 'trader'
  AND u.deleted_at IS NULL;

-- name: ListTraderDetailsByTeam :many
SELECT
    u.id,
    u.team_id,
    u.role,
    u.login,
    u.status,
    u.created_at,
    u.updated_at,
    u.deleted_at,
    p.id AS profile_id,
    p.salary_rate_bps,
    p.external_worker_name,
    p.created_at AS profile_created_at,
    p.updated_at AS profile_updated_at
FROM users u
JOIN trader_profiles p ON p.user_id = u.id
WHERE u.team_id = $1
  AND u.role = 'trader'
  AND u.deleted_at IS NULL
ORDER BY u.id;

-- name: UpdateTrader :one
WITH updated_user AS (
    UPDATE users
    SET
        status = sqlc.arg(status),
        updated_at = now(),
        deleted_at = CASE
            WHEN sqlc.arg(status) = 'deleted' THEN COALESCE(users.deleted_at, now())
            ELSE NULL
        END
    WHERE users.team_id = sqlc.arg(team_id)
      AND users.id = sqlc.arg(trader_id)
      AND users.role = 'trader'
      AND users.deleted_at IS NULL
    RETURNING id, team_id, role, login, status, created_at, updated_at, deleted_at
),
updated_profile AS (
    UPDATE trader_profiles
    SET
        salary_rate_bps = sqlc.arg(salary_rate_bps),
        external_worker_name = sqlc.arg(external_worker_name),
        updated_at = now()
    WHERE user_id = (SELECT id FROM updated_user)
    RETURNING id, user_id, salary_rate_bps, external_worker_name, created_at, updated_at
)
SELECT
    updated_user.id,
    updated_user.team_id,
    updated_user.role,
    updated_user.login,
    updated_user.status,
    updated_user.created_at,
    updated_user.updated_at,
    updated_user.deleted_at,
    updated_profile.id AS profile_id,
    updated_profile.salary_rate_bps,
    updated_profile.external_worker_name,
    updated_profile.created_at AS profile_created_at,
    updated_profile.updated_at AS profile_updated_at
FROM updated_user
CROSS JOIN updated_profile;

-- name: UpdateTraderPasswordHash :exec
UPDATE users
SET password_hash = sqlc.arg(password_hash),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND id = sqlc.arg(trader_id)
  AND role = 'trader'
  AND deleted_at IS NULL;
