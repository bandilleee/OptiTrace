package evaluation

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/open-policy-agent/opa/v1/rego"
)

// Independent small-state oracle: enumerate every consistent completion rather
// than calling the production memoized stopping function.
func plannerOracleCanStop(profile string, partial []int) bool {
	for i, state := range partial {
		if compulsory(profile, i) && state == 3 {
			return false
		}
	}
	outcomes := map[string]bool{}
	for a := 0; a < 3; a++ {
		for b := 0; b < 3; b++ {
			for c := 0; c < 3; c++ {
				completion := []int{a, b, c}
				consistent := true
				for i, state := range partial {
					if state != 3 && completion[i] != state {
						consistent = false
					}
				}
				if consistent {
					outcomes[reference(profile, completion)] = true
				}
			}
		}
	}
	return len(outcomes) == 1
}

func plannerOracleEstimate(t *testing.T, profile string, plan candidate, rows [][]int) candidate {
	t.Helper()
	totalCost := 0.0
	tails := []float64{}
	for _, row := range rows {
		partial := []int{3, 3, 3}
		charge, elapsed := 0.0, 0.0
		for _, batch := range plan.Batches {
			batchLatency := 0.0
			for _, capability := range batch {
				if capability < 0 || capability >= 3 || partial[capability] != 3 {
					t.Fatalf("invalid/repeated capability: %+v", plan)
				}
				partial[capability] = row[capability]
				charge += float64(price(capability))
				batchLatency = math.Max(batchLatency, float64(latency(capability).Milliseconds()))
			}
			elapsed += batchLatency
			if plannerOracleCanStop(profile, partial) {
				break
			}
		}
		if !plannerOracleCanStop(profile, partial) || reference(profile, partial) != reference(profile, row) {
			t.Fatalf("invalid stopping/compulsory obligations: profile=%s plan=%s row=%v partial=%v", profile, plan.Name, row, partial)
		}
		totalCost += charge + elapsed*10
		tails = append(tails, elapsed)
	}
	sort.Float64s(tails)
	plan.ExpectedCost = totalCost / float64(len(rows))
	plan.EstimatedP95MS = tails[(95*len(rows)+99)/100-1]
	return plan
}

func TestPlannerConstrainedOptimalityAgainstOracle(t *testing.T) {
	allStates := [][]int{}
	for a := 0; a < 3; a++ {
		for b := 0; b < 3; b++ {
			for c := 0; c < 3; c++ {
				allStates = append(allStates, []int{a, b, c})
			}
		}
	}
	fixtures := map[string][][]int{
		"frozen_tuning": tuning,
		"all_clear":     {{0, 0, 0}},
		"early_adverse": {{1, 0, 0}, {1, 1, 1}},
		"faults":        {{2, 0, 1}, {0, 2, 0}, {0, 0, 2}},
		"all_27":        allStates,
	}
	for fixtureName, rows := range fixtures {
		for _, profile := range []string{"complete", "conditional"} {
			t.Run(fixtureName+"/"+profile, func(t *testing.T) {
				scored := []candidate{}
				fastest := math.Inf(1)
				for _, plan := range candidates() {
					oracle := plannerOracleEstimate(t, profile, plan, rows)
					production, valid := estimatePlan(profile, plan, rows)
					if !valid || production.ExpectedCost != oracle.ExpectedCost || production.EstimatedP95MS != oracle.EstimatedP95MS {
						t.Fatalf("score differs from independent replay: production=%+v oracle=%+v valid=%v", production, oracle, valid)
					}
					if oracle.EstimatedP95MS <= 500 {
						scored = append(scored, oracle)
						fastest = math.Min(fastest, oracle.EstimatedP95MS)
					}
				}
				admissible := []candidate{}
				for _, plan := range scored {
					if plan.EstimatedP95MS <= fastest*1.1 {
						admissible = append(admissible, plan)
					}
				}
				sort.Slice(admissible, func(i, j int) bool {
					a, b := admissible[i], admissible[j]
					if a.ExpectedCost != b.ExpectedCost {
						return a.ExpectedCost < b.ExpectedCost
					}
					if a.EstimatedP95MS != b.EstimatedP95MS {
						return a.EstimatedP95MS < b.EstimatedP95MS
					}
					return a.Name < b.Name
				})
				got := choosePlanFromRows(profile, rows)
				if !reflect.DeepEqual(got, admissible[0]) || !reflect.DeepEqual(got, choosePlanFromRows(profile, rows)) {
					t.Fatalf("not deterministic constrained optimum: got=%+v want=%+v", got, admissible[0])
				}
				t.Logf("selected %s cost=%.2f p95=%.2f fastest=%.2f", got.Name, got.ExpectedCost, got.EstimatedP95MS, fastest)
			})
		}
	}
}

