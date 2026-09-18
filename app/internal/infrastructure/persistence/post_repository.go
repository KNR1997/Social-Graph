package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence/sqlcgen"
)

type PostRepository struct {
	pool *pgxpool.Pool
}

var _ entity.Repository = (*PostRepository)(nil)

func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{pool: pool}
}

// Add inserts a newly opened post.
func (r *PostRepository) Add(ctx context.Context, p *entity.Post) error {
	err := queries(ctx, r.pool).InsertPost(ctx, sqlcgen.InsertPostParams{
		ID:        p.ID(),
		Name:      p.Name(),
		Version:   p.Version(),
		CreatedAt: p.CreatedAt(),
		UpdatedAt: p.UpdatedAt(),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("persistence - Add: %w", entity.ErrConflict)
		}

		return fmt.Errorf("persistence - Add - InsertPost: %w", err)
	}

	return nil
}

func (r *PostRepository) Update(ctx context.Context, p *entity.Post) error {
	rows, err := queries(ctx, r.pool).UpdatePost(ctx, sqlcgen.UpdatePostParams{
		ID:        p.ID(),
		Name:      p.Name(),
		UpdatedAt: p.UpdatedAt(),
	})
	if err != nil {
		return fmt.Errorf("persistence - Update - UpdatePost: %w", err)
	}

	// Zero rows means either the id is gone or another writer moved the version
	// on. Both are conflicts from this caller's point of view.
	if rows == 0 {
		return fmt.Errorf("persistence - Update: %w", entity.ErrConflict)
	}

	return nil
}

func (r *PostRepository) ByID(ctx context.Context, id uuid.UUID) (*entity.Post, error) {
	row, err := queries(ctx, r.pool).GetPost(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - ByID: %w", entity.ErrNotFound)
		}

		return nil, fmt.Errorf("persistence - ByID - GetPost: %w", err)
	}

	post, err := toPostEntity(&row)
	if err != nil {
		return nil, fmt.Errorf("persistence - ByID: %w", err)
	}

	return post, nil
}

func (r *PostRepository) List(ctx context.Context, page entity.Page) ([]*entity.Post, error) {
	rows, err := queries(ctx, r.pool).ListPosts(ctx, sqlcgen.ListPostsParams{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("persistence - List - ListPosts: %w", err)
	}

	posts := make([]*entity.Post, 0, len(rows))
	// Index rather than range-copy: the generated row struct is large enough that
	// copying it per iteration shows up in profiles on wide pages.
	for i := range rows {
		post, err := toPostEntity(&rows[i])
		if err != nil {
			return nil, fmt.Errorf("persistence - List: %w", err)
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func toPostEntity(row *sqlcgen.Post) (*entity.Post, error) {
	return entity.Reconstitute(
		row.ID,
		row.Name,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}
