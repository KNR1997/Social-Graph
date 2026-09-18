// Package application holds application services: the use-case layer. It
// orchestrates (load, mutate, store) and owns technical concerns like the
// transaction boundary. Business rules belong in the domain, not here.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// TxManager is the transaction boundary port. The implementation puts the
// pgx.Tx on the returned context; repositories pick it up from there, so no
// method signature in the domain has to mention transactions.
//
// Implementations must be reentrant: calling WithinTx inside an existing
// transaction joins it rather than opening a nested one.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Clock is injected so use cases are deterministic under test. Nothing in this
// codebase calls time.Now() outside the system implementation of this port.
type Clock interface {
	Now() time.Time
}

// IDGenerator is injected for the same reason.
type IDGenerator interface {
	NewID() uuid.UUID
}

// ErrPasswordMismatch is what a PasswordHasher returns when a password does not
// match a digest. It belongs to the port, not to the domain: "this argon2
// digest does not verify" is a statement about cryptography. AuthService
// translates it into entity.ErrInvalidCredentials before anyone else sees it.
var ErrPasswordMismatch = errors.New("password does not match the stored hash")

// PasswordHasher is the password-hashing port. The domain deliberately holds no
// cryptography, so the policy (how long a password must be) lives in the
// entity package and the mechanism (which KDF, at what cost) lives behind this
// interface in infrastructure.
//
// Verify takes the digest first and the candidate second, mirroring Hash's
// direction of travel, and reports ErrPasswordMismatch rather than a bool so a
// real failure (a corrupt digest, an unknown algorithm) cannot be mistaken for
// a wrong password.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(hash, plain string) error
}

// TokenGenerator mints and digests session tokens.
//
// New returns a fresh, unguessable, URL-safe token; Hash reduces one to the
// digest stored in the sessions table. Hash must be deterministic and must not
// be salted -- unlike a password, the token is looked up by its digest, so the
// same token has to produce the same key every time.
type TokenGenerator interface {
	New() (string, error)
	Hash(token string) string
}

// SessionTTL is how long a new session stays valid. It is a named type so Wire
// can provide it from configuration without the application layer reading the
// environment, and so a bare time.Duration in the graph is unambiguous.
type SessionTTL time.Duration