func TestPlannerRejectsCheapTailRegression(t *testing.T) {
	selected := choosePlan("conditional")
	cheaperButSlow := false
	for _, plan := range candidates() {
		p, valid := estimatePlan("conditional", plan, tuning)
		if valid && p.ExpectedCost < selected.ExpectedCost && p.EstimatedP95MS > selected.EstimatedP95MS*1.1 {
			cheaperButSlow = true
		}
	}
	if !cheaperButSlow || len(selected.Batches) != 1 || len(selected.Batches[0]) != 3 || selected.EstimatedP95MS != 30 {
		t.Fatalf("latency constraint did not reject the cheaper slow plan: %+v", selected)
	}
	// Alternate explicit tuning data proves selection is not hardcoded parallel.
	early := choosePlanFromRows("conditional", [][]int{{1, 0, 0}})
	if early.EstimatedP95MS != 10 || early.ExpectedCost != 1300 || !reflect.DeepEqual(early.Batches[0], []int{0}) {
		t.Fatalf("safe early decision should remain possible: %+v", early)
	}
}

func TestPlannerInvalidTuningFailsClosed(t *testing.T) {
	for _, rows := range [][][]int{nil, {}, {{0, 0}}, {{0, 0, 3}}, {{0, -1, 0}}} {
		if plan := choosePlanFromRows("conditional", rows); plan.Name != "" {
			t.Fatalf("invalid tuning accepted: %v %+v", rows, plan)
		}
	}
}

func TestBenchmarkSeparatesCostAndLatencyBaselines(t *testing.T) {
	metrics := map[string]BenchmarkMetric{}
	results := map[string][]Result{}
	for strategy, values := range map[string][2]float64{"full_parallel": {100, 1000}, "full_sequential": {200, 1000}, "staged": {180, 800}, "opa": {150, 900}, "optitrace": {120, 600}} {
		metrics[strategy] = BenchmarkMetric{RequestCount: 2, AgreementRate: 1, P95MS: values[0], MeanCostMicrorand: values[1]}
		results[strategy] = []Result{{SupplierCostMicrorand: int64(values[1])}, {SupplierCostMicrorand: int64(values[1])}}
	}
	comparison := benchmarkComparison("conditional", metrics, results)
	if comparison.Baseline != "staged" || comparison.LatencyBaseline != "full_parallel" || comparison.LatencyBaselineP95MS != 100 || math.Abs(comparison.LatencyP95LimitMS-110) > 1e-9 || comparison.LatencyPassed || comparison.ResearchTargetPassed || !comparison.CorrectnessPassed || comparison.SavingsFraction != .25 {
		t.Fatalf("cheap baseline concealed tail regression: %+v", comparison)
	}
	opt := metrics["optitrace"]
	opt.P95MS = 110
	metrics["optitrace"] = opt
	comparison = benchmarkComparison("conditional", metrics, results)
	if !comparison.LatencyPassed || !comparison.ResearchTargetPassed {
		t.Fatalf("exact allowed boundary should pass this synthetic fixture: %+v", comparison)
	}
	fast := metrics["full_parallel"]
	fast.AgreementRate = .5
	metrics["full_parallel"] = fast
	comparison = benchmarkComparison("conditional", metrics, results)
	if comparison.LatencyBaseline != "opa" {
		t.Fatalf("incorrect fast baseline must be excluded: %+v", comparison)
	}
	for _, strategy := range []string{"staged", "opa"} {
		m := metrics[strategy]
		m.CompulsoryViolations = 1
		metrics[strategy] = m
	}
	comparison = benchmarkComparison("conditional", metrics, results)
	if comparison.Baseline != "" || comparison.LatencyBaseline != "full_sequential" || comparison.ResearchTargetPassed {
		t.Fatalf("no staged cost baseline must remain unavailable: %+v", comparison)
	}
}

