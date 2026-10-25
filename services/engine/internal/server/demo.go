package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
)

// Public demonstration requests never open a database transaction, accept a
// tenant/policy identifier, persist evidence, or enqueue background work.
type publicDemo struct {
	evaluate func(context.Context, evaluation.Request) (evaluation.Result, error)
	mu       sync.Mutex
	window   time.Time
	requests int
	active   chan struct{}
}

func newPublicDemo(engine *evaluation.Engine) http.Handler {
	return &publicDemo{evaluate: engine.Evaluate, active: make(chan struct{}, 2)}
}

func (d *publicDemo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	d.mu.Lock()
	if time.Since(d.window) >= time.Minute {
		d.window, d.requests = time.Now(), 0
	}
	limited := d.requests >= 30
	if !limited {
		d.requests++
	}
	d.mu.Unlock()
	if limited {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "demo_limit", "The shared demo limit is reached. Try again in one minute, or run locally.")
		return
	}
	select {
	case d.active <- struct{}{}:
		defer func() { <-d.active }()
	default:
		w.Header().Set("Retry-After", "5")
		fail(w, 429, "demo_busy", "The demo is busy. Try again in a few seconds.")
		return
	}
	var input struct {
		Profile  string `json:"profile"`
		Scenario string `json:"scenario"`
		Strategy string `json:"strategy"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if !decode(w, r, &input) {
		return
	}
	if !has([]string{"complete", "conditional"}, input.Profile) || !has(evaluation.Scenarios, input.Scenario) || !has(evaluation.Strategies, input.Strategy) {
		fail(w, 400, "invalid_demo", "Choose a supported profile, synthetic scenario and execution strategy")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	result, err := d.evaluate(ctx, evaluation.Request{
		Policy:   evaluation.Policy{ID: "demo-" + input.Profile, Version: 1, Profile: input.Profile},
		Scenario: input.Scenario, Strategy: input.Strategy,
	})
	if err != nil {
		fail(w, 503, "demo_unavailable", "The demo could not finish. Try again shortly.")
		return
	}
	// Flatten the actual engine result into the trace shape without tenant data.
	raw, _ := json.Marshal(result)
	var out map[string]any
	json.Unmarshal(raw, &out)
	out["id"], out["policy_id"] = "demo-"+identity.Random(), "demo-"+input.Profile
	out["profile"], out["policy_version"] = input.Profile, 1
	out["scenario"], out["strategy"] = input.Scenario, input.Strategy
	out["created_at"], out["status"] = time.Now().UTC(), "completed"
	out["persisted"], out["public_demo"] = false, true
	write(w, 200, out)
}
