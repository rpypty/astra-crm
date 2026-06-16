-- +goose Up
-- +goose StatementBegin
CREATE TABLE banks (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    sort_order BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO banks (code, name, sort_order)
VALUES
    ('sber', 'Сбер', 10),
    ('tbank', 'Т-Банк', 20),
    ('alfa', 'Альфа-Банк', 30),
    ('vtb', 'ВТБ', 40),
    ('raif', 'Райффайзен', 50),
    ('ozon', 'Ozon Банк', 60),
    ('gazprombank', 'Газпромбанк', 70),
    ('mkb', 'МКБ', 80),
    ('rshb', 'Россельхозбанк', 90),
    ('other', 'Другой банк', 100);

ALTER TABLE requisites
    ADD COLUMN bank_code TEXT,
    ADD COLUMN employee_comment TEXT,
    ADD COLUMN holder_name TEXT,
    ADD COLUMN card_number TEXT,
    ADD COLUMN details_filled_at TIMESTAMPTZ,
    ADD COLUMN details_filled_by BIGINT REFERENCES users(id);

UPDATE requisites
SET bank_code = 'sber'
WHERE bank_code IS NULL;

ALTER TABLE requisites
    ALTER COLUMN bank_code SET NOT NULL,
    ADD CONSTRAINT fk_requisites_bank_code FOREIGN KEY (bank_code) REFERENCES banks(code),
    ADD CONSTRAINT chk_requisites_details_consistent CHECK (
        (holder_name IS NULL AND card_number IS NULL AND details_filled_at IS NULL AND details_filled_by IS NULL)
        OR
        (holder_name IS NOT NULL AND card_number IS NOT NULL AND details_filled_at IS NOT NULL AND details_filled_by IS NOT NULL)
    );

WITH duplicate_proxies AS (
    SELECT
        id,
        proxy,
        row_number() OVER (PARTITION BY proxy ORDER BY id) AS duplicate_index
    FROM requisites
    WHERE deleted_at IS NULL
      AND proxy IS NOT NULL
)
UPDATE requisites r
SET proxy = duplicate_proxies.proxy || '#' || r.id::text
FROM duplicate_proxies
WHERE r.id = duplicate_proxies.id
  AND duplicate_proxies.duplicate_index > 1;

CREATE UNIQUE INDEX uq_requisites_active_team_phone_bank
ON requisites(team_id, phone, bank_code)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_requisites_active_proxy_global
ON requisites(proxy)
WHERE deleted_at IS NULL AND proxy IS NOT NULL;

CREATE INDEX idx_requisites_bank_code ON requisites(bank_code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_requisites_bank_code;
DROP INDEX IF EXISTS uq_requisites_active_proxy_global;
DROP INDEX IF EXISTS uq_requisites_active_team_phone_bank;

ALTER TABLE requisites
    DROP CONSTRAINT IF EXISTS chk_requisites_details_consistent,
    DROP CONSTRAINT IF EXISTS fk_requisites_bank_code,
    DROP COLUMN IF EXISTS details_filled_by,
    DROP COLUMN IF EXISTS details_filled_at,
    DROP COLUMN IF EXISTS card_number,
    DROP COLUMN IF EXISTS holder_name,
    DROP COLUMN IF EXISTS employee_comment,
    DROP COLUMN IF EXISTS bank_code;

DROP TABLE IF EXISTS banks;
-- +goose StatementEnd
