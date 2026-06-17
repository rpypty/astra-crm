-- +goose Up
-- +goose StatementBegin
ALTER TABLE requisites
    ADD COLUMN total_inbound_turnover_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_inbound_turnover_minor >= 0),
    ADD COLUMN total_outbound_turnover_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_outbound_turnover_minor >= 0),
    ADD COLUMN last_closing_balance_minor BIGINT NOT NULL DEFAULT 0 CHECK (last_closing_balance_minor >= 0),
    ADD COLUMN last_activity_status TEXT,
    ADD COLUMN last_activity_at TIMESTAMPTZ,
    ADD COLUMN last_shift_requisite_id BIGINT REFERENCES shift_requisites(id);

WITH totals AS (
    SELECT
        requisite_id,
        COALESCE(SUM(inbound_turnover_minor), 0)::bigint AS total_inbound_turnover_minor,
        COALESCE(SUM(outbound_turnover_minor), 0)::bigint AS total_outbound_turnover_minor
    FROM shift_requisites
    GROUP BY requisite_id
),
latest AS (
    SELECT DISTINCT ON (requisite_id)
        requisite_id,
        id AS shift_requisite_id,
        status,
        closing_balance_minor,
        COALESCE(released_at, taken_at, updated_at) AS activity_at
    FROM shift_requisites
    ORDER BY requisite_id, COALESCE(released_at, taken_at, updated_at) DESC, id DESC
)
UPDATE requisites r
SET total_inbound_turnover_minor = COALESCE(t.total_inbound_turnover_minor, 0),
    total_outbound_turnover_minor = COALESCE(t.total_outbound_turnover_minor, 0),
    last_closing_balance_minor = COALESCE(l.closing_balance_minor, 0),
    last_activity_status = l.status,
    last_activity_at = l.activity_at,
    last_shift_requisite_id = l.shift_requisite_id
FROM totals t
FULL JOIN latest l ON l.requisite_id = t.requisite_id
WHERE r.id = COALESCE(t.requisite_id, l.requisite_id);

CREATE INDEX idx_shift_requisites_requisite_activity
ON shift_requisites(requisite_id, COALESCE(released_at, taken_at, updated_at) DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_shift_requisites_requisite_activity;

ALTER TABLE requisites
    DROP COLUMN IF EXISTS last_shift_requisite_id,
    DROP COLUMN IF EXISTS last_activity_at,
    DROP COLUMN IF EXISTS last_activity_status,
    DROP COLUMN IF EXISTS last_closing_balance_minor,
    DROP COLUMN IF EXISTS total_outbound_turnover_minor,
    DROP COLUMN IF EXISTS total_inbound_turnover_minor;
-- +goose StatementEnd
