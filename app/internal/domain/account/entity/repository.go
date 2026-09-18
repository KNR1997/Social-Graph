package entity

import (
	"context"

	"github.com/google/uuid"
)

// Page is a simple offset window for list queries.
type Page struct {
	Limit  int32
	Offset int32
}

// Repository is the persistence port for the account aggregate.
//
// It is declared here, in the domain, because the domain is what needs it. The
// implementation lives in internal/infrastructure/persistence, so the domain
// never learns that Postgres exists.
//
// Add and Update are separate rather than a single Save so that optimistic
// locking is explicit: Update must return ErrConflict when the stored version
// no longer matches the loaded one.
type Repository interface {
	Add(ctx context.Context, a *Account) error
	Update(ctx context.Context, a *Account) error
	ByID(ctx context.Context, id uuid.UUID) (*Account, error)
	List(ctx context.Context, page Page) ([]*Account, error)
}
