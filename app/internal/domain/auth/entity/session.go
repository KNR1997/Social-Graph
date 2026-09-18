package entity

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session errors.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrInvalidTTL      = errors.New("session lifetime must be positive")
	ErrEmptyTokenHash  = errors.New("session token hash must not be empty")
)

// Session is a user's proof of having signed in. It is an aggregate in its own
// right rather than a field on User, because it has its own lifecycle: one user
// has many sessions, and revoking one must not touch the others.
//
// Invariants:
//   - tokenHash is non-empty, and is a digest -- never the token itself
//   - expiresAt is strictly after issuedAt
//   - an expired session authenticates nobody
type Session struct {
	id        uuid.UUID
	userID    uuid.UUID
	tokenHash string
	issuedAt  time.Time
	expiresAt time.Time
}

// IssueSession creates a session that expires ttl after now.
//
// It takes the token's *hash*, not the token. The plaintext token exists only
// in the response that sets the cookie and in the cookie itself; the database
// holds a digest, so a leaked table dump cannot be replayed as a login. That is
// the same reason the password is stored as a hash, applied to the credential
// the client actually presents on every request.
func IssueSession(
	id, userID uuid.UUID,
	tokenHash string,
	ttl time.Duration,
	now time.Time,
) (*Session, error) {
	if tokenHash == "" {
		return nil, ErrEmptyTokenHash
	}

	if ttl <= 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTTL, ttl)
	}

	issued := now.UTC()

	return &Session{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		issuedAt:  issued,
		expiresAt: issued.Add(ttl),
	}, nil
}

// Credential is a session together with the plaintext token that addresses it.
//
// It exists so that issuing a session has one return value instead of two
// loosely related ones, and it lives in the domain so the layers above can name
// the result of signing in without importing the application package.
//
// The Token field is the only place the plaintext ever appears. It travels from
// the use case to the handler that writes the cookie and is then dropped; what
// persists is Session.TokenHash.
type Credential struct {
	Session *Session
	Token   string
}

// ReconstituteSession rebuilds a session from storage.
func ReconstituteSession(
	id, userID uuid.UUID,
	tokenHash string,
	issuedAt, expiresAt time.Time,
) *Session {
	return &Session{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		issuedAt:  issuedAt.UTC(),
		expiresAt: expiresAt.UTC(),
	}
}

// IsExpired reports whether the session is no longer valid at now.
//
// Expiry is checked here as well as in the query that loads the session. The
// query keeps expired rows from being fetched at all; this keeps a session that
// expired between the read and the check from authenticating anyone.
func (s *Session) IsExpired(now time.Time) bool {
	return !now.UTC().Before(s.expiresAt)
}

// ID returns the identifier.
func (s *Session) ID() uuid.UUID { return s.id }

// UserID returns the owning user.
func (s *Session) UserID() uuid.UUID { return s.userID }

// TokenHash returns the stored digest of the session token.
func (s *Session) TokenHash() string { return s.tokenHash }

// IssuedAt returns when the session started.
func (s *Session) IssuedAt() time.Time { return s.issuedAt }

// ExpiresAt returns when the session stops being valid.
func (s *Session) ExpiresAt() time.Time { return s.expiresAt }
