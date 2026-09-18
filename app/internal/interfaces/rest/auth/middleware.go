package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
)

// userKey is an unexported context key, so nothing outside this package can put
// a user on a context and claim a request is authenticated.
type userKey struct{}

// RequireSession is the authentication middleware. Routes wrapped in it are
// reached only by a request carrying a live session cookie; the authenticated
// user is placed on the context for handlers that need it.
//
// It lives in this package rather than in rest because it is the one place in
// the interfaces layer that already names the identity aggregate. Keeping it
// here leaves the router free of any domain import at all.
func (h *Handler) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.service.Authenticate(r.Context(), h.cookie.token(r))
		if err != nil {
			// A rejected token is a dead token. Clearing it stops the browser from
			// replaying the same failure on every subsequent request, and it means
			// an expired session presents to the SPA as signed out rather than as a
			// login that mysteriously does nothing.
			h.cookie.clear(w)
			h.fail(w, r, err)

			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

// UserFrom returns the authenticated user placed on the context by
// RequireSession. The second result is false on any request that did not pass
// through the middleware.
func UserFrom(ctx context.Context) (*entity.User, bool) {
	user, ok := ctx.Value(userKey{}).(*entity.User)

	return user, ok
}

// mustUser returns the authenticated user for a handler that is only ever
// mounted behind RequireSession.
//
// The error path is not defensive programming for its own sake: it is what
// turns "somebody mounted this route outside the middleware" into a 401 with a
// logged internal error, rather than a nil dereference that takes the process
// down.
func mustUser(r *http.Request) (*entity.User, error) {
	user, ok := UserFrom(r.Context())
	if !ok {
		return nil, entity.ErrSessionNotFound
	}

	return user, nil
}

// userID is a small convenience for the handlers that only need the identifier.
func userID(r *http.Request) (uuid.UUID, error) {
	user, err := mustUser(r)
	if err != nil {
		return uuid.Nil, err
	}

	return user.ID(), nil
}
