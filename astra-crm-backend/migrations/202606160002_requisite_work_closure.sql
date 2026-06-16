-- +goose Up
-- +goose StatementBegin
ALTER TABLE shift_requisites
    DROP CONSTRAINT IF EXISTS shift_requisites_status_check,
    ADD COLUMN inbound_turnover_minor BIGINT NOT NULL DEFAULT 0 CHECK (inbound_turnover_minor >= 0),
    ADD COLUMN outbound_turnover_minor BIGINT NOT NULL DEFAULT 0 CHECK (outbound_turnover_minor >= 0),
    ADD CONSTRAINT shift_requisites_status_check CHECK (status IN ('active', 'worked', 'correction', 'released', 'blocked'));

ALTER TABLE requisites
    DROP CONSTRAINT IF EXISTS requisites_status_check,
    ADD CONSTRAINT requisites_status_check CHECK (status IN ('active', 'disabled', 'archived', 'blocked'));

DROP INDEX IF EXISTS uq_shift_requisite_once;

CREATE UNIQUE INDEX uq_shift_requisite_active_once
ON shift_requisites(shift_id, requisite_id)
WHERE status IN ('active', 'correction');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_shift_requisite_active_once;

CREATE UNIQUE INDEX uq_shift_requisite_once ON shift_requisites(shift_id, requisite_id);

ALTER TABLE requisites
    DROP CONSTRAINT IF EXISTS requisites_status_check,
    ADD CONSTRAINT requisites_status_check CHECK (status IN ('active', 'disabled', 'archived'));

ALTER TABLE shift_requisites
    DROP CONSTRAINT IF EXISTS shift_requisites_status_check,
    DROP COLUMN IF EXISTS outbound_turnover_minor,
    DROP COLUMN IF EXISTS inbound_turnover_minor,
    ADD CONSTRAINT shift_requisites_status_check CHECK (status IN ('active', 'released'));
-- +goose StatementEnd
