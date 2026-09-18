// Package rest is the inbound HTTP adapter. It owns the router, the middleware
// stack, and the operational endpoints, and nothing else: each resource lives
// in its own subpackage (rest/account, rest/post) importing exactly one domain
// package, and the transport helpers they share live in rest/httpx.
package rest

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/kethaka-creskit/go-ddd-service/config"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/account"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/auth"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/post"
	"github.com/kethaka-creskit/go-ddd-service/pkg/logger"
)

// readinessTimeout bounds the database check behind /readyz.
const readinessTimeout = 2 * time.Second

// Pinger is the readiness dependency, declared here so the interfaces layer
// never imports a database driver. The pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewRouter assembles the HTTP surface. It returns http.Handler rather than
// chi.Router so that pkg/httpserver never learns which router is in use.
func NewRouter(
	cfg *config.Config,
	sessions *auth.Handler,
	accounts *account.Handler,
	posts *post.Handler,
	db Pinger,
	log *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	// RealIP is deliberately not used: it rewrites RemoteAddr from client-
	// controlled headers, which is spoofable unless a trusted proxy is known to
	// overwrite them (see GHSA-3fxj-6jh8-hvhx). Resolve the real client IP in the
	// edge layer instead.
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.HTTP.HandlerTimeout))

	// Operational endpoints, deliberately outside /v1 and unversioned.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/readyz", readiness(db))

	r.Route("/v1", func(r chi.Router) {
		// Sign-in and sign-up have to be reachable without a session, so the auth
		// routes carry their own guard internally rather than sitting behind one
		// here. Everything else is authenticated.
		r.Mount("/auth", sessions.Routes())

		r.Group(func(r chi.Router) {
			r.Use(sessions.RequireSession)

			r.Mount("/accounts", accounts.Routes())
			r.Mount("/posts", posts.Routes())
		})
	})

	return r
}

// readiness reports whether the service can reach its database. Liveness
// (/healthz) must not depend on the database, or a brief database blip gets the
// pods killed instead of just drained.
func readiness(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// requestLogger logs one structured line per request and seeds the context with
// chi's request id, so every log line emitted downstream carries it too.
func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ctx := logger.WithRequestID(r.Context(), middleware.GetReqID(r.Context()))
			r = r.WithContext(ctx)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			log.InfoContext(ctx, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
