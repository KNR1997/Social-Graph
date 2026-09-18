package application_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kethaka-creskit/go-ddd-service/internal/application"
	authentity "github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
)

// --- test doubles -----------------------------------------------------------

type fakeUserRepo struct {
	users    map[uuid.UUID]*authentity.User
	failNext error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[uuid.UUID]*authentity.User{}}
}

func (r *fakeUserRepo) Add(_ context.Context, u *authentity.User) error {
	if err := r.take(); err != nil {
		return err
	}
	for _, existing := range r.users {
		if existing.Email() == u.Email() {
			return authentity.ErrEmailTaken
		}
	}
	r.users[u.ID()] = u
	return nil
}

func (r *fakeUserRepo) Update(_ context.Context, u *authentity.User) error {
	if err := r.take(); err != nil {
		return err
	}
	if _, ok := r.users[u.ID()]; !ok {
		return authentity.ErrUserNotFound
	}
	r.users[u.ID()] = u
	return nil
}

func (r *fakeUserRepo) ByID(_ context.Context, id uuid.UUID) (*authentity.User, error) {
	if err := r.take(); err != nil {
		return nil, err
	}
	u, ok := r.users[id]
	if !ok {
		return nil, authentity.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) ByEmail(_ context.Context, email authentity.Email) (*authentity.User, error) {
	if err := r.take(); err != nil {
		return nil, err
	}
	for _, u := range r.users {
		if u.Email() == email {
			return u, nil
		}
	}
	return nil, authentity.ErrUserNotFound
}

func (r *fakeUserRepo) take() error {
	err := r.failNext
	r.failNext = nil
	return err
}

// fakeSessionRepo deliberately does not filter expired rows the way the real
// query does. That leaves the aggregate's own expiry check as the thing under
// test: if it ever stopped working, the production query would hide it here.
type fakeSessionRepo struct {
	sessions map[uuid.UUID]*authentity.Session
	deleted  int
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: map[uuid.UUID]*authentity.Session{}}
}

func (r *fakeSessionRepo) Add(_ context.Context, s *authentity.Session) error {
	r.sessions[s.ID()] = s
	return nil
}

func (r *fakeSessionRepo) ByTokenHash(
	_ context.Context,
	tokenHash string,
	_ time.Time,
) (*authentity.Session, error) {
	for _, s := range r.sessions {
		if s.TokenHash() == tokenHash {
			return s, nil
		}
	}
	return nil, authentity.ErrSessionNotFound
}

func (r *fakeSessionRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.sessions[id]; ok {
		r.deleted++
	}
	delete(r.sessions, id)
	return nil
}

func (r *fakeSessionRepo) DeleteByUser(_ context.Context, userID uuid.UUID) error {
	for id, s := range r.sessions {
		if s.UserID() == userID {
			r.deleted++
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *fakeSessionRepo) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for id, s := range r.sessions {
		if s.IsExpired(now) {
			delete(r.sessions, id)
			n++
		}
	}
	return n, nil
}

// fakeHasher stands in for Argon2id. It is reversible on purpose: these tests
// are about the use case, and a real KDF would add seconds to every run.
type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }

func (fakeHasher) Verify(hash, plain string) error {
	if hash != "hashed:"+plain {
		return application.ErrPasswordMismatch
	}
	return nil
}

// seqTokens hands out predictable tokens so assertions can name them.
type seqTokens struct{ n int }

func (g *seqTokens) New() (string, error) {
	g.n++
	return "token-" + string(rune('a'+g.n-1)), nil
}

func (g *seqTokens) Hash(token string) string { return "digest:" + token }

// --- fixture ----------------------------------------------------------------

const (
	testEmail    = "ada@example.com"
	testName     = "Ada Lovelace"
	testPassword = "correct-horse-battery"
	testTTL      = 24 * time.Hour
)

type authFixture struct {
	svc      *application.AuthService
	users    *fakeUserRepo
	sessions *fakeSessionRepo
	tx       *inlineTx
	clock    *movableClock
}

// movableClock is a Clock whose time the test can advance, which is how session
// expiry is exercised without sleeping.
type movableClock struct{ t time.Time }

func (c *movableClock) Now() time.Time { return c.t }

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()

	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	tx := &inlineTx{}
	clock := &movableClock{t: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := application.NewAuthService(
		users, sessions, tx, clock, &seqIDs{}, fakeHasher{}, &seqTokens{},
		application.SessionTTL(testTTL), log,
	)

	return &authFixture{svc: svc, users: users, sessions: sessions, tx: tx, clock: clock}
}

func (f *authFixture) register(t *testing.T) (*authentity.User, authentity.Credential) {
	t.Helper()

	user, cred, err := f.svc.Register(context.Background(), testEmail, testName, testPassword)
	require.NoError(t, err)

	return user, cred
}

