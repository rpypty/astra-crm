package requisites

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/pagination"
	"github.com/ashpak/astra-crm-backend/internal/users"
)

var (
	ErrInvalidInput         = errors.New("invalid requisite input")
	ErrInactiveTrader       = errors.New("trader is not active")
	ErrRequisiteInOpenShift = errors.New("requisite is already in open shift")
)

const defaultMethodType = "SBP"

type Store interface {
	Create(ctx context.Context, params CreateRecord) (Requisite, error)
	GetDetails(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error)
	ListDetails(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[RequisiteDetails], error)
	Update(ctx context.Context, params UpdateRecord) (Requisite, error)
	Assign(ctx context.Context, params AssignRecord) (Assignment, error)
	CreatePlan(ctx context.Context, params CreatePlanRecord) (Assignment, error)
	Unassign(ctx context.Context, teamID int64, requisiteID int64) (Assignment, error)
	AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64, page pagination.Params) (pagination.Result[Assignment], error)
	ListPlans(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[AssignmentWorkRow], error)
	ListActivity(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[AssignmentWorkRow], error)
	Report(ctx context.Context, teamID int64, requisiteID int64) (RequisiteReport, error)
	GetAssignment(ctx context.Context, teamID int64, assignmentID int64) (Assignment, error)
	UpdatePlan(ctx context.Context, params UpdatePlanRecord) (Assignment, error)
	CancelPlan(ctx context.Context, teamID int64, assignmentID int64) (Assignment, error)
	CreateAssignmentEvent(ctx context.Context, params AssignmentEventRecord) (AssignmentEvent, error)
	AssignmentEvents(ctx context.Context, teamID int64, assignmentID int64, page pagination.Params) (pagination.Result[AssignmentEvent], error)
}

type TraderReader interface {
	GetTraderByID(ctx context.Context, teamID int64, traderID int64) (users.Trader, error)
}

type AuditService interface {
	Write(ctx context.Context, event audit.Event) error
}

type Service struct {
	store   Store
	traders TraderReader
	audit   AuditService
}

func NewService(store Store, traders TraderReader, auditService AuditService) *Service {
	return &Service{
		store:   store,
		traders: traders,
		audit:   auditService,
	}
}

type CreateParams struct {
	ActorID          int64
	TeamID           int64
	Phone            string
	MethodType       string
	BankCode         string
	CardNumber       string
	Proxy            *string
	EmployeeComment  *string
	AssignedTraderID *int64
	AssignComment    *string
}

type PatchParams struct {
	ActorID         int64
	TeamID          int64
	RequisiteID     int64
	Phone           *string
	MethodType      *string
	BankCode        *string
	CardNumber      *string
	Proxy           *string
	EmployeeComment *string
	Status          *string
}

type AssignParams struct {
	ActorID     int64
	TeamID      int64
	RequisiteID int64
	TraderID    int64
	Comment     *string
}

type PlanParams struct {
	ActorID             int64
	TeamID              int64
	AssignmentID        int64
	RequisiteID         int64
	TraderID            int64
	AssignedForDate     time.Time
	TargetTurnoverMinor int64
	Comment             *string
}

type ListParams struct {
	Search               string
	BankCode             string
	Status               string
	TraderID             string
	AvailableForPlanning bool
}

