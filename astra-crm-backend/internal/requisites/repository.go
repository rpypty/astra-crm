package requisites

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

var (
	ErrNotFound           = errors.New("requisite not found")
	ErrAssignmentNotFound = errors.New("requisite assignment not found")
	ErrPhoneBankDuplicate = errors.New("requisite phone and bank duplicate")
	ErrProxyDuplicate     = errors.New("requisite proxy duplicate")
	ErrBankNotFound       = errors.New("requisite bank not found")
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusArchived = "archived"
)

const (
	AssignmentStatusPlanned   = "planned"
	AssignmentStatusAssigned  = "assigned"
	AssignmentStatusInWork    = "in_work"
	AssignmentStatusWorked    = "worked"
	AssignmentStatusBlocked   = "blocked"
	AssignmentStatusCancelled = "cancelled"
	AssignmentStatusExpired   = "expired"
)

type Requisite struct {
	ID              int64
	TeamID          int64
	Phone           string
	MethodType      string
	BankCode        string
	BankName        string
	Proxy           *string
	EmployeeComment *string
	HolderName      *string
	CardNumber      *string
	DetailsFilledAt *time.Time
	DetailsFilledBy *int64
	Status          string
	CreatedBy       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type RequisiteDetails struct {
	Requisite
	ActiveAssignmentID  *int64
	AssignedTraderID    *int64
	AssignedTraderLogin *string
	AssignmentStatus    *string
	AssignedForDate     *time.Time
	TargetTurnoverMinor int64
}

type Assignment struct {
	ID                  int64
	TeamID              int64
	RequisiteID         int64
	TraderID            int64
	AssignedBy          int64
	AssignedAt          time.Time
	UnassignedAt        *time.Time
	Comment             *string
	Status              string
	AssignedForDate     time.Time
	TargetTurnoverMinor int64
	StartedAt           *time.Time
	CompletedAt         *time.Time
	CancelledAt         *time.Time
	ShiftRequisiteID    *int64
	UpdatedAt           time.Time
	WasReassign         bool
}

type AssignmentEvent struct {
	ID           int64
	TeamID       int64
	AssignmentID int64
	ActorID      int64
	Action       string
	BeforeJSON   []byte
	AfterJSON    []byte
	Comment      *string
	CreatedAt    time.Time
}

type AssignmentWorkRow struct {
	AssignmentID          int64
	TeamID                int64
	RequisiteID           int64
	Phone                 string
	BankCode              string
	BankName              string
	Proxy                 *string
	TraderID              int64
	TraderLogin           string
	Status                string
	AssignedForDate       time.Time
	TargetTurnoverMinor   int64
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	ClosingBalanceMinor   int64
	CardNumber            *string
	HolderName            *string
	TakenAt               *time.Time
	ReleasedAt            *time.Time
	Comment               *string
	AssignedAt            time.Time
	StartedAt             *time.Time
	CompletedAt           *time.Time
	UpdatedAt             time.Time
	ShiftRequisiteID      *int64
}

type RequisiteReport struct {
	Summary RequisiteReportSummary
	Shifts  []RequisiteReportShift
}

type RequisiteReportSummary struct {
	ID                         int64
	TeamID                     int64
	Phone                      string
	MethodType                 string
	BankCode                   string
	BankName                   string
	Proxy                      *string
	EmployeeComment            *string
	HolderName                 *string
	CardNumber                 *string
	Status                     string
	TotalInboundTurnoverMinor  int64
	TotalOutboundTurnoverMinor int64
	LastClosingBalanceMinor    int64
	LatestStatus               string
	LastActivityAt             *time.Time
	LastShiftRequisiteID       *int64
}

type RequisiteReportShift struct {
	ShiftRequisiteID      int64
	ShiftID               int64
	TeamID                int64
	RequisiteID           int64
	TraderID              int64
	TraderLogin           string
	ShiftStartedAt        time.Time
	ShiftClosedAt         *time.Time
	ShiftStatus           string
	TakenAt               time.Time
	ReleasedAt            *time.Time
	RequisiteStatus       string
	InboundTurnoverMinor  int64
	OutboundTurnoverMinor int64
	TargetTurnoverMinor   int64
	ClosingBalanceMinor   int64
	CardNumber            *string
	HolderName            *string
	AssignedForDate       *time.Time
	AssignmentStatus      string
}

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	queries *db.Queries
	db      txBeginner
}

