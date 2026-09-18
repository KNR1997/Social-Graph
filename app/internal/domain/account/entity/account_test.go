package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
)

var (
	testID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testNow = time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
)

func mustOpen(t *testing.T) *entity.Account {
	t.Helper()

	a, err := entity.Open(testID, "Ada Lovelace", "USD", testNow)
	require.NoError(t, err)

	return a
}

func mustMoney(t *testing.T, minor int64, currency string) entity.Money {
	t.Helper()

	m, err := entity.NewMoney(minor, currency)
	require.NoError(t, err)

	return m
}

func TestOpen(t *testing.T) {
	t.Parallel()

	t.Run("opens an active account with a zero balance", func(t *testing.T) {
		t.Parallel()

		a, err := entity.Open(testID, "Ada Lovelace", "usd", testNow)
		require.NoError(t, err)

		assert.Equal(t, testID, a.ID())
		assert.Equal(t, "Ada Lovelace", a.Owner())
		assert.Equal(t, entity.StatusActive, a.Status())
		assert.True(t, a.Balance().IsZero())
		assert.Equal(t, entity.Currency("USD"), a.Balance().Currency())
		assert.Equal(t, int64(1), a.Version())
	})

	t.Run("rejects an empty owner", func(t *testing.T) {
		t.Parallel()

		_, err := entity.Open(testID, "   ", "USD", testNow)
		require.ErrorIs(t, err, entity.ErrEmptyOwner)
	})

	t.Run("rejects an invalid currency", func(t *testing.T) {
		t.Parallel()

		_, err := entity.Open(testID, "Ada", "DOLLARS", testNow)
		require.ErrorIs(t, err, entity.ErrInvalidCurrency)
	})
}

func TestDeposit(t *testing.T) {
	t.Parallel()

	t.Run("credits the balance", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Deposit(mustMoney(t, 5000, "USD"), testNow))

		assert.Equal(t, int64(5000), a.Balance().Minor())
	})

	t.Run("rejects a zero or negative amount", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.ErrorIs(t, a.Deposit(mustMoney(t, 0, "USD"), testNow), entity.ErrNotPositive)
		require.ErrorIs(t, a.Deposit(mustMoney(t, -1, "USD"), testNow), entity.ErrNotPositive)
	})

	t.Run("rejects a foreign currency", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.ErrorIs(t, a.Deposit(mustMoney(t, 100, "EUR"), testNow), entity.ErrCurrencyMismatch)
	})

	t.Run("rejects a frozen account", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Freeze(testNow))
		require.ErrorIs(t, a.Deposit(mustMoney(t, 100, "USD"), testNow), entity.ErrNotActive)
	})
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	t.Run("debits the balance", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Deposit(mustMoney(t, 5000, "USD"), testNow))
		require.NoError(t, a.Withdraw(mustMoney(t, 2000, "USD"), testNow))

		assert.Equal(t, int64(3000), a.Balance().Minor())
	})

	t.Run("allows draining the balance to exactly zero", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Deposit(mustMoney(t, 5000, "USD"), testNow))
		require.NoError(t, a.Withdraw(mustMoney(t, 5000, "USD"), testNow))

		assert.True(t, a.Balance().IsZero())
	})

	t.Run("refuses to overdraw and leaves the balance untouched", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Deposit(mustMoney(t, 5000, "USD"), testNow))

		err := a.Withdraw(mustMoney(t, 5001, "USD"), testNow)
		require.ErrorIs(t, err, entity.ErrInsufficientFunds)
		assert.Equal(t, int64(5000), a.Balance().Minor(), "a rejected withdrawal must not mutate state")
	})
}

func TestLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("refuses to close an account holding funds", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Deposit(mustMoney(t, 1, "USD"), testNow))

		require.ErrorIs(t, a.Close(testNow), entity.ErrBalanceNotZero)
		assert.Equal(t, entity.StatusActive, a.Status())
	})

	t.Run("closes an empty account", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Close(testNow))
		assert.Equal(t, entity.StatusClosed, a.Status())
	})

	t.Run("freeze and close are idempotent", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Freeze(testNow))
		require.NoError(t, a.Freeze(testNow))
		assert.Equal(t, entity.StatusFrozen, a.Status())

		require.NoError(t, a.Unfreeze(testNow))
		require.NoError(t, a.Close(testNow))
		require.NoError(t, a.Close(testNow))
		assert.Equal(t, entity.StatusClosed, a.Status())
	})

	t.Run("a closed account cannot be reopened", func(t *testing.T) {
		t.Parallel()

		a := mustOpen(t)
		require.NoError(t, a.Close(testNow))
		require.ErrorIs(t, a.Unfreeze(testNow), entity.ErrNotActive)
		require.ErrorIs(t, a.Freeze(testNow), entity.ErrNotActive)
	})
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"active", "frozen", "closed"} {
		got, err := entity.ParseStatus(s)
		require.NoError(t, err)
		assert.Equal(t, entity.Status(s), got)
	}

	_, err := entity.ParseStatus("pending")
	require.ErrorIs(t, err, entity.ErrInvalidStatus)
}
