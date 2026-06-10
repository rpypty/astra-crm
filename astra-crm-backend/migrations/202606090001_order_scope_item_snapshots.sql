-- +goose Up
-- +goose StatementBegin
ALTER TABLE order_scope_items
    ADD COLUMN external_id TEXT,
    ADD COLUMN external_inner_id TEXT,
    ADD COLUMN worker_name TEXT,
    ADD COLUMN trader_id BIGINT REFERENCES users(id),
    ADD COLUMN requisite_raw TEXT,
    ADD COLUMN requisite_phone TEXT,
    ADD COLUMN method_type TEXT,
    ADD COLUMN method_name TEXT,
    ADD COLUMN amount_minor BIGINT,
    ADD COLUMN currency TEXT,
    ADD COLUMN raw_status TEXT,
    ADD COLUMN normalized_status TEXT,
    ADD COLUMN created_at_external TIMESTAMPTZ;

UPDATE order_scope_items osi
SET
    external_id = eo.external_id,
    external_inner_id = eo.external_inner_id,
    worker_name = eo.worker_name,
    trader_id = eo.trader_id,
    requisite_raw = eo.requisite_raw,
    requisite_phone = eo.requisite_phone,
    method_type = eo.method_type,
    method_name = eo.method_name,
    amount_minor = eo.amount_minor,
    currency = eo.currency,
    raw_status = eo.raw_status,
    normalized_status = eo.normalized_status,
    created_at_external = eo.created_at_external
FROM external_orders eo
WHERE eo.id = osi.external_order_id;

ALTER TABLE order_scope_items
    ALTER COLUMN external_id SET NOT NULL,
    ALTER COLUMN external_inner_id SET NOT NULL,
    ALTER COLUMN worker_name SET NOT NULL,
    ALTER COLUMN amount_minor SET NOT NULL,
    ALTER COLUMN currency SET NOT NULL,
    ALTER COLUMN raw_status SET NOT NULL,
    ALTER COLUMN normalized_status SET NOT NULL,
    ALTER COLUMN created_at_external SET NOT NULL,
    ADD CONSTRAINT chk_order_scope_amount_non_negative CHECK (amount_minor >= 0),
    ADD CONSTRAINT chk_order_scope_normalized_status CHECK (normalized_status IN ('success', 'corrected', 'failed', 'cancelled', 'unknown'));

CREATE INDEX idx_order_scope_snapshot_inner_id
ON order_scope_items(team_id, direction, external_inner_id)
WHERE is_active = TRUE;

CREATE INDEX idx_order_scope_snapshot_worker
ON order_scope_items(team_id, direction, worker_name)
WHERE is_active = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_order_scope_snapshot_worker;
DROP INDEX IF EXISTS idx_order_scope_snapshot_inner_id;

ALTER TABLE order_scope_items
    DROP CONSTRAINT IF EXISTS chk_order_scope_normalized_status,
    DROP CONSTRAINT IF EXISTS chk_order_scope_amount_non_negative,
    DROP COLUMN IF EXISTS normalized_status,
    DROP COLUMN IF EXISTS raw_status,
    DROP COLUMN IF EXISTS created_at_external,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS amount_minor,
    DROP COLUMN IF EXISTS method_name,
    DROP COLUMN IF EXISTS method_type,
    DROP COLUMN IF EXISTS requisite_phone,
    DROP COLUMN IF EXISTS requisite_raw,
    DROP COLUMN IF EXISTS trader_id,
    DROP COLUMN IF EXISTS worker_name,
    DROP COLUMN IF EXISTS external_inner_id,
    DROP COLUMN IF EXISTS external_id;
-- +goose StatementEnd
