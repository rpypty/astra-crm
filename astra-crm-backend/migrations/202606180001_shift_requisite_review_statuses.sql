-- +goose Up
ALTER TABLE shift_requisites
    DROP CONSTRAINT IF EXISTS shift_requisites_status_check;

UPDATE shift_requisites
SET status = 'worked_pending_review'
WHERE status = 'worked';

ALTER TABLE shift_requisites
    ADD CONSTRAINT shift_requisites_status_check CHECK (
        status IN (
            'active',
            'worked_pending_review',
            'worked_verified',
            'worked_discrepancy',
            'correction',
            'released',
            'blocked'
        )
    );

-- +goose Down
ALTER TABLE shift_requisites
    DROP CONSTRAINT IF EXISTS shift_requisites_status_check;

UPDATE shift_requisites
SET status = 'worked'
WHERE status IN ('worked_pending_review', 'worked_verified', 'worked_discrepancy');

ALTER TABLE shift_requisites
    ADD CONSTRAINT shift_requisites_status_check CHECK (
        status IN ('active', 'worked', 'correction', 'released', 'blocked')
    );
