package httpx

import (
	"net/http"
	"strconv"
)

// Pagination bounds for list endpoints.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Page is the transport-level view of a list window. Each aggregate declares
// its own entity.Page, so the shared parser stays in the REST vocabulary and
// each resource package converts to the type its service expects.
type Page struct {
	Limit  int32
	Offset int32
}

// ParsePage reads limit and offset, clamping rather than erroring so a sloppy
// client still gets a usable page.
func ParsePage(r *http.Request) Page {
	q := r.URL.Query()

	limit := int32(DefaultLimit)
	if v, err := strconv.ParseInt(q.Get("limit"), 10, 32); err == nil && v > 0 {
		limit = int32(min(v, MaxLimit))
	}

	var offset int32
	if v, err := strconv.ParseInt(q.Get("offset"), 10, 32); err == nil && v > 0 {
		offset = int32(v)
	}

	return Page{Limit: limit, Offset: offset}
}
