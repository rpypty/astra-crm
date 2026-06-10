package orders

import (
	"context"
	"errors"
)

type Store interface {
	ListTraderOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error)
	ListTeamleadOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error)
	TraderDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error)
	TeamleadDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListTraderOrders(ctx context.Context, teamID int64, traderID int64, direction string, filters Filters) (ListResult, error) {
	if teamID <= 0 || traderID <= 0 || !validDirection(direction) {
		return ListResult{}, ErrInvalidInput
	}

	result, err := s.store.ListTraderOrders(ctx, Scope{
		TeamID:    teamID,
		TraderID:  &traderID,
		Direction: direction,
	}, normalizeFilters(filters))
	if errors.Is(err, ErrNoCurrentShift) {
		return emptyList(filters), nil
	}
	return result, err
}

func (s *Service) ListTeamleadOrders(ctx context.Context, teamID int64, direction string, filters Filters) (ListResult, error) {
	if teamID <= 0 || !validDirection(direction) {
		return ListResult{}, ErrInvalidInput
	}

	return s.store.ListTeamleadOrders(ctx, Scope{
		TeamID:    teamID,
		Direction: direction,
	}, normalizeFilters(filters))
}

func (s *Service) TraderDashboard(ctx context.Context, teamID int64, traderID int64, direction string, filters Filters) (Dashboard, error) {
	if teamID <= 0 || traderID <= 0 || !validDirection(direction) {
		return Dashboard{}, ErrInvalidInput
	}

	dashboard, err := s.store.TraderDashboard(ctx, Scope{
		TeamID:    teamID,
		TraderID:  &traderID,
		Direction: direction,
	}, normalizeFilters(filters))
	if errors.Is(err, ErrNoCurrentShift) {
		return Dashboard{}, nil
	}
	return dashboard, err
}

func (s *Service) TeamleadDashboard(ctx context.Context, teamID int64, direction string, filters Filters) (Dashboard, error) {
	if teamID <= 0 || !validDirection(direction) {
		return Dashboard{}, ErrInvalidInput
	}

	return s.store.TeamleadDashboard(ctx, Scope{
		TeamID:    teamID,
		Direction: direction,
	}, normalizeFilters(filters))
}

func normalizeFilters(filters Filters) Filters {
	if filters.Page <= 0 {
		filters.Page = DefaultPage
	}
	if filters.PageSize <= 0 {
		filters.PageSize = DefaultPageSize
	}
	if filters.PageSize > MaxPageSize {
		filters.PageSize = MaxPageSize
	}
	if !validSort(filters.Sort) {
		filters.Sort = SortCreatedAtDesc
	}

	return filters
}

func validDirection(direction string) bool {
	return direction == DirectionInbound || direction == DirectionOutbound
}

func validSort(sort string) bool {
	switch sort {
	case "", SortCreatedAtDesc, SortCreatedAtAsc, SortAmountAsc, SortAmountDesc:
		return true
	default:
		return false
	}
}

func emptyList(filters Filters) ListResult {
	filters = normalizeFilters(filters)
	return ListResult{
		Items:    []Order{},
		Page:     filters.Page,
		PageSize: filters.PageSize,
		Total:    0,
	}
}
