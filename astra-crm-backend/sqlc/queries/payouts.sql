-- name: ListPayoutOrdersForCurrentShift :many
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
SELECT
    mpo.id,
    mpo.team_id,
    mpo.shift_id,
    mpo.trader_id,
    mpo.destination_bank,
    mpo.destination_requisite,
    mpo.amount_minor,
    mpo.status,
    mpo.created_at,
    mpo.updated_at,
    mpo.deleted_at,
    COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor,
    (mpo.amount_minor - COALESCE(sum(mpt.amount_minor), 0))::bigint AS remaining_amount_minor
FROM manual_payout_orders mpo
JOIN current_shift cs ON cs.id = mpo.shift_id
LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
WHERE mpo.team_id = sqlc.arg(team_id)
  AND mpo.trader_id = sqlc.arg(trader_id)
  AND mpo.deleted_at IS NULL
  AND (
      sqlc.arg(status)::text = ''
      OR sqlc.arg(status)::text = 'all'
      OR (sqlc.arg(status)::text = 'open' AND mpo.status IN ('draft', 'in_progress'))
      OR mpo.status = sqlc.arg(status)::text
  )
GROUP BY mpo.id
ORDER BY mpo.created_at DESC, mpo.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountPayoutOrdersForCurrentShift :one
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
SELECT count(*)::bigint
FROM manual_payout_orders mpo
JOIN current_shift cs ON cs.id = mpo.shift_id
WHERE mpo.team_id = sqlc.arg(team_id)
  AND mpo.trader_id = sqlc.arg(trader_id)
  AND mpo.deleted_at IS NULL
  AND (
      sqlc.arg(status)::text = ''
      OR sqlc.arg(status)::text = 'all'
      OR (sqlc.arg(status)::text = 'open' AND mpo.status IN ('draft', 'in_progress'))
      OR mpo.status = sqlc.arg(status)::text
  );

-- name: ListPayoutOrderHistoryForTrader :many
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
SELECT
    mpo.id,
    mpo.team_id,
    mpo.shift_id,
    mpo.trader_id,
    mpo.destination_bank,
    mpo.destination_requisite,
    mpo.amount_minor,
    mpo.status,
    mpo.created_at,
    mpo.updated_at,
    mpo.deleted_at,
    COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor,
    (mpo.amount_minor - COALESCE(sum(mpt.amount_minor), 0))::bigint AS remaining_amount_minor
FROM manual_payout_orders mpo
LEFT JOIN current_shift cs ON cs.id = mpo.shift_id
LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
WHERE mpo.team_id = sqlc.arg(team_id)
  AND mpo.trader_id = sqlc.arg(trader_id)
  AND (cs.id IS NULL OR mpo.deleted_at IS NOT NULL OR mpo.status = 'cancelled')
GROUP BY mpo.id
ORDER BY COALESCE(mpo.deleted_at, mpo.updated_at, mpo.created_at) DESC, mpo.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountPayoutOrderHistoryForTrader :one
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
SELECT count(*)::bigint
FROM manual_payout_orders mpo
LEFT JOIN current_shift cs ON cs.id = mpo.shift_id
WHERE mpo.team_id = sqlc.arg(team_id)
  AND mpo.trader_id = sqlc.arg(trader_id)
  AND (cs.id IS NULL OR mpo.deleted_at IS NOT NULL OR mpo.status = 'cancelled');

-- name: GetPayoutOrderForTrader :one
SELECT
    mpo.id,
    mpo.team_id,
    mpo.shift_id,
    mpo.trader_id,
    mpo.destination_bank,
    mpo.destination_requisite,
    mpo.amount_minor,
    mpo.status,
    mpo.created_at,
    mpo.updated_at,
    mpo.deleted_at,
    COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor,
    (mpo.amount_minor - COALESCE(sum(mpt.amount_minor), 0))::bigint AS remaining_amount_minor
FROM manual_payout_orders mpo
LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = mpo.id
WHERE mpo.id = sqlc.arg(payout_id)
  AND mpo.team_id = sqlc.arg(team_id)
  AND mpo.trader_id = sqlc.arg(trader_id)
GROUP BY mpo.id;

