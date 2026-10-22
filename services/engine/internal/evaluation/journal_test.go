package evaluation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestJournalFailurePreventsSourceCall(t *testing.T) {
	var calls atomic.Int64
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); SimulatorHandler().ServeHTTP(w, r) }))
	defer source.Close()
	engine, err := New(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: "clear", Strategy: "full_parallel", Journal: func(context.Context, string, Check) error { return errors.New("durable store unavailable") }})
	if err == nil || calls.Load() != 0 || result.SupplierCostMicrorand != 0 || result.Outcome != Indeterminate {
		t.Fatalf("unrecorded work started: calls=%d result=%+v err=%v", calls.Load(), result, err)
	}
}

func TestJournalPairsEveryStartedSource(t *testing.T) {
	engine := testEngine(t)
	var mu sync.Mutex
	events := map[string][]string{}
	result, err := engine.Evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: "clear", Strategy: "full_parallel", Journal: func(_ context.Context, event string, c Check) error {
		mu.Lock()
		defer mu.Unlock()
		events[c.Capability] = append(events[c.Capability], event)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != Verified || len(events) != 3 {
		t.Fatalf("unexpected result %+v events=%v", result, events)
	}
	for capability, actions := range events {
		if len(actions) != 2 || actions[0] != "starting" || actions[1] != "completed" {
			t.Fatalf("journal order %s: %v", capability, actions)
		}
	}
}

func TestCompletionJournalFailureRetainsChargesAndStopsFurtherCalls(t *testing.T) {
	var calls atomic.Int64
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); SimulatorHandler().ServeHTTP(w, r) }))
	defer source.Close()
	engine, err := New(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: "clear", Strategy: "full_sequential", Journal: func(_ context.Context, event string, _ Check) error {
		if event == "completed" {
			return errors.New("completion persistence failed")
		}
		return nil
	}})
	if err == nil || calls.Load() != 1 || result.SupplierCostMicrorand != price(0) || result.Outcome != Indeterminate {
		t.Fatalf("source replay/lost charge: calls=%d result=%+v err=%v", calls.Load(), result, err)
	}
}
