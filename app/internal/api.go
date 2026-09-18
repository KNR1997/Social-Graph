package internal

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/pkg/httpserver"
)

// API is the assembled service: everything main needs to run and stop it.
type API struct {
	Server  *httpserver.Server
	Handler http.Handler
	Pool    *pgxpool.Pool
	Log     *slog.Logger
}

// Close releases the resources the API owns.
func (a *API) Close() {
	if a.Pool != nil {
		a.Pool.Close()
	}
}
