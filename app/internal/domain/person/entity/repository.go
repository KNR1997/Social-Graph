package entity

import (
	"context"

	"github.com/google/uuid"
)

type Page struct {
	Limit  int32
	Offset int32
}

type Filter struct {
	Page   Page
	Search string
}

type Repository interface {
	Add(ctx context.Context, person *Person) error
	Update(ctx context.Context, person *Person) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
	ByID(ctx context.Context, userID, id uuid.UUID) (*Person, error)
	// BySelf loads the owner's own node in the graph, created at registration.
	BySelf(ctx context.Context, userID uuid.UUID) (*Person, error)
	List(ctx context.Context, userID uuid.UUID, filter Filter) ([]*Person, error)
	Count(ctx context.Context, userID uuid.UUID, filter Filter) (int64, error)
}
