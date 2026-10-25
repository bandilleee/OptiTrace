// Package portfolio hosts the API, loopback-only synthetic adapters and one
// lease-based benchmark worker in a single free-tier web-service process.
package portfolio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/server"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
)

// ValidateEnvironment runs before opening ports or a database connection. The
// portfolio process never launches cloud jobs, uploads reports, seeds data or
// exposes a simulator/identity test double on its public HTTP listener.
func ValidateEnvironment() error {
	for _, key := range []string{"BENCHMARK_JOB_NAME", "REPORT_BUCKET", "SIMULATOR_AUDIENCE"} {
		if os.Getenv(key) != "" {
			return fmt.Errorf("%s is not supported by the portfolio runtime", key)
		}
	}
	if value := os.Getenv("SIMULATOR_URL"); value != "" {
		u, err := url.Parse(value)
		if err != nil || u.Scheme != "http" || u.User != nil || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
			return errors.New("portfolio synthetic adapters must be loopback-only")
		}
	}
	if os.Getenv("APP_ENV") != "production" {
		return nil
	}
	if os.Getenv("AUTH_MODE") != "supabase" {
		return errors.New("hosted portfolio runtime requires Supabase Google authentication")
	}
	for _, key := range []string{"MOCK_ISSUER", "MOCK_INTERNAL_URL", "MIGRATION_DATABASE_URL", "NEXT_PUBLIC_SUPABASE_SECRET_KEY"} {
		if os.Getenv(key) != "" {
			return fmt.Errorf("%s must not be configured in the hosted portfolio runtime", key)
		}
	}
	origin, err := url.Parse(os.Getenv("APP_ORIGIN"))
	if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || strings.EqualFold(origin.Hostname(), "localhost") || net.ParseIP(origin.Hostname()) != nil || origin.User != nil || origin.Port() != "" || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || strings.ContainsAny(origin.Host, "*\\") {
		return errors.New("APP_ORIGIN must be one exact HTTPS application origin")
	}
	if os.Getenv("API_PUBLIC_URL") != origin.String()+"/api" {
		return errors.New("API_PUBLIC_URL must be APP_ORIGIN plus /api")
	}
	db, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || (db.Scheme != "postgres" && db.Scheme != "postgresql") || db.Hostname() == "" || db.User == nil {
		return errors.New("portfolio requires a PostgreSQL DATABASE_URL")
	}
	ssl := db.Query()["sslmode"]
	if len(ssl) != 1 || ssl[0] != "verify-full" {
		return errors.New("hosted database requires sslmode=verify-full exactly once")
	}
	if db.Port() == "6543" {
		return errors.New("use the Supabase session pooler on port 5432, not transaction pooling")
	}
	if values, ok := db.Query()["pool_max_conns"]; ok {
		n, e := strconv.Atoi(values[0])
		if len(values) != 1 || e != nil || n < 1 || n > 6 {
			return errors.New("portfolio pool_max_conns must occur once and be between 1 and 6")
		}
	}
	if name := strings.Split(db.User.Username(), ".")[0]; name != "optitrace_login" {
		return errors.New("hosted database requires the dedicated optitrace_login role")
	}
	return nil
}

func publicAddress() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", errors.New("PORT must be between 1 and 65535")
	}
	return ":" + port, nil
}

// WorkerLoop polls only while the process is awake. Render may stop it on idle;
// PostgreSQL leases, not this loop's memory, recover jobs on the next wake.
func WorkerLoop(ctx context.Context, poll time.Duration, run func(context.Context) (bool, error)) {
	for ctx.Err() == nil {
		worked, err := run(ctx)
		if err != nil {
			slog.Warn("portfolio benchmark worker operation failed", "error_type", fmt.Sprintf("%T", err))
		}
		if worked && err == nil {
			continue
		}
		timer := time.NewTimer(poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func Run(ctx context.Context) error {
	if err := ValidateEnvironment(); err != nil {
		return err
	}
	address, err := publicAddress()
	if err != nil {
		return err
	}
	if err = os.Setenv("RUNTIME_PROFILE", "portfolio"); err != nil {
		return err
	}
	setup, cancelSetup := context.WithTimeout(ctx, 30*time.Second)
	defer cancelSetup()
	dbURL := os.Getenv("DATABASE_URL")
	// Bound the pool on a small instance without changing other deployment
	// profiles. Connection URLs remain private; parse errors never echo them.
	if u, e := url.Parse(dbURL); e == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		q := u.Query()
		if q.Get("pool_max_conns") == "" {
			q.Set("pool_max_conns", "6")
		}
		n, e := strconv.Atoi(q.Get("pool_max_conns"))
		if e != nil || n < 1 || n > 6 {
			return errors.New("portfolio pool_max_conns must be between 1 and 6")
		}
		u.RawQuery = q.Encode()
		dbURL = u.String()
	}
	db, err := storage.Open(setup, dbURL)
	if err != nil {
		return fmt.Errorf("portfolio database connection failed (%T)", err)
	}
	defer db.Close()
	if err = storage.CheckRuntimeRole(setup, db); err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	simulatorURL := "http://" + listener.Addr().String()
	mux := http.NewServeMux()
	mux.Handle("/", evaluation.SimulatorHandler())
	if os.Getenv("APP_ENV") != "production" && os.Getenv("AUTH_MODE") == "mock" {
		issuer := os.Getenv("MOCK_ISSUER")
		if issuer == "" {
			issuer = simulatorURL
			os.Setenv("MOCK_ISSUER", issuer)
		}
		mock, e := identity.NewMock(issuer)
		if e != nil {
			return e
		}
		for _, path := range []string{"/.well-known/", "/authorize", "/token", "/jwks", "/identity/"} {
			mux.Handle(path, mock.Handler())
		}
		// OAuth discovery/token requests reach this private listener even when a
		// local browser uses the configured localhost mock issuer.
		os.Setenv("MOCK_INTERNAL_URL", simulatorURL)
	}
	simulator := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 5 * time.Second}
	simulatorErrors := make(chan error, 1)
	go func() { simulatorErrors <- simulator.Serve(listener) }()
	defer simulator.Close()
	provider, err := identity.New(setup)
	if err != nil {
		return fmt.Errorf("portfolio identity configuration failed (%T)", err)
	}
	engine, err := evaluation.New(simulatorURL)
	if err != nil {
		return err
	}
	api := &http.Server{Addr: address, Handler: server.New(db, engine, provider).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	defer api.Close()
	workerCtx, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		WorkerLoop(workerCtx, 5*time.Second, func(c context.Context) (bool, error) { return server.RunJob(c, db, engine) })
	}()
	apiErrors := make(chan error, 1)
	go func() { apiErrors <- api.ListenAndServe() }()
	slog.Info("portfolio runtime listening", "address", address, "simulator", "loopback only", "worker_poll_seconds", 5)
	select {
	case <-ctx.Done():
	case err = <-apiErrors:
	case err = <-simulatorErrors:
	}
	stopWorker()
	drain, cancelDrain := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelDrain()
	// Keep loopback adapters alive while admitted API evaluations drain.
	shutdownErr := shutdownAPI(drain, api)
	select {
	case <-workerDone:
	case <-drain.Done():
		shutdownErr = errors.Join(shutdownErr, drain.Err())
	}
	simulator.Close()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return errors.Join(err, shutdownErr)
}

func shutdownAPI(ctx context.Context, api *http.Server) error {
	if err := api.Shutdown(ctx); err != nil {
		// Shutdown alone leaves active requests alive after the grace period.
		// Closing their connections cancels SQL request contexts before pool.Close.
		return errors.Join(err, api.Close())
	}
	return nil
}
