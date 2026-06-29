package readmodels

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
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
	DateFrom       time.Time
	DateTo         time.Time
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

type TraderProfile struct {
	ID                              int64
	Login                           string
	SalaryRateBps                   int64
	ExternalWorkerName              string
	CurrentShiftSuccessInboundMinor int64
	CurrentShiftSalaryMinor         int64
	PeriodID                        *int64
	PeriodTitle                     *string
	PeriodSuccessInboundMinor       int64
	PeriodSalaryMinor               int64
}

type PeriodFilter struct {
	DateFrom *time.Time
	DateTo   *time.Time
}

func (s *Service) ListPeriods(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[AccountingPeriod], error) {
	if s == nil || s.pool == nil {
		return pagination.Result[AccountingPeriod]{}, fmt.Errorf("readmodels: repository is not configured")
	}
	page = pagination.Normalize(page)

	rows, err := s.pool.Query(ctx, `
SELECT
	ap.id,
	'Период ' || to_char(ap.date_from, 'DD.MM.YYYY') || ' - ' || to_char(ap.date_to, 'DD.MM.YYYY') AS title,
	ap.date_from,
	ap.date_to,
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
ORDER BY ap.date_from DESC, ap.id DESC
LIMIT $2
OFFSET $3`, teamID, page.PageSize, pagination.Offset(page))
	if err != nil {
		return pagination.Result[AccountingPeriod]{}, fmt.Errorf("list periods: %w", err)
	}
	defer rows.Close()

	items := []AccountingPeriod{}
	for rows.Next() {
		var item AccountingPeriod
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.DateFrom,
			&item.DateTo,
			&item.DateRange,
			&item.InboundStatus,
			&item.OutboundStatus,
			&item.Status,
		); err != nil {
			return pagination.Result[AccountingPeriod]{}, fmt.Errorf("scan period: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return pagination.Result[AccountingPeriod]{}, fmt.Errorf("iterate periods: %w", err)
	}

	var total int64
	if err := s.pool.QueryRow(ctx, `
SELECT count(*)::bigint
FROM accounting_periods ap
WHERE ap.team_id = $1`, teamID).Scan(&total); err != nil {
		return pagination.Result[AccountingPeriod]{}, fmt.Errorf("count periods: %w", err)
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *Service) ListAudit(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[AuditLogEntry], error) {
	if s == nil || s.pool == nil {
		return pagination.Result[AuditLogEntry]{}, fmt.Errorf("readmodels: repository is not configured")
	}
	page = pagination.Normalize(page)

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
LIMIT $2
OFFSET $3`, teamID, page.PageSize, pagination.Offset(page))
	if err != nil {
		return pagination.Result[AuditLogEntry]{}, fmt.Errorf("list audit: %w", err)
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
			return pagination.Result[AuditLogEntry]{}, fmt.Errorf("scan audit: %w", err)
		}
		item.MaskedPayload = map[string]any{}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &item.MaskedPayload); err != nil {
				return pagination.Result[AuditLogEntry]{}, fmt.Errorf("decode audit payload: %w", err)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return pagination.Result[AuditLogEntry]{}, fmt.Errorf("iterate audit: %w", err)
	}

	var total int64
	if err := s.pool.QueryRow(ctx, `
SELECT count(*)::bigint
FROM audit_logs al
WHERE al.team_id = $1`, teamID).Scan(&total); err != nil {
		return pagination.Result[AuditLogEntry]{}, fmt.Errorf("count audit: %w", err)
	}

	return pagination.NewResult(items, page, total), nil
}

func (s *Service) TraderProfile(ctx context.Context, teamID int64, traderID int64, filters PeriodFilter) (TraderProfile, error) {
	if s == nil || s.pool == nil {
		return TraderProfile{}, fmt.Errorf("readmodels: repository is not configured")
	}

	row := s.pool.QueryRow(ctx, traderProfileQuery, teamID, traderID, dateValue(filters.DateFrom), dateValue(filters.DateTo))

	var profile TraderProfile
	if err := row.Scan(
		&profile.ID,
		&profile.Login,
		&profile.SalaryRateBps,
		&profile.ExternalWorkerName,
		&profile.CurrentShiftSuccessInboundMinor,
		&profile.CurrentShiftSalaryMinor,
		&profile.PeriodID,
		&profile.PeriodTitle,
		&profile.PeriodSuccessInboundMinor,
		&profile.PeriodSalaryMinor,
	); err != nil {
		return TraderProfile{}, fmt.Errorf("get trader profile readmodel: %w", err)
	}

	return profile, nil
}

const traderProfileQuery = `
WITH current_period AS (
	SELECT id,
		date_from,
		date_to,
		'Период ' || to_char(date_from, 'DD.MM.YYYY') || ' - ' || to_char(date_to, 'DD.MM.YYYY') AS title
	FROM accounting_periods
	WHERE team_id = $1
	  AND $3::date IS NULL
	  AND $4::date IS NULL
	ORDER BY
		CASE WHEN status = 'open' THEN 0 ELSE 1 END,
		date_to DESC,
		id DESC
	LIMIT 1
),
selected_period AS (
	SELECT
		CASE
			WHEN $3::date IS NOT NULL OR $4::date IS NOT NULL THEN NULL::bigint
			ELSE (SELECT id FROM current_period)
		END AS id,
		CASE
			WHEN $3::date IS NOT NULL AND $4::date IS NOT NULL THEN 'Период ' || to_char($3::date, 'DD.MM.YYYY') || ' - ' || to_char($4::date, 'DD.MM.YYYY')
			WHEN $3::date IS NOT NULL THEN 'Период с ' || to_char($3::date, 'DD.MM.YYYY')
			WHEN $4::date IS NOT NULL THEN 'Период до ' || to_char($4::date, 'DD.MM.YYYY')
			ELSE (SELECT title FROM current_period)
		END AS title
),
shift_success AS (
	SELECT COALESCE(sum(amount_minor), 0)::bigint AS amount_minor
	FROM order_scope_items osi
	JOIN trader_shifts ts ON ts.id = osi.shift_id
	WHERE osi.team_id = $1
	  AND osi.scope_type = 'trader_shift'
	  AND osi.trader_id = $2
	  AND osi.direction = 'inbound'
	  AND osi.is_active = TRUE
	  AND osi.normalized_status IN ('success', 'corrected')
	  AND ts.status IN ('closed', 'closed_with_discrepancy')
	  AND ts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')
	  AND osi.created_at_external::date = current_date
),
period_success AS (
	SELECT COALESCE(sum(amount_minor), 0)::bigint AS amount_minor
	FROM order_scope_items osi
	JOIN trader_shifts ts ON ts.id = osi.shift_id
	WHERE osi.team_id = $1
	  AND osi.scope_type = 'trader_shift'
	  AND osi.trader_id = $2
	  AND osi.direction = 'inbound'
	  AND osi.is_active = TRUE
	  AND osi.normalized_status IN ('success', 'corrected')
	  AND ts.status IN ('closed', 'closed_with_discrepancy')
	  AND ts.inbound_reconciliation_status IN ('matched', 'accepted_with_comment')
	  AND (
		  $3::date IS NOT NULL
		  OR $4::date IS NOT NULL
		  OR osi.created_at_external::date BETWEEN (SELECT date_from FROM current_period) AND (SELECT date_to FROM current_period)
	  )
	  AND ($3::date IS NULL OR osi.created_at_external::date >= $3::date)
	  AND ($4::date IS NULL OR osi.created_at_external::date <= $4::date)
)
SELECT
	u.id,
	u.login,
	tp.salary_rate_bps,
	tp.external_worker_name,
	(SELECT amount_minor FROM shift_success) AS current_shift_success_inbound_minor,
	((SELECT amount_minor FROM shift_success) * tp.salary_rate_bps / 10000)::bigint AS current_shift_salary_minor,
	(SELECT id FROM selected_period) AS period_id,
	(SELECT title FROM selected_period) AS period_title,
	(SELECT amount_minor FROM period_success) AS period_success_inbound_minor,
	((SELECT amount_minor FROM period_success) * tp.salary_rate_bps / 10000)::bigint AS period_salary_minor
FROM users u
JOIN trader_profiles tp ON tp.user_id = u.id
WHERE u.team_id = $1
  AND u.id = $2
  AND u.role = 'trader'
  AND u.deleted_at IS NULL`

func dateValue(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}

	return pgtype.Date{Time: *value, Valid: true}
}