func TestBenchmarkAbsoluteLatencyAndMissingPairs(t *testing.T) {
	metrics := map[string]BenchmarkMetric{
		"staged":    {RequestCount: 1, AgreementRate: 1, P95MS: 490, MeanCostMicrorand: 1000},
		"optitrace": {RequestCount: 1, AgreementRate: 1, P95MS: 501, MeanCostMicrorand: 500},
	}
	results := map[string][]Result{"staged": {{SupplierCostMicrorand: 1000}}, "optitrace": {{SupplierCostMicrorand: 500}}}
	comparison := benchmarkComparison("conditional", metrics, results)
	if comparison.LatencyP95LimitMS != 500 || comparison.LatencyPassed || comparison.ResearchTargetPassed {
		t.Fatalf("absolute500ms gate lost: %+v", comparison)
	}
	results["optitrace"] = nil
	comparison = benchmarkComparison("conditional", metrics, results)
	if comparison.ResearchTargetPassed || comparison.Status != "cost observations missing or unpaired" {
		t.Fatalf("missing paired data invented evidence: %+v", comparison)
	}
}

func TestBenchmarkDisagreementDiagnosticsPreserveFrozenOracle(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer source.Close()
	e, err := New(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.Benchmark(context.Background(), 42, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Disagreements) != maxBenchmarkDisagreements || report.Manifest.PlannerVersion != "latency-constrained-v2" || report.Manifest.Version != "synthetic-v1" {
		t.Fatalf("missing bounded/versioned diagnostics: count=%d manifest=%+v", len(report.Disagreements), report.Manifest)
	}
	for _, disagreement := range report.Disagreements {
		if disagreement.Measurement != "measured_http_engine_only" || disagreement.ExpectedOutcome == Indeterminate || disagreement.ActualOutcome != Indeterminate || !disagreement.SourceStatesDiffer || len(disagreement.Result.Checks) != 3 {
			t.Fatalf("transport fault changed frozen oracle: %+v", disagreement)
		}
		charged := int64(0)
		for _, check := range disagreement.Result.Checks {
			if check.Compulsory && check.Status == "skipped" {
				t.Fatalf("missing compulsory failure path: %+v", check)
			}
			if check.Status != "skipped" && (check.Status != "timeout" || check.CostMicrorand == 0) {
				t.Fatalf("diagnostic lost transport status/charge: %+v", check)
			}
			charged += check.CostMicrorand
		}
		if disagreement.Result.SupplierCostMicrorand != charged || charged == 0 {
			t.Fatalf("diagnostic lost started-call charge total: %+v", disagreement)
		}
	}
	// All measured outcomes are indeterminate, but the expected outcomes remain
	// those of the seeded rows. The first20 diagnostic cap cannot hide failures
	// from the aggregate agreement rate or reduce a strategy's measured count.
	for _, metric := range report.Results {
		if metric.Measurement == "virtual_time_estimate" {
			if metric.AgreementRate != 1 {
				t.Fatalf("real faults contaminated virtual oracle: %+v", metric)
			}
			continue
		}
		count, agreed := 0, 0
		for _, row := range heldout(42, 20) {
			if row.profile != metric.Profile {
				continue
			}
			count++
			observations := []Check{}
			for i := range Capabilities {
				observations = append(observations, sourceCheck(i, row.scenario, report.Manifest.ObservationTime))
			}
			// Normalize freshness/event semantics exactly as the frozen-row
			// oracle does; stale observations must not count as available.
			for i := range observations {
				normalize(&observations[i], i, report.Manifest.ObservationTime)
			}
			if reference(row.profile, states(observations)) == Indeterminate {
				agreed++
			}
		}
		if metric.RequestCount != count || metric.AgreementRate != float64(agreed)/float64(count) || metric.AgreementRate == 1 {
			t.Fatalf("aggregate hid input-instability disagreement: %+v want_agreed=%d/%d", metric, agreed, count)
		}
	}
}

func TestBenchmarkDisagreementDistinguishesSameInputPolicyFailure(t *testing.T) {
	e := testEngine(t)
	// Private test substitution demonstrates policy/output failure separately
	// from the changed transport observations exercised above.
	query, err := rego.New(rego.Query("data.optitrace.outcome"), rego.Module("mismatch.rego", `package optitrace
outcome := "indeterminate"`)).PrepareForEval(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	e.queries["complete"] = query
	e.queries["conditional"] = query
	report, err := e.Benchmark(context.Background(), 42, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Disagreements) == 0 {
		t.Fatal("policy/output mismatch was hidden")
	}
	for _, disagreement := range report.Disagreements {
		if disagreement.SourceStatesDiffer || (disagreement.Strategy != "opa" && disagreement.Strategy != "optitrace") {
			t.Fatalf("same-input policy mismatch mislabeled as transport difference: %+v", disagreement)
		}
	}
}
