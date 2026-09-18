package entity

import (
	"context"

	"github.com/google/uuid"
)

type Page struct {
	Limit  int32
	Offset int32
}

type Repository interface {
	Add(ctx context.Context, p *Post) error
	Update(ctx context.Context, a *Post) error
	ByID(ctx context.Context, id uuid.UUID) (*Post, error)
	List(ctx context.Context, page Page) ([]*Post, error)
}
