-- +goose Up
-- +goose StatementBegin
CREATE TABLE debug_fin_all_import_jobs (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    actor_id BIGINT NOT NULL REFERENCES users(id),
    file_name TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    dry_run BOOLEAN NOT NULL DEFAULT TRUE,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    result_json JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_debug_fin_all_import_jobs_team_created
ON debug_fin_all_import_jobs(team_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS debug_fin_all_import_jobs;
-- +goose StatementEnd
