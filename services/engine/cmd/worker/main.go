package main

import (
	"context"
	"flag"

	"github.com/optitrace/optitrace/services/engine/internal/server"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	once := flag.Bool("once", false, "Drain queue and exit")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, e := storage.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		panic(e)
	}
	defer db.Close()
	if e = storage.CheckRuntimeRole(ctx, db); e != nil {
		panic(e)
	}
	engine, e := server.NewEngine(ctx)
	if e != nil {
		panic(e)
	}
	for ctx.Err() == nil {
		worked, e := server.RunJob(ctx, db, engine)
		if e != nil {
			slog.Error("worker", "error", e)
			if *once {
				os.Exit(1)
			}
		}
		if !worked {
			if *once {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}