func NewRepository(queries *db.Queries, txDB ...txBeginner) *Repository {
	var database txBeginner
	if len(txDB) > 0 {
		database = txDB[0]
	}
	return &Repository{queries: queries, db: database}
}

func (r *Repository) GetByIDForTeam(ctx context.Context, id int64, teamID int64) (Requisite, error) {
	row, err := r.queries.GetRequisiteByIDForTeam(ctx, db.GetRequisiteByIDForTeamParams{
		ID:     id,
		TeamID: teamID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Requisite{}, ErrNotFound
	}
	if err != nil {
		return Requisite{}, err
	}

	return Requisite{
		ID:              row.ID,
		TeamID:          row.TeamID,
		Phone:           row.Phone,
		MethodType:      row.MethodType,
		BankCode:        row.BankCode,
		Proxy:           textPtr(row.Proxy),
		EmployeeComment: textPtr(row.EmployeeComment),
		HolderName:      textPtr(row.HolderName),
		CardNumber:      textPtr(row.CardNumber),
		DetailsFilledAt: timePtr(row.DetailsFilledAt),
		DetailsFilledBy: int64Ptr(row.DetailsFilledBy),
		Status:          row.Status,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) Create(ctx context.Context, params CreateRecord) (Requisite, error) {
	row, err := r.queries.CreateRequisite(ctx, db.CreateRequisiteParams{
		TeamID:          params.TeamID,
		Phone:           params.Phone,
		MethodType:      params.MethodType,
		BankCode:        params.BankCode,
		Proxy:           textValue(params.Proxy),
		EmployeeComment: textValue(params.EmployeeComment),
		CreatedBy:       params.CreatedBy,
	})
	if err != nil {
		return Requisite{}, mapRequisiteWriteError(err)
	}

	return fromCreateRow(row), nil
}

func (r *Repository) GetDetails(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error) {
	row, err := r.queries.GetRequisiteDetailsByIDForTeam(ctx, db.GetRequisiteDetailsByIDForTeamParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return RequisiteDetails{}, ErrNotFound
	}
	if err != nil {
		return RequisiteDetails{}, err
	}

	return fromDetailsRow(row), nil
}

func (r *Repository) ListDetails(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[RequisiteDetails], error) {
	page = pagination.Normalize(page)
	params = normalizeListParams(params)
	queryParams := db.ListRequisiteDetailsByTeamParams{
		TeamID:               teamID,
		BankCode:             params.BankCode,
		Status:               params.Status,
		AvailableForPlanning: params.AvailableForPlanning,
		TraderFilter:         params.TraderID,
		Search:               params.Search,
		SearchDigits:         digitsOnly(params.Search),
		OffsetCount:          paginationOffset32(page),
		LimitCount:           paginationLimit32(page),
	}
	rows, err := r.queries.ListRequisiteDetailsByTeam(ctx, queryParams)
	if err != nil {
		return pagination.Result[RequisiteDetails]{}, err
	}

	items := make([]RequisiteDetails, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromListDetailsRow(row))
	}

	total, err := r.queries.CountRequisiteDetailsByTeam(ctx, db.CountRequisiteDetailsByTeamParams{
		TeamID:               queryParams.TeamID,
		BankCode:             queryParams.BankCode,
		Status:               queryParams.Status,
		AvailableForPlanning: queryParams.AvailableForPlanning,
		TraderFilter:         queryParams.TraderFilter,
		Search:               queryParams.Search,
		SearchDigits:         queryParams.SearchDigits,
	})
	if err != nil {
		return pagination.Result[RequisiteDetails]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) Update(ctx context.Context, params UpdateRecord) (Requisite, error) {
	row, err := r.queries.UpdateRequisite(ctx, db.UpdateRequisiteParams{
		TeamID:          params.TeamID,
		RequisiteID:     params.RequisiteID,
		Phone:           params.Phone,
		MethodType:      params.MethodType,
		BankCode:        params.BankCode,
		Proxy:           textValue(params.Proxy),
		EmployeeComment: textValue(params.EmployeeComment),
		Status:          params.Status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Requisite{}, ErrNotFound
	}
	if err != nil {
		return Requisite{}, mapRequisiteWriteError(err)
	}

	return fromUpdateRow(row), nil
}

func (r *Repository) Assign(ctx context.Context, params AssignRecord) (Assignment, error) {
	if r.db == nil {
		return Assignment{}, errors.New("requisites repository: transaction db is not configured")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Assignment{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := r.queries.WithTx(tx)
	_, err = queries.LockAssignableRequisiteForAssignment(ctx, db.LockAssignableRequisiteForAssignmentParams{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrRequisiteInOpenShift
	}
	if err != nil {
		return Assignment{}, err
	}

	closedAssignments, err := queries.CloseActiveRequisiteAssignment(ctx, db.CloseActiveRequisiteAssignmentParams{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
	})
	if err != nil {
		return Assignment{}, err
	}

	row, err := queries.CreateRequisiteAssignment(ctx, db.CreateRequisiteAssignmentParams{
		TeamID:              params.TeamID,
		RequisiteID:         params.RequisiteID,
		TraderID:            params.TraderID,
		AssignedBy:          params.AssignedBy,
		Comment:             textValue(params.Comment),
		Status:              assignmentStatusOrDefault(params.Status),
		AssignedForDate:     dateValue(assignmentDateOrToday(params.AssignedForDate)),
		TargetTurnoverMinor: params.TargetTurnoverMinor,
	})
	if err != nil {
		return Assignment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Assignment{}, err
	}
	committed = true

	assignment := fromDBAssignment(row)
	assignment.WasReassign = len(closedAssignments) > 0
	return assignment, nil
}

func (r *Repository) CreatePlan(ctx context.Context, params CreatePlanRecord) (Assignment, error) {
	row, err := r.queries.CreateRequisiteAssignment(ctx, db.CreateRequisiteAssignmentParams{
		TeamID:              params.TeamID,
		RequisiteID:         params.RequisiteID,
		TraderID:            params.TraderID,
		AssignedBy:          params.AssignedBy,
		Comment:             textValue(params.Comment),
		Status:              AssignmentStatusPlanned,
		AssignedForDate:     dateValue(params.AssignedForDate),
		TargetTurnoverMinor: params.TargetTurnoverMinor,
	})
	if err != nil {
		return Assignment{}, err
	}

	return fromDBAssignment(row), nil
}

func (r *Repository) Unassign(ctx context.Context, teamID int64, requisiteID int64) (Assignment, error) {
	row, err := r.queries.UnassignRequisite(ctx, db.UnassignRequisiteParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound
	}
	if err != nil {
		return Assignment{}, err
	}

	return fromDBAssignment(row), nil
}

func (r *Repository) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64, page pagination.Params) (pagination.Result[Assignment], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListRequisiteAssignmentHistory(ctx, db.ListRequisiteAssignmentHistoryParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[Assignment]{}, err
	}

	items := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBAssignment(row))
	}

	total, err := r.queries.CountRequisiteAssignmentHistory(ctx, db.CountRequisiteAssignmentHistoryParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if err != nil {
		return pagination.Result[Assignment]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) ListActiveAssignmentsByTrader(ctx context.Context, teamID int64, traderID int64) ([]Assignment, error) {
	rows, err := r.queries.ListActiveRequisiteAssignmentsByTrader(ctx, db.ListActiveRequisiteAssignmentsByTraderParams{
		TeamID:   teamID,
		TraderID: traderID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBAssignment(row))
	}

	return items, nil
}

func (r *Repository) ListPlans(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[AssignmentWorkRow], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListTeamleadRequisitePlans(ctx, db.ListTeamleadRequisitePlansParams{
		TeamID:      teamID,
		OffsetCount: paginationOffset32(page),
		LimitCount:  paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignmentWorkRow]{}, err
	}

	items := make([]AssignmentWorkRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromPlanRow(row))
	}

	total, err := r.queries.CountTeamleadRequisitePlans(ctx, teamID)
	if err != nil {
		return pagination.Result[AssignmentWorkRow]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) ListActivity(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[AssignmentWorkRow], error) {
	page = pagination.Normalize(page)
	params = normalizeListParams(params)
	rows, err := r.queries.ListTeamleadRequisiteActivity(ctx, db.ListTeamleadRequisiteActivityParams{
		TeamID:       teamID,
		BankCode:     params.BankCode,
		Status:       params.Status,
		TraderFilter: params.TraderID,
		Search:       params.Search,
		SearchDigits: digitsOnly(params.Search),
		OffsetCount:  paginationOffset32(page),
		LimitCount:   paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignmentWorkRow]{}, err
	}

	items := make([]AssignmentWorkRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromActivityRow(row))
	}

	total, err := r.queries.CountTeamleadRequisiteActivity(ctx, db.CountTeamleadRequisiteActivityParams{
		TeamID:       teamID,
		BankCode:     params.BankCode,
		Status:       params.Status,
		TraderFilter: params.TraderID,
		Search:       params.Search,
		SearchDigits: digitsOnly(params.Search),
	})
	if err != nil {
		return pagination.Result[AssignmentWorkRow]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

func (r *Repository) Report(ctx context.Context, teamID int64, requisiteID int64) (RequisiteReport, error) {
	summaryRow, err := r.queries.GetRequisiteReportSummary(ctx, db.GetRequisiteReportSummaryParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return RequisiteReport{}, ErrNotFound
	}
	if err != nil {
		return RequisiteReport{}, err
	}

	shiftRows, err := r.queries.ListRequisiteReportShifts(ctx, db.ListRequisiteReportShiftsParams{
		TeamID:      teamID,
		RequisiteID: requisiteID,
	})
	if err != nil {
		return RequisiteReport{}, err
	}

	shifts := make([]RequisiteReportShift, 0, len(shiftRows))
	for _, row := range shiftRows {
		shifts = append(shifts, fromReportShiftRow(row))
	}

	return RequisiteReport{
		Summary: fromReportSummaryRow(summaryRow),
		Shifts:  shifts,
	}, nil
}

func (r *Repository) GetAssignment(ctx context.Context, teamID int64, assignmentID int64) (Assignment, error) {
	row, err := r.queries.GetRequisiteAssignmentByIDForTeam(ctx, db.GetRequisiteAssignmentByIDForTeamParams{
		TeamID:       teamID,
		AssignmentID: assignmentID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound
	}
	if err != nil {
		return Assignment{}, err
	}

	return fromDBAssignment(row), nil
}

func (r *Repository) UpdatePlan(ctx context.Context, params UpdatePlanRecord) (Assignment, error) {
	row, err := r.queries.UpdateRequisiteAssignmentPlan(ctx, db.UpdateRequisiteAssignmentPlanParams{
		TeamID:              params.TeamID,
		AssignmentID:        params.AssignmentID,
		TraderID:            params.TraderID,
		RequisiteID:         params.RequisiteID,
		AssignedForDate:     dateValue(params.AssignedForDate),
		TargetTurnoverMinor: params.TargetTurnoverMinor,
		Comment:             textValue(params.Comment),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound
	}
	if err != nil {
		return Assignment{}, err
	}

	return fromDBAssignment(row), nil
}

func (r *Repository) CancelPlan(ctx context.Context, teamID int64, assignmentID int64) (Assignment, error) {
	row, err := r.queries.CancelRequisiteAssignmentPlan(ctx, db.CancelRequisiteAssignmentPlanParams{
		TeamID:       teamID,
		AssignmentID: assignmentID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound
	}
	if err != nil {
		return Assignment{}, err
	}

	return fromDBAssignment(row), nil
}

func (r *Repository) CreateAssignmentEvent(ctx context.Context, params AssignmentEventRecord) (AssignmentEvent, error) {
	row, err := r.queries.CreateRequisiteAssignmentEvent(ctx, db.CreateRequisiteAssignmentEventParams{
		TeamID:       params.TeamID,
		AssignmentID: params.AssignmentID,
		ActorID:      params.ActorID,
		Action:       params.Action,
		BeforeJson:   params.BeforeJSON,
		AfterJson:    params.AfterJSON,
		Comment:      textValue(params.Comment),
	})
	if err != nil {
		return AssignmentEvent{}, err
	}

	return fromDBAssignmentEvent(row), nil
}

func (r *Repository) AssignmentEvents(ctx context.Context, teamID int64, assignmentID int64, page pagination.Params) (pagination.Result[AssignmentEvent], error) {
	page = pagination.Normalize(page)
	rows, err := r.queries.ListRequisiteAssignmentEvents(ctx, db.ListRequisiteAssignmentEventsParams{
		TeamID:       teamID,
		AssignmentID: assignmentID,
		OffsetCount:  paginationOffset32(page),
		LimitCount:   paginationLimit32(page),
	})
	if err != nil {
		return pagination.Result[AssignmentEvent]{}, err
	}

	items := make([]AssignmentEvent, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromDBAssignmentEvent(row))
	}

	total, err := r.queries.CountRequisiteAssignmentEvents(ctx, db.CountRequisiteAssignmentEventsParams{
		TeamID:       teamID,
		AssignmentID: assignmentID,
	})
	if err != nil {
		return pagination.Result[AssignmentEvent]{}, err
	}

	return pagination.NewResult(items, page, total), nil
}

type CreateRecord struct {
	TeamID          int64
	Phone           string
	MethodType      string
	BankCode        string
	Proxy           *string
	EmployeeComment *string
	CreatedBy       int64
}

type UpdateRecord struct {
	TeamID          int64
	RequisiteID     int64
	Phone           string
	MethodType      string
	BankCode        string
	Proxy           *string
	EmployeeComment *string
	Status          string
}

type AssignRecord struct {
	TeamID              int64
	RequisiteID         int64
	TraderID            int64
	AssignedBy          int64
	Comment             *string
	Status              string
	AssignedForDate     time.Time
	TargetTurnoverMinor int64
}

type CreatePlanRecord struct {
	TeamID              int64
	RequisiteID         int64
	TraderID            int64
	AssignedBy          int64
	AssignedForDate     time.Time
	TargetTurnoverMinor int64
	Comment             *string
}

type UpdatePlanRecord struct {
	TeamID              int64
	AssignmentID        int64
	RequisiteID         int64
	TraderID            int64
	AssignedForDate     time.Time
	TargetTurnoverMinor int64
	Comment             *string
}

type AssignmentEventRecord struct {
	TeamID       int64
	AssignmentID int64
	ActorID      int64
	Action       string
	BeforeJSON   []byte
	AfterJSON    []byte
	Comment      *string
}

func fromCreateRow(row db.CreateRequisiteRow) Requisite {
	return Requisite{
		ID:              row.ID,
		TeamID:          row.TeamID,
		Phone:           row.Phone,
		MethodType:      row.MethodType,
		BankCode:        row.BankCode,
		Proxy:           textPtr(row.Proxy),
		EmployeeComment: textPtr(row.EmployeeComment),
		HolderName:      textPtr(row.HolderName),
		CardNumber:      textPtr(row.CardNumber),
		DetailsFilledAt: timePtr(row.DetailsFilledAt),
		DetailsFilledBy: int64Ptr(row.DetailsFilledBy),
		Status:          row.Status,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func fromUpdateRow(row db.UpdateRequisiteRow) Requisite {
	return Requisite{
		ID:              row.ID,
		TeamID:          row.TeamID,
		Phone:           row.Phone,
		MethodType:      row.MethodType,
		BankCode:        row.BankCode,
		Proxy:           textPtr(row.Proxy),
		EmployeeComment: textPtr(row.EmployeeComment),
		HolderName:      textPtr(row.HolderName),
		CardNumber:      textPtr(row.CardNumber),
		DetailsFilledAt: timePtr(row.DetailsFilledAt),
		DetailsFilledBy: int64Ptr(row.DetailsFilledBy),
		Status:          row.Status,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func fromDetailsRow(row db.GetRequisiteDetailsByIDForTeamRow) RequisiteDetails {
	return RequisiteDetails{
		Requisite: Requisite{
			ID:              row.ID,
			TeamID:          row.TeamID,
			Phone:           row.Phone,
			MethodType:      row.MethodType,
			BankCode:        row.BankCode,
			BankName:        row.BankName,
			Proxy:           textPtr(row.Proxy),
			EmployeeComment: textPtr(row.EmployeeComment),
			HolderName:      textPtr(row.HolderName),
			CardNumber:      textPtr(row.CardNumber),
			DetailsFilledAt: timePtr(row.DetailsFilledAt),
			DetailsFilledBy: int64Ptr(row.DetailsFilledBy),
			Status:          row.Status,
			CreatedBy:       row.CreatedBy,
			CreatedAt:       row.CreatedAt.Time,
			UpdatedAt:       row.UpdatedAt.Time,
		},
		ActiveAssignmentID:  int64ZeroPtr(row.ActiveAssignmentID),
		AssignedTraderID:    int64ZeroPtr(row.AssignedTraderID),
		AssignedTraderLogin: stringPtr(row.AssignedTraderLogin),
		AssignmentStatus:    stringPtr(row.AssignmentStatus),
		AssignedForDate:     datePtr(row.AssignedForDate),
		TargetTurnoverMinor: row.TargetTurnoverMinor,
	}
}

func fromListDetailsRow(row db.ListRequisiteDetailsByTeamRow) RequisiteDetails {
	return RequisiteDetails{
		Requisite: Requisite{
			ID:              row.ID,
			TeamID:          row.TeamID,
			Phone:           row.Phone,
			MethodType:      row.MethodType,
			BankCode:        row.BankCode,
			BankName:        row.BankName,
			Proxy:           textPtr(row.Proxy),
			EmployeeComment: textPtr(row.EmployeeComment),
			HolderName:      textPtr(row.HolderName),
			CardNumber:      textPtr(row.CardNumber),
			DetailsFilledAt: timePtr(row.DetailsFilledAt),
			DetailsFilledBy: int64Ptr(row.DetailsFilledBy),
			Status:          row.Status,
			CreatedBy:       row.CreatedBy,
			CreatedAt:       row.CreatedAt.Time,
			UpdatedAt:       row.UpdatedAt.Time,
		},
		ActiveAssignmentID:  int64ZeroPtr(row.ActiveAssignmentID),
		AssignedTraderID:    int64ZeroPtr(row.AssignedTraderID),
		AssignedTraderLogin: stringPtr(row.AssignedTraderLogin),
		AssignmentStatus:    stringPtr(row.AssignmentStatus),
		AssignedForDate:     datePtr(row.AssignedForDate),
		TargetTurnoverMinor: row.TargetTurnoverMinor,
	}
}

func fromDBAssignment(row db.RequisiteAssignment) Assignment {
	return Assignment{
		ID:                  row.ID,
		TeamID:              row.TeamID,
		RequisiteID:         row.RequisiteID,
		TraderID:            row.TraderID,
		AssignedBy:          row.AssignedBy,
		AssignedAt:          row.AssignedAt.Time,
		UnassignedAt:        timePtr(row.UnassignedAt),
		Comment:             textPtr(row.Comment),
		Status:              row.Status,
		AssignedForDate:     row.AssignedForDate.Time,
		TargetTurnoverMinor: row.TargetTurnoverMinor,
		StartedAt:           timePtr(row.StartedAt),
		CompletedAt:         timePtr(row.CompletedAt),
		CancelledAt:         timePtr(row.CancelledAt),
		ShiftRequisiteID:    int64Ptr(row.ShiftRequisiteID),
		UpdatedAt:           row.UpdatedAt.Time,
	}
}

func fromPlanRow(row db.ListTeamleadRequisitePlansRow) AssignmentWorkRow {
	return AssignmentWorkRow{
		AssignmentID:          row.AssignmentID,
		TeamID:                row.TeamID,
		RequisiteID:           row.RequisiteID,
		Phone:                 row.Phone,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		TraderID:              row.TraderID,
		TraderLogin:           row.TraderLogin,
		Status:                row.Status,
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
		Comment:               textPtr(row.Comment),
		AssignedAt:            row.AssignedAt.Time,
		StartedAt:             timePtr(row.StartedAt),
		CompletedAt:           timePtr(row.CompletedAt),
		UpdatedAt:             row.UpdatedAt.Time,
		ShiftRequisiteID:      int64Ptr(row.ShiftRequisiteID),
	}
}

func fromActivityRow(row db.ListTeamleadRequisiteActivityRow) AssignmentWorkRow {
	return AssignmentWorkRow{
		AssignmentID:          row.AssignmentID,
		TeamID:                row.TeamID,
		RequisiteID:           row.RequisiteID,
		Phone:                 row.Phone,
		BankCode:              row.BankCode,
		BankName:              row.BankName,
		Proxy:                 textPtr(row.Proxy),
		TraderID:              row.TraderID,
		TraderLogin:           row.TraderLogin,
		Status:                row.Status,
		AssignedForDate:       row.AssignedForDate.Time,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
		CardNumber:            textPtr(row.CardNumber),
		HolderName:            textPtr(row.HolderName),
		TakenAt:               timePtr(row.TakenAt),
		ReleasedAt:            timePtr(row.ReleasedAt),
		Comment:               textPtr(row.Comment),
		AssignedAt:            row.AssignedAt.Time,
		StartedAt:             timePtr(row.StartedAt),
		CompletedAt:           timePtr(row.CompletedAt),
		ShiftRequisiteID:      int64Ptr(row.ShiftRequisiteID),
	}
}

func fromDBAssignmentEvent(row db.RequisiteAssignmentEvent) AssignmentEvent {
	return AssignmentEvent{
		ID:           row.ID,
		TeamID:       row.TeamID,
		AssignmentID: row.AssignmentID,
		ActorID:      row.ActorID,
		Action:       row.Action,
		BeforeJSON:   row.BeforeJson,
		AfterJSON:    row.AfterJson,
		Comment:      textPtr(row.Comment),
		CreatedAt:    row.CreatedAt.Time,
	}
}

func fromReportSummaryRow(row db.GetRequisiteReportSummaryRow) RequisiteReportSummary {
	return RequisiteReportSummary{
		ID:                         row.ID,
		TeamID:                     row.TeamID,
		Phone:                      row.Phone,
		MethodType:                 row.MethodType,
		BankCode:                   row.BankCode,
		BankName:                   row.BankName,
		Proxy:                      textPtr(row.Proxy),
		EmployeeComment:            textPtr(row.EmployeeComment),
		HolderName:                 textPtr(row.HolderName),
		CardNumber:                 textPtr(row.CardNumber),
		Status:                     row.Status,
		TotalInboundTurnoverMinor:  row.TotalInboundTurnoverMinor,
		TotalOutboundTurnoverMinor: row.TotalOutboundTurnoverMinor,
		LastClosingBalanceMinor:    row.LastClosingBalanceMinor,
		LatestStatus:               row.LatestStatus,
		LastActivityAt:             timePtr(row.LastActivityAt),
		LastShiftRequisiteID:       int64Ptr(row.LastShiftRequisiteID),
	}
}

func fromReportShiftRow(row db.ListRequisiteReportShiftsRow) RequisiteReportShift {
	return RequisiteReportShift{
		ShiftRequisiteID:      row.ShiftRequisiteID,
		ShiftID:               row.ShiftID,
		TeamID:                row.TeamID,
		RequisiteID:           row.RequisiteID,
		TraderID:              row.TraderID,
		TraderLogin:           row.TraderLogin,
		ShiftStartedAt:        row.ShiftStartedAt.Time,
		ShiftClosedAt:         timePtr(row.ShiftClosedAt),
		ShiftStatus:           row.ShiftStatus,
		TakenAt:               row.TakenAt.Time,
		ReleasedAt:            timePtr(row.ReleasedAt),
		RequisiteStatus:       row.RequisiteStatus,
		InboundTurnoverMinor:  row.InboundTurnoverMinor,
		OutboundTurnoverMinor: row.OutboundTurnoverMinor,
		TargetTurnoverMinor:   row.TargetTurnoverMinor,
		ClosingBalanceMinor:   row.ClosingBalanceMinor,
		CardNumber:            stringPtr(row.CardNumber),
		HolderName:            stringPtr(row.HolderName),
		AssignedForDate:       datePtr(row.AssignedForDate),
		AssignmentStatus:      row.AssignmentStatus,
	}
}

func mapRequisiteWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch {
	case pgErr.Code == "23505" && pgErr.ConstraintName == "uq_requisites_active_team_phone_bank":
		return ErrPhoneBankDuplicate
	case pgErr.Code == "23505" && pgErr.ConstraintName == "uq_requisites_active_proxy_global":
		return ErrProxyDuplicate
	case pgErr.Code == "23503" && pgErr.ConstraintName == "fk_requisites_bank_code":
		return ErrBankNotFound
	default:
		return err
	}
}

func textValue(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func paginationOffset32(params pagination.Params) int32 {
	offset := pagination.Offset(params)
	if offset > math.MaxInt32 {
		return math.MaxInt32
	}

	return int32(offset)
}

func paginationLimit32(params pagination.Params) int32 {
	params = pagination.Normalize(params)
	return int32(params.PageSize)
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func int64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	return &value.Int64
}

func int64ZeroPtr(value int64) *int64 {
	if value == 0 {
		return nil
	}

	return &value
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

func datePtr(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

func dateValue(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

func assignmentDateOrToday(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}

	return value
}

func assignmentStatusOrDefault(status string) string {
	if status == "" {
		return AssignmentStatusAssigned
	}

	return status
}
