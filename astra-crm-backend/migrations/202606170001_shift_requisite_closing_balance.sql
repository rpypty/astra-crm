-- +goose Up
-- +goose StatementBegin
ALTER TABLE shift_requisites
    ADD COLUMN closing_balance_minor BIGINT NOT NULL DEFAULT 0 CHECK (closing_balance_minor >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE shift_requisites
    DROP COLUMN IF EXISTS closing_balance_minor;
-- +goose StatementEnd
