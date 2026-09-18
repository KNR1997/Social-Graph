//go:build integration

package internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/config"
	"github.com/kethaka-creskit/go-ddd-service/internal"
	"github.com/kethaka-creskit/go-ddd-service/internal/test/containers"
)

// sessionCookie is the cookie name from .env.example, which these tests load.
const sessionCookie = "sid"

// client drives the API over an in-process handler, carrying a session cookie
// the way a browser would.
//
// It exists because every resource route is now behind authentication: without
// somewhere to keep the cookie, each test would have to thread one through by
// hand and the assertions would disappear into plumbing.
type client struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
}

// newAPI builds the real service on a throwaway database.
//
// This is the payoff of splitting the Wire graph: the test exercises the exact
// production wiring (real repository, real transaction manager, real router)
// with only the pool substituted. There is no second, drifting object graph.
func newAPI(t *testing.T) http.Handler {
	t.Helper()

	dsn := containers.StartPostgres(t)
	// Sessions before users: the foreign key cascades, but truncating in
	// dependency order keeps the intent readable.
	containers.Truncate(t, dsn, "sessions", "users", "accounts", "posts")

	cfg, err := config.LoadFromFile("../.env.example")
	require.NoError(t, err)
	cfg.DB.URL = dsn
	cfg.Log.Level = "error"

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return internal.InitializeAPIWithPool(cfg, pool).Handler
}

// newAnonClient builds a client with no session.
func newAnonClient(t *testing.T) *client {
	t.Helper()

	return &client{t: t, h: newAPI(t)}
}

// newClient builds a client that has already registered and is signed in,
// which is the starting state for every test about something other than auth.
func newClient(t *testing.T) *client {
	t.Helper()

	c := newAnonClient(t)
	c.register("ada@example.com", "Ada Lovelace", "correct-horse-battery")

	return c
}

// do issues a request carrying the client's session cookie, if it has one.
func (c *client) do(method, target string, body any) *httptest.ResponseRecorder {
	c.t.Helper()

	var buf bytes.Buffer
	if body != nil {
		require.NoError(c.t, json.NewEncoder(&buf).Encode(body))
	}

	req := httptest.NewRequestWithContext(context.Background(), method, target, &buf)
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}

	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)

	// Track Set-Cookie the way a browser would, so a re-issued session (after a
	// password change) or a cleared one (after sign-out) is picked up without
	// the test having to know it happened.
	c.absorb(rec)

	return rec
}

// absorb applies any session cookie the response set.
func (c *client) absorb(rec *httptest.ResponseRecorder) {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name != sessionCookie {
			continue
		}

		if cookie.MaxAge < 0 || cookie.Value == "" {
			c.cookie = nil

			continue
		}

		c.cookie = cookie
	}
}

