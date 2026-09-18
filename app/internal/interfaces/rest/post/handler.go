// Package post is the HTTP adapter for the post aggregate. It translates JSON
// into use-case calls and post sentinels into status codes, and contains no
// business rules.
//
// It imports exactly one domain package. Anything a second resource would also
// need belongs in rest/httpx, phrased in transport terms.
package post

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Service is the port this handler needs, declared here on the consuming side
// rather than exported by the application package.
type Service interface {
	CreatePost(ctx context.Context, name string) (*entity.Post, error)
	ListPosts(ctx context.Context, page entity.Page) ([]*entity.Post, error)
	GetPost(ctx context.Context, id uuid.UUID) (*entity.Post, error)
}

// Handler serves the post endpoints.
type Handler struct {
	service Service
	log     *slog.Logger
}

// NewHandler builds the handler.
func NewHandler(service Service, log *slog.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Routes mounts the post endpoints on their own sub-router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Route("/{postID}", func(r chi.Router) {
		r.Get("/", h.get)
	})

	return r
}

// fail writes an error response, binding this package's sentinel mapping to the
// shared writer so every call site stays one line.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteError(w, r, h.log, err, statusFor)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.Decode[openPostRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	created, err := h.service.CreatePost(r.Context(), req.Name)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	w.Header().Set("Location", "/v1/posts/"+created.ID().String())
	httpx.WriteJSON(w, h.log, http.StatusCreated, toResponse(created))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := postID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	found, err := h.service.GetPost(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(found))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePage(r)
	page := entity.Page{Limit: p.Limit, Offset: p.Offset}

	posts, err := h.service.ListPosts(r.Context(), page)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, listResponse{
		Posts:  toResponses(posts),
		Limit:  page.Limit,
		Offset: page.Offset,
	})
}

// postID pulls the path parameter and validates it.
func postID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "postID")

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.NewRequestError("%q is not a valid post id", raw)
	}

	return id, nil
}
