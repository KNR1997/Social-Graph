// Package persistence adapts the domain's ports to Postgres. It is the only
// package in the module that knows sqlc and pgx exist.
package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence/sqlcgen"
)

// txKey is an unexported context key, so nothing outside this package can put a
// transaction on a context or take one off it.
type txKey struct{}

// TxManager implements application.TxManager on a pgx pool.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager builds the transaction manager.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithinTx runs fn inside a transaction, committing on success and rolling back
// on any error or panic. If ctx already carries a transaction, fn joins it
// rather than opening a nested one, so use cases compose safely.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("persistence - WithinTx - pool.BeginTx: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		// Rollback on a cancelled ctx would fail, so use a detached context.
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil &&
			!errors.Is(rbErr, pgx.ErrTxClosed) {
			// Nothing useful to do here: the caller is already returning the
			// real error, and the connection is discarded by the pool.
			_ = rbErr
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("persistence - WithinTx - tx.Commit: %w", err)
	}
	committed = true

	return nil
}

// queries returns a sqlc querier bound to the ambient transaction if there is
// one, and to the pool otherwise. Every repository method goes through this,
// which is what makes the transaction boundary invisible to the domain.
func queries(ctx context.Context, pool *pgxpool.Pool) *sqlcgen.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return sqlcgen.New(tx)
	}

	return sqlcgen.New(pool)
}
