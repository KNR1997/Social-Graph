package post

import (
	"time"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
)

// openPostRequest is the POST /v1/posts body.
type openPostRequest struct {
	Name string `json:"name"`
}

// postResponse is the wire representation of a post. It is a separate type from
// the aggregate on purpose: the API shape can stay stable while the aggregate
// evolves, and the aggregate never has to grow json tags.
type postResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// listResponse wraps a page of posts in an object rather than returning a bare
// array, so pagination metadata can be added later without a breaking change.
type listResponse struct {
	Posts  []postResponse `json:"posts"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
}

func toResponse(p *entity.Post) postResponse {
	return postResponse{
		ID:        p.ID(),
		Name:      p.Name(),
		CreatedAt: p.CreatedAt(),
		UpdatedAt: p.UpdatedAt(),
	}
}

func toResponses(posts []*entity.Post) []postResponse {
	out := make([]postResponse, 0, len(posts))
	for _, p := range posts {
		out = append(out, toResponse(p))
	}

	return out
}
