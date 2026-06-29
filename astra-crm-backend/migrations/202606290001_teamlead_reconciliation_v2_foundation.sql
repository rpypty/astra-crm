-- +goose Up
-- +goose StatementBegin
ALTER TABLE requisites
    DROP CONSTRAINT IF EXISTS chk_requisites_details_consistent;

ALTER TABLE requisites
    ADD COLUMN normalized_phone TEXT GENERATED ALWAYS AS (
        CASE
            WHEN length(regexp_replace(phone, '[^0-9]', '', 'g')) = 10
                THEN '7' || regexp_replace(phone, '[^0-9]', '', 'g')
            WHEN length(regexp_replace(phone, '[^0-9]', '', 'g')) = 11
                AND left(regexp_replace(phone, '[^0-9]', '', 'g'), 1) = '8'
                THEN '7' || right(regexp_replace(phone, '[^0-9]', '', 'g'), 10)
            ELSE regexp_replace(phone, '[^0-9]', '', 'g')
        END
    ) STORED,
    ADD COLUMN normalized_card_number TEXT GENERATED ALWAYS AS (
        regexp_replace(COALESCE(card_number, ''), '[^0-9]', '', 'g')
    ) STORED,
    ADD CONSTRAINT chk_requisites_normalized_phone_not_empty CHECK (normalized_phone <> ''),
    ADD CONSTRAINT chk_requisites_details_consistent CHECK (
        (holder_name IS NULL AND details_filled_at IS NULL AND details_filled_by IS NULL)
        OR
        (holder_name IS NOT NULL AND details_filled_at IS NOT NULL AND details_filled_by IS NOT NULL)
    );

DROP INDEX IF EXISTS uq_requisites_active_team_phone_bank;

CREATE UNIQUE INDEX uq_requisites_active_identity
ON requisites(team_id, bank_code, normalized_phone, normalized_card_number)
WHERE deleted_at IS NULL;

CREATE INDEX idx_requisites_active_card_identity
ON requisites(team_id, bank_code, normalized_card_number)
WHERE deleted_at IS NULL
  AND normalized_card_number <> '';

CREATE INDEX idx_requisites_active_phone_identity
ON requisites(team_id, bank_code, normalized_phone)
WHERE deleted_at IS NULL;

CREATE TABLE teamlead_reconciliations (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'analyzing', 'matched', 'mismatch', 'apply_queued', 'applying', 'applied', 'apply_failed', 'rejected')),
    created_by BIGINT NOT NULL REFERENCES users(id),
    confirmed_by BIGINT REFERENCES users(id),
    rejected_by BIGINT REFERENCES users(id),
    inbound_import_batch_id BIGINT REFERENCES import_batches(id),
    outbound_import_batch_id BIGINT REFERENCES import_batches(id),
    comment TEXT,
    mismatch_count BIGINT NOT NULL DEFAULT 0 CHECK (mismatch_count >= 0),
    conflict_count BIGINT NOT NULL DEFAULT 0 CHECK (conflict_count >= 0),
    blocked_count BIGINT NOT NULL DEFAULT 0 CHECK (blocked_count >= 0),
    pipeline_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    inbound_summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    outbound_summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    preview_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    apply_result_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    analyzed_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    apply_queued_at TIMESTAMPTZ,
    applied_at TIMESTAMPTZ,
    CHECK (date_to >= date_from),
    CHECK (inbound_import_batch_id IS NOT NULL OR outbound_import_batch_id IS NOT NULL),
    CHECK ((status = 'rejected') = (rejected_at IS NOT NULL)),
    CHECK (confirmed_at IS NULL OR confirmed_by IS NOT NULL),
    CHECK (rejected_at IS NULL OR rejected_by IS NOT NULL)
);

CREATE INDEX idx_teamlead_reconciliations_team_created
ON teamlead_reconciliations(team_id, created_at DESC, id DESC);

CREATE INDEX idx_teamlead_reconciliations_team_period
ON teamlead_reconciliations(team_id, date_from, date_to, created_at DESC, id DESC);

CREATE INDEX idx_teamlead_reconciliations_team_status
ON teamlead_reconciliations(team_id, status, created_at DESC, id DESC);

CREATE TABLE teamlead_reconciliation_items (
    id BIGSERIAL PRIMARY KEY,
    teamlead_reconciliation_id BIGINT NOT NULL REFERENCES teamlead_reconciliations(id),
    team_id BIGINT NOT NULL REFERENCES teams(id),
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    stage TEXT NOT NULL CHECK (stage IN ('normalization', 'matching', 'turnover_check', 'transaction_check', 'preview', 'apply')),
    issue_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'blocker')),
    external_order_id BIGINT REFERENCES external_orders(id),
    external_inner_id TEXT,
    trader_id BIGINT REFERENCES users(id),
    requisite_id BIGINT REFERENCES requisites(id),
    shift_id BIGINT REFERENCES trader_shifts(id),
    before_json JSONB,
    after_json JSONB,
    message TEXT,
    is_blocking BOOLEAN NOT NULL DEFAULT FALSE,
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_teamlead_reconciliation_items_run
ON teamlead_reconciliation_items(teamlead_reconciliation_id, id);

