-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    constraint_name text;
BEGIN
    FOR constraint_name IN
        SELECT con.conname
        FROM pg_constraint con
        WHERE con.conrelid = 'shift_requisites'::regclass
          AND con.contype = 'c'
          AND pg_get_constraintdef(con.oid) ILIKE '%released_at%'
          AND pg_get_constraintdef(con.oid) ILIKE '%taken_at%'
    LOOP
        EXECUTE format('ALTER TABLE shift_requisites DROP CONSTRAINT %I', constraint_name);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE shift_requisites
    ADD CONSTRAINT shift_requisites_released_at_after_taken_at_check
    CHECK (released_at IS NULL OR released_at >= taken_at);
-- +goose StatementEnd
