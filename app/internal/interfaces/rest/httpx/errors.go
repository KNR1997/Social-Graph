package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// RequestError is a transport-level rejection: the request was malformed and
// never reached the domain. Its message is written here, so it is safe to show
// to the client verbatim.
//
// Transport validation gets its own type rather than borrowing a domain
// sentinel: "this is not valid JSON" is not a statement about accounts.
type RequestError struct {
	msg string
}

// NewRequestError builds a client-safe rejection from a formatted message.
func NewRequestError(format string, args ...any) error {
	return &RequestError{msg: fmt.Sprintf(format, args...)}
}

func (e *RequestError) Error() string { return e.msg }

// Machine-readable error codes that belong to no single resource. Clients
// branch on these, so they are part of the API contract: add new ones freely,
// never reword an existing one. Resource-specific codes ("account_not_found")
// live in that resource's package.
const (
	CodeInvalidRequest = "invalid_request"
	CodeInternal       = "internal_error"
)

// Status is the wire outcome of an error: a status code, a stable
// machine-readable code, and a client-facing message.
type Status struct {
	HTTP    int
	Code    string
	Message string
}

// Mapper translates one resource's domain sentinels into a Status. It reports
// false for anything it does not own, which is how WriteError falls through to
// a 500 rather than guessing.
//
// Each resource package implements exactly one of these, and it is the only
// place in that package that knows both the domain and HTTP vocabularies.
type Mapper func(error) (Status, bool)

// errorBody is the single error shape the API returns.
type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// WriteError maps an error to a response. The client sees the curated message
// from the mapper; the log sees the whole wrapped chain.
func WriteError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error, m Mapper) {
	st := resolve(err, m)

	attrs := []any{
		slog.Any("error", err),
		slog.String("code", st.Code),
		slog.Int("status", st.HTTP),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	}

	if st.HTTP >= http.StatusInternalServerError {
		log.ErrorContext(r.Context(), "rest - request failed", attrs...)
	} else {
		log.InfoContext(r.Context(), "rest - request rejected", attrs...)
	}

	WriteJSON(w, log, st.HTTP, errorBody{Error: st.Message, Code: st.Code})
}

// resolve applies the fixed precedence: transport rejections first, then the
// resource's own sentinels, then an opaque 500.
//
// The 500 message is written out here instead of being derived from the error
// text. That keeps the wire contract stable when an internal message is
// reworded, and it guarantees that no wrapped internal detail (a table name, or
// a balance the caller may not be entitled to see) escapes. The full error
// still goes to the log.
func resolve(err error, m Mapper) Status {
	var reqErr *RequestError
	if errors.As(err, &reqErr) {
		return Status{HTTP: http.StatusBadRequest, Code: CodeInvalidRequest, Message: reqErr.Error()}
	}

	if m != nil {
		if st, ok := m(err); ok {
			return st
		}
	}

	return Status{HTTP: http.StatusInternalServerError, Code: CodeInternal, Message: "internal server error"}
}
