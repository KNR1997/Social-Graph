package entity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository is the persistence port for users, declared here by the
// consumer rather than exported by the adapter that satisfies it.
type UserRepository interface {
	Add(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	ByID(ctx context.Context, id uuid.UUID) (*User, error)
	ByEmail(ctx context.Context, email Email) (*User, error)
}

// SessionRepository is the persistence port for sessions.
//
// Lookup is by token hash rather than by id: the client presents a token, and
// the service must never have to search for a session by anything the client
// could enumerate.
type SessionRepository interface {
	Add(ctx context.Context, s *Session) error
	ByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
