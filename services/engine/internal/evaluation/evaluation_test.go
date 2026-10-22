package evaluation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	s := httptest.NewServer(SimulatorHandler())
	t.Cleanup(s.Close)
	e, err := New(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestExhaustivePolicyAndStopping(t *testing.T) {
	e := testEngine(t)
	for _, p := range []string{"complete", "conditional"} {
		for a := 0; a < 4; a++ {
			for b := 0; b < 4; b++ {
				for c := 0; c < 4; c++ {
					s := []int{a, b, c}
					partialOutcome, partialErr := opaDecision(context.Background(), e.queries[p], s)
					if partialErr != nil || partialOutcome != reference(p, s) {
						t.Fatalf("OPA partial-state mismatch %s %v: %s %v", p, s, partialOutcome, partialErr)
					}
					got, stop := stoppingOracle(p, s)
					pending := false
					for i, v := range s {
						if compulsory(p, i) && v == 3 {
							pending = true
						}
					}
					possible := map[string]bool{}
					for x := 0; x < 3; x++ {
						for y := 0; y < 3; y++ {
							for z := 0; z < 3; z++ {
								v := []int{x, y, z}
								for i := range s {
									if s[i] != 3 {
										v[i] = s[i]
									}
								}
								possible[reference(p, v)] = true
							}
						}
					}
					if stop != (len(possible) == 1 && !pending) || stop && !possible[got] {
						t.Fatalf("unsafe proof %s %v %s %v", p, s, got, possible)
					}
					v := append([]int(nil), s...)
					for i, x := range v {
						if x == 3 {
							v[i] = 2
						}
					}
					out, err := opaDecision(context.Background(), e.queries[p], v)
					if err != nil || out != reference(p, v) {
						t.Fatalf("OPA %s %v %s %v", p, v, out, err)
					}
				}
			}
		}
	}
}
func TestPlans(t *testing.T) {
	ps := candidates()
	if len(ps) != 13 {
		t.Fatal(len(ps))
	}
	seen := map[string]bool{}
	for _, p := range ps {
		if seen[p.Name] {
			t.Fatal("duplicate plan")
		}
		seen[p.Name] = true
		mask := 0
		for _, batch := range p.Batches {
			for _, i := range batch {
				if mask&(1<<i) != 0 {
					t.Fatal("duplicate fetch")
				}
				mask |= 1 << i
			}
		}
		if mask != 7 {
			t.Fatal("missing fetch")
		}
	}
	for _, profile := range []string{"complete", "conditional"} {
		p := choosePlan(profile)
		t.Logf("synthetic tuning plan profile=%s plan=%s estimated_cost_microrand=%.2f estimated_tail_ms=%.2f", profile, p.Name, p.ExpectedCost, p.EstimatedP95MS)
	}
}
func TestAllStrategiesScenarios(t *testing.T) {
	e := testEngine(t)
	for _, p := range []string{"complete", "conditional"} {
		for _, scenario := range Scenarios {
			want := ""
			for _, strategy := range Strategies {
				r, err := e.evaluate(context.Background(), Request{Policy: Policy{Profile: p}, Scenario: scenario, Strategy: strategy}, time.Now().UTC(), true)
				if err != nil {
					t.Fatal(err)
				}
				if want == "" {
					want = r.Outcome
				}
				if r.Outcome != want {
					t.Fatalf("%s %s %s %s != %s", p, scenario, strategy, r.Outcome, want)
				}
				for _, c := range r.Checks {
					if c.Compulsory && c.Status == "skipped" {
						t.Fatal("compulsory skipped")
					}
					if c.Status != "skipped" && c.CostMicrorand == 0 {
						t.Fatal("free attempted call")
					}
				}
			}
		}
	}
}
func TestFreshnessScopeEvents(t *testing.T) {
	at := time.Now().UTC()
	for _, tc := range []struct {
		name                   string
		age, eventAge          time.Duration
		scope, subject, status string
		adverse                bool
	}{{"inclusive", 60 * time.Second, 24 * time.Hour, "ZA:account-verification", "synthetic-subject", "available", true}, {"stale", 60*time.Second + 1, time.Hour, "ZA:account-verification", "synthetic-subject", "stale", false}, {"old", time.Second, 24*time.Hour + 1, "ZA:account-verification", "synthetic-subject", "available", false}, {"future source", -1, time.Hour, "ZA:account-verification", "synthetic-subject", "unavailable", false}, {"future event", time.Second, -1, "ZA:account-verification", "synthetic-subject", "unavailable", false}, {"scope", time.Second, time.Hour, "KE:other", "synthetic-subject", "unavailable", false}, {"subject", time.Second, time.Hour, "ZA:account-verification", "other", "unavailable", false}} {
		t.Run(tc.name, func(t *testing.T) {
			event := at.Add(-tc.eventAge)
			c := Check{Status: "available", Value: ptr(true), SourceTime: at.Add(-tc.age), EventTime: &event, Scope: tc.scope, Subject: tc.subject}
			normalize(&c, 1, at)
			if c.Status != tc.status {
				t.Fatal(c.Status)
			}
			if c.Status == "available" && *c.Value != tc.adverse {
				t.Fatal("event boundary")
			}
			if c.Status != "available" && c.Value != nil {
				t.Fatal("unknown value retained")
			}
		})
	}
}
func TestExplicitFailureSemantics(t *testing.T) {
	for _, tc := range []struct {
		p    string
		s    []int
		want string
	}{{"complete", []int{1, 2, 0}, Indeterminate}, {"conditional", []int{1, 2, 0}, Additional}, {"conditional", []int{2, 1, 0}, Indeterminate}, {"conditional", []int{0, 2, 0}, Indeterminate}, {"conditional", []int{0, 2, 1}, Additional}} {
		if reference(tc.p, tc.s) != tc.want {
			t.Fatal(tc)
		}
	}
}
func TestCancellationChargesStarted(t *testing.T) {
	e := testEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	r, err := e.Evaluate(ctx, Request{Policy: Policy{Profile: "complete"}, Scenario: "clear", Strategy: "full_parallel"})
	if err != nil {
		t.Fatal(err)
	}
	if r.SupplierCostMicrorand != 6600 || r.Outcome != Indeterminate {
		t.Fatalf("%+v", r)
	}
	for _, c := range r.Checks {
		if c.Status == "skipped" {
			t.Fatal("no compulsory failure path")
		}
	}
}
func TestDistinctHTTPFaults(t *testing.T) {
	e := testEngine(t)
	for _, tc := range []struct{ s, status string }{{"stale", "stale"}, {"timeout", "timeout"}, {"rate_limit", "rate_limited"}, {"permission_denied", "permission_denied"}, {"source_failure", "unavailable"}} {
		r, err := e.Evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: tc.s, Strategy: "full_parallel"})
		if err != nil || r.Checks[1].Status != tc.status || r.Outcome != Indeterminate || r.SupplierCostMicrorand != 6600 {
			t.Fatalf("%s %+v %v", tc.s, r, err)
		}
	}
}
func TestNoCrossRequestReuse(t *testing.T) {
	e := testEngine(t)
	for i := 0; i < 2; i++ {
		r, err := e.Evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: "clear"})
		if err != nil || r.SupplierCostMicrorand != 6600 {
			t.Fatalf("%+v %v", r, err)
		}
	}
}

