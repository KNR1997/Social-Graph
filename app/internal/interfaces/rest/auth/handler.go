// Package auth is the HTTP adapter for the identity aggregate. It translates
// JSON into use-case calls and identity sentinels into status codes, owns the
// session cookie, and contains no business rules.
//
// It imports exactly one domain package. It is also the only REST package that
// exports middleware: RequireSession is what every other resource is mounted
// behind, and it lives here because this is the package that already knows what
// a user is.
package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Service is the port this handler needs, declared here on the consuming side
// rather than exported by the application package. Every type in it belongs to
// the domain, so the interfaces layer never names an application type.
type Service interface {
	Register(ctx context.Context, email, name, password string) (*entity.User, entity.Credential, error)
	Login(ctx context.Context, email, password string) (*entity.User, entity.Credential, error)
	Logout(ctx context.Context, token string) error
	Authenticate(ctx context.Context, token string) (*entity.User, error)
	CurrentUser(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Rename(ctx context.Context, id uuid.UUID, name string) (*entity.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, current, next string) (*entity.User, entity.Credential, error)
}

// Handler serves the authentication endpoints.
type Handler struct {
	service Service
	cookie  CookieConfig
	log     *slog.Logger
}

// NewHandler builds the handler.
func NewHandler(service Service, cookie CookieConfig, log *slog.Logger) *Handler {
	return &Handler{service: service, cookie: cookie, log: log}
}

// Routes mounts the authentication endpoints on their own sub-router.
//
// The split is the point: register and login are the only routes in the whole
// API reachable without a session, so they are the only ones outside the
// RequireSession group. Adding a route to the wrong half is visible here rather
// than buried in a middleware chain.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.register)
	r.Post("/login", h.login)

	// Sign-out is authenticated so that a cross-site request cannot log a user
	// out; it stays idempotent for a token the server has already forgotten.
	r.Group(func(r chi.Router) {
		r.Use(h.RequireSession)

		r.Post("/logout", h.logout)
		r.Get("/me", h.me)
		r.Patch("/me", h.rename)
		r.Post("/me/password", h.changePassword)
	})

	return r
}

// fail writes an error response, binding this package's sentinel mapping to the
// shared writer so every call site stays one line.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteError(w, r, h.log, err, statusFor)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.Decode[registerRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	user, cred, err := h.service.Register(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	h.cookie.set(w, cred.Token, cred.Session.ExpiresAt())
	httpx.WriteJSON(w, h.log, http.StatusCreated, toSessionResponse(user, cred))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.Decode[loginRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	user, cred, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	h.cookie.set(w, cred.Token, cred.Session.ExpiresAt())
	httpx.WriteJSON(w, h.log, http.StatusOK, toSessionResponse(user, cred))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r.Context(), h.cookie.token(r)); err != nil {
		h.fail(w, r, err)

		return
	}

	// Clear the cookie whatever the server-side state was: the caller asked to
	// be signed out, and the browser is the half of that they can see.
	h.cookie.clear(w)
	httpx.WriteJSON(w, h.log, http.StatusNoContent, nil)
}

// me returns the signed-in user. It re-reads through the service rather than
// serialising the copy the middleware attached, so a profile edit made in
// another tab shows up here.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	id, err := userID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	user, err := h.service.CurrentUser(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(user))
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	id, err := userID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	req, err := httpx.Decode[renameRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	user, err := h.service.Rename(r.Context(), id, req.Name)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(user))
}

// changePassword rotates the password and re-issues the caller's cookie.
//
// The service has already destroyed every session for this user, including the
// one that made this request. Setting the new cookie here is what keeps the
// browser doing the change signed in while everything else is evicted.
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	id, err := userID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	req, err := httpx.Decode[changePasswordRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	user, cred, err := h.service.ChangePassword(r.Context(), id, req.CurrentPassword, req.NewPassword)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	h.cookie.set(w, cred.Token, cred.Session.ExpiresAt())
	httpx.WriteJSON(w, h.log, http.StatusOK, toSessionResponse(user, cred))
}
