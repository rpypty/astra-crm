-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    constraint_name text;
BEGIN
    FOR constraint_name IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'import_batches'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) LIKE '%teamlead_period%'
          AND pg_get_constraintdef(oid) LIKE '%accounting_period_id IS NOT NULL%'
    LOOP
        EXECUTE format('ALTER TABLE import_batches DROP CONSTRAINT %I', constraint_name);
    END LOOP;

    FOR constraint_name IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'order_scope_items'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) LIKE '%teamlead_period%'
          AND pg_get_constraintdef(oid) LIKE '%accounting_period_id IS NOT NULL%'
    LOOP
        EXECUTE format('ALTER TABLE order_scope_items DROP CONSTRAINT %I', constraint_name);
    END LOOP;

    FOR constraint_name IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'reconciliation_runs'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) LIKE '%teamlead_period%'
          AND pg_get_constraintdef(oid) LIKE '%accounting_period_id IS NOT NULL%'
    LOOP
        EXECUTE format('ALTER TABLE reconciliation_runs DROP CONSTRAINT %I', constraint_name);
    END LOOP;
END $$;

ALTER TABLE import_batches DROP CONSTRAINT IF EXISTS import_batches_scope_shape_check;
ALTER TABLE order_scope_items DROP CONSTRAINT IF EXISTS order_scope_items_scope_shape_check;
ALTER TABLE reconciliation_runs DROP CONSTRAINT IF EXISTS reconciliation_runs_scope_shape_check;

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_scope_shape_check CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND shift_id IS NULL)
    );

ALTER TABLE order_scope_items
    ADD CONSTRAINT order_scope_items_scope_shape_check CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND shift_id IS NULL)
    );

ALTER TABLE reconciliation_runs
    ADD CONSTRAINT reconciliation_runs_scope_shape_check CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND shift_id IS NULL)
    );

CREATE INDEX IF NOT EXISTS idx_import_batches_teamlead_current_scope
ON import_batches(team_id, direction, applied_at DESC, id DESC)
WHERE scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL
  AND superseded_by_batch_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_order_scope_teamlead_current_active
ON order_scope_items(team_id, direction, external_inner_id)
WHERE scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL
  AND is_active = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_order_scope_teamlead_current_active;
DROP INDEX IF EXISTS idx_import_batches_teamlead_current_scope;

ALTER TABLE reconciliation_runs DROP CONSTRAINT IF EXISTS reconciliation_runs_scope_shape_check;
ALTER TABLE order_scope_items DROP CONSTRAINT IF EXISTS order_scope_items_scope_shape_check;
ALTER TABLE import_batches DROP CONSTRAINT IF EXISTS import_batches_scope_shape_check;

ALTER TABLE import_batches
    ADD CONSTRAINT import_batches_scope_shape_check CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND accounting_period_id IS NOT NULL AND shift_id IS NULL)
    );

ALTER TABLE order_scope_items
    ADD CONSTRAINT order_scope_items_scope_shape_check CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND accounting_period_id IS NOT NULL AND shift_id IS NULL)
    );

ALTER TABLE reconciliation_runs
    ADD CONSTRAINT reconciliation_runs_scope_shape_check CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND accounting_period_id IS NOT NULL AND shift_id IS NULL)
    );
-- +goose StatementEnd
