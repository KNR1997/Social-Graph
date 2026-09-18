package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
)

func TestParseCurrency(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in      string
		want    entity.Currency
		wantErr error
	}{
		"uppercase":      {in: "USD", want: "USD"},
		"normalised":     {in: " eur ", want: "EUR"},
		"too short":      {in: "US", wantErr: entity.ErrInvalidCurrency},
		"too long":       {in: "USDX", wantErr: entity.ErrInvalidCurrency},
		"non alphabetic": {in: "U5D", wantErr: entity.ErrInvalidCurrency},
		"empty":          {in: "", wantErr: entity.ErrInvalidCurrency},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := entity.ParseCurrency(tc.in)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMoneyArithmeticRequiresMatchingCurrency(t *testing.T) {
	t.Parallel()

	usd, err := entity.NewMoney(1000, "USD")
	require.NoError(t, err)
	eur, err := entity.NewMoney(1000, "EUR")
	require.NoError(t, err)

	_, err = usd.Add(eur)
	require.ErrorIs(t, err, entity.ErrCurrencyMismatch)

	_, err = usd.Sub(eur)
	require.ErrorIs(t, err, entity.ErrCurrencyMismatch)
}

func TestMoneyAddSub(t *testing.T) {
	t.Parallel()

	a, err := entity.NewMoney(1000, "USD")
	require.NoError(t, err)
	b, err := entity.NewMoney(250, "USD")
	require.NoError(t, err)

	sum, err := a.Add(b)
	require.NoError(t, err)
	assert.Equal(t, int64(1250), sum.Minor())

	diff, err := a.Sub(b)
	require.NoError(t, err)
	assert.Equal(t, int64(750), diff.Minor())

	// Sub is allowed to go negative; the aggregate is what forbids it.
	over, err := b.Sub(a)
	require.NoError(t, err)
	assert.True(t, over.IsNegative())
}

func TestMoneyIsImmutable(t *testing.T) {
	t.Parallel()

	original, err := entity.NewMoney(1000, "USD")
	require.NoError(t, err)

	added, err := entity.NewMoney(500, "USD")
	require.NoError(t, err)

	_, err = original.Add(added)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), original.Minor(), "Add must not mutate the receiver")
}
