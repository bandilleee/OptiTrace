package main

import (
	"context"

	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/server"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdown, e := server.ConfigureTracing(ctx)
	if e != nil {
		panic(e)
	}
	defer shutdown(context.Background())
	db, e := storage.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		slog.Error("database", "error", e)
		os.Exit(1)
	}
	defer db.Close()
	if e = storage.CheckRuntimeRole(ctx, db); e != nil {
		panic(e)
	}
	provider, e := identity.New(ctx)
	if e != nil {
		slog.Error("identity configuration", "error", e)
		os.Exit(1)
	}
	engine, e := server.NewEngine(ctx)
	if e != nil {
		slog.Error("engine", "error", e)
		os.Exit(1)
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":" + os.Getenv("PORT")
		if addr == ":" {
			addr = ":8080"
		}
	}
	h := &http.Server{Addr: addr, Handler: server.New(db, engine, provider).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		h.Shutdown(c)
	}()
	slog.Info("api listening", "address", addr)
	if e = h.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		slog.Error("serve", "error", e)
		os.Exit(1)
	}
}
