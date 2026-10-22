package evaluation

import (
	"context"
	"time"
)

const (
	Verified      = "verified"
	Additional    = "additional_verification_required"
	Indeterminate = "indeterminate"
)

var Capabilities = []string{"number_verification", "sim_swap", "device_swap"}
var Scenarios = []string{"clear", "number_mismatch", "sim_change", "device_change", "all_adverse", "stale", "timeout", "rate_limit", "permission_denied", "source_failure", "correlated"}
var Strategies = []string{"full_sequential", "full_parallel", "staged", "opa", "optitrace"}

type Policy struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Profile string `json:"profile"`
}
type Request struct {
	Policy   Policy `json:"policy"`
	Scenario string `json:"scenario"`
	Strategy string `json:"strategy"`
	// Journal must atomically persist each action before returning. Parallel
	// plans may invoke it concurrently for different capabilities.
	Journal func(context.Context, string, Check) error `json:"-"`
}
type Check struct {
	Capability    string     `json:"capability"`
	Status        string     `json:"status"`
	Value         *bool      `json:"value,omitempty"`
	SourceTime    time.Time  `json:"source_time"`
	EventTime     *time.Time `json:"event_time,omitempty"`
	Subject       string     `json:"subject,omitempty"`
	Scope         string     `json:"scope,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	ElapsedMS     float64    `json:"elapsed_ms"`
	CostMicrorand int64      `json:"cost_microrand"`
	Compulsory    bool       `json:"compulsory"`
}
type Result struct {
	Outcome               string  `json:"outcome"`
	StoppingReason        string  `json:"stopping_reason"`
	Checks                []Check `json:"checks"`
	SupplierCostMicrorand int64   `json:"supplier_cost_microrand"`
	RuntimeCostMicrorand  int64   `json:"runtime_cost_microrand"`
	ElapsedMS             float64 `json:"elapsed_ms"`
	Plan                  string  `json:"plan"`
}
type DecisionTable struct {
	Profile          string `json:"profile"`
	FreshnessSeconds int    `json:"freshness_seconds"`
	RecentHours      int    `json:"recent_hours"`
}

func Table(profile string) DecisionTable    { return DecisionTable{profile, 60, 24} }
func compulsory(profile string, i int) bool { return profile == "complete" || i == 0 }
func ptr(v bool) *bool                      { return &v }
func price(i int) int64                     { return []int64{1200, 2200, 3200}[i] }
func latency(i int) time.Duration {
	return []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond}[i]
}
func validName(v string, choices []string) bool {
	for _, x := range choices {
		if x == v {
			return true
		}
	}
	return false
}
