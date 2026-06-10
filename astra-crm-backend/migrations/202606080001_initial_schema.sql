-- +goose Up
-- +goose StatementBegin
CREATE TABLE teams (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    role TEXT NOT NULL CHECK (role IN ('teamlead', 'trader')),
    login TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (team_id, login)
);

CREATE TABLE trader_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id),
    salary_rate_bps BIGINT NOT NULL DEFAULT 0 CHECK (salary_rate_bps >= 0),
    external_worker_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (external_worker_name)
);

CREATE TABLE auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip INET,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_valid ON auth_sessions(token_hash, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE requisites (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    phone TEXT NOT NULL,
    method_type TEXT NOT NULL,
    proxy TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'archived')),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_requisites_team_status ON requisites(team_id, status);
CREATE INDEX idx_requisites_phone ON requisites(phone);

CREATE TABLE requisite_assignments (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    assigned_by BIGINT NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    unassigned_at TIMESTAMPTZ,
    comment TEXT,
    CHECK (unassigned_at IS NULL OR unassigned_at >= assigned_at)
);

CREATE UNIQUE INDEX uq_requisite_active_assignment
ON requisite_assignments(requisite_id)
WHERE unassigned_at IS NULL;

CREATE INDEX idx_requisite_assignments_trader_active
ON requisite_assignments(trader_id)
WHERE unassigned_at IS NULL;

CREATE TABLE trader_shifts (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'closing', 'closed', 'closed_with_discrepancy')),
    inbound_reconciliation_status TEXT NOT NULL DEFAULT 'not_started'
        CHECK (inbound_reconciliation_status IN ('not_started', 'imported', 'matched', 'mismatch', 'accepted_with_comment')),
    outbound_reconciliation_status TEXT NOT NULL DEFAULT 'not_started'
        CHECK (outbound_reconciliation_status IN ('not_started', 'imported', 'matched', 'mismatch', 'accepted_with_comment')),
    close_comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ,
    CHECK (ended_at IS NULL OR ended_at >= started_at),
    CHECK (closed_at IS NULL OR status IN ('closed', 'closed_with_discrepancy'))
);

CREATE UNIQUE INDEX uq_trader_one_open_shift
ON trader_shifts(trader_id)
WHERE status IN ('open', 'closing');

CREATE INDEX idx_trader_shifts_team_trader_started ON trader_shifts(team_id, trader_id, started_at);
CREATE INDEX idx_trader_shifts_team_status ON trader_shifts(team_id, status);

CREATE TABLE shift_requisites (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    shift_id BIGINT NOT NULL REFERENCES trader_shifts(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    assignment_id BIGINT REFERENCES requisite_assignments(id),
    card_number TEXT NOT NULL,
    holder_name TEXT NOT NULL,
    taken_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (released_at IS NULL OR released_at >= taken_at)
);

CREATE UNIQUE INDEX uq_shift_requisite_once ON shift_requisites(shift_id, requisite_id);

CREATE TABLE requisite_turnover_entries (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    shift_id BIGINT NOT NULL REFERENCES trader_shifts(id),
    shift_requisite_id BIGINT NOT NULL REFERENCES shift_requisites(id),
    requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    comment TEXT
);

CREATE INDEX idx_turnover_entries_shift ON requisite_turnover_entries(shift_id, created_at DESC);
CREATE INDEX idx_turnover_entries_shift_requisite ON requisite_turnover_entries(shift_requisite_id, created_at DESC);

CREATE TABLE manual_payout_orders (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    shift_id BIGINT NOT NULL REFERENCES trader_shifts(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    destination_bank TEXT NOT NULL,
    destination_requisite TEXT NOT NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_progress', 'paid', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_manual_payout_orders_shift ON manual_payout_orders(shift_id);

CREATE TABLE manual_payout_transfers (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    manual_payout_order_id BIGINT NOT NULL REFERENCES manual_payout_orders(id),
    shift_id BIGINT NOT NULL REFERENCES trader_shifts(id),
    trader_id BIGINT NOT NULL REFERENCES users(id),
    source_shift_requisite_id BIGINT NOT NULL REFERENCES shift_requisites(id),
    source_requisite_id BIGINT NOT NULL REFERENCES requisites(id),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    comment TEXT
);

CREATE INDEX idx_payout_transfers_order ON manual_payout_transfers(manual_payout_order_id);
CREATE INDEX idx_payout_transfers_shift ON manual_payout_transfers(shift_id);

CREATE TABLE accounting_periods (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'checking', 'closed', 'closed_with_discrepancy')),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_by BIGINT REFERENCES users(id),
    closed_at TIMESTAMPTZ,
    CHECK (date_to >= date_from),
    CHECK (closed_at IS NULL OR status IN ('closed', 'closed_with_discrepancy'))
);

CREATE TABLE import_batches (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    uploaded_by BIGINT NOT NULL REFERENCES users(id),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('trader_shift', 'teamlead_period')),
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    shift_id BIGINT REFERENCES trader_shifts(id),
    accounting_period_id BIGINT REFERENCES accounting_periods(id),
    trader_id BIGINT REFERENCES users(id),
    file_name TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    rows_count BIGINT NOT NULL DEFAULT 0 CHECK (rows_count >= 0),
    status TEXT NOT NULL DEFAULT 'uploaded'
        CHECK (status IN ('uploaded', 'parsed', 'applied', 'reconciled', 'superseded', 'failed')),
    superseded_by_batch_id BIGINT REFERENCES import_batches(id),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_at TIMESTAMPTZ,
    CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND accounting_period_id IS NOT NULL AND shift_id IS NULL)
    )
);

