// Package httpserver wraps net/http with timeouts and graceful shutdown. It is
// router-agnostic: it takes an http.Handler, so swapping chi for anything else
// touches only the interfaces layer.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kethaka-creskit/go-ddd-service/config"
)

// Server owns the listener lifecycle.
type Server struct {
	server          *http.Server
	notify          chan error
	shutdownTimeout time.Duration
}

// New builds the server. It does not start listening: call Start, so that main
// stays in control of when the process starts taking traffic.
func New(cfg *config.Config, handler http.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:              net.JoinHostPort("", cfg.HTTP.Port),
			Handler:           handler,
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			ReadHeaderTimeout: cfg.HTTP.ReadTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
		},
		notify:          make(chan error, 1),
		shutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}
}

// Addr reports the configured listen address.
func (s *Server) Addr() string { return s.server.Addr }

// Start begins serving in the background.
func (s *Server) Start() {
	go func() {
		defer close(s.notify)

		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.notify <- fmt.Errorf("httpserver - ListenAndServe: %w", err)
		}
	}()
}

// Notify reports a listener failure. It is closed on clean shutdown.
func (s *Server) Notify() <-chan error { return s.notify }

// Shutdown drains in-flight requests, up to the configured timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("httpserver - Shutdown: %w", err)
	}

	return nil
}
