package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/optitrace/optitrace/services/engine/internal/portfolio"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := portfolio.Run(ctx); err != nil {
		slog.Error("portfolio runtime stopped", "error", err)
		os.Exit(1)
	}
}
