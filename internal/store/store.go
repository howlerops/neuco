// Package store provides database access methods for all domain entities.
// All queries use parameterised placeholders ($1, $2, …) and are scoped by
// a tenant identifier (org_id or project_id) to enforce isolation between
// customers. The Store type wraps a pgxpool.Pool and exposes all persistence
// operations used by the application layer.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the central persistence layer. All query methods are defined as
// pointer receiver methods on Store so they share the connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store backed by the given connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Pool exposes the underlying connection pool for callers that need direct
// pool access (e.g. River queue integration).
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// PageParams carries pagination parameters for list queries.
type PageParams struct {
	Limit  int
	Offset int
}

// Page is a convenience constructor that clamps limit to sensible boundaries.
func Page(limit, offset int) PageParams {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return PageParams{Limit: limit, Offset: offset}
}

// withTx executes fn inside a database transaction. If fn returns an error the
// transaction is rolled back; otherwise it is committed.
func (s *Store) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("store: begin transaction: %w", err)
	}
	defer func() {
		// Best-effort rollback; ignored if tx was already committed or
		// the connection is gone.
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit transaction: %w", err)
	}
	return nil
}