CREATE INDEX idx_teamlead_reconciliation_items_filters
ON teamlead_reconciliation_items(team_id, direction, stage, issue_type, severity, created_at DESC, id DESC);

CREATE INDEX idx_teamlead_reconciliation_items_mismatches
ON teamlead_reconciliation_items(teamlead_reconciliation_id, direction, severity, id)
WHERE severity IN ('warning', 'error', 'blocker') OR is_blocking = TRUE;

CREATE INDEX idx_teamlead_reconciliation_items_trader
ON teamlead_reconciliation_items(team_id, trader_id, created_at DESC, id DESC)
WHERE trader_id IS NOT NULL;

CREATE INDEX idx_teamlead_reconciliation_items_requisite
ON teamlead_reconciliation_items(team_id, requisite_id, created_at DESC, id DESC)
WHERE requisite_id IS NOT NULL;

ALTER TABLE external_orders
    ADD COLUMN tl_reconciliation_status TEXT NOT NULL DEFAULT 'not_checked'
        CHECK (tl_reconciliation_status IN ('not_checked', 'confirmed_by_tl', 'updated_by_tl', 'tl_discrepancy', 'tl_accepted')),
    ADD COLUMN last_teamlead_reconciliation_id BIGINT REFERENCES teamlead_reconciliations(id),
    ADD COLUMN tl_reconciled_at TIMESTAMPTZ;

CREATE INDEX idx_external_orders_tl_reconciliation_status
ON external_orders(team_id, tl_reconciliation_status, updated_at DESC, id DESC);

ALTER TABLE trader_shifts
    ADD COLUMN tl_reconciliation_status TEXT NOT NULL DEFAULT 'not_checked'
        CHECK (tl_reconciliation_status IN ('not_checked', 'confirmed_by_tl', 'updated_by_tl', 'tl_discrepancy', 'tl_accepted')),
    ADD COLUMN last_teamlead_reconciliation_id BIGINT REFERENCES teamlead_reconciliations(id),
    ADD COLUMN tl_reconciled_at TIMESTAMPTZ;

CREATE INDEX idx_trader_shifts_tl_reconciliation_status
ON trader_shifts(team_id, tl_reconciliation_status, started_at DESC, id DESC);

ALTER TABLE shift_requisites
    ADD COLUMN tl_reconciliation_status TEXT NOT NULL DEFAULT 'not_checked'
        CHECK (tl_reconciliation_status IN ('not_checked', 'confirmed_by_tl', 'updated_by_tl', 'tl_discrepancy', 'tl_accepted')),
    ADD COLUMN last_teamlead_reconciliation_id BIGINT REFERENCES teamlead_reconciliations(id),
    ADD COLUMN tl_reconciled_at TIMESTAMPTZ;

CREATE INDEX idx_shift_requisites_tl_reconciliation_status
ON shift_requisites(team_id, tl_reconciliation_status, updated_at DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_shift_requisites_tl_reconciliation_status;

ALTER TABLE shift_requisites
    DROP COLUMN IF EXISTS tl_reconciled_at,
    DROP COLUMN IF EXISTS last_teamlead_reconciliation_id,
    DROP COLUMN IF EXISTS tl_reconciliation_status;

DROP INDEX IF EXISTS idx_trader_shifts_tl_reconciliation_status;

ALTER TABLE trader_shifts
    DROP COLUMN IF EXISTS tl_reconciled_at,
    DROP COLUMN IF EXISTS last_teamlead_reconciliation_id,
    DROP COLUMN IF EXISTS tl_reconciliation_status;

DROP INDEX IF EXISTS idx_external_orders_tl_reconciliation_status;

ALTER TABLE external_orders
    DROP COLUMN IF EXISTS tl_reconciled_at,
    DROP COLUMN IF EXISTS last_teamlead_reconciliation_id,
    DROP COLUMN IF EXISTS tl_reconciliation_status;

DROP TABLE IF EXISTS teamlead_reconciliation_items;
DROP TABLE IF EXISTS teamlead_reconciliations;

DROP INDEX IF EXISTS idx_requisites_active_phone_identity;
DROP INDEX IF EXISTS idx_requisites_active_card_identity;
DROP INDEX IF EXISTS uq_requisites_active_identity;

ALTER TABLE requisites
    DROP CONSTRAINT IF EXISTS chk_requisites_details_consistent,
    DROP CONSTRAINT IF EXISTS chk_requisites_normalized_phone_not_empty,
    DROP COLUMN IF EXISTS normalized_card_number,
    DROP COLUMN IF EXISTS normalized_phone,
    ADD CONSTRAINT chk_requisites_details_consistent CHECK (
        (holder_name IS NULL AND card_number IS NULL AND details_filled_at IS NULL AND details_filled_by IS NULL)
        OR
        (holder_name IS NOT NULL AND card_number IS NOT NULL AND details_filled_at IS NOT NULL AND details_filled_by IS NOT NULL)
    );

CREATE UNIQUE INDEX uq_requisites_active_team_phone_bank
ON requisites(team_id, phone, bank_code)
WHERE deleted_at IS NULL;
-- +goose StatementEnd
