package server

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func Main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	srv, err := initServer()
	if err != nil {
		slog.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	cancelFn, errch := srv.Run(ctx)
	defer cancelFn()

	select {
	case err := <-errch:
		if err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		if err := <-errch; err != nil {
			slog.Error("server error during shutdown", "error", err)
		}
	}
}