func (s *Service) Create(ctx context.Context, params CreateParams) (RequisiteDetails, error) {
	phone, ok := normalizePhone(params.Phone)
	methodType := cleanMethodType(params.MethodType)
	bankCode := normalizeBankCode(params.BankCode)
	cardNumber, cardOK := normalizeCardNumber(params.CardNumber)
	proxy := cleanOptionalString(params.Proxy)
	employeeComment := cleanOptionalString(params.EmployeeComment)
	if params.ActorID <= 0 || params.TeamID <= 0 || !ok || bankCode == "" || !cardOK {
		return RequisiteDetails{}, ErrInvalidInput
	}
	if params.AssignedTraderID != nil {
		trader, err := s.traders.GetTraderByID(ctx, params.TeamID, *params.AssignedTraderID)
		if err != nil {
			return RequisiteDetails{}, err
		}
		if trader.Status != users.StatusActive {
			return RequisiteDetails{}, ErrInactiveTrader
		}
	}

	created, err := s.store.Create(ctx, CreateRecord{
		TeamID:          params.TeamID,
		Phone:           phone,
		MethodType:      methodType,
		BankCode:        bankCode,
		CardNumber:      cardNumber,
		Proxy:           proxy,
		EmployeeComment: employeeComment,
		CreatedBy:       params.ActorID,
	})
	if err != nil {
		return RequisiteDetails{}, err
	}

	details := RequisiteDetails{Requisite: created}
	if params.AssignedTraderID != nil {
		assignment, err := s.store.Assign(ctx, AssignRecord{
			TeamID:      params.TeamID,
			RequisiteID: created.ID,
			TraderID:    *params.AssignedTraderID,
			AssignedBy:  params.ActorID,
			Comment:     cleanOptionalString(params.AssignComment),
		})
		if err != nil {
			return RequisiteDetails{}, err
		}
		details.ActiveAssignmentID = &assignment.ID
		details.AssignedTraderID = &assignment.TraderID
		if err := s.writeAudit(ctx, audit.Event{
			TeamID:     params.TeamID,
			ActorID:    params.ActorID,
			Action:     audit.ActionRequisiteAssigned,
			EntityType: "requisite",
			EntityID:   strconv.FormatInt(created.ID, 10),
			After:      PublicAssignment(assignment),
			Comment:    cleanOptionalString(params.AssignComment),
		}); err != nil {
			return RequisiteDetails{}, err
		}
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     audit.ActionRequisiteCreated,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(created.ID, 10),
		After:      PublicRequisiteFromDetails(details),
	}); err != nil {
		return RequisiteDetails{}, err
	}

	return details, nil
}

func (s *Service) List(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[RequisiteDetails], error) {
	return s.store.ListDetails(ctx, teamID, normalizeListParams(params), page)
}

func (s *Service) Get(ctx context.Context, teamID int64, requisiteID int64) (RequisiteDetails, error) {
	return s.store.GetDetails(ctx, teamID, requisiteID)
}

