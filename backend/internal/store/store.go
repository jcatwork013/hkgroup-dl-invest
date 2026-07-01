package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hkgroup/backend/internal/db"
)

// Store is the data-access seam. It embeds the sqlc Queries (pool-bound) and exposes ExecTx so
// usecases can run several queries atomically — the mechanism that makes invariants 3 and 8 hold.
type Store struct {
	*db.Queries
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{Queries: db.New(pool), pool: pool}
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// ExecTx runs fn inside a single transaction. If fn returns an error the whole tx rolls back, so
// share issuance (ledger + shareholding + offering + audit) is all-or-nothing.
func (s *Store) ExecTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit

	if err := fn(s.Queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Connect opens a pgx pool and verifies connectivity.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Pgx error helpers re-exported for usecases that branch on no-rows.
var ErrNoRows = pgx.ErrNoRows
