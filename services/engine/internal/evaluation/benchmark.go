package evaluation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"time"
)

type BenchmarkMetric struct {
	Scenario                  string  `json:"scenario,omitempty"`
	Strategy                  string  `json:"strategy"`
	Profile                   string  `json:"profile"`
	Measurement               string  `json:"measurement"`
	RequestCount              int     `json:"request_count"`
	AgreementRate             float64 `json:"agreement_rate"`
	CompulsoryViolations      int     `json:"compulsory_violations"`
	MeanCostMicrorand         float64 `json:"mean_cost_microrand"`
	MeanSupplierCostMicrorand float64 `json:"mean_supplier_cost_microrand"`
	MeanRuntimeCostMicrorand  float64 `json:"mean_runtime_cost_microrand"`
	P50MS                     float64 `json:"p50_ms"`
	P95MS                     float64 `json:"p95_ms"`
	P99MS                     float64 `json:"p99_ms"`
	MeanCalls                 float64 `json:"mean_calls"`
	IndeterminateRate         float64 `json:"indeterminate_rate"`
}
type Comparison struct {
	Measurement          string  `json:"measurement"`
	Profile              string  `json:"profile"`
	Baseline             string  `json:"baseline"`
	PairedCostDifference float64 `json:"paired_cost_difference_microrand"`
	Lower95              float64 `json:"lower_95_microrand"`
	Upper95              float64 `json:"upper_95_microrand"`
	SavingsFraction      float64 `json:"savings_fraction"`
	CorrectnessPassed    bool    `json:"correctness_passed"`
	LatencyPassed        bool    `json:"latency_passed"`
	ResearchTargetPassed bool    `json:"research_target_passed"`
	Status               string  `json:"status"`
}
type BenchmarkManifest struct {
	ObservationSHA256 string           `json:"observation_sha256"`
	Seed              int64            `json:"seed"`
	Count             int              `json:"count"`
	Version           string           `json:"version"`
	Data              string           `json:"data"`
	Dataset           string           `json:"dataset"`
	Tuning            string           `json:"tuning"`
	Prices            map[string]int64 `json:"prices_microrand"`
	RuntimeRate       int              `json:"runtime_microrand_per_ms"`
	RecentHours       int              `json:"recent_hours"`
	FreshnessSeconds  int              `json:"freshness_seconds"`
	OS                string           `json:"os"`
	Arch              string           `json:"arch"`
	GoVersion         string           `json:"go_version"`
	Hardware          string           `json:"hardware"`
	MeasuredCount     int              `json:"measured_count"`
	ScenarioCounts    map[string]int   `json:"scenario_counts"`
	ObservationTime   time.Time        `json:"observation_time"`
}
type BenchmarkReport struct {
	ArtifactObject  string            `json:"artifact_object,omitempty"`
	ScenarioResults []BenchmarkMetric `json:"scenario_results"`
	Manifest        BenchmarkManifest `json:"manifest"`
	Results         []BenchmarkMetric `json:"results"`
	Comparisons     []Comparison      `json:"comparisons"`
	Notes           []string          `json:"notes"`
}
type sample struct {
	scenario string
	profile  string
}

