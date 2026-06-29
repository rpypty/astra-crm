package readmodels

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

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
	ap.date_from,
	ap.date_to,
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
			&item.DateFrom,
			&item.DateTo,
			&item.InboundStatus,
			&item.OutboundStatus,
			&item.Status,
		); err != nil {
			return pagination.Result[AccountingPeriod]{}, fmt.Errorf("scan period: %w", err)
		}
		item.Title = periodTitle(item.DateFrom, item.DateTo)
		item.DateRange = periodDateRange(item.DateFrom, item.DateTo)
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

	period, periodRange, periodTitleValue, err := s.resolveTraderProfilePeriod(ctx, teamID, filters)
	if err != nil {
		return TraderProfile{}, err
	}

	var profile TraderProfile
	todayFrom, todayTo := currentDayRange(time.Now())

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return s.pool.QueryRow(groupCtx, traderProfileBaseQuery, teamID, traderID).Scan(
			&profile.ID,
			&profile.Login,
			&profile.SalaryRateBps,
			&profile.ExternalWorkerName,
		)
	})
	g.Go(func() error {
		amount, err := s.traderSuccessInboundAmount(groupCtx, teamID, traderID, &todayFrom, &todayTo)
		if err != nil {
			return err
		}
		profile.CurrentShiftSuccessInboundMinor = amount
		return nil
	})
	g.Go(func() error {
		if !periodRange.Active {
			profile.PeriodSuccessInboundMinor = 0
			return nil
		}
		amount, err := s.traderSuccessInboundAmount(groupCtx, teamID, traderID, periodRange.From, periodRange.ToExclusive)
		if err != nil {
			return err
		}
		profile.PeriodSuccessInboundMinor = amount
		return nil
	})
	if err := g.Wait(); err != nil {
		return TraderProfile{}, fmt.Errorf("get trader profile readmodel: %w", err)
	}

	if period != nil {
		profile.PeriodID = &period.ID
	}
	profile.PeriodTitle = periodTitleValue
	profile.CurrentShiftSalaryMinor = salaryMinor(profile.CurrentShiftSuccessInboundMinor, profile.SalaryRateBps)
	profile.PeriodSalaryMinor = salaryMinor(profile.PeriodSuccessInboundMinor, profile.SalaryRateBps)

	return profile, nil
}

type traderProfilePeriod struct {
	ID       int64
	DateFrom time.Time
	DateTo   time.Time
}

type optionalTimeRange struct {
	From        *time.Time
	ToExclusive *time.Time
	Active      bool
}

const traderProfileBaseQuery = `
SELECT u.id, u.login, tp.salary_rate_bps, tp.external_worker_name
FROM users u
JOIN trader_profiles tp ON tp.user_id = u.id
WHERE u.team_id = $1
  AND u.id = $2
  AND u.role = 'trader'
  AND u.deleted_at IS NULL`

const currentAccountingPeriodQuery = `
SELECT id, date_from, date_to
FROM accounting_periods
WHERE team_id = $1
ORDER BY (status <> 'open'), date_to DESC, id DESC
LIMIT 1`

const traderSuccessInboundAmountQuery = `
SELECT COALESCE(sum(osi.amount_minor), 0)::bigint
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
  AND ($3::timestamptz IS NULL OR osi.created_at_external >= $3::timestamptz)
  AND ($4::timestamptz IS NULL OR osi.created_at_external < $4::timestamptz)`

func (s *Service) resolveTraderProfilePeriod(ctx context.Context, teamID int64, filters PeriodFilter) (*traderProfilePeriod, optionalTimeRange, *string, error) {
	if filters.DateFrom != nil || filters.DateTo != nil {
		filterTitle := filteredPeriodTitle(filters.DateFrom, filters.DateTo)
		return nil, optionalTimeRange{
			From:        dayStartPtr(filters.DateFrom),
			ToExclusive: dayAfterPtr(filters.DateTo),
			Active:      true,
		}, &filterTitle, nil
	}

	var period traderProfilePeriod
	if err := s.pool.QueryRow(ctx, currentAccountingPeriodQuery, teamID).Scan(&period.ID, &period.DateFrom, &period.DateTo); err != nil {
		if err == pgx.ErrNoRows {
			return nil, optionalTimeRange{}, nil, nil
		}
		return nil, optionalTimeRange{}, nil, fmt.Errorf("current accounting period: %w", err)
	}

	title := periodTitle(period.DateFrom, period.DateTo)
	return &period, optionalTimeRange{
		From:        timePtr(dayStart(period.DateFrom)),
		ToExclusive: timePtr(dayAfter(period.DateTo)),
		Active:      true,
	}, &title, nil
}

func (s *Service) traderSuccessInboundAmount(ctx context.Context, teamID int64, traderID int64, from *time.Time, toExclusive *time.Time) (int64, error) {
	var amount int64
	if err := s.pool.QueryRow(ctx, traderSuccessInboundAmountQuery, teamID, traderID, from, toExclusive).Scan(&amount); err != nil {
		return 0, err
	}
	return amount, nil
}

func salaryMinor(amountMinor int64, salaryRateBps int64) int64 {
	return amountMinor * salaryRateBps / 10000
}

func periodTitle(dateFrom time.Time, dateTo time.Time) string {
	return "Период " + periodDateRange(dateFrom, dateTo)
}

func periodDateRange(dateFrom time.Time, dateTo time.Time) string {
	return formatPeriodDate(dateFrom) + " - " + formatPeriodDate(dateTo)
}

func filteredPeriodTitle(dateFrom *time.Time, dateTo *time.Time) string {
	switch {
	case dateFrom != nil && dateTo != nil:
		return "Период " + periodDateRange(*dateFrom, *dateTo)
	case dateFrom != nil:
		return "Период с " + formatPeriodDate(*dateFrom)
	case dateTo != nil:
		return "Период до " + formatPeriodDate(*dateTo)
	default:
		return ""
	}
}

func formatPeriodDate(value time.Time) string {
	return value.Format("02.01.2006")
}

func currentDayRange(now time.Time) (time.Time, time.Time) {
	from := dayStart(now)
	return from, from.AddDate(0, 0, 1)
}

func dayStart(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func dayAfter(value time.Time) time.Time {
	return dayStart(value).AddDate(0, 0, 1)
}

func dayStartPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	return timePtr(dayStart(*value))
}

func dayAfterPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	return timePtr(dayAfter(*value))
}

func timePtr(value time.Time) *time.Time {
	return &value
}
