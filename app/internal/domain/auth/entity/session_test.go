package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
)

var testUserID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

const testTokenHash = "b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"

func TestIssueSession(t *testing.T) {
	t.Parallel()

	t.Run("expires one ttl after now", func(t *testing.T) {
		t.Parallel()

		session, err := entity.IssueSession(testID, testUserID, testTokenHash, time.Hour, testNow)
		require.NoError(t, err)

		assert.Equal(t, testUserID, session.UserID())
		assert.Equal(t, testTokenHash, session.TokenHash())
		assert.Equal(t, testNow, session.IssuedAt())
		assert.Equal(t, testNow.Add(time.Hour), session.ExpiresAt())
	})

	t.Run("refuses an empty token hash", func(t *testing.T) {
		t.Parallel()

		_, err := entity.IssueSession(testID, testUserID, "", time.Hour, testNow)
		assert.ErrorIs(t, err, entity.ErrEmptyTokenHash)
	})

	t.Run("refuses a non-positive lifetime", func(t *testing.T) {
		t.Parallel()

		// A zero or negative ttl would mint a session that is already expired,
		// which presents to a user as a login that silently does nothing.
		for _, ttl := range []time.Duration{0, -time.Second} {
			_, err := entity.IssueSession(testID, testUserID, testTokenHash, ttl, testNow)
			assert.ErrorIs(t, err, entity.ErrInvalidTTL)
		}
	})
}

func TestSessionIsExpired(t *testing.T) {
	t.Parallel()

	session, err := entity.IssueSession(testID, testUserID, testTokenHash, time.Hour, testNow)
	require.NoError(t, err)

	assert.False(t, session.IsExpired(testNow))
	assert.False(t, session.IsExpired(testNow.Add(59*time.Minute)))

	// Expiry is inclusive: a session is dead at the instant it expires, not one
	// tick afterwards.
	assert.True(t, session.IsExpired(testNow.Add(time.Hour)))
	assert.True(t, session.IsExpired(testNow.Add(2*time.Hour)))
}
