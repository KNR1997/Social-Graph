// Package httpx holds the transport helpers shared by every REST resource
// package. It knows about JSON, status codes, and query strings, and nothing
// about accounts or posts: a depguard rule in .golangci.yml keeps it that way.
// Anything that needs to name a domain type belongs in that resource's package.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// MaxBodyBytes caps request bodies. Without it a client can stream forever into
// the JSON decoder.
const MaxBodyBytes = 1 << 20 // 1 MiB

// Decode reads a JSON body strictly: unknown fields and trailing content are
// errors. DisallowUnknownFields plus the handler's explicit required-field
// checks make malformed input fail for the stated reason.
func Decode[T any](r *http.Request) (T, error) {
	var body T

	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&body); err != nil {
		return body, NewRequestError("malformed JSON body: %s", err)
	}

	// Reject a second JSON document in the same body.
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return body, NewRequestError("body must contain a single JSON object")
	}

	return body, nil
}

// WriteJSON encodes a success response.
func WriteJSON(w http.ResponseWriter, log *slog.Logger, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if body == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The status line is already on the wire; all that is left is to record it.
		log.Error("rest - failed to encode response", slog.Any("error", err))
	}
}