type authTransport struct{ base http.RoundTripper }

func (a authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	copy.Header.Set("Authorization", "Bearer test-id-token")
	return a.base.RoundTrip(copy)
}
func TestInjectedIdentityTransport(t *testing.T) {
	underlying := SimulatorHandler()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-id-token" {
			w.WriteHeader(401)
			return
		}
		underlying.ServeHTTP(w, r)
	}))
	defer s.Close()
	e, err := NewWithHTTPClient(s.URL, &http.Client{Transport: authTransport{http.DefaultTransport}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.Evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: "clear"})
	if err != nil || r.Outcome != Verified {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestInvalidEvidenceNeverBecomesClear(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Check{Status: "available", Value: ptr(false), Scope: "wrong", Subject: "another-subject", SourceTime: time.Now().Add(-time.Second)})
	}))
	defer s.Close()
	e, err := New(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.Evaluate(context.Background(), Request{Policy: Policy{Profile: "conditional"}, Scenario: "clear"})
	if err != nil || r.Outcome != Indeterminate {
		t.Fatalf("%+v %v", r, err)
	}
	if r.Checks[0].Status != "unavailable" {
		t.Fatal("compulsory malformed evidence accepted")
	}
	for _, c := range r.Checks {
		if (c.Status != "unavailable" && c.Status != "skipped") || c.Value != nil {
			t.Fatalf("%+v", c)
		}
	}
}

