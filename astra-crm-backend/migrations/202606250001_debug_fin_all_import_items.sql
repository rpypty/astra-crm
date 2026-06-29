-- +goose Up
-- +goose StatementBegin
CREATE TABLE debug_fin_all_import_items (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    source_hash TEXT NOT NULL,
    source_sheet TEXT NOT NULL,
    source_row BIGINT NOT NULL,
    source_circle BIGINT NOT NULL,
    trader_id BIGINT NOT NULL REFERENCES users(id),
    requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    assignment_id BIGINT REFERENCES requisite_assignments(id),
    shift_id BIGINT REFERENCES trader_shifts(id),
    shift_requisite_id BIGINT REFERENCES shift_requisites(id),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, source_hash, source_sheet, source_row, source_circle)
);

CREATE INDEX idx_debug_fin_all_import_items_team_created
ON debug_fin_all_import_items(team_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS debug_fin_all_import_items;
-- +goose StatementEnd
