package main

import (
	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	if os.Getenv("APP_ENV") == "production" && os.Getenv("AUTH_MODE") == "mock" {
		panic("production rejects mock identity provider")
	}
	m := http.NewServeMux()
	m.Handle("/", evaluation.SimulatorHandler())
	if os.Getenv("APP_ENV") != "production" && os.Getenv("AUTH_MODE") == "mock" {
		issuer := os.Getenv("MOCK_ISSUER")
		if issuer == "" {
			issuer = "http://localhost:8090"
		}
		mock, e := identity.NewMock(issuer)
		if e != nil {
			panic(e)
		}
		for _, path := range []string{"/.well-known/", "/authorize", "/token", "/jwks", "/identity/"} {
			m.Handle(path, mock.Handler())
		}
	}
	addr := ":8090"
	if os.Getenv("PORT") != "" {
		addr = ":" + os.Getenv("PORT")
	}
	s := &http.Server{Addr: addr, Handler: m, ReadHeaderTimeout: 5 * time.Second}
	slog.Info("simulator listening", "address", addr)
	if e := s.ListenAndServe(); e != nil {
		panic(e)
	}
}
