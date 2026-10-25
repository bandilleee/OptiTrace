package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"github.com/optitrace/optitrace/services/engine/internal/storage/dbgen"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func (s *Server) policies(w http.ResponseWriter, r *http.Request, tx pgx.Tx, p principal, tid string) error {
	id, action := r.PathValue("id"), r.PathValue("action")
	if r.Method == "GET" {
		if id != "" {
			v, e := load(r.Context(), tx, "policy", id)
			if e == pgx.ErrNoRows {
				fail(w, 404, "not_found", "Policy not found")
				return nil
			}
			if e != nil {
				return e
			}
			write(w, 200, v)
			return nil
		}
		v, e := list(r.Context(), tx, "policy")
		if e != nil {
			return e
		}
		write(w, 200, map[string]any{"items": v})
		return nil
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return nil
	}
	if id == "" {
		var b struct {
			Name    string `json:"name"`
			Profile string `json:"profile"`
		}
		if !decode(w, r, &b) {
			return nil
		}
		b.Name = strings.TrimSpace(b.Name)
		if !resourceNameValid(b.Name) || b.Profile != "complete" && b.Profile != "conditional" {
			fail(w, 400, "invalid_policy", "Name and valid profile required")
			return nil
		}
		id = identity.Random()
		if _, e := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, tid+":"+b.Name); e != nil {
			return e
		}
		var version int
		if e := tx.QueryRow(r.Context(), `SELECT coalesce(max((data->>'version')::integer),0)+1 FROM resources WHERE kind='policy' AND data->>'name'=$1`, b.Name).Scan(&version); e != nil {
			return e
		}
		v := map[string]any{"id": id, "tenant_id": tid, "name": b.Name, "profile": b.Profile, "version": version, "status": "draft", "author_id": p.id, "created_at": time.Now().UTC()}
		if e := persist(r.Context(), tx, id, tid, "policy", v); e != nil {
			return e
		}
		if e := audit(r.Context(), tx, tid, p.id, "policy.created", id); e != nil {
			return e
		}
		return commitWrite(w, r, tx, 201, v)
	}
	if _, e := tx.Exec(r.Context(), `SELECT id FROM resources WHERE id=$1 AND kind='policy' FOR UPDATE`, id); e != nil {
		return e
	}
	v, e := load(r.Context(), tx, "policy", id)
	if e == pgx.ErrNoRows {
		fail(w, 404, "not_found", "Policy not found")
		return nil
	}
	if e != nil {
		return e
	}
	status := v["status"]
	switch action {
	case "submit":
		if status != "draft" || v["author_id"] != p.id {
			fail(w, 409, "invalid_transition", "Only the author may submit their draft")
			return nil
		}
		v["status"] = "submitted"
	case "approve":
		if status != "submitted" || v["author_id"] == p.id {
			fail(w, 409, "invalid_approval", "Submitted policy requires an independent reviewer")
			return nil
		}
		v["status"] = "approved"
		v["reviewer_id"] = p.id
	case "activate":
		if status != "approved" || v["author_id"] == p.id {
			fail(w, 409, "invalid_transition", "Independent reviewer may activate an approved policy")
			return nil
		}
		v["status"] = "active"
	default:
		fail(w, 404, "not_found", "Policy action not found")
		return nil
	}
	b, _ := json.Marshal(v)
	if _, e = tx.Exec(r.Context(), `UPDATE resources SET data=$2 WHERE id=$1`, id, b); e != nil {
		return e
	}
	if e = audit(r.Context(), tx, tid, p.id, "policy."+action, id); e != nil {
		return e
	}
	return commitWrite(w, r, tx, 200, v)
}
func (s *Server) evaluations(w http.ResponseWriter, r *http.Request, tx pgx.Tx, p principal, tid string) error {
	id := r.PathValue("id")
	if r.Method == "GET" {
		if id != "" {
			v, e := load(r.Context(), tx, "evaluation", id)
			if e == pgx.ErrNoRows {
				v, e = load(r.Context(), tx, "evaluation_attempt", id)
			}
			if e == pgx.ErrNoRows {
				fail(w, 404, "not_found", "Evaluation not found")
				return nil
			}
			if e != nil {
				return e
			}
			if e = storage.EnrichEvaluationJournals(r.Context(), tx, []map[string]any{v}); e != nil {
				return e
			}
			write(w, 200, v)
			return nil
		}
		rows, e := dbgen.New(tx).ListEvaluations(r.Context())
		if e != nil {
			return e
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			var item map[string]any
			if e = json.Unmarshal(row, &item); e != nil {
				return e
			}
			items = append(items, item)
		}
		if e = storage.EnrichEvaluationJournals(r.Context(), tx, items); e != nil {
			return e
		}
		v := []json.RawMessage{}
		for _, item := range items {
			row, marshalErr := json.Marshal(item)
			if marshalErr != nil {
				return marshalErr
			}
			v = append(v, json.RawMessage(row))
		}
		write(w, 200, map[string]any{"items": v})
		return nil
	}
	if r.Method != "POST" || id != "" {
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return nil
	}
	var b struct {
		PolicyID string `json:"policy_id"`
		Scenario string `json:"scenario"`
		Strategy string `json:"strategy"`
	}
	if !decode(w, r, &b) {
		return nil
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 1 || len(key) > 200 {
		fail(w, 400, "idempotency_required", "Idempotency-Key required (max 200 characters)")
		return nil
	}
	if b.Strategy == "" {
		b.Strategy = "optitrace"
	}
	if !has([]string{"clear", "number_mismatch", "sim_change", "device_change", "all_adverse", "stale", "timeout", "rate_limit", "permission_denied", "source_failure", "correlated"}, b.Scenario) || !has([]string{"optitrace", "full_sequential", "full_parallel", "staged", "opa"}, b.Strategy) {
		fail(w, 400, "invalid_evaluation", "Unknown synthetic scenario or strategy")
		return nil
	}
	raw, _ := json.Marshal(b)
	hash := identity.Hash(string(raw))
	queries := dbgen.New(tx)
	reserved, e := queries.ReserveIdempotency(r.Context(), dbgen.ReserveIdempotencyParams{TenantID: tid, Key: key, RequestHash: hash})
	if e != nil {
		return e
	}
	reservation, e := queries.GetIdempotency(r.Context(), dbgen.GetIdempotencyParams{TenantID: tid, Key: key})
	if e != nil {
		return e
	}
	previous, response := reservation.RequestHash, reservation.Response
	if previous != hash {
		fail(w, 409, "idempotency_conflict", "Key already used for a different request")
		return nil
	}
	if len(response) > 0 {
		write(w, 200, json.RawMessage(response))
		return nil
	}
	if reserved == 0 {
		fail(w, 409, "execution_unconfirmed", "This request already started; inspect its execution trace. It will not be automatically repeated.")
		return nil
	}
	policy, e := load(r.Context(), tx, "policy", b.PolicyID)
	if e == pgx.ErrNoRows {
		fail(w, 404, "not_found", "Policy not found")
		return nil
	}
	if e != nil {
		return e
	}
	if policy["status"] != "active" {
		fail(w, 409, "inactive_policy", "Policy is not active")
		return nil
	}
	id = identity.Random()
	created := time.Now().UTC()
	attempt := map[string]any{"id": id, "tenant_id": tid, "policy_id": b.PolicyID, "policy_version": policy["version"], "profile": policy["profile"], "scenario": b.Scenario, "strategy": b.Strategy, "outcome": "indeterminate", "status": "execution_unconfirmed", "stopping_reason": "Execution reserved. Until finalized, call completion and charges are uncertain; this key is never automatically repeated.", "checks": []any{}, "journal": []any{}, "supplier_cost_microrand": nil, "runtime_cost_microrand": nil, "elapsed_ms": nil, "created_at": created}
	if e = persist(r.Context(), tx, id, tid, "evaluation_attempt", attempt); e != nil {
		return e
	}
	if e = audit(r.Context(), tx, tid, p.id, "evaluation.reserved", id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	journal := func(ctx context.Context, event string, check evaluation.Check) error {
		raw, e := json.Marshal(check)
		if e != nil {
			return e
		}
		e = storage.AppendEvaluationEvent(ctx, s.DB, tid, id, check.Capability, event, raw)
		if e != nil {
			// Never put provider tokens, database URLs or evidence values in logs.
			slog.Error("evaluation journal persistence failed", "evaluation_id", id, "phase", event, "capability", check.Capability, "error_type", fmt.Sprintf("%T", e), "context_error", ctx.Err())
		}
		return e
	}
	result, e := s.Engine.Evaluate(r.Context(), evaluation.Request{Policy: evaluation.Policy{ID: b.PolicyID, Profile: policy["profile"].(string), Version: int(policy["version"].(float64))}, Scenario: b.Scenario, Strategy: b.Strategy, Journal: journal})
	if e != nil {
		fail(w, 502, "evaluation_failed", "Adapter execution failed")
		return nil
	}
	tx, e = storage.Tenant(r.Context(), s.DB, tid)
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	queries = dbgen.New(tx)
	out := map[string]any{"id": id, "tenant_id": tid, "policy_id": b.PolicyID, "policy_version": policy["version"], "profile": policy["profile"], "scenario": b.Scenario, "strategy": b.Strategy, "status": "completed", "outcome": result.Outcome, "stopping_reason": result.StoppingReason, "checks": result.Checks, "supplier_cost_microrand": result.SupplierCostMicrorand, "runtime_cost_microrand": result.RuntimeCostMicrorand, "elapsed_ms": result.ElapsedMS, "created_at": created}
	raw, _ = json.Marshal(out)
	if _, e = queries.FinalizeEvaluation(r.Context(), dbgen.FinalizeEvaluationParams{ID: id, Data: raw}); e != nil {
		return e
	}
	raw, _ = json.Marshal(out)
	if e = queries.CompleteIdempotency(r.Context(), dbgen.CompleteIdempotencyParams{TenantID: tid, Key: key, Response: raw}); e != nil {
		return e
	}
	if e = audit(r.Context(), tx, tid, p.id, "evaluation.executed", id); e != nil {
		return e
	}
	return commitWrite(w, r, tx, 201, out)
}
func (s *Server) benchmarks(w http.ResponseWriter, r *http.Request, tx pgx.Tx, p principal, tid string) error {
	if r.Method == "POST" && r.PathValue("id") == "" {
		var b struct {
			Seed  int64 `json:"seed"`
			Count int   `json:"count"`
		}
		if !decode(w, r, &b) {
			return nil
		}
		if b.Count == 0 {
			b.Count = 50
		}
		if b.Seed == 0 {
			b.Seed = 42
		}
		if b.Count < 2 || b.Count > 1000 {
			fail(w, 400, "invalid_count", "Count must be between 2 and 1000")
			return nil
		}
		if os.Getenv("RUNTIME_PROFILE") == "portfolio" && b.Count > 100 {
			fail(w, 400, "portfolio_benchmark_limit", "Portfolio benchmarks accept at most 100 observations")
			return nil
		}
		if _, e := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, tid+":benchmark"); e != nil {
			return e
		}
		var queued int
		if e := tx.QueryRow(r.Context(), `SELECT count(*) FROM jobs WHERE tenant_id=$1 AND status IN ('queued','running')`, tid).Scan(&queued); e != nil {
			return e
		}
		if queued >= 3 {
			fail(w, 429, "benchmark_limit", "At most three queued/running benchmarks per tenant")
			return nil
		}
		id := identity.Random()
		_, e := tx.Exec(r.Context(), `INSERT INTO jobs(id,tenant_id,seed,count) VALUES($1,$2,$3,$4)`, id, tid, b.Seed, b.Count)
		if e != nil {
			return e
		}
		if e = audit(r.Context(), tx, tid, p.id, "benchmark.queued", id); e != nil {
			return e
		}
		if e = commitWrite(w, r, tx, 202, map[string]any{"id": id, "status": "queued", "seed": b.Seed, "count": b.Count, "created_at": time.Now().UTC()}); e != nil {
			return e
		}
		go launchBenchmark()
		return nil
	}
	if r.Method != "GET" {
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return nil
	}
	rows, e := tx.Query(r.Context(), `SELECT id,status,seed,count,created_at,result,coalesce(error,'') FROM jobs WHERE tenant_id=$1 AND ($2='' OR id=$2) ORDER BY created_at DESC LIMIT 100`, tid, r.PathValue("id"))
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, status, err string
		var seed int64
		var count int
		var created time.Time
		var result []byte
		if e = rows.Scan(&id, &status, &seed, &count, &created, &result, &err); e != nil {
			return e
		}
		v := map[string]any{"id": id, "status": status, "seed": seed, "count": count, "created_at": created}
		if len(result) > 0 {
			var report map[string]any
			if e = json.Unmarshal(result, &report); e != nil {
				return e
			}
			v["report"] = report
			v["results"] = report["results"]
		}
		if err != "" {
			v["error"] = err
		}
		items = append(items, v)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if r.PathValue("id") != "" {
		if len(items) == 0 {
			fail(w, 404, "not_found", "Benchmark not found")
			return nil
		}
		if r.PathValue("action") == "export" {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="benchmark-%s.json"`, r.PathValue("id")))
		}
		write(w, 200, items[0])
	} else {
		write(w, 200, map[string]any{"items": items})
	}
	return nil
}
