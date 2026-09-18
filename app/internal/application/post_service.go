package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
)

type PostService struct {
	posts entity.Repository
	tx    TxManager
	clock Clock
	ids   IDGenerator
	log   *slog.Logger
}

func NewPostService(
	posts entity.Repository,
	tx TxManager,
	clock Clock,
	ids IDGenerator,
	log *slog.Logger,
) *PostService {
	return &PostService{
		posts: posts,
		tx:    tx,
		clock: clock,
		ids:   ids,
		log:   log,
	}
}

func (s *PostService) CreatePost(ctx context.Context, name string) (*entity.Post, error) {
	s.log.InfoContext(ctx, "creating post",
		slog.String("name", name),
	)
	post, err := entity.Create(s.ids.NewID(), name, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("application - CreatePost - entity.Create: %w", err)
	}

	if err := s.posts.Add(ctx, post); err != nil {
		return nil, fmt.Errorf("application - CreatePost - posts.Add: %w", err)
	}

	s.log.InfoContext(ctx, "post created",
		slog.String("post_id", post.ID().String()),
		slog.String("name", post.Name()),
	)

	return post, nil
}

func (s *PostService) ListPosts(ctx context.Context, page entity.Page) ([]*entity.Post, error) {
	posts, err := s.posts.List(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("application - ListPosts - posts.List: %w", err)
	}

	return posts, nil
}

func (s *PostService) GetPost(ctx context.Context, id uuid.UUID) (*entity.Post, error) {
	post, err := s.posts.ByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("application - GetPost - posts.ByID: %w", err)
	}

	return post, nil
}
