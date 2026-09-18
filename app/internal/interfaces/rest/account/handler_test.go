package account_test

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

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/account"
)

// stubService implements account.Service. Because the handler depends on an
// interface it declares itself, these tests need no database and no container.
type stubService struct {
	account *entity.Account
	err     error

	gotID       uuid.UUID
	gotMinor    int64
	gotCurrency string
}

func (s *stubService) OpenAccount(_ context.Context, owner, currency string) (*entity.Account, error) {
	if s.err != nil {
		return nil, s.err
	}

	return entity.Open(uuid.MustParse("22222222-2222-2222-2222-222222222222"), owner, currency, testNow)
}

func (s *stubService) GetAccount(_ context.Context, id uuid.UUID) (*entity.Account, error) {
	s.gotID = id

	return s.account, s.err
}

func (s *stubService) ListAccounts(_ context.Context, _ entity.Page) ([]*entity.Account, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.account == nil {
		return nil, nil
	}

	return []*entity.Account{s.account}, nil
}

func (s *stubService) Deposit(_ context.Context, id uuid.UUID, minor int64, currency string) (*entity.Account, error) {
	s.gotID, s.gotMinor, s.gotCurrency = id, minor, currency

	return s.account, s.err
}

func (s *stubService) Withdraw(_ context.Context, id uuid.UUID, minor int64, currency string) (*entity.Account, error) {
	s.gotID, s.gotMinor, s.gotCurrency = id, minor, currency

	return s.account, s.err
}

func (s *stubService) CloseAccount(_ context.Context, id uuid.UUID) (*entity.Account, error) {
	s.gotID = id

	return s.account, s.err
}

// newHandlerRouter mounts just the account routes, so these tests exercise the
// real chi routing and the real error mapping without the full application
// router, which needs a database pool for its readiness probe.
func newHandlerRouter(svc account.Service) http.Handler {
	h := account.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	r := chi.NewRouter()
	r.Mount("/v1/accounts", h.Routes())

	return r
}

// do issues a request. A string body is sent verbatim so a test can supply
// deliberately malformed JSON; anything else is marshalled.
func do(t *testing.T, handler http.Handler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader

	switch b := body.(type) {
	case nil:
	case string:
		reader = bytes.NewBufferString(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)

		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequestWithContext(context.Background(), method, target, reader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

func TestOpenAccountEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("returns 201 with a Location header", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{})

		rec := do(t, h, http.MethodPost, "/v1/accounts", map[string]string{
			"owner":    "Ada Lovelace",
			"currency": "USD",
		})

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.NotEmpty(t, rec.Header().Get("Location"))

		// Decoded into a typed struct rather than map[string]any, so balance_minor
		// is compared as the integer it is instead of as a float.
		var got struct {
			Owner        string `json:"owner"`
			Currency     string `json:"currency"`
			BalanceMinor int64  `json:"balance_minor"`
			Status       string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, "Ada Lovelace", got.Owner)
		assert.Equal(t, "USD", got.Currency)
		assert.Equal(t, int64(0), got.BalanceMinor)
		assert.Equal(t, "active", got.Status)
	})

	t.Run("maps a domain validation error to 400", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{})

		rec := do(t, h, http.MethodPost, "/v1/accounts", map[string]string{
			"owner":    "",
			"currency": "USD",
		})

		require.Equal(t, http.StatusBadRequest, rec.Code)

		var got map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, "invalid_request", got["code"])
	})

	t.Run("rejects malformed JSON as malformed", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{})

		// The trailing comma makes this genuinely malformed. The assertion is
		// that it is rejected for that reason, not mistaken for a missing field.
		rec := do(t, h, http.MethodPost, "/v1/accounts", "{\"owner\":\"Ada\",}")

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "malformed JSON")
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{})

		rec := do(t, h, http.MethodPost, "/v1/accounts",
			"{\"owner\":\"Ada\",\"currency\":\"USD\",\"admin\":true}")

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestGetAccountEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("rejects a malformed id before reaching the service", func(t *testing.T) {
		t.Parallel()

		svc := &stubService{}
		h := newHandlerRouter(svc)

		rec := do(t, h, http.MethodGet, "/v1/accounts/not-a-uuid", nil)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, uuid.Nil, svc.gotID, "the service must not be called with a bad id")
	})

	t.Run("maps ErrNotFound to 404", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{err: entity.ErrNotFound})

		rec := do(t, h, http.MethodGet, "/v1/accounts/"+uuid.New().String(), nil)

		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "account_not_found")
	})
}

func TestBalanceEndpoints(t *testing.T) {
	t.Parallel()

	openAccount := func(t *testing.T) *entity.Account {
		t.Helper()

		a, err := entity.Open(uuid.New(), "Ada", "USD", testNow)
		require.NoError(t, err)

		return a
	}

	t.Run("passes the amount through to the service", func(t *testing.T) {
		t.Parallel()

		svc := &stubService{account: openAccount(t)}
		h := newHandlerRouter(svc)
		id := uuid.New()

		rec := do(t, h, http.MethodPost, "/v1/accounts/"+id.String()+"/deposit", map[string]any{
			"amount_minor": 2500,
			"currency":     "USD",
		})

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, id, svc.gotID)
		assert.Equal(t, int64(2500), svc.gotMinor)
		assert.Equal(t, "USD", svc.gotCurrency)
	})

	t.Run("rejects a missing amount rather than defaulting it to zero", func(t *testing.T) {
		t.Parallel()

		svc := &stubService{account: openAccount(t)}
		h := newHandlerRouter(svc)

		rec := do(t, h, http.MethodPost, "/v1/accounts/"+uuid.New().String()+"/deposit", map[string]any{
			"currency": "USD",
		})

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "amount_minor is required")
		assert.Equal(t, int64(0), svc.gotMinor, "the service must not be called")
	})

	t.Run("maps insufficient funds to 422", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{err: entity.ErrInsufficientFunds})

		rec := do(t, h, http.MethodPost, "/v1/accounts/"+uuid.New().String()+"/withdraw", map[string]any{
			"amount_minor": 10,
			"currency":     "USD",
		})

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "insufficient_funds")
	})

	t.Run("maps an optimistic-lock conflict to 409", func(t *testing.T) {
		t.Parallel()

		h := newHandlerRouter(&stubService{err: entity.ErrConflict})

		rec := do(t, h, http.MethodPost, "/v1/accounts/"+uuid.New().String()+"/deposit", map[string]any{
			"amount_minor": 10,
			"currency":     "USD",
		})

		require.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestListEndpointClampsPagination(t *testing.T) {
	t.Parallel()

	h := newHandlerRouter(&stubService{})

	rec := do(t, h, http.MethodGet, "/v1/accounts?limit=99999&offset=-5", nil)

	require.Equal(t, http.StatusOK, rec.Code)

	var got struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, int32(100), got.Limit, "limit must be clamped to the maximum")
	assert.Equal(t, int32(0), got.Offset, "a negative offset must fall back to zero")
}

func TestUnexpectedErrorDoesNotLeakDetail(t *testing.T) {
	t.Parallel()

	h := newHandlerRouter(&stubService{err: errBoom})

	rec := do(t, h, http.MethodGet, "/v1/accounts/"+uuid.New().String(), nil)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "boom", "internal error text must not reach the client")
	assert.Contains(t, rec.Body.String(), "internal server error")
}
