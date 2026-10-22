package evaluation

import (
	"context"
	"fmt"
	"github.com/open-policy-agent/opa/v1/rego"
	"strings"
	"sync"
)

// GenerateRego only accepts the bounded decision-table contract. User supplied
// executable Rego is deliberately unsupported. Observations are normalized first.
func GenerateRego(t DecisionTable) (string, error) {
	if (t.Profile != "complete" && t.Profile != "conditional") || t.FreshnessSeconds != 60 || t.RecentHours != 24 {
		return "", fmt.Errorf("unsupported decision table")
	}
	required := "input.states[0] >= 2"
	if t.Profile == "complete" {
		required = "some s in input.states; s >= 2"
	}
	return fmt.Sprintf(`package optitrace
 default outcome := "indeterminate"
 mandatory_failed if { %s }
 adverse if { some s in input.states; s == 1 }
 all_clear if { every s in input.states { s == 0 } }
 outcome := "additional_verification_required" if { not mandatory_failed; adverse }
 outcome := "verified" if { not mandatory_failed; not adverse; all_clear }
 `, required), nil
}

// State 0 = known clear, 1 = known adverse, 2 = resolved unavailable,
// 3 = not yet fetched. Pending values are never treated as reassuring.
func states(checks []Check) []int {
	s := make([]int, len(checks))
	for i, c := range checks {
		s[i] = 2
		if c.Status == "skipped" {
			s[i] = 3
		} else if c.Status == "available" && c.Value != nil {
			s[i] = 0
			if *c.Value {
				s[i] = 1
			}
		}
	}
	return s
}
func reference(profile string, s []int) string {
	for i, v := range s {
		if compulsory(profile, i) && v != 0 && v != 1 {
			return Indeterminate
		}
	}
	for _, v := range s {
		if v == 1 {
			return Additional
		}
	}
	for _, v := range s {
		if v != 0 {
			return Indeterminate
		}
	}
	return Verified
}
func opaDecision(ctx context.Context, q rego.PreparedEvalQuery, s []int) (string, error) {
	r, err := q.Eval(ctx, rego.EvalInput(map[string]any{"states": s}))
	if err != nil {
		return "", err
	}
	if len(r) != 1 || len(r[0].Expressions) != 1 {
		return "", fmt.Errorf("undefined policy decision")
	}
	v, ok := r[0].Expressions[0].Value.(string)
	if !ok {
		return "", fmt.Errorf("invalid policy output")
	}
	return v, nil
}

// stoppingOracle exhaustively proves output equivalence over all valid future
// clear/adverse/unavailable completions, then checks compulsory obligations.
// The memo is bounded by 2 * 4^3 and is safe across concurrent evaluations.
var oracleMemo sync.Map

func stoppingOracle(profile string, s []int) (string, bool) {
	key := profile + fmt.Sprint(s)
	if cached, ok := oracleMemo.Load(key); ok {
		v := cached.(string)
		return v, v != ""
	}
	for i, v := range s {
		if compulsory(profile, i) && v == 3 {
			oracleMemo.Store(key, "")
			return "", false
		}
	}
	outcomes := map[string]bool{}
	copyStates := append([]int(nil), s...)
	var visit func(int)
	visit = func(i int) {
		if i == len(copyStates) {
			outcomes[reference(profile, copyStates)] = true
			return
		}
		if s[i] != 3 {
			visit(i + 1)
			return
		}
		for v := 0; v < 3; v++ {
			copyStates[i] = v
			visit(i + 1)
		}
	}
	visit(0)
	result := ""
	if len(outcomes) == 1 {
		for x := range outcomes {
			result = x
		}
	}
	oracleMemo.Store(key, result)
	return result, result != ""
}

type candidate struct {
	Batches        [][]int
	Name           string
	ExpectedCost   float64
	EstimatedP95MS float64
}

// Ordered set partitions enumerate all sequential orders, parallel groups, and
// mixed batches. With three capabilities the candidate space has 13 plans.
func candidates() []candidate {
	var out []candidate
	var walk func(int, [][]int)
	walk = func(mask int, batches [][]int) {
		if mask == 7 {
			parts := []string{}
			cp := [][]int{}
			for _, b := range batches {
				names := []string{}
				cp = append(cp, append([]int(nil), b...))
				for _, i := range b {
					names = append(names, Capabilities[i])
				}
				parts = append(parts, strings.Join(names, "+"))
			}
			out = append(out, candidate{Batches: cp, Name: strings.Join(parts, " -> ")})
			return
		}
		remaining := 7 ^ mask
		for subset := 1; subset <= 7; subset++ {
			if subset&remaining != subset {
				continue
			}
			b := []int{}
			for i := 0; i < 3; i++ {
				if subset&(1<<i) != 0 {
					b = append(b, i)
				}
			}
			walk(mask|subset, append(batches, b))
		}
	}
	walk(0, nil)
	return out
}

// Fixed tuning observations are distinct from held-out seeded evaluation rows.
// They include correlation; neither planning nor pricing reads the request's
// future source observations.
var tuning = [][]int{{0, 0, 0}, {0, 0, 0}, {0, 1, 1}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0, 2, 0}, {2, 0, 0}}

func choosePlan(profile string) candidate {
	best := candidate{ExpectedCost: 1e30}
	for _, p := range candidates() {
		total := float64(0)
		maxLatency := float64(0)
		for _, row := range tuning {
			s := []int{3, 3, 3}
			cost := float64(0)
			elapsed := float64(0)
			for _, batch := range p.Batches {
				batchMS := float64(0)
				for _, i := range batch {
					s[i] = row[i]
					cost += float64(price(i))
					ms := float64(latency(i).Milliseconds())
					if ms > batchMS {
						batchMS = ms
					}
				}
				elapsed += batchMS
				if _, ok := stoppingOracle(profile, s); ok {
					break
				}
			}
			total += cost + elapsed*10
			if elapsed > maxLatency {
				maxLatency = elapsed
			}
		}
		p.ExpectedCost = total / float64(len(tuning))
		p.EstimatedP95MS = maxLatency
		if maxLatency <= 500 && (p.ExpectedCost < best.ExpectedCost || (p.ExpectedCost == best.ExpectedCost && p.EstimatedP95MS < best.EstimatedP95MS)) {
			best = p
		}
	}
	return best
}
