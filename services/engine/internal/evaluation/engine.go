package evaluation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/v1/rego"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Engine struct {
	baseURL string
	client  *http.Client
	queries map[string]rego.PreparedEvalQuery
	plans   map[string]candidate
}

func New(simulatorURL string) (*Engine, error) {
	return NewWithHTTPClient(simulatorURL, &http.Client{Timeout: 200 * time.Millisecond})
}

// NewWithHTTPClient permits a Google ADC ID-token transport for private Cloud
// Run adapters. Identity acquisition stays in infrastructure initialization;
// each billable source request still has a bounded transport timeout.
func NewWithHTTPClient(simulatorURL string, client *http.Client) (*Engine, error) {
	u, err := url.Parse(simulatorURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("valid simulator URL required")
	}
	if client == nil {
		return nil, fmt.Errorf("HTTP client required")
	}
	boundedClient := *client
	boundedClient.Timeout = 200 * time.Millisecond
	transport := boundedClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	boundedClient.Transport = otelhttp.NewTransport(transport)
	boundedClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	e := &Engine{baseURL: u.String(), client: &boundedClient, queries: map[string]rego.PreparedEvalQuery{}, plans: map[string]candidate{}}
	for _, profile := range []string{"complete", "conditional"} {
		source, _ := GenerateRego(Table(profile))
		q, err := rego.New(rego.Query("data.optitrace.outcome"), rego.Module("generated.rego", source)).PrepareForEval(context.Background())
		if err != nil {
			return nil, err
		}
		e.queries[profile] = q
		e.plans[profile] = choosePlan(profile)
		if e.plans[profile].Name == "" {
			return nil, fmt.Errorf("no admissible plan for profile %s", profile)
		}
	}
	return e, nil
}
func validate(req Request) error {
	if req.Policy.Profile != "complete" && req.Policy.Profile != "conditional" {
		return fmt.Errorf("unsupported profile")
	}
	if !validName(req.Scenario, Scenarios) {
		return fmt.Errorf("unsupported scenario")
	}
	if req.Strategy != "" && !validName(req.Strategy, Strategies) {
		return fmt.Errorf("unsupported strategy")
	}
	return nil
}
func (e *Engine) Evaluate(ctx context.Context, req Request) (Result, error) {
	return e.evaluate(ctx, req, time.Now().UTC(), false)
}
func (e *Engine) evaluate(ctx context.Context, req Request, at time.Time, virtual bool) (Result, error) {
	if err := validate(req); err != nil {
		return Result{}, err
	}
	if req.Strategy == "" {
		req.Strategy = "optitrace"
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	start := time.Now()
	checks := make([]Check, 3)
	for i := range checks {
		checks[i] = Check{Capability: Capabilities[i], Status: "skipped", Compulsory: compulsory(req.Policy.Profile, i)}
	}
	plan := candidate{Batches: [][]int{{0}, {1}, {2}}, Name: "number_verification -> sim_swap -> device_swap"}
	if req.Strategy == "full_parallel" {
		plan = candidate{Batches: [][]int{{0, 1, 2}}, Name: "all capabilities parallel"}
	} else if req.Strategy == "optitrace" {
		plan = e.plans[req.Policy.Profile]
	}
	virtualMS := float64(0)
	var journalErr error
	for _, batch := range plan.Batches {
		if ctx.Err() != nil {
			break
		}
		var wg sync.WaitGroup
		batchErrors := make([]error, len(checks))
		for _, index := range batch {
			i := index
			wg.Add(1)
			go func() {
				defer wg.Done()
				if virtual {
					checks[i] = sourceCheck(i, req.Scenario, at)
					checks[i].StartedAt = at.Add(time.Duration(virtualMS * float64(time.Millisecond)))
					checks[i].ElapsedMS = float64(latency(i).Milliseconds())
					if req.Scenario == "timeout" && i == 1 {
						checks[i].ElapsedMS = 200
					}
					checks[i].CostMicrorand = price(i)
					checks[i].Compulsory = compulsory(req.Policy.Profile, i)
					normalize(&checks[i], i, at)
				} else {
					checks[i], batchErrors[i] = e.fetchJournal(ctx, i, req.Scenario, at, compulsory(req.Policy.Profile, i), req.Journal)
				}
			}()
		}
		wg.Wait()
		for _, i := range batch {
			if batchErrors[i] != nil {
				journalErr = errors.Join(journalErr, batchErrors[i])
			}
		}
		if journalErr != nil {
			break
		}
		batchMS := float64(0)
		for _, i := range batch {
			if checks[i].ElapsedMS > batchMS {
				batchMS = checks[i].ElapsedMS
			}
		}
		virtualMS += batchMS
		if req.Strategy != "full_parallel" && req.Strategy != "full_sequential" {
			if _, ok := stoppingOracle(req.Policy.Profile, states(checks)); ok {
				break
			}
		}
	}
	// Cancellation is an explicit required-work failure, never a free skipped call.
	if ctx.Err() != nil || journalErr != nil {
		for i := range checks {
			if checks[i].Status == "skipped" && checks[i].Compulsory {
				checks[i].Status = "unavailable"
			}
		}
	}
	s := states(checks)
	for i, v := range s {
		if v == 3 {
			s[i] = 2
		}
	}
	outcome := reference(req.Policy.Profile, s)
	policyFailed := false
	if req.Strategy == "opa" || req.Strategy == "optitrace" {
		// Once billable work starts, preserve its trace even if the caller
		// cancels during the final prepared policy evaluation.
		// Generated tables have bounded inputs and interpreter work. A wall
		// timer here misclassified valid evidence when the host was descheduled.
		policyCtx := context.WithoutCancel(ctx)
		evaluated, err := opaDecision(policyCtx, e.queries[req.Policy.Profile], s)
		if err != nil {
			slog.ErrorContext(policyCtx, "policy evaluation failed", "profile", req.Policy.Profile, "error", err, "context_error", policyCtx.Err())
			outcome = Indeterminate
			policyFailed = true
		} else {
			outcome = evaluated
		}
	}
	elapsed := float64(time.Since(start)) / float64(time.Millisecond)
	if virtual {
		elapsed = virtualMS
	}
	reason := "all required observations resolved"
	for _, c := range checks {
		if c.Status == "skipped" {
			reason = "all valid unresolved completions give the same outcome; compulsory obligations satisfied"
			break
		}
	}
	if outcome == Indeterminate {
		reason = "required evidence unavailable, stale, or invalid"
	}
	if ctx.Err() != nil {
		reason = "request cancelled; completed attempts remain charged; compulsory remainder follows unavailable failure path"
	}
	if policyFailed {
		reason = "policy evaluator failure; billable attempts retained; outcome indeterminate"
	}
	if journalErr != nil {
		outcome = Indeterminate
		reason = "durable source journal failed; partial attempts retained; evaluation requires reconciliation"
	}
	result := Result{Outcome: outcome, StoppingReason: reason, Checks: checks, ElapsedMS: elapsed, RuntimeCostMicrorand: int64(elapsed * 10), Plan: plan.Name}
	for _, c := range checks {
		result.SupplierCostMicrorand += c.CostMicrorand
	}
	return result, journalErr
}

// A persisted starting record is an uncertain billing intention: a process can
// die between its commit and the network call. Only completed confirms an
// observed result; storage failure must never permit a fresh source replay.
func (e *Engine) fetchJournal(ctx context.Context, i int, scenario string, at time.Time, required bool, journal func(context.Context, string, Check) error) (Check, error) {
	if journal == nil {
		return e.fetch(ctx, i, scenario, at, required), nil
	}
	intention := Check{Capability: Capabilities[i], Status: "unavailable", StartedAt: time.Now().UTC(), CostMicrorand: price(i), Compulsory: required}
	if err := journal(ctx, "starting", intention); err != nil {
		return Check{Capability: Capabilities[i], Status: "unavailable", Compulsory: required}, fmt.Errorf("journal starting %s: %w", Capabilities[i], err)
	}
	c := e.fetch(ctx, i, scenario, at, required)
	journalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if err := journal(journalCtx, "completed", c); err != nil {
		return c, fmt.Errorf("journal completed %s: %w", Capabilities[i], err)
	}
	return c, nil
}
func (e *Engine) fetch(ctx context.Context, i int, scenario string, at time.Time, required bool) Check {
	start := time.Now().UTC()
	c := Check{Capability: Capabilities[i], Status: "unavailable", StartedAt: start, CostMicrorand: price(i), Compulsory: required}
	deferElapsed := func() Check { c.ElapsedMS = float64(time.Since(start)) / float64(time.Millisecond); return c }
	u := e.baseURL + "/capabilities/" + Capabilities[i] + "?scenario=" + url.QueryEscape(scenario) + "&at=" + url.QueryEscape(at.Format(time.RFC3339Nano))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return deferElapsed()
	}
	response, err := e.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			c.Status = "timeout"
		}
		return deferElapsed()
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case 429:
		c.Status = "rate_limited"
	case 403:
		c.Status = "permission_denied"
	case 408, 504:
		c.Status = "timeout"
	case 200:
		var observation Check
		if err := json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(&observation); err == nil {
			c.Status = observation.Status
			c.Value = observation.Value
			c.SourceTime = observation.SourceTime
			c.EventTime = observation.EventTime
			c.Subject = observation.Subject
			c.Scope = observation.Scope
			normalize(&c, i, at)
		}
	}
	return deferElapsed()
}
func normalize(c *Check, i int, at time.Time) {
	if !validName(c.Status, []string{"available", "stale", "timeout", "rate_limited", "permission_denied", "unavailable"}) {
		c.Status = "unavailable"
	}
	if c.Status != "available" {
		c.Value = nil
		return
	}
	if c.Subject != "synthetic-subject" || c.Scope != "ZA:account-verification" || c.SourceTime.IsZero() || c.SourceTime.After(at) {
		c.Status = "unavailable"
		c.Value = nil
		return
	}
	if at.Sub(c.SourceTime) > 60*time.Second {
		c.Status = "stale"
		c.Value = nil
		return
	}
	if c.Value == nil {
		c.Status = "unavailable"
		return
	}
	if i > 0 {
		if c.EventTime != nil {
			if c.EventTime.After(at) {
				c.Status = "unavailable"
				c.Value = nil
				return
			}
			c.Value = ptr(at.Sub(*c.EventTime) <= 24*time.Hour)
		} else if *c.Value {
			c.Status = "unavailable"
			c.Value = nil
		}
	}
}
func sourceCheck(i int, scenario string, at time.Time) Check {
	c := Check{Capability: Capabilities[i], Status: "available", Value: ptr(false), SourceTime: at.Add(-time.Second), Subject: "synthetic-subject", Scope: "ZA:account-verification"}
	if scenario == "number_mismatch" && i == 0 || scenario == "sim_change" && i == 1 || scenario == "device_change" && i == 2 || scenario == "all_adverse" || scenario == "correlated" && i > 0 {
		c.Value = ptr(true)
		if i > 0 {
			event := at.Add(-time.Hour)
			c.EventTime = &event
		}
	}
	if i == 1 {
		switch scenario {
		case "stale":
			c.SourceTime = at.Add(-61 * time.Second)
		case "timeout":
			c.Status = "timeout"
		case "rate_limit":
			c.Status = "rate_limited"
		case "permission_denied":
			c.Status = "permission_denied"
		case "source_failure":
			c.Status = "unavailable"
		}
	}
	return c
}

// SimulatorHandler exposes synthetic observations only, with deterministic source
// time anchored to the evaluation rather than the wall-clock response time.
func SimulatorHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"ok","data":"synthetic"}`))
	})
	for index, name := range Capabilities {
		i := index
		mux.HandleFunc("GET /capabilities/"+name, func(w http.ResponseWriter, r *http.Request) {
			scenario := r.URL.Query().Get("scenario")
			at, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("at"))
			if err != nil || !validName(scenario, Scenarios) {
				http.Error(w, "invalid synthetic request", 400)
				return
			}
			delay := latency(i)
			if scenario == "timeout" && i == 1 {
				delay = 250 * time.Millisecond
			}
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-r.Context().Done():
				return
			case <-timer.C:
			}
			c := sourceCheck(i, scenario, at)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Synthetic-Price-Microrand", strconv.FormatInt(price(i), 10))
			switch c.Status {
			case "rate_limited":
				w.WriteHeader(429)
			case "permission_denied":
				w.WriteHeader(403)
			case "unavailable":
				w.WriteHeader(503)
			case "timeout":
				w.WriteHeader(504)
			}
			_ = json.NewEncoder(w).Encode(c)
		})
	}
	return mux
}
