package evaluation

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown"
)

// Run this same regression in a CPU-quota container to exercise scheduler
// pauses without changing policy complexity, observations or timeout settings.
func TestPolicyFinalizationUnderCPUQuota(t *testing.T) {
	e, err := New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		profile := []string{"complete", "conditional"}[i%2]
		strategy := []string{"opa", "optitrace"}[(i/2)%2]
		result, err := e.evaluate(context.Background(), Request{Policy: Policy{Profile: profile}, Scenario: "clear", Strategy: strategy}, time.Now().UTC(), true)
		if err != nil || result.Outcome != Verified || result.SupplierCostMicrorand != 6600 {
			t.Fatalf("sample %d profile=%s strategy=%s result=%+v error=%v", i, profile, strategy, result, err)
		}
		for _, check := range result.Checks {
			if check.Compulsory && check.Status != "available" {
				t.Fatalf("compulsory check lost: %+v", check)
			}
		}
	}
}

func TestPolicyInputAndBudgetBoundaries(t *testing.T) {
	e := testEngine(t)
	for _, input := range [][]int{nil, {}, {0}, {0, 0}, {0, 0, 0, 0}, {-1, 0, 0}, {0, 4, 0}} {
		if got, err := opaDecision(context.Background(), e.queries["complete"], input); err == nil || got != "" {
			t.Fatalf("out-of-domain input accepted: %v got=%s error=%v", input, got, err)
		}
	}
	for _, limit := range []int{-1, 0, 1} {
		if got, err := opaDecisionWithBudget(context.Background(), e.queries["complete"], []int{0, 0, 0}, limit); !errors.Is(err, errPolicyBudget) || got != "" {
			t.Fatalf("budget %d: got=%s error=%v", limit, got, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := opaDecision(ctx, e.queries["complete"], []int{0, 0, 0}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled boundary: %v", err)
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := opaDecision(ctx, e.queries["complete"], []int{0, 0, 0}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired boundary: %v", err)
	}
}

func TestExhaustivePolicyEventBudgetHeadroom(t *testing.T) {
	e := testEngine(t)
	maxEvents := 0
	for _, profile := range []string{"complete", "conditional"} {
		for a := 0; a < 4; a++ {
			for b := 0; b < 4; b++ {
				for c := 0; c < 4; c++ {
					input := []int{a, b, c}
					budget := &policyBudget{limit: policyEventBudget, cancel: topdown.NewCancel()}
					_, err := e.queries[profile].Eval(context.Background(), rego.EvalInput(map[string]any{"states": input}), rego.EvalQueryTracer(budget), rego.EvalExternalCancel(budget.cancel))
					if err != nil || budget.exhausted {
						t.Fatalf("budget exceeded for %s %v: events=%d error=%v", profile, input, budget.events, err)
					}
					if budget.events > maxEvents {
						maxEvents = budget.events
					}
				}
			}
		}
	}
	if maxEvents == 0 || maxEvents > policyEventBudget/4 {
		t.Fatalf("insufficient observed budget headroom: max=%d budget=%d", maxEvents, policyEventBudget)
	}
	t.Logf("all 128 partial/terminal profile states: max_events=%d budget=%d headroom=%.1fx", maxEvents, policyEventBudget, float64(policyEventBudget)/float64(maxEvents))
}

func TestPolicyBudgetFailureRetainsChargedEvidence(t *testing.T) {
	e := testEngine(t)
	// Private test substitution only: executable policy is never accepted from
	// application users. Force excessive interpreter work without I/O.
	query, err := rego.New(rego.Query("data.optitrace.outcome"), rego.Module("budget.rego", `package optitrace
outcome := "verified" if { every n in numbers.range(0, 10000) { n >= 0 } }`)).PrepareForEval(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	e.queries["complete"] = query
	result, err := e.evaluate(context.Background(), Request{Policy: Policy{Profile: "complete"}, Scenario: "clear", Strategy: "opa"}, time.Now().UTC(), true)
	if err != nil || result.Outcome != Indeterminate || result.SupplierCostMicrorand != 6600 || result.StoppingReason != "policy evaluator failure; billable attempts retained; outcome indeterminate" {
		t.Fatalf("budget failure lost billing/fail-closed evidence: %+v error=%v", result, err)
	}
	for _, check := range result.Checks {
		if check.Status != "available" || !check.Compulsory || check.CostMicrorand == 0 {
			t.Fatalf("budget failure altered source evidence: %+v", check)
		}
	}
}

func TestDisconnectedCallerRetainsCompletedOPADecision(t *testing.T) {
	e := testEngine(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completed := 0
	result, err := e.Evaluate(ctx, Request{Policy: Policy{Profile: "complete"}, Scenario: "clear", Strategy: "opa", Journal: func(_ context.Context, phase string, _ Check) error {
		if phase == "completed" {
			completed++
			if completed == 3 {
				cancel()
			}
		}
		return nil
	}})
	if err != nil || result.Outcome != Verified || result.SupplierCostMicrorand != 6600 || ctx.Err() != context.Canceled {
		t.Fatalf("completed evidence lost on disconnect: %+v error=%v", result, err)
	}
	for _, check := range result.Checks {
		if check.Status != "available" || !check.Compulsory {
			t.Fatalf("disconnect discarded completed check: %+v", check)
		}
	}
}

func TestPreparedPolicyRecoversAfterCancellation(t *testing.T) {
	e, err := New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for sample := 0; sample < 64; sample++ {
				profile := []string{"complete", "conditional"}[sample%2]
				cancelled, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := opaDecision(cancelled, e.queries[profile], []int{0, 0, 0}); err == nil {
					t.Error("cancelled evaluation unexpectedly succeeded")
				}
				input := []int{sample % 4, (sample / 4) % 4, (sample / 16) % 4}
				got, err := opaDecision(context.Background(), e.queries[profile], input)
				if err != nil || got != reference(profile, input) {
					t.Errorf("fresh evaluation after cancellation: %s %v got=%s error=%v", profile, input, got, err)
				}
			}
		}()
	}
	workers.Wait()
}
