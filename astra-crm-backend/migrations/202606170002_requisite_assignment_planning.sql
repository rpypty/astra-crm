-- +goose Up
-- +goose StatementBegin
ALTER TABLE requisite_assignments
    ADD COLUMN status TEXT NOT NULL DEFAULT 'assigned',
    ADD COLUMN assigned_for_date DATE NOT NULL DEFAULT CURRENT_DATE,
    ADD COLUMN target_turnover_minor BIGINT NOT NULL DEFAULT 0 CHECK (target_turnover_minor >= 0),
    ADD COLUMN started_at TIMESTAMPTZ,
    ADD COLUMN completed_at TIMESTAMPTZ,
    ADD COLUMN cancelled_at TIMESTAMPTZ,
    ADD COLUMN shift_requisite_id BIGINT REFERENCES shift_requisites(id),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD CONSTRAINT requisite_assignments_status_check CHECK (status IN ('planned', 'assigned', 'in_work', 'worked', 'blocked', 'cancelled', 'expired'));

UPDATE requisite_assignments
SET assigned_for_date = assigned_at::date,
    status = CASE WHEN unassigned_at IS NULL THEN 'assigned' ELSE 'worked' END,
    updated_at = now();

DROP INDEX IF EXISTS uq_requisite_active_assignment;

CREATE UNIQUE INDEX uq_requisite_assignment_active_date
ON requisite_assignments(requisite_id, assigned_for_date)
WHERE unassigned_at IS NULL
  AND status IN ('planned', 'assigned', 'in_work');

CREATE INDEX idx_requisite_assignments_team_date_status
ON requisite_assignments(team_id, assigned_for_date, status);

CREATE INDEX idx_requisite_assignments_trader_date_status
ON requisite_assignments(trader_id, assigned_for_date, status);

CREATE TABLE requisite_assignment_events (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    assignment_id BIGINT NOT NULL REFERENCES requisite_assignments(id),
    actor_id BIGINT NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    before_json JSONB,
    after_json JSONB,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_requisite_assignment_events_assignment
ON requisite_assignment_events(assignment_id, created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS requisite_assignment_events;

DROP INDEX IF EXISTS idx_requisite_assignments_trader_date_status;
DROP INDEX IF EXISTS idx_requisite_assignments_team_date_status;
DROP INDEX IF EXISTS uq_requisite_assignment_active_date;

CREATE UNIQUE INDEX uq_requisite_active_assignment
ON requisite_assignments(requisite_id)
WHERE unassigned_at IS NULL;

ALTER TABLE requisite_assignments
    DROP CONSTRAINT IF EXISTS requisite_assignments_status_check,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS shift_requisite_id,
    DROP COLUMN IF EXISTS cancelled_at,
    DROP COLUMN IF EXISTS completed_at,
    DROP COLUMN IF EXISTS started_at,
    DROP COLUMN IF EXISTS target_turnover_minor,
    DROP COLUMN IF EXISTS assigned_for_date,
    DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
