package post_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/post"
)

const unknownID = "33333333-3333-3333-3333-333333333333"

// stubService implements post.Service. Because the handler depends on an
// interface it declares itself, these tests need no database and no container.
type stubService struct {
	post *entity.Post
	err  error
}

func (s *stubService) CreatePost(_ context.Context, _ string) (*entity.Post, error) {
	return s.post, s.err
}

func (s *stubService) ListPosts(_ context.Context, _ entity.Page) ([]*entity.Post, error) {
	if s.err != nil {
		return nil, s.err
	}

	return nil, nil
}

func (s *stubService) GetPost(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
	return s.post, s.err
}

// newHandlerRouter mounts just the post routes, so these tests exercise the real
// chi routing and the real error mapping.
func newHandlerRouter(svc post.Service) http.Handler {
	h := post.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	r := chi.NewRouter()
	r.Mount("/v1/posts", h.Routes())

	return r
}

func do(t *testing.T, handler http.Handler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader

	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)

		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequestWithContext(context.Background(), method, target, reader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// decodeError reads the single error shape the API returns.
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) (message, code string) {
	t.Helper()

	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	return body.Error, body.Code
}

// TestPostErrorMapping is the regression guard on the post sentinels reaching
// the status table at all. Before the REST layer was split per resource they
// were unreachable from the account-only mapper, and every one of these
// returned 500 internal_error.
func TestPostErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", entity.ErrNotFound, http.StatusNotFound, "post_not_found"},
		{"conflict", entity.ErrConflict, http.StatusConflict, "concurrent_modification"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newHandlerRouter(&stubService{err: tt.err})

			rec := do(t, h, http.MethodGet, "/v1/posts/"+unknownID, nil)

			require.Equal(t, tt.wantStatus, rec.Code)

			_, code := decodeError(t, rec)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestCreatePostEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("rejects an empty name with 400", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{err: entity.ErrEmptyName})

		rec := do(t, h, http.MethodPost, "/v1/posts", map[string]string{"name": ""})

		require.Equal(t, http.StatusBadRequest, rec.Code)

		message, code := decodeError(t, rec)
		assert.Equal(t, "invalid_request", code)
		assert.Equal(t, "name must not be empty", message)
	})

	t.Run("returns 201 with a Location header", func(t *testing.T) {
		t.Parallel()

		id := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		created, err := entity.Create(id, "first post", testNow)
		require.NoError(t, err)

		h := newHandlerRouter(&stubService{post: created})

		rec := do(t, h, http.MethodPost, "/v1/posts", map[string]string{"name": "first post"})

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "/v1/posts/"+id.String(), rec.Header().Get("Location"))
		assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	})
}

// TestGetPostEndpoint covers the transport rejection, which never reaches the
// service and so is mapped by httpx rather than by this package's statusFor.
func TestGetPostEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("rejects a malformed id with 400", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{})

		rec := do(t, h, http.MethodGet, "/v1/posts/not-a-uuid", nil)

		require.Equal(t, http.StatusBadRequest, rec.Code)

		_, code := decodeError(t, rec)
		assert.Equal(t, "invalid_request", code)
	})
}

// TestListPostsEndpoint checks the shared pagination parser is wired to the post
// aggregate's own Page type.
func TestListPostsEndpoint(t *testing.T) {
	t.Parallel()

	h := newHandlerRouter(&stubService{})

	rec := do(t, h, http.MethodGet, "/v1/posts?limit=999", nil)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Posts  []any `json:"posts"`
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, int32(100), body.Limit, "limit must be clamped to MaxLimit")
	assert.Empty(t, body.Posts)
}
