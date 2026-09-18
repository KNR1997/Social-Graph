package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyName = errors.New("name must not be empty")
	ErrConflict  = errors.New("account was modified concurrently")
	ErrNotFound  = errors.New("account not found")
)

type Post struct {
	id        uuid.UUID
	name      string
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

func Create(id uuid.UUID, name string, now time.Time) (*Post, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	return &Post{
		id:        id,
		name:      name,
		version:   1,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	name string,
	createdAt, updatedAt time.Time,
) *Post {
	return &Post{
		id:        id,
		name:      name,
		createdAt: createdAt.UTC(),
		updatedAt: updatedAt.UTC(),
	}
}

func (p *Post) ID() uuid.UUID { return p.id }

func (p *Post) Name() string { return p.name }

func (p *Post) Version() int64 { return p.version }

func (p *Post) CreatedAt() time.Time { return p.createdAt }

func (p *Post) UpdatedAt() time.Time { return p.updatedAt }
