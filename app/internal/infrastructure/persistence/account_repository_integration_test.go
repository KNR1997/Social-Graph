//go:build integration

package persistence_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence"
	"github.com/kethaka-creskit/go-ddd-service/internal/test/containers"
)

var (
	dsn     string
	dsnOnce sync.Once
)

// harness gives each test a live repository against a migrated database.
type harness struct {
	pool *pgxpool.Pool
	repo *persistence.AccountRepository
	tx   *persistence.TxManager
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	dsnOnce.Do(func() { dsn = containers.StartPostgres(t) })
	containers.Truncate(t, dsn, "accounts")

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return &harness{
		pool: pool,
		repo: persistence.NewAccountRepository(pool),
		tx:   persistence.NewTxManager(pool),
	}
}

func openAccount(t *testing.T, owner string) *entity.Account {
	t.Helper()

	a, err := entity.Open(uuid.New(), owner, "USD", time.Now())
	require.NoError(t, err)

	return a
}

func TestAddAndLoad(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada Lovelace")
	require.NoError(t, h.repo.Add(ctx, acc))

	loaded, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)

	// The round trip must preserve every field the aggregate cares about,
	// including the currency, which lives in a separate column.
	assert.Equal(t, acc.ID(), loaded.ID())
	assert.Equal(t, "Ada Lovelace", loaded.Owner())
	assert.Equal(t, entity.StatusActive, loaded.Status())
	assert.Equal(t, acc.Balance().Minor(), loaded.Balance().Minor())
	assert.Equal(t, acc.Balance().Currency(), loaded.Balance().Currency())
	assert.Equal(t, int64(1), loaded.Version())
	assert.WithinDuration(t, acc.CreatedAt(), loaded.CreatedAt(), time.Millisecond)
}

func TestByIDReturnsNotFound(t *testing.T) {
	h := newHarness(t)

	_, err := h.repo.ByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestAddRejectsDuplicateID(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")
	require.NoError(t, h.repo.Add(ctx, acc))
	require.ErrorIs(t, h.repo.Add(ctx, acc), entity.ErrConflict)
}

func TestUpdateBumpsVersion(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")
	require.NoError(t, h.repo.Add(ctx, acc))

	require.NoError(t, acc.Deposit(mustMoney(t, 5000), time.Now()))
	require.NoError(t, h.repo.Update(ctx, acc))

	loaded, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)
	assert.Equal(t, int64(5000), loaded.Balance().Minor())
	assert.Equal(t, int64(2), loaded.Version(), "a successful update must advance the version")
}

func TestUpdateDetectsStaleWrite(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")
	require.NoError(t, h.repo.Add(ctx, acc))

	// Two readers load the same version.
	first, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)
	second, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)

	require.NoError(t, first.Deposit(mustMoney(t, 100), time.Now()))
	require.NoError(t, h.repo.Update(ctx, first))

	// The second writer is now working from a stale version and must be refused
	// rather than silently overwriting the first deposit.
	require.NoError(t, second.Deposit(mustMoney(t, 100), time.Now()))
	require.ErrorIs(t, h.repo.Update(ctx, second), entity.ErrConflict)

	loaded, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)
	assert.Equal(t, int64(100), loaded.Balance().Minor(), "the lost update must not have landed")
}

func TestWithinTxRollsBack(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")

	sentinel := entity.ErrNotActive
	err := h.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := h.repo.Add(ctx, acc); err != nil {
			return err
		}

		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	// The insert happened inside the transaction, so it must be gone.
	_, err = h.repo.ByID(ctx, acc.ID())
	require.ErrorIs(t, err, entity.ErrNotFound)
}

func TestWithinTxCommits(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")

	require.NoError(t, h.tx.WithinTx(ctx, func(ctx context.Context) error {
		return h.repo.Add(ctx, acc)
	}))

	_, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)
}

func TestWithinTxIsReentrant(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")

	// A nested WithinTx must join the outer transaction, not open a second one,
	// or a use case calling another use case would deadlock or half-commit.
	require.NoError(t, h.tx.WithinTx(ctx, func(outer context.Context) error {
		return h.tx.WithinTx(outer, func(inner context.Context) error {
			return h.repo.Add(inner, acc)
		})
	}))

	_, err := h.repo.ByID(ctx, acc.ID())
	require.NoError(t, err)
}

func TestListIsNewestFirst(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	base := time.Now().UTC().Truncate(time.Millisecond)
	for i := range 3 {
		acc, err := entity.Open(uuid.New(), "owner", "USD", base.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
		require.NoError(t, h.repo.Add(ctx, acc))
	}

	page, err := h.repo.List(ctx, entity.Page{Limit: 2, Offset: 0})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.True(t, page[0].CreatedAt().After(page[1].CreatedAt()), "list must be newest first")

	second, err := h.repo.List(ctx, entity.Page{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, second, 1, "the offset must page through the result set")
}

func TestDatabaseRejectsNegativeBalance(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	acc := openAccount(t, "Ada")
	require.NoError(t, h.repo.Add(ctx, acc))

	// The aggregate makes this unreachable through the application, so go
	// straight to SQL: the CHECK constraint is the backstop for anything that
	// writes to the table without going through the domain.
	_, err := h.pool.Exec(ctx, "UPDATE accounts SET balance_minor = -1 WHERE id = $1", acc.ID())
	require.Error(t, err, "the accounts_balance_not_neg constraint must reject this")
}

func mustMoney(t *testing.T, minor int64) entity.Money {
	t.Helper()

	m, err := entity.NewMoney(minor, "USD")
	require.NoError(t, err)

	return m
}
