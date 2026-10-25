package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
)

func TestPublicDemoActualEngine(t *testing.T) {
	sim := httptest.NewServer(evaluation.SimulatorHandler())
	defer sim.Close()
	engine, err := evaluation.New(sim.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		profile, scenario, strategy, outcome string
		calls                                int
	}{
		{"complete", "clear", "optitrace", evaluation.Verified, 3},
		{"conditional", "number_mismatch", "staged", evaluation.Additional, 1},
		{"conditional", "stale", "optitrace", evaluation.Indeterminate, 3},
	} {
		body := `{"profile":"` + tc.profile + `","scenario":"` + tc.scenario + `","strategy":"` + tc.strategy + `"}`
		w := httptest.NewRecorder()
		newPublicDemo(engine).ServeHTTP(w, httptest.NewRequest("POST", "/v1/demo/evaluations", strings.NewReader(body)))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", tc.scenario, w.Code, w.Body.String())
		}
		var out struct {
			evaluation.Result
			Persisted bool `json:"persisted"`
			Demo      bool `json:"public_demo"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		calls, cost := 0, int64(0)
		for _, c := range out.Checks {
			if c.Status != "skipped" {
				calls++
			}
			cost += c.CostMicrorand
		}
		if out.Outcome != tc.outcome || calls != tc.calls || out.Persisted || !out.Demo || cost != out.SupplierCostMicrorand {
			t.Fatalf("%s: invalid demo result %+v calls=%d", tc.scenario, out, calls)
		}
		if strings.Contains(w.Body.String(), "tenant_id") {
			t.Fatal("public response contains tenant identity")
		}
	}
}

func TestPublicDemoRejectsTenantAndUnboundedInput(t *testing.T) {
	d := &publicDemo{active: make(chan struct{}, 2), evaluate: func(context.Context, evaluation.Request) (evaluation.Result, error) {
		t.Fatal("invalid input reached engine")
		return evaluation.Result{}, nil
	}}
	for _, body := range []string{
		`{"profile":"conditional","scenario":"clear","strategy":"staged","tenant_id":"optitrace"}`,
		`{"profile":"complete","scenario":"real-phone","strategy":"staged"}`,
		`{"profile":"custom","scenario":"clear","strategy":"staged"}`,
		`{"profile":"complete","scenario":"clear","strategy":"custom"}`,
		`{"profile":"complete","scenario":"clear","strategy":"staged"} {}`,
		`{"profile":"` + strings.Repeat("x", 2100) + `"}`,
	} {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("POST", "/", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	}
}

func TestPublicDemoLimitsAndCancellation(t *testing.T) {
	d := &publicDemo{window: time.Now(), requests: 30, active: make(chan struct{}, 2)}
	w := httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("rate limit missing")
	}
	d.window = time.Now().Add(-time.Minute)
	d.active <- struct{}{}
	d.active <- struct{}{}
	w = httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	if w.Code != 429 {
		t.Fatal("concurrency limit missing")
	}
	<-d.active
	<-d.active
	d.evaluate = func(ctx context.Context, _ evaluation.Request) (evaluation.Result, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("deadline missing")
		}
		return evaluation.Result{}, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"profile":"complete","scenario":"clear","strategy":"staged"}`)).WithContext(ctx))
	if w.Code != 503 || len(d.active) != 0 {
		t.Fatal("cancelled request retained capacity")
	}
}

func TestPublicDemoOptIn(t *testing.T) {
	t.Setenv("PUBLIC_DEMO_ENABLED", "false")
	w := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/demo/evaluations", nil))
	if w.Code != 404 {
		t.Fatal("public demo must be opt-in")
	}
}

func TestPublicDemoDoesNotAuthorizeTenantRoutes(t *testing.T) {
	t.Setenv("PUBLIC_DEMO_ENABLED", "true")
	h := (&Server{}).Handler()
	for _, path := range []string{"/v1/tenants/optitrace/evaluations", "/v1/tenants/optitrace/policies", "/v1/tenants/optitrace/memberships", "/v1/tenants/optitrace/benchmarks"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 401 {
			t.Fatalf("anonymous tenant access: %s %d", path, w.Code)
		}
	}
}
