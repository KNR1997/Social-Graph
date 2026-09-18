package security_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/application"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/security"
)

func TestHashAndVerify(t *testing.T) {
	t.Parallel()

	hasher := security.NewHasher()

	hash, err := hasher.Hash("correct-horse-battery")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(hash, "$argon2id$"),
		"the digest carries its own parameters, so it must be in PHC format")
	assert.NotContains(t, hash, "correct-horse-battery")

	require.NoError(t, hasher.Verify(hash, "correct-horse-battery"))
	assert.ErrorIs(t, hasher.Verify(hash, "wrong-password"), application.ErrPasswordMismatch)
}

func TestHashIsSalted(t *testing.T) {
	t.Parallel()

	hasher := security.NewHasher()

	first, err := hasher.Hash("same-password-twice")
	require.NoError(t, err)

	second, err := hasher.Hash("same-password-twice")
	require.NoError(t, err)

	// Equal digests for equal passwords would let anyone with the table see
	// which users share a password, and would make a rainbow table worth
	// building.
	assert.NotEqual(t, first, second)
	require.NoError(t, hasher.Verify(first, "same-password-twice"))
	require.NoError(t, hasher.Verify(second, "same-password-twice"))
}

func TestVerifyRejectsAMalformedDigest(t *testing.T) {
	t.Parallel()

	hasher := security.NewHasher()

	valid, err := hasher.Hash("correct-horse-battery")
	require.NoError(t, err)

	cases := map[string]string{
		"empty":            "",
		"not phc":          "plaintext-password",
		"wrong algorithm":  strings.Replace(valid, "argon2id", "argon2i", 1),
		"truncated":        valid[:len(valid)/2],
		"bad base64 salt":  strings.Replace(valid, "$m=", "$!!$m=", 1),
		"missing segments": "$argon2id$v=19$m=65536,t=3,p=2",
	}

	for name, digest := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := hasher.Verify(digest, "correct-horse-battery")

			// A corrupt row must be a real error, never a plain mismatch: if it
			// degraded to "wrong password" every user of that row would be locked
			// out with no signal that anything was broken.
			require.Error(t, err)
			assert.NotErrorIs(t, err, application.ErrPasswordMismatch)
		})
	}
}

func TestTokensAreUniqueAndHashedStably(t *testing.T) {
	t.Parallel()

	gen := security.NewTokenGenerator()

	first, err := gen.New()
	require.NoError(t, err)

	second, err := gen.New()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.NotEmpty(t, first)

	// Hashing is the lookup key, so it has to be deterministic -- the opposite
	// requirement to the password hasher above.
	digest := gen.Hash(first)
	assert.Equal(t, digest, gen.Hash(first), "the same token must always produce the same key")
	assert.NotEqual(t, gen.Hash(first), gen.Hash(second))

	// A 64-character hex digest, which is what the sessions table's CHECK
	// constraint expects.
	assert.Len(t, gen.Hash(first), 64)
	assert.NotContains(t, gen.Hash(first), first)
}
