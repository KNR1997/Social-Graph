package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/kethaka-creskit/go-ddd-service/internal/application"
)

// tokenBytes is the entropy in a session token. 256 bits is far past the point
// where guessing is the weak link, and it costs 43 characters in a cookie.
const tokenBytes = 32

// TokenGenerator implements application.TokenGenerator.
type TokenGenerator struct{}

var _ application.TokenGenerator = (*TokenGenerator)(nil)

// NewTokenGenerator builds the token generator.
func NewTokenGenerator() TokenGenerator { return TokenGenerator{} }

// New returns a fresh URL-safe token drawn from the system CSPRNG.
func (TokenGenerator) New() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("security - New - rand.Read: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Hash reduces a token to the digest stored in the sessions table.
//
// A plain unsalted SHA-256 is the right tool here, and a password KDF would be
// the wrong one. The input is 256 bits of uniform randomness, so there is no
// dictionary to attack and nothing for a slow hash to buy; meanwhile the digest
// is a lookup key, which requires that the same token always hash to the same
// value. What it does buy is that a stolen database dump contains no usable
// session tokens.
func (TokenGenerator) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}
