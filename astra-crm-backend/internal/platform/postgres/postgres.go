package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	inner *pgxpool.Pool
}

func NewPool(ctx context.Context, databaseURL string) (*Pool, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("postgres: database URL is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Pool{inner: pool}, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.inner == nil {
		return errors.New("postgres: pool is not initialized")
	}

	return p.inner.Ping(ctx)
}

func (p *Pool) Close() {
	if p == nil || p.inner == nil {
		return
	}

	p.inner.Close()
}

func (p *Pool) Raw() *pgxpool.Pool {
	if p == nil {
		return nil
	}

	return p.inner
}
