package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"

	"sync"
	"testing"
	"time"
)

func TestPostgresAuthenticatedWorkflow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL required; integration test uses a dedicated disposable database")
	}
	ctx := context.Background()
	owner, e := storage.Open(ctx, databaseURL)
	if e != nil {
		t.Fatal(e)
	}
	defer owner.Close()
	if e = storage.Migrate(ctx, owner); e != nil {
		t.Fatal(e)
	}
	if e = storage.Seed(ctx, owner); e != nil {
		t.Fatal(e)
	}
	if _, e = owner.Exec(ctx, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='optitrace_test_login') THEN CREATE ROLE optitrace_test_login LOGIN PASSWORD 'test-only'; END IF; END $$; GRANT optitrace_app TO optitrace_test_login`); e != nil {
		t.Fatal(e)
	}
	parsed, e := url.Parse(databaseURL)
	if e != nil {
		t.Fatal(e)
	}
	parsed.User = url.UserPassword("optitrace_test_login", "test-only")
	appURL := parsed.String()
	app, e := storage.Open(ctx, appURL)
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	var mock *identity.Mock
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mock.Handler().ServeHTTP(w, r) }))
	defer providerServer.Close()
	mock, e = identity.NewMock(providerServer.URL)
	if e != nil {
		t.Fatal(e)
	}
	sim := httptest.NewServer(evaluation.SimulatorHandler())
	defer sim.Close()
	eng, e := evaluation.New(sim.URL)
	if e != nil {
		t.Fatal(e)
	}
	s := New(app, eng, nil)
	api := httptest.NewServer(s.Handler())
	defer api.Close()
	t.Setenv("AUTH_MODE", "mock")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_ORIGIN", api.URL)
	t.Setenv("API_PUBLIC_URL", api.URL)
	t.Setenv("MOCK_ISSUER", providerServer.URL)
	t.Setenv("MOCK_INTERNAL_URL", "")
	s.Origin = api.URL
	s.Identity, e = identity.New(ctx)
	if e != nil {
		t.Fatal(e)
	}
	loginAs := func(name string) (*http.Client, string) {
		t.Helper()
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := client.Get(api.URL + "/v1/auth/login")
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		res, e = client.Get(res.Header.Get("Location") + "&email=" + name + "%40optitrace.test")
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		res, e = client.Get(res.Header.Get("Location"))
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 302 {
			t.Fatalf("login %s: %d %s", name, res.StatusCode, b)
		}
		res, e = client.Get(api.URL + "/v1/auth/me")
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		var me map[string]any
		json.NewDecoder(res.Body).Decode(&me)
		csrf, _ := me["csrf_token"].(string)
		if csrf == "" {
			t.Fatal(me)
		}
		return client, csrf
	}
	request := func(c *http.Client, csrf, method, path, body string, want int) map[string]any {
		t.Helper()
		req, _ := http.NewRequest(method, api.URL+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", api.URL)
		req.Header.Set("X-CSRF-Token", csrf)
		req.Header.Set("Idempotency-Key", "test-"+identity.Random())
		res, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s => %d want %d: %s", method, path, res.StatusCode, want, raw)
		}
		v := map[string]any{}
		json.Unmarshal(raw, &v)
		return v
	}
	admin, adminCSRF := loginAs("admin")
	request(admin, adminCSRF, "GET", "/v1/tenants/other/policies", "", 403)
	request(admin, "wrong", "POST", "/v1/tenants/demo/invitations", `{"email":"x@optitrace.test","roles":["auditor"]}`, 403)
	ev := request(admin, adminCSRF, "POST", "/v1/tenants/demo/evaluations", `{"policy_id":"policy-conditional","scenario":"number_mismatch"}`, 201)
	if ev["outcome"] != "additional_verification_required" {
		t.Fatal(ev)
	}
	author, authorCSRF := loginAs("author")
	policy := request(author, authorCSRF, "POST", "/v1/tenants/demo/policies", `{"name":"Independent approval","profile":"conditional"}`, 201)
	pid := policy["id"].(string)
	request(author, authorCSRF, "POST", "/v1/tenants/demo/policies/"+pid+"/submit", `{}`, 200)
	request(author, authorCSRF, "POST", "/v1/tenants/demo/policies/"+pid+"/approve", `{}`, 403)
	reviewer, reviewerCSRF := loginAs("reviewer")
	request(reviewer, reviewerCSRF, "POST", "/v1/tenants/demo/policies/"+pid+"/approve", `{}`, 200)
	request(reviewer, reviewerCSRF, "POST", "/v1/tenants/demo/policies/"+pid+"/activate", `{}`, 200)
	auditor, auditorCSRF := loginAs("auditor")
	request(auditor, auditorCSRF, "POST", "/v1/tenants/demo/evaluations", `{"policy_id":"policy-complete","scenario":"clear"}`, 403)
	key := request(admin, adminCSRF, "POST", "/v1/tenants/demo/credentials", `{"name":"test","scopes":["evaluations:read"]}`, 201)
	req, _ := http.NewRequest("GET", api.URL+"/v1/tenants/demo/evaluations", nil)
	req.Header.Set("Authorization", "Bearer "+key["secret"].(string))
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.StatusCode)
	}
	job := request(admin, adminCSRF, "POST", "/v1/tenants/demo/benchmarks", `{"seed":42,"count":2}`, 202)
	worked, e := RunJob(ctx, app, eng)
	if e != nil || !worked {
		t.Fatalf("worker %v %v", worked, e)
	}
	report := request(admin, adminCSRF, "GET", fmt.Sprintf("/v1/tenants/demo/benchmarks/%s/export", job["id"]), "", 200)
	if report["status"] != "completed" {
		t.Fatal(report)
	}
	tx, e := storage.Tenant(ctx, app, "other")
	if e != nil {
		t.Fatal(e)
	}
	var count int
	e = tx.QueryRow(ctx, `SELECT count(*) FROM resources WHERE tenant_id='demo'`).Scan(&count)
	tx.Rollback(ctx)
	if e != nil || count != 0 {
		t.Fatalf("RLS failed: %d %v", count, e)
	}
	tx, e = storage.Tenant(ctx, app, "demo")
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, `UPDATE resources SET data=jsonb_set(data,'{name}','"tampered"') WHERE id='policy-complete'`)
	tx.Rollback(ctx)
	if e == nil {
		t.Fatal("approved policy mutable")
	}
	// Approved versions cannot be downgraded and then edited through direct SQL.
	approvedID := "guard-" + identity.Random()
	tx, e = storage.Tenant(ctx, app, "demo")
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, `INSERT INTO resources(id,tenant_id,kind,data) VALUES($1,'demo','policy','{"status":"approved","author_id":"a","reviewer_id":"b"}')`, approvedID)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	for _, mutation := range []string{
		`data=jsonb_set(data,'{status}','"draft"')`,
		`data=data-'status'`,
		`kind='other'`,
		`id=id||'-changed'`,
		`data=jsonb_set(data,'{author_id}','"b"')`,
	} {
		tx, e = storage.Tenant(ctx, app, "demo")
		if e != nil {
			t.Fatal(e)
		}
		_, e = tx.Exec(ctx, "UPDATE resources SET "+mutation+" WHERE id=$1", approvedID)
		tx.Rollback(ctx)
		if e == nil {
			t.Fatalf("approved policy accepted mutation: %s", mutation)
		}
	}
	tx, e = storage.Tenant(ctx, app, "demo")
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, `INSERT INTO resources(id,tenant_id,kind,data) VALUES($1,'demo','policy','{"status":"active","author_id":"a","reviewer_id":"a"}')`, "invalid-"+identity.Random())
	tx.Rollback(ctx)
	if e == nil {
		t.Fatal("self-approved policy insert accepted")
	}
	// A disconnected caller must not erase a started billable execution trace.
	started := make(chan struct{})
	var startedOnce sync.Once
	slowSimulator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedOnce.Do(func() { close(started) })
		time.Sleep(40 * time.Millisecond)
		evaluation.SimulatorHandler().ServeHTTP(w, r)
	}))
	defer slowSimulator.Close()
	slowEngine, e := evaluation.New(slowSimulator.URL)
	if e != nil {
		t.Fatal(e)
	}
	detachedServer := New(app, slowEngine, s.Identity)
	detachedServer.Origin = api.URL
	detachedAPI := httptest.NewServer(detachedServer.Handler())
	defer detachedAPI.Close()
	requestCtx, disconnect := context.WithCancel(ctx)
	req, _ = http.NewRequestWithContext(requestCtx, "POST", detachedAPI.URL+"/v1/tenants/demo/evaluations", bytes.NewBufferString(`{"policy_id":"policy-complete","scenario":"clear"}`))
	req.Header.Set("Origin", api.URL)
	req.Header.Set("X-CSRF-Token", adminCSRF)
	req.Header.Set("Content-Type", "application/json")
	durableKey := "disconnect-" + identity.Random()
	req.Header.Set("Idempotency-Key", durableKey)
	done := make(chan struct{})
	go func() {
		defer close(done)
		res, err := admin.Do(req)
		if err == nil {
			res.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("adapter never started")
	}
	disconnect()
	<-done
	var persisted bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		e = owner.QueryRow(ctx, `SELECT response IS NOT NULL FROM idempotency WHERE tenant_id='demo' AND key=$1`, durableKey).Scan(&persisted)
		if e == nil && persisted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !persisted {
		t.Fatal("disconnected evaluation trace was lost")
	}
	// A process can stop after committing its reservation but before finalizing.
	// Reusing that key must never repeat potentially billable source work.
	abandonedKey := "abandoned-" + identity.Random()
	abandonedID := "attempt-" + identity.Random()
	reservationTx, err := storage.Tenant(ctx, app, "demo")
	if err != nil {
		t.Fatal(err)
	}
	abandonedHash := identity.Hash(`{"policy_id":"policy-complete","scenario":"clear","strategy":"optitrace"}`)
	if _, err = reservationTx.Exec(ctx, `INSERT INTO idempotency(tenant_id,key,request_hash) VALUES('demo',$1,$2)`, abandonedKey, abandonedHash); err != nil {
		t.Fatal(err)
	}
	if err = persist(ctx, reservationTx, abandonedID, "demo", "evaluation_attempt", map[string]any{"id": abandonedID, "status": "execution_unconfirmed", "supplier_cost_microrand": nil, "outcome": "indeterminate"}); err != nil {
		t.Fatal(err)
	}
	if err = reservationTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	replay, _ := http.NewRequest("POST", api.URL+"/v1/tenants/demo/evaluations", bytes.NewBufferString(`{"policy_id":"policy-complete","scenario":"clear","strategy":"optitrace"}`))
	replay.Header.Set("Content-Type", "application/json")
	replay.Header.Set("Origin", api.URL)
	replay.Header.Set("X-CSRF-Token", adminCSRF)
	replay.Header.Set("Idempotency-Key", abandonedKey)
	replayResponse, err := admin.Do(replay)
	if err != nil {
		t.Fatal(err)
	}
	replayBody, _ := io.ReadAll(replayResponse.Body)
	replayResponse.Body.Close()
	if replayResponse.StatusCode != 409 || !bytes.Contains(replayBody, []byte("execution_unconfirmed")) {
		t.Fatalf("abandoned execution was repeated: %d %s", replayResponse.StatusCode, replayBody)
	}
	abandonedTrace := request(admin, adminCSRF, "GET", "/v1/tenants/demo/evaluations/"+abandonedID, "", 200)
	if abandonedTrace["status"] != "execution_unconfirmed" || abandonedTrace["supplier_cost_microrand"] != nil {
		t.Fatalf("uncertain billing fabricated: %v", abandonedTrace)
	}
	mock.SetDisabled("seed-auditor", true)
	if _, e = owner.Exec(ctx, `UPDATE sessions SET provider_checked=now()-interval '6 minutes' WHERE user_id='seed-auditor'`); e != nil {
		t.Fatal(e)
	}
	request(auditor, auditorCSRF, "GET", "/v1/auth/me", "", 401)
	mock.SetDisabled("seed-auditor", false)
	if _, e = owner.Exec(ctx, `UPDATE sessions SET expires_at=now()-interval '1 minute' WHERE user_id='seed-reviewer'`); e != nil {
		t.Fatal(e)
	}
	request(reviewer, reviewerCSRF, "GET", "/v1/auth/me", "", 401)
	request(admin, adminCSRF, "POST", "/v1/auth/logout", `{}`, 204)
	request(admin, adminCSRF, "GET", "/v1/auth/me", "", 401)
}