func (s *Service) Patch(ctx context.Context, params PatchParams) (RequisiteDetails, error) {
	current, err := s.store.GetDetails(ctx, params.TeamID, params.RequisiteID)
	if err != nil {
		return RequisiteDetails{}, err
	}

	next := current.Requisite
	if params.Phone != nil {
		phone, ok := normalizePhone(*params.Phone)
		if !ok {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.Phone = phone
	}
	if params.MethodType != nil {
		methodType := cleanMethodType(*params.MethodType)
		if methodType == "" {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.MethodType = methodType
	}
	if params.BankCode != nil {
		bankCode := normalizeBankCode(*params.BankCode)
		if bankCode == "" {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.BankCode = bankCode
	}
	if params.CardNumber != nil {
		cardNumber, ok := normalizeCardNumber(*params.CardNumber)
		if !ok {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.CardNumber = &cardNumber
	}
	if params.Proxy != nil {
		next.Proxy = cleanOptionalString(params.Proxy)
	}
	if params.EmployeeComment != nil {
		next.EmployeeComment = cleanOptionalString(params.EmployeeComment)
	}
	if params.Status != nil {
		status := strings.TrimSpace(*params.Status)
		if !validStatus(status) {
			return RequisiteDetails{}, ErrInvalidInput
		}
		next.Status = status
	}

	updated, err := s.store.Update(ctx, UpdateRecord{
		TeamID:          params.TeamID,
		RequisiteID:     params.RequisiteID,
		Phone:           next.Phone,
		MethodType:      next.MethodType,
		BankCode:        next.BankCode,
		CardNumber:      next.CardNumber,
		Proxy:           next.Proxy,
		EmployeeComment: next.EmployeeComment,
		Status:          next.Status,
	})
	if err != nil {
		return RequisiteDetails{}, err
	}

	details := current
	details.Requisite = updated
	action := audit.ActionRequisiteUpdated
	if updated.Status == StatusArchived {
		action = audit.ActionRequisiteArchived
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     action,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(updated.ID, 10),
		Before:     PublicRequisiteFromDetails(current),
		After:      PublicRequisiteFromDetails(details),
	}); err != nil {
		return RequisiteDetails{}, err
	}

	return details, nil
}

func (s *Service) Delete(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error {
	status := StatusArchived
	_, err := s.Patch(ctx, PatchParams{
		ActorID:     actorID,
		TeamID:      teamID,
		RequisiteID: requisiteID,
		Status:      &status,
	})
	return err
}

func (s *Service) Assign(ctx context.Context, params AssignParams) (Assignment, error) {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.RequisiteID <= 0 || params.TraderID <= 0 {
		return Assignment{}, ErrInvalidInput
	}

	requisite, err := s.store.GetDetails(ctx, params.TeamID, params.RequisiteID)
	if err != nil {
		return Assignment{}, err
	}
	if requisite.Status != StatusActive {
		return Assignment{}, ErrInvalidInput
	}

	trader, err := s.traders.GetTraderByID(ctx, params.TeamID, params.TraderID)
	if err != nil {
		return Assignment{}, err
	}
	if trader.Status != users.StatusActive {
		return Assignment{}, ErrInactiveTrader
	}

	comment := cleanOptionalString(params.Comment)
	assignment, err := s.store.Assign(ctx, AssignRecord{
		TeamID:      params.TeamID,
		RequisiteID: params.RequisiteID,
		TraderID:    params.TraderID,
		AssignedBy:  params.ActorID,
		Comment:     comment,
	})
	if err != nil {
		return Assignment{}, err
	}

	action := audit.ActionRequisiteAssigned
	if assignment.WasReassign {
		action = audit.ActionRequisiteReassigned
	}

	if err := s.writeAudit(ctx, audit.Event{
		TeamID:     params.TeamID,
		ActorID:    params.ActorID,
		Action:     action,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(params.RequisiteID, 10),
		After:      PublicAssignment(assignment),
		Comment:    comment,
	}); err != nil {
		return Assignment{}, err
	}

	return assignment, nil
}

func (s *Service) Unassign(ctx context.Context, actorID int64, teamID int64, requisiteID int64) error {
	assignment, err := s.store.Unassign(ctx, teamID, requisiteID)
	if err != nil {
		return err
	}

	return s.writeAudit(ctx, audit.Event{
		TeamID:     teamID,
		ActorID:    actorID,
		Action:     audit.ActionRequisiteUnassigned,
		EntityType: "requisite",
		EntityID:   strconv.FormatInt(requisiteID, 10),
		After:      PublicAssignment(assignment),
	})
}

func (s *Service) AssignmentHistory(ctx context.Context, teamID int64, requisiteID int64, page pagination.Params) (pagination.Result[Assignment], error) {
	return s.store.AssignmentHistory(ctx, teamID, requisiteID, page)
}

func (s *Service) Plans(ctx context.Context, teamID int64, page pagination.Params) (pagination.Result[AssignmentWorkRow], error) {
	return s.store.ListPlans(ctx, teamID, page)
}

func (s *Service) Activity(ctx context.Context, teamID int64, params ListParams, page pagination.Params) (pagination.Result[AssignmentWorkRow], error) {
	return s.store.ListActivity(ctx, teamID, normalizeListParams(params), page)
}

func (s *Service) Report(ctx context.Context, teamID int64, requisiteID int64) (RequisiteReport, error) {
	if teamID <= 0 || requisiteID <= 0 {
		return RequisiteReport{}, ErrInvalidInput
	}

	return s.store.Report(ctx, teamID, requisiteID)
}

func (s *Service) CreatePlan(ctx context.Context, params PlanParams) (Assignment, error) {
	if err := s.validatePlanParams(ctx, params); err != nil {
		return Assignment{}, err
	}

	comment := cleanOptionalString(params.Comment)
	assignment, err := s.store.CreatePlan(ctx, CreatePlanRecord{
		TeamID:              params.TeamID,
		RequisiteID:         params.RequisiteID,
		TraderID:            params.TraderID,
		AssignedBy:          params.ActorID,
		AssignedForDate:     normalizeDate(params.AssignedForDate),
		TargetTurnoverMinor: params.TargetTurnoverMinor,
		Comment:             comment,
	})
	if err != nil {
		return Assignment{}, err
	}

	if err := s.writeAssignmentEvent(ctx, params.TeamID, params.ActorID, assignment.ID, "created", nil, PublicAssignment(assignment), comment); err != nil {
		return Assignment{}, err
	}

	return assignment, nil
}

func (s *Service) UpdatePlan(ctx context.Context, params PlanParams) (Assignment, error) {
	if params.AssignmentID <= 0 {
		return Assignment{}, ErrInvalidInput
	}
	if err := s.validatePlanParams(ctx, params); err != nil {
		return Assignment{}, err
	}

	before, err := s.store.GetAssignment(ctx, params.TeamID, params.AssignmentID)
	if err != nil {
		return Assignment{}, err
	}

	comment := cleanOptionalString(params.Comment)
	updated, err := s.store.UpdatePlan(ctx, UpdatePlanRecord{
		TeamID:              params.TeamID,
		AssignmentID:        params.AssignmentID,
		RequisiteID:         params.RequisiteID,
		TraderID:            params.TraderID,
		AssignedForDate:     normalizeDate(params.AssignedForDate),
		TargetTurnoverMinor: params.TargetTurnoverMinor,
		Comment:             comment,
	})
	if err != nil {
		return Assignment{}, err
	}

	if err := s.writeAssignmentEvent(ctx, params.TeamID, params.ActorID, updated.ID, "updated", PublicAssignment(before), PublicAssignment(updated), comment); err != nil {
		return Assignment{}, err
	}

	return updated, nil
}

func (s *Service) CancelPlan(ctx context.Context, actorID int64, teamID int64, assignmentID int64) (Assignment, error) {
	if actorID <= 0 || teamID <= 0 || assignmentID <= 0 {
		return Assignment{}, ErrInvalidInput
	}

	before, err := s.store.GetAssignment(ctx, teamID, assignmentID)
	if err != nil {
		return Assignment{}, err
	}
	cancelled, err := s.store.CancelPlan(ctx, teamID, assignmentID)
	if err != nil {
		return Assignment{}, err
	}

	if err := s.writeAssignmentEvent(ctx, teamID, actorID, assignmentID, "cancelled", PublicAssignment(before), PublicAssignment(cancelled), nil); err != nil {
		return Assignment{}, err
	}

	return cancelled, nil
}

func (s *Service) AssignmentEvents(ctx context.Context, teamID int64, assignmentID int64, page pagination.Params) (pagination.Result[AssignmentEvent], error) {
	if teamID <= 0 || assignmentID <= 0 {
		return pagination.Result[AssignmentEvent]{}, ErrInvalidInput
	}

	return s.store.AssignmentEvents(ctx, teamID, assignmentID, page)
}

func (s *Service) writeAudit(ctx context.Context, event audit.Event) error {
	if s.audit == nil {
		return nil
	}

	return s.audit.Write(ctx, event)
}

func (s *Service) validatePlanParams(ctx context.Context, params PlanParams) error {
	if params.ActorID <= 0 || params.TeamID <= 0 || params.RequisiteID <= 0 || params.TraderID <= 0 || params.TargetTurnoverMinor <= 0 || params.AssignedForDate.IsZero() {
		return ErrInvalidInput
	}

	planDate := normalizeDate(params.AssignedForDate)
	if planDate.IsZero() {
		return ErrInvalidInput
	}

	requisite, err := s.store.GetDetails(ctx, params.TeamID, params.RequisiteID)
	if err != nil {
		return err
	}
	if requisite.Status != StatusActive {
		return ErrInvalidInput
	}

	trader, err := s.traders.GetTraderByID(ctx, params.TeamID, params.TraderID)
	if err != nil {
		return err
	}
	if trader.Status != users.StatusActive {
		return ErrInactiveTrader
	}

	return nil
}

func (s *Service) writeAssignmentEvent(ctx context.Context, teamID int64, actorID int64, assignmentID int64, action string, before any, after any, comment *string) error {
	beforeJSON, err := marshalOptionalJSON(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalOptionalJSON(after)
	if err != nil {
		return err
	}

	_, err = s.store.CreateAssignmentEvent(ctx, AssignmentEventRecord{
		TeamID:       teamID,
		AssignmentID: assignmentID,
		ActorID:      actorID,
		Action:       action,
		BeforeJSON:   beforeJSON,
		AfterJSON:    afterJSON,
		Comment:      comment,
	})

	return err
}

func marshalOptionalJSON(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}

	return json.Marshal(value)
}

func normalizeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func cleanMethodType(value string) string {
	methodType := strings.TrimSpace(value)
	if methodType == "" {
		return defaultMethodType
	}

	return methodType
}

func normalizeBankCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validStatus(status string) bool {
	switch status {
	case StatusActive, StatusDisabled, StatusArchived:
		return true
	default:
		return false
	}
}

func normalizeListParams(params ListParams) ListParams {
	return ListParams{
		Search:               strings.TrimSpace(params.Search),
		BankCode:             normalizeBankCode(params.BankCode),
		Status:               strings.TrimSpace(params.Status),
		TraderID:             strings.TrimSpace(params.TraderID),
		AvailableForPlanning: params.AvailableForPlanning,
	}
}

func requisiteMatchesListParams(item RequisiteDetails, params ListParams) bool {
	if params.BankCode != "" && params.BankCode != "all" && item.BankCode != params.BankCode {
		return false
	}
	if params.Status != "" && params.Status != "all" && item.Status != params.Status {
		return false
	}
	if params.AvailableForPlanning && (item.Status != StatusActive || stringPtrValue(item.AssignmentStatus) == AssignmentStatusBlocked) {
		return false
	}
	if params.TraderID != "" && params.TraderID != "all" {
		if params.TraderID == "unassigned" {
			if item.AssignedTraderID != nil {
				return false
			}
		} else if item.AssignedTraderID == nil || strconv.FormatInt(*item.AssignedTraderID, 10) != params.TraderID {
			return false
		}
	}
	if params.Search == "" {
		return true
	}

	return requisiteMatchesSearch(item, params.Search)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func requisiteMatchesSearch(item RequisiteDetails, search string) bool {
	query := strings.ToLower(strings.TrimSpace(search))
	queryDigits := digitsOnly(query)
	if queryDigits != "" && strings.Contains(digitsOnly(item.Phone), queryDigits) {
		return true
	}
	if queryDigits != "" && item.CardNumber != nil && strings.Contains(digitsOnly(*item.CardNumber), queryDigits) {
		return true
	}

	for _, value := range []string{
		item.Phone,
		item.BankName,
		valueOrEmpty(item.Proxy),
		valueOrEmpty(item.EmployeeComment),
		valueOrEmpty(item.HolderName),
		valueOrEmpty(item.CardNumber),
	} {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}

	return false
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if unicode.IsDigit(char) {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizePhone(value string) (string, bool) {
	digits := make([]rune, 0, 11)
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits = append(digits, char)
		}
	}
	if len(digits) == 10 {
		digits = append([]rune{'7'}, digits...)
	}
	if len(digits) == 11 && digits[0] == '8' {
		digits[0] = '7'
	}
	if len(digits) != 11 || digits[0] != '7' {
		return "", false
	}
	return string(digits), true
}

func normalizeCardNumber(value string) (string, bool) {
	digits := digitsOnly(value)
	if len(digits) < 8 {
		return "", false
	}
	return digits, true
}
