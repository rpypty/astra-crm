-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_team_id_login_key;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        GROUP BY login
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'users.login contains duplicates; rename duplicate users before applying global unique constraint';
    END IF;
END $$;

ALTER TABLE users
    ADD CONSTRAINT users_login_key UNIQUE (login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_login_key;

ALTER TABLE users
    ADD CONSTRAINT users_team_id_login_key UNIQUE (team_id, login);
-- +goose StatementEnd