// --- tests ------------------------------------------------------------------

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("stores the user and signs them in", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		user, cred, err := f.svc.Register(context.Background(), "  Ada@Example.com ", testName, testPassword)
		require.NoError(t, err)

		assert.Equal(t, authentity.Email(testEmail), user.Email())
		assert.Equal(t, authentity.RoleMember, user.Role(), "registration never grants admin")
		assert.NotEmpty(t, cred.Token)
		assert.Equal(t, f.clock.Now().Add(testTTL), cred.Session.ExpiresAt())
		assert.Len(t, f.sessions.sessions, 1)
		assert.Equal(t, 1, f.tx.calls, "the user and the session commit together")
	})

	t.Run("stores what the hasher returned, not the password", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		user, _ := f.register(t)

		// The fake hasher is reversible, so this cannot assert the digest is
		// unreadable -- that is the real hasher's test. What it does assert is
		// that the password went through the port at all, rather than being
		// handed to the aggregate untouched.
		expected, err := fakeHasher{}.Hash(testPassword)
		require.NoError(t, err)

		assert.Equal(t, expected, user.PasswordHash())
		assert.NotEqual(t, testPassword, user.PasswordHash())
	})

	t.Run("stores only the token digest", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		_, cred := f.register(t)

		assert.NotEqual(t, cred.Token, cred.Session.TokenHash(),
			"a stolen table dump must not contain replayable tokens")
	})

	t.Run("refuses a duplicate address", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, _, err := f.svc.Register(context.Background(), "ADA@example.com", "Impostor", "another-password")
		require.ErrorIs(t, err, authentity.ErrEmailTaken)
		assert.Len(t, f.users.users, 1)
	})

	t.Run("refuses a weak password before touching the repository", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)

		_, _, err := f.svc.Register(context.Background(), testEmail, testName, "short")
		require.ErrorIs(t, err, authentity.ErrWeakPassword)
		assert.Empty(t, f.users.users)
		assert.Equal(t, 0, f.tx.calls, "a rejected password opens no transaction")
	})

	t.Run("rolls back when the session cannot be stored", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.users.failNext = assert.AnError

		_, _, err := f.svc.Register(context.Background(), testEmail, testName, testPassword)
		require.Error(t, err)
		assert.Equal(t, 1, f.tx.rolledBk)
	})
}

func TestLogin(t *testing.T) {
	t.Parallel()

	t.Run("issues a second, independent session", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		_, first := f.register(t)

		_, second, err := f.svc.Login(context.Background(), testEmail, testPassword)
		require.NoError(t, err)

		assert.NotEqual(t, first.Token, second.Token)
		assert.Len(t, f.sessions.sessions, 2, "signing in on a second device must not evict the first")
	})

	t.Run("reports the same error for a wrong password and an unknown address", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, _, wrongPassword := f.svc.Login(context.Background(), testEmail, "not-the-password")
		_, _, unknownUser := f.svc.Login(context.Background(), "nobody@example.com", testPassword)
		_, _, malformed := f.svc.Login(context.Background(), "not-an-address", testPassword)

		require.ErrorIs(t, wrongPassword, authentity.ErrInvalidCredentials)
		require.ErrorIs(t, unknownUser, authentity.ErrInvalidCredentials)
		require.ErrorIs(t, malformed, authentity.ErrInvalidCredentials,
			"even a malformed address must not be distinguishable from a wrong password")
	})

	t.Run("issues no session on failure", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, _, err := f.svc.Login(context.Background(), testEmail, "not-the-password")
		require.Error(t, err)
		assert.Len(t, f.sessions.sessions, 1, "only the session from registration exists")
	})
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	t.Run("resolves a live token to its user", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		registered, cred := f.register(t)

		user, err := f.svc.Authenticate(context.Background(), cred.Token)
		require.NoError(t, err)
		assert.Equal(t, registered.ID(), user.ID())
	})

	t.Run("rejects an unknown or empty token", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, unknown := f.svc.Authenticate(context.Background(), "not-a-real-token")
		_, empty := f.svc.Authenticate(context.Background(), "")

		require.ErrorIs(t, unknown, authentity.ErrSessionNotFound)
		assert.ErrorIs(t, empty, authentity.ErrSessionNotFound)
	})

	t.Run("rejects and deletes an expired session", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		_, cred := f.register(t)

		f.clock.t = f.clock.t.Add(testTTL + time.Second)

		_, err := f.svc.Authenticate(context.Background(), cred.Token)
		require.ErrorIs(t, err, authentity.ErrSessionExpired)
		assert.Empty(t, f.sessions.sessions, "an expired session is cleaned up as it is rejected")
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("destroys the session server-side", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		_, cred := f.register(t)

		require.NoError(t, f.svc.Logout(context.Background(), cred.Token))
		assert.Empty(t, f.sessions.sessions)

		// The token is dead, not merely forgotten by the browser.
		_, err := f.svc.Authenticate(context.Background(), cred.Token)
		assert.ErrorIs(t, err, authentity.ErrSessionNotFound)
	})

	t.Run("is idempotent", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		_, cred := f.register(t)

		require.NoError(t, f.svc.Logout(context.Background(), cred.Token))
		require.NoError(t, f.svc.Logout(context.Background(), cred.Token))
		require.NoError(t, f.svc.Logout(context.Background(), ""))
	})

	t.Run("leaves the user's other sessions alone", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		_, first := f.register(t)

		_, second, err := f.svc.Login(context.Background(), testEmail, testPassword)
		require.NoError(t, err)

		require.NoError(t, f.svc.Logout(context.Background(), first.Token))

		_, err = f.svc.Authenticate(context.Background(), second.Token)
		assert.NoError(t, err, "signing out of one device must not sign out the others")
	})
}

