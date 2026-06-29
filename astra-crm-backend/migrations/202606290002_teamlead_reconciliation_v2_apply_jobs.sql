-- +goose Up
CREATE TABLE teamlead_reconciliation_apply_jobs (
    id BIGSERIAL PRIMARY KEY,
    teamlead_reconciliation_id BIGINT NOT NULL REFERENCES teamlead_reconciliations(id),
    team_id BIGINT NOT NULL REFERENCES teams(id),
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    result_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    UNIQUE (teamlead_reconciliation_id)
);

CREATE INDEX idx_teamlead_reconciliation_apply_jobs_team_status
ON teamlead_reconciliation_apply_jobs(team_id, status, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_teamlead_reconciliation_apply_jobs_team_status;
DROP TABLE IF EXISTS teamlead_reconciliation_apply_jobs;
