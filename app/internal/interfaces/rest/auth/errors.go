package auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Machine-readable error codes specific to identity. Clients branch on these,
// so they are part of the API contract: add new ones freely, never reword an
// existing one. The resource-independent codes live in httpx.
const (
	codeUnauthenticated  = "unauthenticated"
	codeInvalidLogin     = "invalid_credentials"
	codeEmailTaken       = "email_taken"
	codeWeakPassword     = "weak_password"
	codeUserNotFound     = "user_not_found"
	codeConflict         = "concurrent_modification"
	codeInvalidEmailAddr = "invalid_email"
)

// statusFor maps an identity sentinel to a status, a stable machine-readable
// code, and a client-facing message. It reports false for anything that is not
// an identity error, leaving httpx to fall through to a 500.
//
// Two decisions here are security decisions rather than style ones:
//
//   - ErrInvalidCredentials is 401 with one flat message. The service has
//     already collapsed "no such user" and "wrong password" into this sentinel;
//     re-separating them here would undo that and hand back the account
//     enumeration oracle it exists to prevent.
//   - A missing session and an expired one are both 401 with the same code. The
//     SPA's only reasonable response to either is to show the sign-in page, and
//     telling an unauthenticated caller that a token *used* to be valid tells
//     them something they should not learn from a 401.
//
// This function is the only place in this package that knows both the domain
// vocabulary and the HTTP one.
func statusFor(err error) (httpx.Status, bool) {
	switch {
	case errors.Is(err, entity.ErrSessionNotFound), errors.Is(err, entity.ErrSessionExpired):
		return httpx.Status{
			HTTP:    http.StatusUnauthorized,
			Code:    codeUnauthenticated,
			Message: "sign in to continue",
		}, true

	case errors.Is(err, entity.ErrInvalidCredentials):
		return httpx.Status{
			HTTP:    http.StatusUnauthorized,
			Code:    codeInvalidLogin,
			Message: "email or password is incorrect",
		}, true

	case errors.Is(err, entity.ErrEmailTaken):
		return httpx.Status{
			HTTP:    http.StatusConflict,
			Code:    codeEmailTaken,
			Message: "that email address is already registered",
		}, true

	case errors.Is(err, entity.ErrWeakPassword):
		return httpx.Status{
			HTTP: http.StatusUnprocessableEntity,
			Code: codeWeakPassword,
			// Derived from the domain constants rather than typed out, so the
			// policy and the message it produces cannot drift apart.
			Message: fmt.Sprintf("password must be between %d and %d characters",
				entity.MinPasswordLen, entity.MaxPasswordLen),
		}, true

	case errors.Is(err, entity.ErrInvalidEmail):
		return httpx.Status{
			HTTP:    http.StatusBadRequest,
			Code:    codeInvalidEmailAddr,
			Message: "that is not a valid email address",
		}, true

	case errors.Is(err, entity.ErrEmptyName):
		return httpx.Status{
			HTTP:    http.StatusBadRequest,
			Code:    httpx.CodeInvalidRequest,
			Message: "name must not be empty",
		}, true

	case errors.Is(err, entity.ErrUserNotFound):
		return httpx.Status{
			HTTP:    http.StatusNotFound,
			Code:    codeUserNotFound,
			Message: "user not found",
		}, true

	case errors.Is(err, entity.ErrConflict):
		return httpx.Status{
			HTTP:    http.StatusConflict,
			Code:    codeConflict,
			Message: "the account was modified concurrently, retry the request",
		}, true

	default:
		return httpx.Status{}, false
	}
}
