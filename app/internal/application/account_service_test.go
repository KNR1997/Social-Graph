package application_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/application"
	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
)

// --- test doubles -----------------------------------------------------------
//
// Hand-written, because there are three tiny ports. Reach for mockgen when the
// interfaces grow, not before.

type fakeRepo struct {
	accounts map[uuid.UUID]*entity.Account
	updates  int
	failNext error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{accounts: map[uuid.UUID]*entity.Account{}}
}

func (r *fakeRepo) Add(_ context.Context, a *entity.Account) error {
	if err := r.take(); err != nil {
		return err
	}
	r.accounts[a.ID()] = a
	return nil
}

func (r *fakeRepo) Update(_ context.Context, a *entity.Account) error {
	if err := r.take(); err != nil {
		return err
	}
	if _, ok := r.accounts[a.ID()]; !ok {
		return entity.ErrNotFound
	}
	r.accounts[a.ID()] = a
	r.updates++
	return nil
}

func (r *fakeRepo) ByID(_ context.Context, id uuid.UUID) (*entity.Account, error) {
	if err := r.take(); err != nil {
		return nil, err
	}
	a, ok := r.accounts[id]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return a, nil
}

func (r *fakeRepo) List(_ context.Context, _ entity.Page) ([]*entity.Account, error) {
	if err := r.take(); err != nil {
		return nil, err
	}
	out := make([]*entity.Account, 0, len(r.accounts))
	for _, a := range r.accounts {
		out = append(out, a)
	}
	return out, nil
}

func (r *fakeRepo) take() error {
	err := r.failNext
	r.failNext = nil
	return err
}

// inlineTx runs the callback directly and records whether it was rolled back,
// which is what lets a unit test assert the transaction boundary exists.
type inlineTx struct {
	calls    int
	rolledBk int
}

func (t *inlineTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	t.calls++
	if err := fn(ctx); err != nil {
		t.rolledBk++
		return err
	}
	return nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// seqIDs hands out predictable ids so assertions can name them.
type seqIDs struct {
	n int
}

func (g *seqIDs) NewID() uuid.UUID {
	g.n++
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", g.n))
}

// --- fixture ----------------------------------------------------------------

type fixture struct {
	svc  *application.AccountService
	repo *fakeRepo
	tx   *inlineTx
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	repo := newFakeRepo()
	tx := &inlineTx{}
	clock := fixedClock{t: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &fixture{
		svc:  application.NewAccountService(repo, tx, clock, &seqIDs{}, log),
		repo: repo,
		tx:   tx,
	}
}

// --- tests ------------------------------------------------------------------

func TestOpenAccount(t *testing.T) {
	t.Parallel()

	t.Run("persists a new account", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		acc, err := f.svc.OpenAccount(context.Background(), "Ada", "USD")
		require.NoError(t, err)

		assert.Equal(t, "Ada", acc.Owner())
		assert.Len(t, f.repo.accounts, 1)
	})

	t.Run("surfaces a domain validation failure without touching the repository", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		_, err := f.svc.OpenAccount(context.Background(), "", "USD")
		require.ErrorIs(t, err, entity.ErrEmptyOwner)
		assert.Empty(t, f.repo.accounts)
	})
}

func TestDeposit(t *testing.T) {
	t.Parallel()

	t.Run("credits inside a transaction", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		opened, err := f.svc.OpenAccount(context.Background(), "Ada", "USD")
		require.NoError(t, err)

		acc, err := f.svc.Deposit(context.Background(), opened.ID(), 5000, "USD")
		require.NoError(t, err)

		assert.Equal(t, int64(5000), acc.Balance().Minor())
		assert.Equal(t, 1, f.tx.calls, "the balance change must run in a transaction")
		assert.Equal(t, 0, f.tx.rolledBk)
	})

	t.Run("rolls back when the aggregate rejects the change", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		opened, err := f.svc.OpenAccount(context.Background(), "Ada", "USD")
		require.NoError(t, err)

		_, err = f.svc.Deposit(context.Background(), opened.ID(), -100, "USD")
		require.ErrorIs(t, err, entity.ErrNotPositive)
		assert.Equal(t, 0, f.repo.updates, "nothing may be written when the domain refuses")
	})

	t.Run("returns not found for an unknown account", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		_, err := f.svc.Deposit(context.Background(), uuid.New(), 100, "USD")
		require.ErrorIs(t, err, entity.ErrNotFound)
		assert.Equal(t, 1, f.tx.rolledBk, "a failed use case must roll back")
	})
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	t.Run("refuses to overdraw and rolls back", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		opened, err := f.svc.OpenAccount(context.Background(), "Ada", "USD")
		require.NoError(t, err)
		_, err = f.svc.Deposit(context.Background(), opened.ID(), 1000, "USD")
		require.NoError(t, err)

		_, err = f.svc.Withdraw(context.Background(), opened.ID(), 1001, "USD")
		require.ErrorIs(t, err, entity.ErrInsufficientFunds)

		reloaded, err := f.svc.GetAccount(context.Background(), opened.ID())
		require.NoError(t, err)
		assert.Equal(t, int64(1000), reloaded.Balance().Minor())
	})
}

func TestCloseAccount(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	opened, err := f.svc.OpenAccount(context.Background(), "Ada", "USD")
	require.NoError(t, err)

	_, err = f.svc.Deposit(context.Background(), opened.ID(), 500, "USD")
	require.NoError(t, err)

	_, err = f.svc.CloseAccount(context.Background(), opened.ID())
	require.ErrorIs(t, err, entity.ErrBalanceNotZero)

	_, err = f.svc.Withdraw(context.Background(), opened.ID(), 500, "USD")
	require.NoError(t, err)

	closed, err := f.svc.CloseAccount(context.Background(), opened.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.StatusClosed, closed.Status())
}
