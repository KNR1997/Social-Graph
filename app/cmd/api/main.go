// Command api runs the HTTP service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kethaka-creskit/go-ddd-service/config"
	"github.com/kethaka-creskit/go-ddd-service/internal"
)

func main() {
	if err := run(); err != nil {
		// The logger may not exist yet, so fail on stderr and let the exit code
		// speak. This is the only place in the service that calls os.Exit.
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// SIGINT and SIGTERM cancel the root context, which propagates into every
	// in-flight request, query and transaction.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	api, err := internal.InitializeAPI(ctx, cfg)
	if err != nil {
		return err
	}
	defer api.Close()

	api.Server.Start()
	api.Log.Info("service started",
		slog.String("addr", api.Server.Addr()),
		slog.String("env", cfg.App.Env),
	)

	// Whichever happens first: a signal, or the listener falling over.
	var serveErr error
	select {
	case <-ctx.Done():
		api.Log.Info("shutdown signal received")
	case serveErr = <-api.Server.Notify():
		api.Log.Error("http server failed", slog.Any("error", serveErr))
	}

	// Drain with a context that is not already cancelled by the signal.
	if err := api.Server.Shutdown(context.WithoutCancel(ctx)); err != nil {
		api.Log.Error("graceful shutdown failed", slog.Any("error", err))
		serveErr = errors.Join(serveErr, err)
	}

	api.Log.Info("service stopped")

	return serveErr
}