func heldout(seed int64, count int) []sample {
	r := rand.New(rand.NewSource(seed))
	rows := make([]sample, count)
	for i := range rows {
		profile := "complete"
		if i%2 == 1 {
			profile = "conditional"
		}
		rows[i] = sample{Scenarios[r.Intn(len(Scenarios))], profile}
	}
	return rows
}
func quantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	index := int(math.Ceil(q*float64(len(cp)))) - 1
	if index < 0 {
		index = 0
	}
	return cp[index]
}
func aggregate(strategy, profile, label string, rs []Result, expected []string) BenchmarkMetric {
	m := BenchmarkMetric{Strategy: strategy, Profile: profile, Measurement: label, RequestCount: len(rs)}
	latencies := []float64{}
	agreed := 0
	for i, r := range rs {
		if r.Outcome == expected[i] {
			agreed++
		}
		m.MeanCostMicrorand += float64(r.SupplierCostMicrorand + r.RuntimeCostMicrorand)
		m.MeanSupplierCostMicrorand += float64(r.SupplierCostMicrorand)
		m.MeanRuntimeCostMicrorand += float64(r.RuntimeCostMicrorand)
		latencies = append(latencies, r.ElapsedMS)
		if r.Outcome == Indeterminate {
			m.IndeterminateRate++
		}
		for _, c := range r.Checks {
			if c.Compulsory && c.Status == "skipped" {
				m.CompulsoryViolations++
			}
			if c.CostMicrorand > 0 {
				m.MeanCalls++
			}
		}
	}
	n := float64(len(rs))
	if n > 0 {
		m.AgreementRate = float64(agreed) / n
		m.MeanCostMicrorand /= n
		m.MeanSupplierCostMicrorand /= n
		m.MeanRuntimeCostMicrorand /= n
		m.MeanCalls /= n
		m.IndeterminateRate /= n
	}
	m.P50MS = quantile(latencies, .5)
	m.P95MS = quantile(latencies, .95)
	m.P99MS = quantile(latencies, .99)
	return m
}
func paired(base, opt []Result) (float64, float64, float64) {
	if len(base) == 0 {
		return 0, 0, 0
	}
	diffs := make([]float64, len(base))
	mean := float64(0)
	for i, b := range base {
		diffs[i] = float64(b.SupplierCostMicrorand + b.RuntimeCostMicrorand - opt[i].SupplierCostMicrorand - opt[i].RuntimeCostMicrorand)
		mean += diffs[i]
	}
	mean /= float64(len(diffs))
	variance := float64(0)
	for _, d := range diffs {
		variance += (d - mean) * (d - mean)
	}
	se := float64(0)
	if len(diffs) > 1 {
		se = math.Sqrt(variance / float64(len(diffs)-1) / float64(len(diffs)))
	}
	return mean, mean - 1.96*se, mean + 1.96*se
}
func (e *Engine) Benchmark(ctx context.Context, seed int64, count int) (BenchmarkReport, error) {
	if count < 2 || count > 1000 {
		return BenchmarkReport{}, fmt.Errorf("count must be between 2 and 1000")
	}
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	rows := heldout(seed, count)
	measured := count
	if measured > 20 {
		measured = 20
	}
	manifest := BenchmarkManifest{Seed: seed, Count: count, Version: "synthetic-v1", Data: "synthetic; configured demonstration prices; no observed MTN savings", Dataset: "seeded held-out rows, equal alternating profile allocation", Tuning: "fixed eight-row tuning table in policy.go; disjoint held-out generator", Prices: map[string]int64{}, RuntimeRate: 10, RecentHours: 24, FreshnessSeconds: 60, OS: runtime.GOOS, Arch: runtime.GOARCH, GoVersion: runtime.Version(), Hardware: fmt.Sprintf("logical CPUs=%d; host model not captured", runtime.NumCPU()), MeasuredCount: measured, ScenarioCounts: map[string]int{}, ObservationTime: at}
	for i, c := range Capabilities {
		manifest.Prices[c] = price(i)
	}
	for _, row := range rows {
		manifest.ScenarioCounts[row.scenario]++
	}
	// Fingerprint the actual immutable source rows, not only the RNG seed.
	hash := sha256.New()
	for _, row := range rows {
		_, _ = hash.Write([]byte(row.profile + ":" + row.scenario))
		for i := range Capabilities {
			b, _ := json.Marshal(sourceCheck(i, row.scenario, at))
			_, _ = hash.Write(b)
		}
	}
	manifest.ObservationSHA256 = hex.EncodeToString(hash.Sum(nil))
	report := BenchmarkReport{Manifest: manifest, Notes: []string{"Virtual-time results estimate configured supplier charges and runtime, not cloud expenditure.", "Measured results use real HTTP and local elapsed time, excluding browser, authentication, DB persistence and cold starts; they do not certify the full API 500ms gate.", "At most 20 seeded held-out rows measured over HTTP per run; small-sample tails are descriptive, not production evidence.", "95% paired mean difference intervals use normal approximation and require larger samples for strong inference.", "staged is a competent number/SIM/device conditional workflow; OPA uses the same early-exit controller with embedded OPA final evaluation.", "No market telemetry, learned model, cross-request cache, or proven commercial benefit."}}
	virtual := map[string]map[string][]Result{}
	expected := map[string][]string{"complete": {}, "conditional": {}}
	for _, strategy := range Strategies {
		virtual[strategy] = map[string][]Result{"complete": {}, "conditional": {}}
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return BenchmarkReport{}, err
		}
		for _, strategy := range Strategies {
			r, err := e.evaluate(ctx, Request{Policy: Policy{ID: "benchmark-" + row.profile, Version: 1, Profile: row.profile}, Scenario: row.scenario, Strategy: strategy}, at, true)
			if err != nil {
				return BenchmarkReport{}, err
			}
			virtual[strategy][row.profile] = append(virtual[strategy][row.profile], r)
			if strategy == "full_sequential" {
				expected[row.profile] = append(expected[row.profile], r.Outcome)
			}
		}
	}
	for _, profile := range []string{"complete", "conditional"} {
		metrics := map[string]BenchmarkMetric{}
		for _, strategy := range Strategies {
			m := aggregate(strategy, profile, "virtual_time_estimate", virtual[strategy][profile], expected[profile])
			metrics[strategy] = m
			report.Results = append(report.Results, m)
		}
		best := ""
		for _, strategy := range []string{"full_sequential", "full_parallel", "staged", "opa"} {
			m := metrics[strategy]
			if m.AgreementRate == 1 && m.CompulsoryViolations == 0 && m.P95MS <= 500 && (best == "" || m.MeanCostMicrorand < metrics[best].MeanCostMicrorand) {
				best = strategy
			}
		}
		comparison := Comparison{Profile: profile, Baseline: best, Measurement: "virtual_time_estimate", Status: "no eligible baseline"}
		if best != "" {
			opt := metrics["optitrace"]
			base := metrics[best]
			mean, lo, hi := paired(virtual[best][profile], virtual["optitrace"][profile])
			comparison.PairedCostDifference = mean
			comparison.Lower95 = lo
			comparison.Upper95 = hi
			if base.MeanCostMicrorand > 0 {
				comparison.SavingsFraction = mean / base.MeanCostMicrorand
			}
			comparison.CorrectnessPassed = opt.AgreementRate == 1 && opt.CompulsoryViolations == 0
			comparison.LatencyPassed = opt.P95MS <= 500 && opt.P95MS <= base.P95MS*1.1
			comparison.ResearchTargetPassed = comparison.CorrectnessPassed && comparison.LatencyPassed && comparison.SavingsFraction >= .2
			comparison.Status = "synthetic target not established"
			if comparison.ResearchTargetPassed {
				comparison.Status = "synthetic workload target met; production claim unproven"
			}
		}
		report.Comparisons = append(report.Comparisons, comparison)
	}
	measuredRows := rows[:measured]
	actual := map[string]map[string][]Result{}
	actualExpected := map[string][]string{"complete": {}, "conditional": {}}
	for _, strategy := range Strategies {
		actual[strategy] = map[string][]Result{"complete": {}, "conditional": {}}
	}
	// Rotate execution order to avoid always assigning warmest connections to one strategy.
	for rowIndex, row := range measuredRows {
		for step := range Strategies {
			strategy := Strategies[(step+rowIndex)%len(Strategies)]
			r, err := e.evaluate(ctx, Request{Policy: Policy{ID: "benchmark-" + row.profile, Version: 1, Profile: row.profile}, Scenario: row.scenario, Strategy: strategy}, at, false)
			if err != nil {
				return BenchmarkReport{}, err
			}
			actual[strategy][row.profile] = append(actual[strategy][row.profile], r)
		}
		source := []int{}
		for i := 0; i < 3; i++ {
			c := sourceCheck(i, row.scenario, at)
			normalize(&c, i, at)
			source = append(source, states([]Check{c})[0])
		}
		actualExpected[row.profile] = append(actualExpected[row.profile], reference(row.profile, source))
	}
	for _, profile := range []string{"complete", "conditional"} {
		for _, strategy := range Strategies {
			report.Results = append(report.Results, aggregate(strategy, profile, "measured_http_engine_only", actual[strategy][profile], actualExpected[profile]))
		}
	}
	for _, profile := range []string{"complete", "conditional"} {
		for _, scenario := range Scenarios {
			for _, strategy := range Strategies {
				var results []Result
				var outcomes []string
				position := 0
				for _, row := range rows {
					if row.profile != profile {
						continue
					}
					if row.scenario == scenario {
						results = append(results, virtual[strategy][profile][position])
						outcomes = append(outcomes, expected[profile][position])
					}
					position++
				}
				if len(results) > 0 {
					m := aggregate(strategy, profile, "virtual_time_estimate", results, outcomes)
					m.Scenario = scenario
					report.ScenarioResults = append(report.ScenarioResults, m)
				}
			}
		}
	}
	return report, nil
}