func TestChangePassword(t *testing.T) {
	t.Parallel()

	t.Run("evicts every other session and re-issues the caller's", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, other, err := f.svc.Login(context.Background(), testEmail, testPassword)
		require.NoError(t, err)

		user, cred, err := f.svc.ChangePassword(context.Background(), firstUserID(t, f), testPassword, "a-brand-new-password")
		require.NoError(t, err)

		assert.Len(t, f.sessions.sessions, 1, "exactly one session survives: the new one")

		_, err = f.svc.Authenticate(context.Background(), other.Token)
		require.ErrorIs(t, err, authentity.ErrSessionNotFound, "the other device is signed out")

		resolved, err := f.svc.Authenticate(context.Background(), cred.Token)
		require.NoError(t, err)
		assert.Equal(t, user.ID(), resolved.ID())
	})

	t.Run("only the new password works afterwards", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, _, err := f.svc.ChangePassword(context.Background(), firstUserID(t, f), testPassword, "a-brand-new-password")
		require.NoError(t, err)

		_, _, err = f.svc.Login(context.Background(), testEmail, testPassword)
		require.ErrorIs(t, err, authentity.ErrInvalidCredentials)

		_, _, err = f.svc.Login(context.Background(), testEmail, "a-brand-new-password")
		require.NoError(t, err, "the new password works")
	})

	t.Run("refuses a wrong current password and changes nothing", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, _, err := f.svc.ChangePassword(context.Background(), firstUserID(t, f), "not-the-password", "a-brand-new-password")
		require.ErrorIs(t, err, authentity.ErrInvalidCredentials)
		assert.Equal(t, 1, f.tx.rolledBk)

		_, _, err = f.svc.Login(context.Background(), testEmail, testPassword)
		assert.NoError(t, err, "the original password still works")
	})

	t.Run("refuses a weak new password", func(t *testing.T) {
		t.Parallel()

		f := newAuthFixture(t)
		f.register(t)

		_, _, err := f.svc.ChangePassword(context.Background(), firstUserID(t, f), testPassword, "short")
		require.ErrorIs(t, err, authentity.ErrWeakPassword)
	})
}

func TestRenameUser(t *testing.T) {
	t.Parallel()

	f := newAuthFixture(t)
	f.register(t)

	user, err := f.svc.Rename(context.Background(), firstUserID(t, f), "  Ada King  ")
	require.NoError(t, err)
	assert.Equal(t, "Ada King", user.Name())

	_, err = f.svc.Rename(context.Background(), firstUserID(t, f), "   ")
	assert.ErrorIs(t, err, authentity.ErrEmptyName)
}

func TestPurgeExpiredSessions(t *testing.T) {
	t.Parallel()

	f := newAuthFixture(t)
	f.register(t)

	deleted, err := f.svc.PurgeExpiredSessions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	f.clock.t = f.clock.t.Add(testTTL + time.Second)

	deleted, err = f.svc.PurgeExpiredSessions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}

func TestPasswordPolicyBoundary(t *testing.T) {
	t.Parallel()

	f := newAuthFixture(t)

	_, _, err := f.svc.Register(context.Background(), testEmail, testName,
		strings.Repeat("a", authentity.MinPasswordLen))
	assert.NoError(t, err, "the shortest allowed password is allowed")
}

// firstUserID returns the id of the single registered user in a fixture.
func firstUserID(t *testing.T, f *authFixture) uuid.UUID {
	t.Helper()

	require.Len(t, f.users.users, 1)
	for id := range f.users.users {
		return id
	}

	return uuid.Nil
}