-- name: CreatePayoutOrder :one
WITH current_shift AS (
    SELECT trader_shifts.id
    FROM trader_shifts
    WHERE trader_shifts.team_id = sqlc.arg(team_id)
      AND trader_shifts.trader_id = sqlc.arg(trader_id)
      AND trader_shifts.status IN ('open', 'closing')
    ORDER BY trader_shifts.started_at DESC, trader_shifts.id DESC
    LIMIT 1
)
INSERT INTO manual_payout_orders (team_id, shift_id, trader_id, destination_bank, destination_requisite, amount_minor, status)
SELECT
    sqlc.arg(team_id),
    id,
    sqlc.arg(trader_id),
    sqlc.arg(destination_bank),
    sqlc.arg(destination_requisite),
    sqlc.arg(amount_minor),
    'in_progress'
FROM current_shift
RETURNING id, team_id, shift_id, trader_id, destination_bank, destination_requisite, amount_minor, status, created_at, updated_at, deleted_at;

-- name: UpdatePayoutOrder :one
WITH order_row AS (
    SELECT id
    FROM manual_payout_orders
    WHERE manual_payout_orders.id = sqlc.arg(payout_id)
      AND manual_payout_orders.team_id = sqlc.arg(team_id)
      AND manual_payout_orders.trader_id = sqlc.arg(trader_id)
      AND manual_payout_orders.deleted_at IS NULL
      AND manual_payout_orders.status <> 'cancelled'
    FOR UPDATE
),
paid AS (
    SELECT COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor
    FROM order_row
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = order_row.id
),
updated AS (
    UPDATE manual_payout_orders
    SET
        destination_bank = sqlc.arg(destination_bank),
        destination_requisite = sqlc.arg(destination_requisite),
        amount_minor = sqlc.arg(amount_minor),
        status = CASE
            WHEN sqlc.arg(amount_minor) = (SELECT paid_amount_minor FROM paid) THEN 'paid'
            WHEN (SELECT paid_amount_minor FROM paid) > 0 THEN 'in_progress'
            ELSE 'in_progress'
        END,
        updated_at = now()
    WHERE manual_payout_orders.id = (SELECT id FROM order_row)
      AND sqlc.arg(amount_minor) >= (SELECT paid_amount_minor FROM paid)
    RETURNING id, team_id, shift_id, trader_id, destination_bank, destination_requisite, amount_minor, status, created_at, updated_at, deleted_at
)
SELECT
    updated.id,
    updated.team_id,
    updated.shift_id,
    updated.trader_id,
    updated.destination_bank,
    updated.destination_requisite,
    updated.amount_minor,
    updated.status,
    updated.created_at,
    updated.updated_at,
    updated.deleted_at,
    (SELECT paid_amount_minor FROM paid) AS paid_amount_minor,
    (updated.amount_minor - (SELECT paid_amount_minor FROM paid))::bigint AS remaining_amount_minor
FROM updated;

-- name: CancelPayoutOrder :one
UPDATE manual_payout_orders
SET status = 'cancelled',
    deleted_at = COALESCE(manual_payout_orders.deleted_at, now()),
    updated_at = now()
WHERE manual_payout_orders.id = sqlc.arg(payout_id)
  AND manual_payout_orders.team_id = sqlc.arg(team_id)
  AND manual_payout_orders.trader_id = sqlc.arg(trader_id)
  AND manual_payout_orders.deleted_at IS NULL
RETURNING id, team_id, shift_id, trader_id, destination_bank, destination_requisite, amount_minor, status, created_at, updated_at, deleted_at;