// register creates a user and leaves the client signed in as them.
func (c *client) register(email, name, password string) *httptest.ResponseRecorder {
	c.t.Helper()

	rec := c.do(http.MethodPost, "/v1/auth/register", map[string]string{
		"email":    email,
		"name":     name,
		"password": password,
	})
	require.Equal(c.t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NotNil(c.t, c.cookie, "register must set a session cookie")

	return rec
}

type accountBody struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	BalanceMinor int64  `json:"balance_minor"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
	Version      int64  `json:"version"`
}

func decodeAccount(t *testing.T, rec *httptest.ResponseRecorder) accountBody {
	t.Helper()

	var got accountBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got), "body: %s", rec.Body.String())

	return got
}

type userBody struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func TestHealthAndReadiness(t *testing.T) {
	c := newAnonClient(t)

	// The operational endpoints stay outside authentication: a load balancer
	// does not have a session, and a readiness probe that can fail on an auth
	// bug is worse than no probe at all.
	assert.Equal(t, http.StatusOK, c.do(http.MethodGet, "/healthz", nil).Code)
	assert.Equal(t, http.StatusOK, c.do(http.MethodGet, "/readyz", nil).Code)
}

func TestResourceRoutesRequireASession(t *testing.T) {
	c := newAnonClient(t)

	for _, target := range []string{"/v1/accounts", "/v1/posts", "/v1/auth/me"} {
		rec := c.do(http.MethodGet, target, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "GET %s", target)
		assert.Contains(t, rec.Body.String(), "unauthenticated", "GET %s", target)
	}
}

func TestRegisterLoginAndSignOut(t *testing.T) {
	c := newAnonClient(t)

	rec := c.register("ada@example.com", "Ada Lovelace", "correct-horse-battery")
	assert.NotContains(t, rec.Body.String(), "correct-horse-battery",
		"the response must never echo the password")
	assert.NotContains(t, rec.Body.String(), "argon2",
		"the response must never carry the password hash")

	// The session cookie is not readable by scripts and is withheld from
	// cross-site mutations.
	require.True(t, c.cookie.HttpOnly, "session cookie must be HttpOnly")
	require.Equal(t, http.SameSiteLaxMode, c.cookie.SameSite)

	// The session works.
	rec = c.do(http.MethodGet, "/v1/auth/me", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var me userBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
	assert.Equal(t, "ada@example.com", me.Email)
	assert.Equal(t, "member", me.Role)

	// Registering the same address again is refused, whatever the casing.
	rec = c.do(http.MethodPost, "/v1/auth/register", map[string]string{
		"email":    "ADA@Example.com",
		"name":     "Impostor",
		"password": "another-long-password",
	})
	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "email_taken")

	// Sign out, and the cookie stops working.
	rec = c.do(http.MethodPost, "/v1/auth/logout", nil)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Nil(t, c.cookie, "sign-out must clear the session cookie")

	rec = c.do(http.MethodGet, "/v1/auth/me", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Sign back in.
	rec = c.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "ada@example.com",
		"password": "correct-horse-battery",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, c.cookie)

	assert.Equal(t, http.StatusOK, c.do(http.MethodGet, "/v1/auth/me", nil).Code)
}

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	c := newAnonClient(t)
	c.register("ada@example.com", "Ada Lovelace", "correct-horse-battery")

	wrongPassword := c.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "ada@example.com",
		"password": "not-the-password",
	})
	unknownUser := c.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "not-the-password",
	})

	require.Equal(t, http.StatusUnauthorized, wrongPassword.Code)
	require.Equal(t, http.StatusUnauthorized, unknownUser.Code)
	assert.JSONEq(t, wrongPassword.Body.String(), unknownUser.Body.String(),
		"a wrong password and an unregistered address must be indistinguishable, "+
			"or the login form becomes an account-enumeration oracle")
}

func TestRejectsAWeakPassword(t *testing.T) {
	c := newAnonClient(t)

	rec := c.do(http.MethodPost, "/v1/auth/register", map[string]string{
		"email":    "short@example.com",
		"name":     "Too Short",
		"password": "sixchr",
	})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "weak_password")
}

func TestChangingThePasswordEvictsOtherSessions(t *testing.T) {
	first := newClient(t)

	// A second sign-in on the same account: think of it as another browser.
	second := &client{t: t, h: first.h}
	rec := second.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "ada@example.com",
		"password": "correct-horse-battery",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, second.cookie)

	// The first browser changes the password.
	rec = first.do(http.MethodPost, "/v1/auth/me/password", map[string]string{
		"current_password": "correct-horse-battery",
		"new_password":     "an-entirely-new-password",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// It keeps its own session, on a freshly issued cookie.
	assert.Equal(t, http.StatusOK, first.do(http.MethodGet, "/v1/auth/me", nil).Code,
		"the browser that changed the password stays signed in")

	// The other one is evicted, which is the entire point of changing it.
	assert.Equal(t, http.StatusUnauthorized, second.do(http.MethodGet, "/v1/auth/me", nil).Code,
		"every other session must be destroyed by a password change")

	// The old password no longer works.
	rec = second.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "ada@example.com",
		"password": "correct-horse-battery",
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWrongCurrentPasswordCannotRotateIt(t *testing.T) {
	c := newClient(t)

	rec := c.do(http.MethodPost, "/v1/auth/me/password", map[string]string{
		"current_password": "not-the-password",
		"new_password":     "an-entirely-new-password",
	})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_credentials")

	// The original password still works, so nothing was half-applied.
	rec = c.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "ada@example.com",
		"password": "correct-horse-battery",
	})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestRenameTheSignedInUser(t *testing.T) {
	c := newClient(t)

	rec := c.do(http.MethodPatch, "/v1/auth/me", map[string]string{"name": "Ada King"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = c.do(http.MethodGet, "/v1/auth/me", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var me userBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
	assert.Equal(t, "Ada King", me.Name)
}

func TestAccountLifecycleOverHTTP(t *testing.T) {
	c := newClient(t)

	// Open.
	rec := c.do(http.MethodPost, "/v1/accounts", map[string]string{
		"owner":    "Ada Lovelace",
		"currency": "USD",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	opened := decodeAccount(t, rec)
	assert.Equal(t, int64(0), opened.BalanceMinor)
	assert.Equal(t, "active", opened.Status)

	base := "/v1/accounts/" + opened.ID

	// Deposit.
	rec = c.do(http.MethodPost, base+"/deposit", map[string]any{
		"amount_minor": 10_000,
		"currency":     "USD",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, int64(10_000), decodeAccount(t, rec).BalanceMinor)

	// Withdraw within the balance.
	rec = c.do(http.MethodPost, base+"/withdraw", map[string]any{
		"amount_minor": 2_500,
		"currency":     "USD",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	withdrawn := decodeAccount(t, rec)
	assert.Equal(t, int64(7_500), withdrawn.BalanceMinor)
	assert.Equal(t, int64(3), withdrawn.Version, "each committed change advances the version")

	// Overdraw is refused, and the balance is unchanged.
	rec = c.do(http.MethodPost, base+"/withdraw", map[string]any{
		"amount_minor": 7_501,
		"currency":     "USD",
	})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "insufficient_funds")

	rec = c.do(http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(7_500), decodeAccount(t, rec).BalanceMinor,
		"the refused withdrawal must have rolled back")

	// A foreign currency is refused.
	rec = c.do(http.MethodPost, base+"/deposit", map[string]any{
		"amount_minor": 100,
		"currency":     "EUR",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// Closing a funded account is refused.
	rec = c.do(http.MethodDelete, base, nil)
	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "balance_not_zero")

	// Drain, then close.
	rec = c.do(http.MethodPost, base+"/withdraw", map[string]any{
		"amount_minor": 7_500,
		"currency":     "USD",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = c.do(http.MethodDelete, base, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "closed", decodeAccount(t, rec).Status)

	// A closed account rejects further movement.
	rec = c.do(http.MethodPost, base+"/deposit", map[string]any{
		"amount_minor": 1,
		"currency":     "USD",
	})
	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "account_not_active")
}

func TestUnknownAccountIs404(t *testing.T) {
	c := newClient(t)

	rec := c.do(http.MethodGet, "/v1/accounts/33333333-3333-3333-3333-333333333333", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "account_not_found")
}

func TestListPagination(t *testing.T) {
	c := newClient(t)

	for _, owner := range []string{"one", "two", "three"} {
		rec := c.do(http.MethodPost, "/v1/accounts", map[string]string{
			"owner":    owner,
			"currency": "USD",
		})
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}

	rec := c.do(http.MethodGet, "/v1/accounts?limit=2", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var got struct {
		Accounts []accountBody `json:"accounts"`
		Limit    int32         `json:"limit"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Len(t, got.Accounts, 2)
	assert.Equal(t, int32(2), got.Limit)
}
