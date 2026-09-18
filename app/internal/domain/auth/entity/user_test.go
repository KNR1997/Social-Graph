package entity_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
)

var (
	testID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testNow = time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
)

const testHash = "$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2E$ZGlnZXN0ZGlnZXN0ZGlnZXN0ZGln"

func TestParseEmail(t *testing.T) {
	t.Parallel()

	t.Run("normalises case and surrounding space", func(t *testing.T) {
		t.Parallel()

		got, err := entity.ParseEmail("  Ada@Example.COM ")
		require.NoError(t, err)
		assert.Equal(t, entity.Email("ada@example.com"), got,
			"the address is the identity, so two spellings must collapse to one")
	})

	t.Run("rejects malformed addresses", func(t *testing.T) {
		t.Parallel()

		cases := map[string]string{
			"empty":          "",
			"no at":          "ada.example.com",
			"no domain":      "ada@",
			"no local part":  "@example.com",
			"embedded space": "ada lovelace@example.com",
			"display name":   "Ada <ada@example.com>",
			"two addresses":  "ada@example.com, bob@example.com",
			"too long":       strings.Repeat("a", 250) + "@example.com",
		}

		for name, input := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := entity.ParseEmail(input)
				assert.ErrorIs(t, err, entity.ErrInvalidEmail)
			})
		}
	})
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	t.Run("accepts a password at the boundaries", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, entity.ValidatePassword(strings.Repeat("a", entity.MinPasswordLen)))
		assert.NoError(t, entity.ValidatePassword(strings.Repeat("a", entity.MaxPasswordLen)))
	})

	t.Run("rejects one that is too short or too long", func(t *testing.T) {
		t.Parallel()

		short := strings.Repeat("a", entity.MinPasswordLen-1)
		long := strings.Repeat("a", entity.MaxPasswordLen+1)

		require.ErrorIs(t, entity.ValidatePassword(short), entity.ErrWeakPassword)
		require.ErrorIs(t, entity.ValidatePassword(long), entity.ErrWeakPassword)
	})

	t.Run("counts characters rather than bytes", func(t *testing.T) {
		t.Parallel()

		// Eight characters that weigh considerably more than eight bytes. A
		// byte-length check would wrongly accept a shorter passphrase than the
		// policy allows, and wrongly reject a longer one.
		require.NoError(t, entity.ValidatePassword(strings.Repeat("é", entity.MinPasswordLen)))
		require.ErrorIs(t,
			entity.ValidatePassword(strings.Repeat("é", entity.MinPasswordLen-1)),
			entity.ErrWeakPassword)
	})
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("creates a member at version one", func(t *testing.T) {
		t.Parallel()

		user, err := entity.Register(testID, " Ada@Example.com ", "  Ada Lovelace  ",
			testHash, entity.RoleMember, testNow)
		require.NoError(t, err)

		assert.Equal(t, testID, user.ID())
		assert.Equal(t, entity.Email("ada@example.com"), user.Email())
		assert.Equal(t, "Ada Lovelace", user.Name(), "the name is trimmed")
		assert.Equal(t, entity.RoleMember, user.Role())
		assert.Equal(t, int64(1), user.Version())
		assert.Equal(t, testNow, user.CreatedAt())
		assert.Equal(t, testNow, user.UpdatedAt())
	})

	t.Run("refuses an empty password hash", func(t *testing.T) {
		t.Parallel()

		// The aggregate cannot tell a plaintext password from a digest, but it
		// can refuse to hold nothing at all, which is the failure mode a broken
		// hasher would otherwise produce silently.
		_, err := entity.Register(testID, "ada@example.com", "Ada", "", entity.RoleMember, testNow)
		assert.ErrorIs(t, err, entity.ErrWeakPassword)
	})

	t.Run("refuses a blank name", func(t *testing.T) {
		t.Parallel()

		_, err := entity.Register(testID, "ada@example.com", "   ", testHash, entity.RoleMember, testNow)
		assert.ErrorIs(t, err, entity.ErrEmptyName)
	})

	t.Run("refuses an unknown role", func(t *testing.T) {
		t.Parallel()

		_, err := entity.Register(testID, "ada@example.com", "Ada", testHash, entity.Role("root"), testNow)
		assert.ErrorIs(t, err, entity.ErrInvalidRole)
	})

	t.Run("refuses an invalid address", func(t *testing.T) {
		t.Parallel()

		_, err := entity.Register(testID, "not-an-address", "Ada", testHash, entity.RoleMember, testNow)
		assert.ErrorIs(t, err, entity.ErrInvalidEmail)
	})
}

func TestRename(t *testing.T) {
	t.Parallel()

	user := newTestUser(t)
	later := testNow.Add(time.Hour)

	require.NoError(t, user.Rename("  Ada King  ", later))
	assert.Equal(t, "Ada King", user.Name())
	assert.Equal(t, later, user.UpdatedAt())

	require.ErrorIs(t, user.Rename("", later), entity.ErrEmptyName)
	assert.Equal(t, "Ada King", user.Name(), "a rejected rename leaves the name alone")
}

func TestChangePassword(t *testing.T) {
	t.Parallel()

	user := newTestUser(t)
	later := testNow.Add(time.Hour)
	next := testHash + "x"

	require.NoError(t, user.ChangePassword(next, later))
	assert.Equal(t, next, user.PasswordHash())
	assert.Equal(t, later, user.UpdatedAt())

	require.ErrorIs(t, user.ChangePassword("", later), entity.ErrWeakPassword)
	assert.Equal(t, next, user.PasswordHash(), "a rejected change leaves the digest alone")
}

func TestParseRole(t *testing.T) {
	t.Parallel()

	for _, role := range []entity.Role{entity.RoleMember, entity.RoleAdmin} {
		got, err := entity.ParseRole(string(role))
		require.NoError(t, err)
		assert.Equal(t, role, got)
	}

	_, err := entity.ParseRole("superuser")
	assert.ErrorIs(t, err, entity.ErrInvalidRole)
}

func newTestUser(t *testing.T) *entity.User {
	t.Helper()

	user, err := entity.Register(testID, "ada@example.com", "Ada Lovelace",
		testHash, entity.RoleMember, testNow)
	require.NoError(t, err)

	return user
}
