package person

import (
	"errors"
	"net/http"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/person/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Machine-readable error codes specific to people. Clients branch on these, so
// they are part of the API contract: add new ones freely, never reword an
// existing one. The resource-independent codes live in httpx.
const (
	codeNotFound  = "person_not_found"
	codeConflict  = "concurrent_modification"
	codeNoSession = "no_session"
)

// statusFor maps a person sentinel to a status, a stable machine-readable code,
// and a client-facing message. It reports false for anything that is not a
// person error, leaving httpx to fall through to a 500.
//
// This function is the only place in this package that knows both the domain
// vocabulary and the HTTP one.
func statusFor(err error) (httpx.Status, bool) {
	// FieldTooLongError is matched first and by type, because its message names
	// the offending field and its bound. That message is written entirely in the
	// domain, so it is safe to return verbatim; nothing from the wrapped error
	// chain reaches the client.
	var tooLong *entity.FieldTooLongError
	if errors.As(err, &tooLong) {
		return badRequest(tooLong.Error()), true
	}

	switch {
	case errors.Is(err, entity.ErrNotFound):
		return status(http.StatusNotFound, codeNotFound, "person not found"), true

	case errors.Is(err, entity.ErrConflict):
		return status(http.StatusConflict, codeConflict,
			"the person was modified concurrently, retry the request"), true

	case errors.Is(err, entity.ErrFullNameRequired):
		return badRequest("full_name must not be empty"), true

	case errors.Is(err, entity.ErrInvalidEmail):
		return badRequest("email must be a valid address"), true

	// An owner that never arrived means the route was mounted outside
	// RequireSession. That is a wiring mistake rather than a client mistake, so
	// it is logged in full and reported as an unauthenticated request instead of
	// dereferencing a user that is not there.
	case errors.Is(err, entity.ErrOwnerRequired), errors.Is(err, errNoOwner):
		return status(http.StatusUnauthorized, codeNoSession, "authentication required"), true

	default:
		return httpx.Status{}, false
	}
}

func status(code int, slug, message string) httpx.Status {
	return httpx.Status{HTTP: code, Code: slug, Message: message}
}

func badRequest(message string) httpx.Status {
	return status(http.StatusBadRequest, httpx.CodeInvalidRequest, message)
}