-- name: AddPayoutTransfer :one
WITH order_row AS (
    SELECT *
    FROM manual_payout_orders
    WHERE manual_payout_orders.id = sqlc.arg(payout_id)
      AND manual_payout_orders.team_id = sqlc.arg(team_id)
      AND manual_payout_orders.trader_id = sqlc.arg(trader_id)
      AND manual_payout_orders.deleted_at IS NULL
      AND manual_payout_orders.status <> 'cancelled'
    FOR UPDATE
),
source_requisite AS (
    SELECT
        sr.id AS source_shift_requisite_id,
        sr.requisite_id AS source_requisite_id,
        r.phone AS source_phone,
        b.name AS source_bank_name
    FROM order_row
    JOIN shift_requisites sr ON sr.id = sqlc.arg(source_shift_requisite_id)
        AND sr.team_id = order_row.team_id
        AND sr.trader_id = order_row.trader_id
        AND sr.shift_id = order_row.shift_id
        AND sr.status = 'active'
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN banks b ON b.code = r.bank_code
),
paid AS (
    SELECT COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor
    FROM order_row
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = order_row.id
),
inserted AS (
    INSERT INTO manual_payout_transfers (
        team_id,
        manual_payout_order_id,
        shift_id,
        trader_id,
        source_shift_requisite_id,
        source_requisite_id,
        amount_minor,
        created_by,
        comment
    )
    SELECT
        order_row.team_id,
        order_row.id,
        order_row.shift_id,
        order_row.trader_id,
        source_requisite.source_shift_requisite_id,
        source_requisite.source_requisite_id,
        sqlc.arg(amount_minor),
        sqlc.arg(created_by),
        sqlc.arg(comment)
    FROM order_row
    CROSS JOIN source_requisite
    CROSS JOIN paid
    WHERE paid.paid_amount_minor + sqlc.arg(amount_minor) <= order_row.amount_minor
    RETURNING id, team_id, manual_payout_order_id, shift_id, trader_id, source_shift_requisite_id, source_requisite_id, amount_minor, created_by, created_at, comment
),
updated_order AS (
    UPDATE manual_payout_orders mpo
    SET status = CASE
            WHEN (SELECT paid_amount_minor FROM paid) + (SELECT amount_minor FROM inserted) = mpo.amount_minor THEN 'paid'
            ELSE 'in_progress'
        END,
        updated_at = now()
    WHERE mpo.id = (SELECT manual_payout_order_id FROM inserted)
    RETURNING mpo.id
)
SELECT inserted.id, inserted.team_id, inserted.manual_payout_order_id, inserted.shift_id, inserted.trader_id, inserted.source_shift_requisite_id, inserted.source_requisite_id, source_requisite.source_phone, source_requisite.source_bank_name, inserted.amount_minor, inserted.created_by, inserted.created_at, inserted.comment
FROM inserted
JOIN source_requisite ON source_requisite.source_shift_requisite_id = inserted.source_shift_requisite_id;

-- name: UpdatePayoutTransfer :one
WITH order_row AS (
    SELECT *
    FROM manual_payout_orders
    WHERE manual_payout_orders.id = sqlc.arg(payout_id)
      AND manual_payout_orders.team_id = sqlc.arg(team_id)
      AND manual_payout_orders.trader_id = sqlc.arg(trader_id)
      AND manual_payout_orders.deleted_at IS NULL
      AND manual_payout_orders.status <> 'cancelled'
    FOR UPDATE
),
current_transfer AS (
    SELECT *
    FROM manual_payout_transfers
    WHERE manual_payout_transfers.id = sqlc.arg(transfer_id)
      AND manual_payout_transfers.team_id = sqlc.arg(team_id)
      AND manual_payout_transfers.trader_id = sqlc.arg(trader_id)
      AND manual_payout_transfers.manual_payout_order_id = (SELECT id FROM order_row)
    FOR UPDATE
),
source_requisite AS (
    SELECT
        sr.id AS source_shift_requisite_id,
        sr.requisite_id AS source_requisite_id,
        r.phone AS source_phone,
        b.name AS source_bank_name
    FROM order_row
    JOIN shift_requisites sr ON sr.id = sqlc.arg(source_shift_requisite_id)
        AND sr.team_id = order_row.team_id
        AND sr.trader_id = order_row.trader_id
        AND sr.shift_id = order_row.shift_id
    JOIN requisites r ON r.id = sr.requisite_id
    JOIN banks b ON b.code = r.bank_code
),
paid_other AS (
    SELECT COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor
    FROM order_row
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = order_row.id
        AND mpt.id <> sqlc.arg(transfer_id)
),
updated AS (
    UPDATE manual_payout_transfers mpt
    SET
        source_shift_requisite_id = source_requisite.source_shift_requisite_id,
        source_requisite_id = source_requisite.source_requisite_id,
        amount_minor = sqlc.arg(amount_minor),
        comment = sqlc.arg(comment)
    FROM order_row, current_transfer, source_requisite, paid_other
    WHERE mpt.id = current_transfer.id
      AND paid_other.paid_amount_minor + sqlc.arg(amount_minor) <= order_row.amount_minor
    RETURNING mpt.id, mpt.team_id, mpt.manual_payout_order_id, mpt.shift_id, mpt.trader_id, mpt.source_shift_requisite_id, mpt.source_requisite_id, mpt.amount_minor, mpt.created_by, mpt.created_at, mpt.comment
),
paid AS (
    SELECT COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor
    FROM order_row
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = order_row.id
),
updated_order AS (
    UPDATE manual_payout_orders mpo
    SET status = CASE
            WHEN (SELECT paid_amount_minor FROM paid) = mpo.amount_minor THEN 'paid'
            ELSE 'in_progress'
        END,
        updated_at = now()
    WHERE mpo.id = (SELECT id FROM order_row)
      AND EXISTS (SELECT 1 FROM updated)
    RETURNING mpo.id
)
SELECT updated.id, updated.team_id, updated.manual_payout_order_id, updated.shift_id, updated.trader_id, updated.source_shift_requisite_id, updated.source_requisite_id, source_requisite.source_phone, source_requisite.source_bank_name, updated.amount_minor, updated.created_by, updated.created_at, updated.comment
FROM updated
JOIN source_requisite ON source_requisite.source_shift_requisite_id = updated.source_shift_requisite_id;

