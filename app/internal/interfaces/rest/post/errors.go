package post

import (
	"errors"
	"net/http"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Machine-readable error codes specific to posts. Clients branch on these, so
// they are part of the API contract: add new ones freely, never reword an
// existing one. The resource-independent codes live in httpx.
const (
	codeNotFound = "post_not_found"
	codeConflict = "concurrent_modification"
)

// statusFor maps a post sentinel to a status, a stable machine-readable code,
// and a client-facing message. It reports false for anything that is not a post
// error, leaving httpx to fall through to a 500.
//
// The message is written out per case rather than taken from the sentinel's
// text: the sentinels themselves are internal, and their wording is free to
// change without moving the wire contract.
//
// This function is the only place in this package that knows both the domain
// vocabulary and the HTTP one.
func statusFor(err error) (httpx.Status, bool) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		return httpx.Status{
			HTTP:    http.StatusNotFound,
			Code:    codeNotFound,
			Message: "post not found",
		}, true

	case errors.Is(err, entity.ErrConflict):
		return httpx.Status{
			HTTP:    http.StatusConflict,
			Code:    codeConflict,
			Message: "the post was modified concurrently, retry the request",
		}, true

	case errors.Is(err, entity.ErrEmptyName):
		return httpx.Status{
			HTTP:    http.StatusBadRequest,
			Code:    httpx.CodeInvalidRequest,
			Message: "name must not be empty",
		}, true

	default:
		return httpx.Status{}, false
	}
}
