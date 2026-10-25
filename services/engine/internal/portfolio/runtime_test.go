package portfolio

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestShutdownCancelsRequestsAfterGrace(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, cancelled := make(chan struct{}), make(chan struct{})
	api := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(cancelled) })}
	defer api.Close()
	go api.Serve(listener)
	go func() {
		r, e := http.Get("http://" + listener.Addr().String())
		if e == nil {
			r.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err = shutdownAPI(ctx, api); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected expired grace: %v", err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("active request context survived forced close")
	}
}

func TestHostedPortfolioRejectsUnsafeConfiguration(t *testing.T) {
	set := func() {
		for _, key := range []string{"SIMULATOR_URL", "SIMULATOR_AUDIENCE", "BENCHMARK_JOB_NAME", "REPORT_BUCKET", "MOCK_ISSUER", "MOCK_INTERNAL_URL", "MIGRATION_DATABASE_URL", "NEXT_PUBLIC_SUPABASE_SECRET_KEY"} {
			t.Setenv(key, "")
		}
		t.Setenv("APP_ENV", "production")
		t.Setenv("AUTH_MODE", "supabase")
		t.Setenv("APP_ORIGIN", "https://optitrace.example.com")
		t.Setenv("API_PUBLIC_URL", "https://optitrace.example.com/api")
		t.Setenv("DATABASE_URL", "postgres://optitrace_login.ref:private-test-only@pooler.example.com:5432/postgres?sslmode=verify-full")
	}
	set()
	if err := ValidateEnvironment(); err != nil {
		t.Fatal(err)
	}
	for _, sample := range [][2]string{{"APP_ORIGIN", "https://127.0.0.1"}, {"MOCK_ISSUER", "http://localhost:8090"}, {"MIGRATION_DATABASE_URL", "must-be-private"}, {"DATABASE_URL", "postgres://optitrace_login.ref:secret@pooler.example.com:5432/postgres?sslmode=verify-full&pool_max_conns=4&pool_max_conns=6"}, {"DATABASE_URL", "postgres://optitrace_login.ref:secret@pooler.example.com:5432/postgres?sslmode=verify-full&pool_max_conns=7"}} {
		set()
		t.Setenv(sample[0], sample[1])
		if err := ValidateEnvironment(); err == nil {
			t.Errorf("accepted unsafe %s", sample[0])
		}
	}
	for _, sample := range [][2]string{{"AUTH_MODE", "mock"}, {"AUTH_MODE", "google"}, {"APP_ORIGIN", "https://*.example.com"}, {"APP_ORIGIN", "https://optitrace.example.com?x=1"}, {"API_PUBLIC_URL", "https://api.example.com"}, {"SIMULATOR_URL", "https://untrusted.example.com"}, {"SIMULATOR_URL", "http://127.0.0.1?x=1"}, {"REPORT_BUCKET", "paid-bucket"}, {"BENCHMARK_JOB_NAME", "projects/other/jobs/worker"}, {"DATABASE_URL", "postgres://postgres:secret@pooler.example.com/postgres?sslmode=verify-full"}, {"DATABASE_URL", "postgres://optitrace_login.ref:secret@pooler.example.com:6543/postgres?sslmode=verify-full"}, {"DATABASE_URL", "postgres://optitrace_login.ref:secret@pooler.example.com/postgres?sslmode=disable"}} {
		set()
		t.Setenv(sample[0], sample[1])
		if err := ValidateEnvironment(); err == nil {
			t.Errorf("accepted unsafe %s", sample[0])
		}
	}
}

func TestPortfolioWorkerRecoversAfterIdleAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		WorkerLoop(ctx, time.Millisecond, func(context.Context) (bool, error) {
			n := calls.Add(1)
			if n == 1 {
				return false, errors.New("temporary database outage")
			}
			if n == 2 {
				return false, nil
			}
			if n == 4 {
				cancel()
			}
			return true, nil
		})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker failed to recover or stop")
	}
	if calls.Load() != 4 {
		t.Fatalf("unexpected worker calls %d", calls.Load())
	}
}
