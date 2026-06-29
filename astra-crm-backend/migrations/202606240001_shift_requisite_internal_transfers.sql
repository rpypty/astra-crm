-- +goose Up
-- +goose StatementBegin
CREATE TABLE shift_requisite_internal_transfers (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    shift_id BIGINT NOT NULL REFERENCES trader_shifts(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    source_shift_requisite_id BIGINT NOT NULL REFERENCES shift_requisites(id),
    source_requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    destination_shift_requisite_id BIGINT NOT NULL REFERENCES shift_requisites(id),
    destination_requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    cancelled_by BIGINT REFERENCES users(id),
    cancelled_at TIMESTAMPTZ,
    comment TEXT,
    CHECK (source_shift_requisite_id <> destination_shift_requisite_id),
    CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL))
);

CREATE INDEX idx_shift_requisite_internal_transfers_shift
ON shift_requisite_internal_transfers(team_id, shift_id, status, created_at DESC);

CREATE INDEX idx_shift_requisite_internal_transfers_source
ON shift_requisite_internal_transfers(source_shift_requisite_id, status, created_at DESC);

CREATE INDEX idx_shift_requisite_internal_transfers_destination
ON shift_requisite_internal_transfers(destination_shift_requisite_id, status, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shift_requisite_internal_transfers;
-- +goose StatementEnd
