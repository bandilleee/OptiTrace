package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"os"
)

func main() {
	seed := flag.Bool("seed", false, "Insert synthetic demonstration fixtures")
	flag.Parse()
	ctx := context.Background()
	db, e := storage.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		panic(e)
	}
	defer db.Close()
	if e = storage.Migrate(ctx, db); e != nil {
		panic(e)
	}
	user := os.Getenv("APP_DATABASE_USER")
	if user == "" {
		user = "optitrace_login"
	}
	var exists bool
	if e = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)", user).Scan(&exists); e != nil {
		panic(e)
	}
	if exists {
		if e = storage.HardenLogin(ctx, db, user); e != nil {
			panic(e)
		}
	} else {
		panic("application login must be provisioned separately before migrations")
	}
	if *seed {
		if os.Getenv("APP_ENV") == "production" {
			panic("refusing synthetic seed in production")
		}
		if e = storage.Seed(ctx, db); e != nil {
			panic(e)
		}
	}
	fmt.Println("Migrations complete")
}