-- name: DeletePayoutTransfer :one
WITH order_row AS (
    SELECT id
    FROM manual_payout_orders
    WHERE manual_payout_orders.id = sqlc.arg(payout_id)
      AND manual_payout_orders.team_id = sqlc.arg(team_id)
      AND manual_payout_orders.trader_id = sqlc.arg(trader_id)
      AND manual_payout_orders.deleted_at IS NULL
      AND manual_payout_orders.status <> 'cancelled'
    FOR UPDATE
),
deleted AS (
    DELETE FROM manual_payout_transfers
    WHERE manual_payout_transfers.id = sqlc.arg(transfer_id)
      AND manual_payout_transfers.team_id = sqlc.arg(team_id)
      AND manual_payout_transfers.trader_id = sqlc.arg(trader_id)
      AND manual_payout_transfers.manual_payout_order_id = (SELECT id FROM order_row)
    RETURNING id, team_id, manual_payout_order_id, shift_id, trader_id, source_shift_requisite_id, source_requisite_id, amount_minor, created_by, created_at, comment
),
paid AS (
    SELECT COALESCE(sum(mpt.amount_minor), 0)::bigint AS paid_amount_minor
    FROM order_row
    LEFT JOIN manual_payout_transfers mpt ON mpt.manual_payout_order_id = order_row.id
),
updated_order AS (
    UPDATE manual_payout_orders mpo
    SET status = CASE
            WHEN (SELECT paid_amount_minor FROM paid) = mpo.amount_minor THEN 'paid'
            ELSE 'in_progress'
        END,
        updated_at = now()
    WHERE mpo.id = (SELECT id FROM order_row)
      AND EXISTS (SELECT 1 FROM deleted)
    RETURNING mpo.id
)
SELECT deleted.id, deleted.team_id, deleted.manual_payout_order_id, deleted.shift_id, deleted.trader_id, deleted.source_shift_requisite_id, deleted.source_requisite_id, r.phone AS source_phone, b.name AS source_bank_name, deleted.amount_minor, deleted.created_by, deleted.created_at, deleted.comment
FROM deleted
JOIN requisites r ON r.id = deleted.source_requisite_id
JOIN banks b ON b.code = r.bank_code;

-- name: ListPayoutTransfers :many
SELECT
    mpt.id,
    mpt.team_id,
    mpt.manual_payout_order_id,
    mpt.shift_id,
    mpt.trader_id,
    mpt.source_shift_requisite_id,
    mpt.source_requisite_id,
    r.phone AS source_phone,
    b.name AS source_bank_name,
    mpt.amount_minor,
    mpt.created_by,
    mpt.created_at,
    mpt.comment
FROM manual_payout_transfers mpt
JOIN requisites r ON r.id = mpt.source_requisite_id
JOIN banks b ON b.code = r.bank_code
WHERE mpt.team_id = sqlc.arg(team_id)
  AND mpt.trader_id = sqlc.arg(trader_id)
  AND mpt.manual_payout_order_id = sqlc.arg(payout_id)
ORDER BY mpt.created_at DESC, mpt.id DESC;