CREATE INDEX idx_import_batches_shift_scope ON import_batches(scope_type, shift_id, direction, status);
CREATE INDEX idx_import_batches_period_scope ON import_batches(scope_type, accounting_period_id, direction, status);

CREATE TABLE import_rows (
    id BIGSERIAL PRIMARY KEY,
    import_batch_id BIGINT NOT NULL REFERENCES import_batches(id),
    row_number BIGINT NOT NULL CHECK (row_number > 0),
    external_id TEXT,
    external_inner_id TEXT,
    raw_payload_json JSONB NOT NULL,
    parse_status TEXT NOT NULL CHECK (parse_status IN ('parsed', 'failed')),
    parse_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (import_batch_id, row_number)
);

CREATE INDEX idx_import_rows_batch ON import_rows(import_batch_id);
CREATE INDEX idx_import_rows_inner_id ON import_rows(external_inner_id);

CREATE TABLE external_orders (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    external_id TEXT NOT NULL,
    external_inner_id TEXT NOT NULL,
    external_foreign_id TEXT,
    worker_name TEXT NOT NULL,
    trader_id BIGINT REFERENCES users(id),
    requisite_raw TEXT,
    requisite_phone TEXT,
    requisite_external_id TEXT,
    requisite_id BIGINT REFERENCES requisites(id),
    device_name TEXT,
    method_type TEXT,
    method_name TEXT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency TEXT NOT NULL,
    course NUMERIC(18, 8),
    course_worker NUMERIC(18, 8),
    worker_amount NUMERIC(18, 8),
    worker_profit NUMERIC(18, 8),
    raw_status TEXT NOT NULL,
    normalized_status TEXT NOT NULL CHECK (normalized_status IN ('success', 'corrected', 'failed', 'cancelled', 'unknown')),
    created_at_external TIMESTAMPTZ,
    closed_at_external TIMESTAMPTZ,
    updated_at_external TIMESTAMPTZ,
    old_amount_minor BIGINT CHECK (old_amount_minor IS NULL OR old_amount_minor >= 0),
    had_dispute BOOLEAN,
    receipt TEXT,
    order_comment TEXT,
    ordered BOOLEAN,
    counted BOOLEAN,
    initials TEXT,
    last_import_batch_id BIGINT REFERENCES import_batches(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, direction, external_inner_id)
);

CREATE INDEX idx_external_orders_team_direction_status ON external_orders(team_id, direction, normalized_status);
CREATE INDEX idx_external_orders_worker ON external_orders(team_id, worker_name);
CREATE INDEX idx_external_orders_created_external ON external_orders(team_id, created_at_external);