func TestAdversarialJointDistributionsAgainstAllStrategies(t *testing.T) {
	// Source outcomes intentionally differ from the tuning distribution. The
	// execution planner may be economically wrong, but must remain correct.
	row := []int{0, 0, 0}
	var rowMu sync.RWMutex
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rowMu.RLock()
		defer rowMu.RUnlock()
		i := 0
		for j, name := range Capabilities {
			if strings.HasSuffix(r.URL.Path, name) {
				i = j
			}
		}
		at, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("at"))
		if err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		c := sourceCheck(i, "clear", at)
		if row[i] == 2 {
			c.Status = "unavailable"
			c.Value = nil
		} else {
			c.Value = ptr(row[i] == 1)
			if i > 0 && row[i] == 1 {
				event := at.Add(-time.Hour)
				c.EventTime = &event
			}
		}
		_ = json.NewEncoder(w).Encode(c)
	}))
	defer s.Close()
	e, err := New(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"complete", "conditional"} {
		for a := 0; a < 3; a++ {
			for b := 0; b < 3; b++ {
				for c := 0; c < 3; c++ {
					rowMu.Lock()
					row = []int{a, b, c}
					rowMu.Unlock()
					for _, strategy := range Strategies {
						r, err := e.Evaluate(context.Background(), Request{Policy: Policy{Profile: profile}, Scenario: "clear", Strategy: strategy})
						if err != nil || r.Outcome != reference(profile, row) {
							t.Fatalf("%s %v %s %+v %v", profile, row, strategy, r, err)
						}
					}
				}
			}
		}
	}
}
func TestBenchmarkReproducibilityBoundsAndLabels(t *testing.T) {
	if !reflect.DeepEqual(heldout(42, 30), heldout(42, 30)) || reflect.DeepEqual(heldout(42, 30), heldout(43, 30)) {
		t.Fatal("seed")
	}
	e := testEngine(t)
	for _, n := range []int{0, 1, 1001} {
		if _, err := e.Benchmark(context.Background(), 1, n); err == nil {
			t.Fatal("unbounded")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Benchmark(ctx, 1, 2); err == nil {
		t.Fatal("cancellation")
	}
	r, err := e.Benchmark(context.Background(), 42, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Results) != 20 || r.Manifest.Data == "" {
		t.Fatal("incomplete report")
	}
	for _, m := range r.Results {
		if m.AgreementRate != 1 || m.CompulsoryViolations != 0 {
			t.Fatalf("%+v", m)
		}
	}
	for _, c := range r.Comparisons {
		t.Logf("synthetic held-out comparison: %+v", c)
		if c.Profile == "complete" && c.ResearchTargetPassed {
			t.Fatal("invented complete-profile savings")
		}
	}
}
func FuzzStoppingProof(f *testing.F) {
	f.Add(uint8(0), uint8(1), uint8(2), false)
	f.Add(uint8(3), uint8(3), uint8(3), true)
	f.Fuzz(func(t *testing.T, a, b, c uint8, complete bool) {
		p := "conditional"
		if complete {
			p = "complete"
		}
		s := []int{int(a % 4), int(b % 4), int(c % 4)}
		out, ok := stoppingOracle(p, s)
		if ok {
			for i, v := range s {
				if compulsory(p, i) && v == 3 {
					t.Fatal("unsafe compulsory")
				}
				if v == 3 {
					s[i] = int(a % 3)
				}
			}
			if reference(p, s) != out {
				t.Fatal("unsafe stop")
			}
		}
	})
}
