package orders

import (
	"context"
	"testing"
)

func TestServiceListTraderOrdersReturnsEmptyWhenNoCurrentShift(t *testing.T) {
	service := NewService(&fakeStore{traderListErr: ErrNoCurrentShift})

	result, err := service.ListTraderOrders(context.Background(), 2, 3, DirectionInbound, Filters{})
	if err != nil {
		t.Fatalf("ListTraderOrders() error = %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("result = %+v, want empty list", result)
	}
	if result.Page != DefaultPage || result.PageSize != DefaultPageSize {
		t.Fatalf("pagination = %d/%d, want defaults", result.Page, result.PageSize)
	}
}

func TestServiceNormalizesPaginationAndSort(t *testing.T) {
	store := &fakeStore{
		teamleadList: ListResult{Total: 1},
	}
	service := NewService(store)

	_, err := service.ListTeamleadOrders(context.Background(), 2, DirectionInbound, Filters{
		Page:     -1,
		PageSize: 1000,
		Sort:     "bad",
	})
	if err != nil {
		t.Fatalf("ListTeamleadOrders() error = %v", err)
	}

	if store.filters.Page != DefaultPage || store.filters.PageSize != MaxPageSize || store.filters.Sort != SortCreatedAtDesc {
		t.Fatalf("filters = %+v, want normalized page/default max page size/default sort", store.filters)
	}
}

func TestServiceRejectsInvalidDirection(t *testing.T) {
	service := NewService(&fakeStore{})

	_, err := service.ListTeamleadOrders(context.Background(), 2, "sideways", Filters{})
	if err != ErrInvalidInput {
		t.Fatalf("ListTeamleadOrders() error = %v, want ErrInvalidInput", err)
	}
}

type fakeStore struct {
	filters       Filters
	traderListErr error
	teamleadList  ListResult
}

func (s *fakeStore) ListTraderOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	s.filters = filters
	if s.traderListErr != nil {
		return ListResult{}, s.traderListErr
	}
	return ListResult{}, nil
}

func (s *fakeStore) ListTeamleadOrders(ctx context.Context, scope Scope, filters Filters) (ListResult, error) {
	s.filters = filters
	return s.teamleadList, nil
}

func (s *fakeStore) TraderDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	return Dashboard{}, nil
}

func (s *fakeStore) TeamleadDashboard(ctx context.Context, scope Scope, filters Filters) (Dashboard, error) {
	return Dashboard{}, nil
}
