package banks

import "context"

type Store interface {
	ListActive(ctx context.Context) ([]Bank, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListActive(ctx context.Context) ([]Bank, error) {
	return s.store.ListActive(ctx)
}
