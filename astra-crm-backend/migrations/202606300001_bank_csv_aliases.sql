-- +goose Up
-- +goose StatementBegin
ALTER TABLE banks
    ADD COLUMN csv_alias TEXT,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX uq_banks_csv_alias_normalized
ON banks (regexp_replace(replace(lower(btrim(csv_alias)), 'ё', 'е'), '[^[:alpha:][:digit:]]+', '', 'g'))
WHERE csv_alias IS NOT NULL
  AND regexp_replace(replace(lower(btrim(csv_alias)), 'ё', 'е'), '[^[:alpha:][:digit:]]+', '', 'g') <> '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_banks_csv_alias_normalized;

ALTER TABLE banks
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS csv_alias;
-- +goose StatementEnd
