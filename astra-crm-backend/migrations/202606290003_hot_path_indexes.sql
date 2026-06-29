-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_order_scope_active_shift_created
ON order_scope_items(team_id, shift_id, direction, created_at_external DESC, id DESC)
WHERE scope_type = 'trader_shift'
  AND is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_order_scope_active_shift_inner
ON order_scope_items(team_id, shift_id, direction, external_inner_id, created_at DESC, id DESC)
WHERE scope_type = 'trader_shift'
  AND is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_order_scope_active_period_inner
ON order_scope_items(team_id, accounting_period_id, direction, external_inner_id, created_at DESC, id DESC)
WHERE scope_type = 'teamlead_period'
  AND accounting_period_id IS NOT NULL
  AND is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_order_scope_active_current_import_inner
ON order_scope_items(team_id, import_batch_id, direction, external_inner_id, created_at DESC, id DESC)
WHERE scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL
  AND is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_shift_requisites_team_shift_taken
ON shift_requisites(team_id, shift_id, taken_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_shift_requisites_team_trader_shift_status
ON shift_requisites(team_id, trader_id, shift_id, status);

CREATE INDEX IF NOT EXISTS idx_requisite_assignments_active_latest
ON requisite_assignments(team_id, requisite_id, assigned_for_date DESC, assigned_at DESC, id DESC)
WHERE unassigned_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_manual_payout_orders_active_shift
ON manual_payout_orders(team_id, trader_id, shift_id, id)
WHERE deleted_at IS NULL
  AND status <> 'cancelled';

CREATE INDEX IF NOT EXISTS idx_manual_payout_transfers_team_shift_source
ON manual_payout_transfers(team_id, shift_id, trader_id, source_shift_requisite_id);

CREATE INDEX IF NOT EXISTS idx_import_batches_teamlead_current_recent
ON import_batches(team_id, uploaded_by, direction, created_at DESC, id DESC)
WHERE scope_type = 'teamlead_period'
  AND accounting_period_id IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_batches_teamlead_current_recent;
DROP INDEX IF EXISTS idx_manual_payout_transfers_team_shift_source;
DROP INDEX IF EXISTS idx_manual_payout_orders_active_shift;
DROP INDEX IF EXISTS idx_requisite_assignments_active_latest;
DROP INDEX IF EXISTS idx_shift_requisites_team_trader_shift_status;
DROP INDEX IF EXISTS idx_shift_requisites_team_shift_taken;
DROP INDEX IF EXISTS idx_order_scope_active_current_import_inner;
DROP INDEX IF EXISTS idx_order_scope_active_period_inner;
DROP INDEX IF EXISTS idx_order_scope_active_shift_inner;
DROP INDEX IF EXISTS idx_order_scope_active_shift_created;
-- +goose StatementEnd
