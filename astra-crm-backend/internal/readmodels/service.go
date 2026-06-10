package readmodels

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type AccountingPeriod struct {
	ID             int64
	Title          string
	DateRange      string
	InboundStatus  string
	OutboundStatus string
	Status         string
}

type AuditLogEntry struct {
	ID            int64
	CreatedAt     time.Time
	ActorLogin    string
	Action        string
	EntityType    string
	EntityID      string
	Comment       *string
	MaskedPayload map[string]any
}

func (s *Service) ListPeriods(ctx context.Context, teamID int64) ([]AccountingPeriod, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("readmodels: repository is not configured")
	}

	rows, err := s.pool.Query(ctx, `
SELECT
	ap.id,
	'Период ' || to_char(ap.date_from, 'DD.MM.YYYY') || ' - ' || to_char(ap.date_to, 'DD.MM.YYYY') AS title,
	to_char(ap.date_from, 'DD.MM.YYYY') || ' - ' || to_char(ap.date_to, 'DD.MM.YYYY') AS date_range,
	COALESCE((
		SELECT rr.status
		FROM reconciliation_runs rr
		WHERE rr.accounting_period_id = ap.id AND rr.type = 'teamlead_period_inbound'
		ORDER BY rr.created_at DESC, rr.id DESC
		LIMIT 1
	), 'matched') AS inbound_status,
	COALESCE((
		SELECT rr.status
		FROM reconciliation_runs rr
		WHERE rr.accounting_period_id = ap.id AND rr.type = 'teamlead_period_outbound'
		ORDER BY rr.created_at DESC, rr.id DESC
		LIMIT 1
	), 'matched') AS outbound_status,
	ap.status
FROM accounting_periods ap
WHERE ap.team_id = $1
ORDER BY ap.date_from DESC, ap.id DESC`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list periods: %w", err)
	}
	defer rows.Close()

	items := []AccountingPeriod{}
	for rows.Next() {
		var item AccountingPeriod
		if err := rows.Scan(&item.ID, &item.Title, &item.DateRange, &item.InboundStatus, &item.OutboundStatus, &item.Status); err != nil {
			return nil, fmt.Errorf("scan period: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate periods: %w", err)
	}

	return items, nil
}

func (s *Service) ListAudit(ctx context.Context, teamID int64) ([]AuditLogEntry, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("readmodels: repository is not configured")
	}

	rows, err := s.pool.Query(ctx, `
SELECT
	al.id,
	al.created_at,
	u.login AS actor_login,
	al.action,
	al.entity_type,
	al.entity_id,
	al.comment,
	COALESCE(al.changed_fields_json, al.after_json, '{}'::jsonb) AS masked_payload
FROM audit_logs al
JOIN users u ON u.id = al.actor_id
WHERE al.team_id = $1
ORDER BY al.created_at DESC, al.id DESC
LIMIT 200`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()

	items := []AuditLogEntry{}
	for rows.Next() {
		var item AuditLogEntry
		var payload []byte
		if err := rows.Scan(
			&item.ID,
			&item.CreatedAt,
			&item.ActorLogin,
			&item.Action,
			&item.EntityType,
			&item.EntityID,
			&item.Comment,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		item.MaskedPayload = map[string]any{}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &item.MaskedPayload); err != nil {
				return nil, fmt.Errorf("decode audit payload: %w", err)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit: %w", err)
	}

	return items, nil
}
