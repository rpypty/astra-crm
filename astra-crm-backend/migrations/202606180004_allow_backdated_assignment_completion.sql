-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    constraint_name text;
BEGIN
    FOR constraint_name IN
        SELECT con.conname
        FROM pg_constraint con
        WHERE con.conrelid = 'requisite_assignments'::regclass
          AND con.contype = 'c'
          AND pg_get_constraintdef(con.oid) ILIKE '%unassigned_at%'
          AND pg_get_constraintdef(con.oid) ILIKE '%assigned_at%'
    LOOP
        EXECUTE format('ALTER TABLE requisite_assignments DROP CONSTRAINT %I', constraint_name);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE requisite_assignments
    ADD CONSTRAINT requisite_assignments_unassigned_at_after_assigned_at_check
    CHECK (unassigned_at IS NULL OR unassigned_at >= assigned_at);
-- +goose StatementEnd