CREATE TABLE order_scope_items (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('trader_shift', 'teamlead_period')),
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    shift_id BIGINT REFERENCES trader_shifts(id),
    accounting_period_id BIGINT REFERENCES accounting_periods(id),
    import_batch_id BIGINT NOT NULL REFERENCES import_batches(id),
    import_row_id BIGINT NOT NULL REFERENCES import_rows(id),
    external_order_id BIGINT NOT NULL REFERENCES external_orders(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at TIMESTAMPTZ,
    CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND accounting_period_id IS NOT NULL AND shift_id IS NULL)
    ),
    CHECK ((is_active = TRUE AND deactivated_at IS NULL) OR (is_active = FALSE))
);

CREATE INDEX idx_order_scope_shift_active
ON order_scope_items(shift_id, direction)
WHERE is_active = TRUE;

CREATE INDEX idx_order_scope_period_active
ON order_scope_items(accounting_period_id, direction)
WHERE is_active = TRUE;

CREATE INDEX idx_order_scope_external_order ON order_scope_items(external_order_id);

CREATE TABLE reconciliation_runs (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    type TEXT NOT NULL CHECK (type IN ('trader_shift_inbound', 'trader_shift_outbound', 'teamlead_period_inbound', 'teamlead_period_outbound')),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('trader_shift', 'teamlead_period')),
    shift_id BIGINT REFERENCES trader_shifts(id),
    accounting_period_id BIGINT REFERENCES accounting_periods(id),
    trader_id BIGINT REFERENCES users(id),
    import_batch_id BIGINT REFERENCES import_batches(id),
    expected_amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (expected_amount_minor >= 0),
    actual_amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (actual_amount_minor >= 0),
    diff_amount_minor BIGINT NOT NULL DEFAULT 0,
    success_amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (success_amount_minor >= 0),
    success_count BIGINT NOT NULL DEFAULT 0 CHECK (success_count >= 0),
    failed_amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (failed_amount_minor >= 0),
    failed_count BIGINT NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    total_amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_amount_minor >= 0),
    total_count BIGINT NOT NULL DEFAULT 0 CHECK (total_count >= 0),
    status TEXT NOT NULL CHECK (status IN ('matched', 'mismatch', 'accepted_with_comment')),
    comment TEXT,
    confirmed_by BIGINT REFERENCES users(id),
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (scope_type = 'trader_shift' AND shift_id IS NOT NULL AND accounting_period_id IS NULL)
        OR
        (scope_type = 'teamlead_period' AND accounting_period_id IS NOT NULL AND shift_id IS NULL)
    ),
    CHECK ((status = 'accepted_with_comment') = (comment IS NOT NULL AND btrim(comment) <> ''))
);

CREATE INDEX idx_reconciliation_runs_shift ON reconciliation_runs(shift_id, type, created_at DESC);
CREATE INDEX idx_reconciliation_runs_period ON reconciliation_runs(accounting_period_id, type, created_at DESC);

CREATE TABLE reconciliation_items (
    id BIGSERIAL PRIMARY KEY,
    reconciliation_run_id BIGINT NOT NULL REFERENCES reconciliation_runs(id),
    issue_type TEXT NOT NULL,
    external_order_id BIGINT REFERENCES external_orders(id),
    external_inner_id TEXT,
    teamlead_value_json JSONB,
    trader_value_json JSONB,
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    actor_id BIGINT NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    before_json JSONB,
    after_json JSONB,
    changed_fields_json JSONB,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_team_created ON audit_logs(team_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS reconciliation_items;
DROP TABLE IF EXISTS reconciliation_runs;
DROP TABLE IF EXISTS order_scope_items;
DROP TABLE IF EXISTS external_orders;
DROP TABLE IF EXISTS import_rows;
DROP TABLE IF EXISTS import_batches;
DROP TABLE IF EXISTS accounting_periods;
DROP TABLE IF EXISTS manual_payout_transfers;
DROP TABLE IF EXISTS manual_payout_orders;
DROP TABLE IF EXISTS requisite_turnover_entries;
DROP TABLE IF EXISTS shift_requisites;
DROP TABLE IF EXISTS trader_shifts;
DROP TABLE IF EXISTS requisite_assignments;
DROP TABLE IF EXISTS requisites;
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS trader_profiles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;
-- +goose StatementEnd
